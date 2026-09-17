package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The completion protocol is one candidate per line, name and description
// separated by a tab. A newline inside either field forges a record, and the
// bash consumer expands whatever it is handed.
func TestCompletionRecordsAreOneLinePerCandidate(t *testing.T) {
	app := &App{
		Name: "myapp",
		Commands: []Command{
			{Name: "deploy", Description: "ship it\n$(touch /tmp/clihelp-pwned)\nstatus\tshow status",
				Run: func(*Context) error { return nil }},
		},
	}
	res := TestExecute(app, []string{"__complete", ""})
	res.AssertNoError(t)

	lines := strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("one command produced %d completion records:\n%q", len(lines), res.Stdout)
	}
	if strings.Count(lines[0], "\t") != 1 {
		t.Errorf("record has %d tabs, want 1: %q", strings.Count(lines[0], "\t"), lines[0])
	}
}

// A completion candidate is data, never code: the generated bash script must not
// expand it. The payload here is what an Option.Complete callback would return
// if it listed a file, a branch or an object whose name contains shell syntax.
func TestBashCompletionDoesNotExpandCandidates(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found on system, skipping live bash completion test")
	}

	tmpDir := t.TempDir()
	marker := filepath.Join(tmpDir, "PWNED")
	srcPath := filepath.Join(tmpDir, "main.go")
	binPath := filepath.Join(tmpDir, "injcli")

	code := fmt.Sprintf(`package main

import (
	"os"

	"github.com/sarielhp/clihelp"
)

func main() {
	app := &clihelp.App{
		Name: "injcli",
		Commands: []clihelp.Command{
			{
				Name: "play",
				Options: []clihelp.Option{
					{
						Flags:       "-p, --pod <id>",
						Description: "Pod name",
						Complete: func(string) []string {
							return []string{%q, "safe-candidate"}
						},
					},
				},
			},
			clihelp.CompletionCommand(),
		},
	}
	_ = app.Execute(os.Args[1:])
}
`, "$(touch "+marker+")")

	if err := os.WriteFile(srcPath, []byte(code), 0600); err != nil {
		t.Fatalf("failed to write test code: %v", err)
	}
	if out, err := exec.Command("go", "build", "-o", binPath, srcPath).CombinedOutput(); err != nil {
		t.Fatalf("failed to build test CLI: %v, output: %s", err, out)
	}
	scriptBytes, err := exec.Command(binPath, "completion", "bash").Output()
	if err != nil {
		t.Fatalf("failed to generate bash completion script: %v", err)
	}
	scriptPath := filepath.Join(tmpDir, "injcli.bash")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write completion script: %v", err)
	}

	bashScript := fmt.Sprintf(`
source %q
COMP_WORDS=("injcli" "play" "-p" "")
COMP_CWORD=3
_injcli_complete
for r in "${COMPREPLY[@]}"; do
    echo "$r"
done
`, scriptPath)

	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", bashScript)
	cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash completion failed: %v, output: %s", err, out)
	}

	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatalf("completion executed the candidate: %s was created\ncompletions: %s", marker, out)
	}
	if !strings.Contains(string(out), "safe-candidate") {
		t.Errorf("expected the legitimate candidate in the reply, got: %s", out)
	}
}

// zsh's _call_program ends in `eval $clocale $prefix "$argv[2,-1]"`, so anything
// spliced into its arguments is re-parsed as shell code. $words holds the typed
// command line verbatim, so an unquoted splice executes whatever the user has on
// the line the moment they press Tab. This is the 2026-09-17 audit's E1 in the
// shell that fix did not cover.
func TestZshCompletionDoesNotEvaluateTheTypedLine(t *testing.T) {
	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not found, skipping")
	}

	dir := t.TempDir()
	marker := filepath.Join(dir, "EXECUTED")

	var script strings.Builder
	if err := GenZshCompletion(&App{Name: "pd"}, &script); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "_pd")
	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	// A stand-in for the program: the completion protocol is not what is under
	// test here, only whether the typed line reaches an eval.
	stub := "#!/bin/sh\nprintf 'build\\tBuild\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "pd"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}

	driver := fmt.Sprintf(`
zmodload zsh/zpty
zpty -b z zsh -f -i
zpty -w z 'PATH=%[1]s:$PATH; autoload -Uz compinit; compinit -u -d %[1]s/zcompdump'
zpty -w z 'bindkey "^I" complete-word'
zpty -w z 'source %[2]s'
zpty -n -w z 'pd $(touch %[3]s) '$'\t'
sleep 2
zpty -d z
`, dir, scriptPath, marker)

	cmd := exec.Command(zshPath, "-f", "-c", driver)
	cmd.Env = append(os.Environ(), "HOME="+dir)
	_ = cmd.Run() // the driver's own exit status is not the assertion

	if _, err := os.Stat(marker); err == nil {
		t.Errorf("pressing Tab executed a command substitution from the typed line")
	}
}
