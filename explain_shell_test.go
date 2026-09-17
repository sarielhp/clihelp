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
	out, err := exec.Command(bin, "completion", "keys", shell).Output()
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
_podctl_clihelp_explain
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
_podctl_clihelp_explain
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
