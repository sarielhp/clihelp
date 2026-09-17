package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildExplainFixture builds the example CLI and writes the key-binding snippet
// for shell into a temporary directory, returning both paths.
func buildExplainFixture(t *testing.T, shell string) (binDir, snippet string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}
	out, err := sandboxedCommand(t, bin, "completion", "keys", shell).Output()
	if err != nil {
		t.Fatalf("failed to generate %s key bindings: %v", shell, err)
	}
	snippet = filepath.Join(dir, "keys."+shell)
	if err := os.WriteFile(snippet, out, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, snippet
}

// Every snippet must at least parse in the shell it is written for; a syntax
// error in an rc file is a broken shell, not a broken feature.
func TestKeyBindingSnippetsParse(t *testing.T) {
	for _, tt := range []struct {
		shell string
		args  []string
	}{
		{"bash", []string{"-n"}},
		{"zsh", []string{"-n"}},
		{"fish", []string{"--no-execute"}},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			shellPath, err := exec.LookPath(tt.shell)
			if err != nil {
				t.Skipf("%s not found, skipping", tt.shell)
			}
			_, snippet := buildExplainFixture(t, tt.shell)
			if out, err := exec.Command(shellPath, append(tt.args, snippet)...).CombinedOutput(); err != nil {
				t.Errorf("%s rejected its own snippet: %v\n%s", tt.shell, err, out)
			}
		})
	}
}

// The bash binding is an ordinary function around READLINE_LINE, so it can be
// driven without a terminal: this is the whole Alt-H behavior, end to end.
func TestLiveBashAltHExpandsAndExplains(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found, skipping")
	}
	binDir, snippet := buildExplainFixture(t, "bash")

	script := fmt.Sprintf(`
source %q 2>/dev/null
LINES=24
COLUMNS=78
READLINE_LINE='podctl b'
READLINE_POINT=${#READLINE_LINE}
_clihelp_explain
echo "EXPANDED:$READLINE_LINE"
echo "POINT:$READLINE_POINT"
`, snippet)

	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash binding failed: %v\n%s", err, out)
	}
	text := string(out)

	if !strings.Contains(text, "EXPANDED:podctl build") {
		t.Errorf("Alt-H did not expand the command line:\n%s", text)
	}
	if !strings.Contains(text, "POINT:12") { // len("podctl build")
		t.Errorf("the cursor was not moved to the end of the expanded line:\n%s", text)
	}
	if !strings.Contains(text, "Usage:  podctl build") {
		t.Errorf("Alt-H printed no help:\n%s", text)
	}

	var printed int
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if strings.HasPrefix(line, "EXPANDED:") || strings.HasPrefix(line, "POINT:") {
			continue
		}
		printed++
	}
	if budget := explainBudget(24); printed > budget {
		t.Errorf("Alt-H printed %d lines on a 24-line screen, over its budget of %d:\n%s", printed, budget, text)
	}
}

// Alt-H belongs to this app only: bash binds it globally, so the function must
// leave someone else's command line alone.
func TestLiveBashAltHIgnoresOtherCommands(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found, skipping")
	}
	binDir, snippet := buildExplainFixture(t, "bash")

	script := fmt.Sprintf(`
source %q 2>/dev/null
READLINE_LINE='git commit -m x'
READLINE_POINT=${#READLINE_LINE}
_clihelp_explain
echo "LINE:$READLINE_LINE"
`, snippet)

	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash binding failed: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "LINE:git commit -m x" {
		t.Errorf("Alt-H touched a command line that is not ours: %q", got)
	}
}

// Sourcing a completion script — "source <(myapp completion zsh)", or a startup
// file that reads it — must be silent. The zsh script ended with the call that
// an $fpath autoload needs, so sourcing it ran the completion function outside
// any completion context and printed "can only be called from completion
// function" at every shell start.
func TestCompletionScriptsAreSafeToSource(t *testing.T) {
	for _, tt := range []struct {
		shell  string
		script func(string, string) string
	}{
		{"bash", func(path, _ string) string {
			return "source " + path + "\necho SOURCED\n"
		}},
		{"zsh", func(path, dir string) string {
			return "autoload -Uz compinit && compinit -u -d " + dir + "/zcompdump\nsource " + path + "\necho SOURCED\n"
		}},
		{"fish", func(path, _ string) string {
			return "source " + path + "\necho SOURCED\n"
		}},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			shellPath, err := exec.LookPath(tt.shell)
			if err != nil {
				t.Skipf("%s not found, skipping", tt.shell)
			}

			dir := t.TempDir()
			bin := filepath.Join(dir, "podctl")
			if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
				t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
			}
			script, err := sandboxedCommand(t, bin, "completion", tt.shell).Output()
			if err != nil {
				t.Fatalf("failed to generate the %s script: %v", tt.shell, err)
			}
			scriptPath := filepath.Join(dir, "completion."+tt.shell)
			if err := os.WriteFile(scriptPath, script, 0o600); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(shellPath, "-c", tt.script(scriptPath, dir))
			if tt.shell != "fish" {
				cmd = exec.Command(shellPath, "-f", "-c", tt.script(scriptPath, dir))
			}
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed to source its own completion script: %v\n%s", tt.shell, err, out)
			}
			if got := strings.TrimSpace(string(out)); got != "SOURCED" {
				t.Errorf("sourcing the %s script was not silent:\n%s", tt.shell, got)
			}
		})
	}
}

// Alt-H is one key for the whole shell, so every clihelp program on the machine
// has to share one dispatcher. A per-program binding was replaced by the next
// program installed, and that program's name check then refused every other
// program's command line: Alt-H went silently dead for all but the last one.
func TestLiveBashAltHServesSeveralPrograms(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found, skipping")
	}

	dir := t.TempDir()
	// Stand-ins for two installed clihelp programs: all the dispatcher asks of
	// them is the __explain protocol — an expanded command line, then the help.
	for _, name := range []string{"alpha", "beta"} {
		// $1 is "__explain", $2 the command line: echo it back expanded, then
		// one line standing in for the help.
		stub := "#!/bin/sh\necho \"$2 EXPANDED\"\necho \"help for " + name + "\"\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(stub), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	var snippets []string
	for _, name := range []string{"alpha", "beta"} {
		var b strings.Builder
		if err := GenKeyBindings(&App{Name: name}, "bash", &b); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "keys."+name)
		if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		snippets = append(snippets, path)
	}

	// The dispatcher has to run in this shell, the way bind -x runs it: inside
	// $( ) its rewrite of READLINE_LINE would die with the subshell.
	script := fmt.Sprintf(`
source %q 2>/dev/null
source %q 2>/dev/null
for line in "alpha build" "beta build" "git commit"; do
    READLINE_LINE="$line"
    _clihelp_explain > %q
    printf '%%s => [%%s] %%s\n' "$line" "$READLINE_LINE" "$(grep -c . %q)"
done
`, snippets[0], snippets[1], filepath.Join(dir, "out"), filepath.Join(dir, "out"))

	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dispatcher failed: %v\n%s", err, out)
	}

	for _, want := range []string{
		"alpha build => [alpha build EXPANDED] 1",
		"beta build => [beta build EXPANDED] 1",
		"git commit => [git commit] 0", // not ours: the line and the key are left alone
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
}

// A generated wrapper is only worth anything if the shell really completes and
// explains through it, so this drives one with the real bash completion script.
func TestLiveBashWrapperScript(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found, skipping")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}

	// The wrapper, and the completion script, exactly as a user would get them.
	wrapper := filepath.Join(dir, "pd")
	script, err := sandboxedCommand(t, bin, "__clihelp", "wrapper", "pd", "deploy").Output()
	if err != nil {
		t.Fatalf("generating the wrapper failed: %v", err)
	}
	if err := os.WriteFile(wrapper, script, 0o700); err != nil {
		t.Fatal(err)
	}
	completion, err := sandboxedCommand(t, bin, "completion", "bash").Output()
	if err != nil {
		t.Fatal(err)
	}
	completionPath := filepath.Join(dir, "completion.bash")
	if err := os.WriteFile(completionPath, completion, 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(t *testing.T, body string) string {
		t.Helper()
		cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c",
			"source "+completionPath+"\ncomplete -F _podctl_complete pd\n"+body)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "LINES=24", "COLUMNS=78")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bash failed: %v\n%s", err, out)
		}
		return string(out)
	}

	t.Run("completion runs in the wrapped command's context", func(t *testing.T) {
		out := run(t, `COMP_WORDS=(pd '--b'); COMP_CWORD=1; _podctl_complete; printf '%s\n' "${COMPREPLY[@]}"`)
		if !strings.Contains(out, "--bucket") {
			t.Errorf("completing 'pd --b' did not reach deploy's flags:\n%s", out)
		}
	})

	t.Run("the wrapper answers __explain without rewriting the line", func(t *testing.T) {
		out := run(t, `pd __explain "pd --bucket x"`)
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if lines[0] != "pd --bucket x" {
			t.Errorf("first line = %q, want the command line as typed", lines[0])
		}
		if !strings.Contains(out, "Usage:  podctl deploy") {
			t.Errorf("the explanation is not for the wrapped command:\n%s", out)
		}
	})

	t.Run("the wrapper still runs the program", func(t *testing.T) {
		out := run(t, `pd --help | head -1`)
		if !strings.Contains(out, "podctl deploy") {
			t.Errorf("running the wrapper did not run the wrapped command:\n%s", out)
		}
	})
}

// The whole point of the one-step install: after it, a shell that reads only its
// own startup file has completion and Alt-H, and nothing ran at startup.
func TestLiveBashOneStepInstall(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found, skipping")
	}

	home := t.TempDir()
	bin := filepath.Join(home, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}
	// A logging stand-in on PATH ahead of the real binary would change what runs,
	// so instead the install is done first and the startup measured after.
	install := sandboxedCommand(t, bin, "completion", "install", "bash")
	install.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"), "CLIHELP_NO_AUTO_COMPLETION=1")
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}

	script := `
source "$HOME/.bashrc"
echo "COMPLETION: $(complete -p podctl 2>/dev/null | grep -c podctl)"
echo "BINDING: $(declare -F _clihelp_explain >/dev/null && echo yes || echo no)"
echo "REGISTERED: $_clihelp_apps"
COMP_WORDS=(podctl 'buil'); COMP_CWORD=1
_podctl_complete
echo "CANDIDATES: ${COMPREPLY[*]}"
`
	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", script)
	cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+home+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the installed startup file failed: %v\n%s", err, out)
	}
	text := string(out)

	for _, want := range []string{"COMPLETION: 1", "BINDING: yes", "REGISTERED:  podctl", "CANDIDATES: build"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q after sourcing only the startup file:\n%s", want, text)
		}
	}
}
