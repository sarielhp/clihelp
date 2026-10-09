package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hintFolders is a completer in the shape that asked for hints: labels are
// written %name, and a bare word gets a hint instead of a list of the
// deprecated spelling.
func hintFolders(toComplete string) []string {
	if toComplete != "" && !strings.HasPrefix(toComplete, "%") {
		return []string{CompletionHint("a folder is named with %, as in %" + toComplete)}
	}
	var out []string
	for _, f := range []string{"%wuna", "%work"} {
		if strings.HasPrefix(f, toComplete) {
			out = append(out, f+"\tfolder")
		}
	}
	return out
}

func hintApp() *App {
	var folder string
	return &App{Name: "hintcli", Commands: []Command{{
		Name:        "open",
		Description: "Open a folder.",
		Args:        MaximumNArgs(1),
		Parameters:  []Param{{Name: "[folder]", Description: "The folder.", Complete: hintFolders}},
		Options:     []Option{String(&folder, "--folder <name>", "", "A folder.")},
		Run:         nopRun,
	}}}
}

// A script that does not read hint records — one written by hand, or generated
// before hints existed — would offer "__hint__" as a candidate, so it is not
// sent one: fish's autoload found exactly such a script in a real setup and
// inserted the word.
func TestCompletionHintGoesOnlyToScriptsThatAskForIt(t *testing.T) {
	t.Setenv(hintsEnv, "")
	res := testExecute(hintApp(), []string{"__complete", "open", "wu"})
	res.AssertNoError(t)
	if res.Stdout != "" {
		t.Errorf("a caller that did not ask for hints got %q", res.Stdout)
	}
}

func TestCompletionHintIsAProtocolRecord(t *testing.T) {
	t.Setenv(hintsEnv, "1")
	app := hintApp()
	app.Commands[0].Options[0].Complete = hintFolders
	long := strings.Repeat("long ", 30) + "end. And a second sentence."
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"positional", []string{"open", "wu"}, "__hint__\ta folder is named with %, as in %wu\n"},
		{"flag value", []string{"open", "--folder", "wu"}, "__hint__\ta folder is named with %, as in %wu\n"},
		{"inline flag value", []string{"open", "--folder=wu"}, "__hint__\ta folder is named with %, as in %wu\n"},
		{"candidates instead", []string{"open", "%w"}, "%wuna\tfolder\n%work\tfolder\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := testExecute(app, append([]string{"__complete"}, tt.args...))
			res.AssertNoError(t)
			if res.Stdout != tt.want {
				t.Errorf("__complete %q = %q, want %q", tt.args, res.Stdout, tt.want)
			}
		})
	}

	// A description is cut to its first sentence and a menu's width; a hint is
	// not in a menu, so it keeps its text — but not a newline that would forge
	// a second record.
	app.Commands[0].Parameters[0].Complete = func(string) []string {
		return []string{CompletionHint(long + "\ninjected\tline")}
	}
	res := testExecute(app, []string{"__complete", "open", "x"})
	res.AssertNoError(t)
	if want := "__hint__\t" + long + " injected line\n"; res.Stdout != want {
		t.Errorf("hint record = %q, want %q", res.Stdout, want)
	}
}

// buildHintCLI builds a program around hintApp's completer and writes its
// completion script for shell into the same directory.
func buildHintCLI(t *testing.T, shell string) (dir, script string) {
	t.Helper()
	if _, err := exec.LookPath(shell); err != nil {
		skipMissingToolf(t, "%s not found, skipping", shell)
	}
	dir = t.TempDir()
	src := `package main

import (
	"os"
	"strings"

	"github.com/sarielhp/clihelp"
)

func main() {
	folders := func(toComplete string) []string {
		if toComplete != "" && !strings.HasPrefix(toComplete, "%") {
			return []string{clihelp.CompletionHint("a folder is named with %, as in %" + toComplete)}
		}
		var out []string
		for _, f := range []string{"%wuna", "%work"} {
			if strings.HasPrefix(f, toComplete) {
				out = append(out, f+"\tfolder")
			}
		}
		return out
	}
	app := &clihelp.App{Name: "hintcli", Commands: []clihelp.Command{
		{Name: "open", Description: "Open a folder.", Args: clihelp.MaximumNArgs(1),
			Parameters: []clihelp.Param{{Name: "[folder]", Description: "The folder.", Complete: folders}},
			Run:        func(*clihelp.Context) error { return nil }},
		clihelp.CompletionCommand(),
	}}
	_ = app.Execute(os.Args[1:])
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hintcli")
	if out, err := exec.Command("go", "build", "-o", bin, filepath.Join(dir, "main.go")).CombinedOutput(); err != nil {
		t.Fatalf("building the hint CLI failed: %v\n%s", err, out)
	}
	body, err := sandboxedCommand(t, bin, "completion", shell).Output()
	if err != nil {
		t.Fatalf("generating the %s script failed: %v", shell, err)
	}
	script = filepath.Join(dir, "hintcli."+shell)
	if err := os.WriteFile(script, body, 0o600); err != nil {
		t.Fatal(err)
	}
	// A file the files fallback would offer for "wu": a hint must replace it.
	if err := os.WriteFile(filepath.Join(dir, "wufile"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, script
}

// runIn runs script in shell from dir, with dir on $PATH.
func runIn(t *testing.T, dir, shell string, args ...string) string {
	t.Helper()
	path, _ := exec.LookPath(shell)
	cmd := sandboxedCommand(t, path, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", shell, err, out)
	}
	return string(out)
}

// The hint reaches the person, takes the place of the files fallback, and
// stays out of the way when there are candidates to show.
func TestLiveCompletionHint(t *testing.T) {
	t.Run("bash", func(t *testing.T) {
		dir, script := buildHintCLI(t, "bash")
		// COMP_TYPE is set by readline; the hint is drawn only for it.
		complete := func(word string) string {
			line := "hintcli open " + word
			return runIn(t, dir, "bash", "--norc", "--noprofile", "-c", `
PS1='$ '
source `+script+`
COMP_WORDS=(hintcli open '`+word+`'); COMP_CWORD=2
COMP_LINE='`+line+`'; COMP_POINT=${#COMP_LINE}; COMP_TYPE=9
_hintcli_complete
echo "CANDIDATES: ${COMPREPLY[*]}"
`)
		}
		out := complete("wu")
		wantLines(t, out, "a folder is named with %, as in %wu", "$ hintcli open wu", "CANDIDATES: \n")
		out = complete("%w")
		wantLines(t, out, "CANDIDATES: %wuna %work")
		if strings.Contains(out, "a folder is named") {
			t.Errorf("the hint was shown beside candidates:\n%s", out)
		}
	})

	t.Run("zsh", func(t *testing.T) {
		dir, script := buildHintCLI(t, "zsh")
		complete := func(word string) string {
			return runIn(t, dir, "zsh", "-f", "-c", `
_describe() { shift 3; local n; for n in "$@"; do print -r -- "CANDIDATE: ${${(P)n}[@]}"; done }
compadd() { [[ $1 == -a ]] && print -r -- "CANDIDATE: ${${(P)2}[@]}"; return 0 }
_message() { print -r -- "MESSAGE: $2" }
_files() { print -r -- "FILES" }
source `+script+`
words=(hintcli open '`+word+`'); CURRENT=3
_hintcli
`)
		}
		out := complete("wu")
		// The % is doubled for zsh's prompt expansion, which turns it back.
		wantLines(t, out, "MESSAGE: a folder is named with %%, as in %%wu")
		if strings.Contains(out, "FILES") {
			t.Errorf("the files fallback ran beside the hint:\n%s", out)
		}
		out = complete("%w")
		wantLines(t, out, "CANDIDATE: %wuna:folder %work:folder")
		if strings.Contains(out, "MESSAGE") {
			t.Errorf("the hint was shown beside candidates:\n%s", out)
		}
	})

	t.Run("fish", func(t *testing.T) {
		dir, script := buildHintCLI(t, "fish")
		// Scripted completion has no prompt to draw a hint under; what matters
		// here is that the hint is never a candidate, and that it stops the
		// files fallback.
		out := runIn(t, dir, "fish", "--no-config", "-c", `
source `+script+`
set -l bare (complete -C "hintcli open wu")
set -l sigil (complete -C "hintcli open %w")
echo "BARE: [$bare]"
echo "SIGIL: [$sigil]"
`)
		// fish sorts the candidates.
		wantLines(t, out, "BARE: []", "SIGIL: [%work\tfolder %wuna\tfolder]")
		if strings.Contains(out, "__hint__") || strings.Contains(out, "wufile") {
			t.Errorf("the hint leaked into the candidates, or files were offered:\n%s", out)
		}
	})
}
