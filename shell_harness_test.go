package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Driving the generated shell code without a terminal.
//
// bash's key binding is an ordinary function, so it can be called directly, and
// the tests in explain_shell_test.go do. zsh's widget and fish's binding cannot
// be — but they are ordinary functions too, and in both shells a function
// shadows a builtin. So stubbing `zle`, `bindkey`, `commandline` and `bind` with
// functions that record their arguments drives the real generated code
// headlessly, with no pty and no new dependency. Everything below is what those
// two shells were missing: until now they were only syntax-checked.

// generateSnippet writes a generated artifact into dir and returns its path.
func generateSnippet(t *testing.T, dir, name string, gen func(*App, string, *strings.Builder) error, app *App, shell string) string {
	t.Helper()
	var b strings.Builder
	if err := gen(app, shell, &b); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func keySnippet(t *testing.T, dir, shell string, app *App) string {
	t.Helper()
	return generateSnippet(t, dir, "keys."+shell, func(a *App, s string, b *strings.Builder) error {
		return GenKeyBindings(a, s, b)
	}, app, shell)
}

// driveShell runs body in shell with the snippet sourced and the interactive
// builtins stubbed, and returns everything it printed.
func driveShell(t *testing.T, shell, dir, snippet, stubs, body string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not found, skipping", shell)
	}

	script := stubs + "\nsource " + snippet + "\n" + body + "\n"

	var args []string
	switch shell {
	case "bash":
		args = []string{"--norc", "--noprofile", "-c", script}
	case "fish":
		args = []string{"--no-config", "-c", script}
	default:
		args = []string{"-f", "-c", script}
	}
	cmd := exec.Command(path, args...)
	cmd.Env = append(os.Environ(), "HOME="+dir, "PATH="+dir+":"+os.Getenv("PATH"), "LINES=24", "COLUMNS=78")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", shell, err, out)
	}
	return string(out)
}

// zshStubs shadows the ZLE builtins a widget uses, recording each call.
const zshStubs = `
zle() { print -r -- "ZLE:$*" }
bindkey() { print -r -- "BINDKEY:$*" }
`

// fishStubs shadows fish's interactive builtins. commandline answers the buffer
// it is given and records a replacement.
const fishStubs = `
set -g __buffer ""
function commandline
    if test (count $argv) -eq 0
        echo $__buffer
        return
    end
    switch $argv[1]
        case '-r' '--replace'
            set -g __buffer $argv[-1]
            echo "SET:$argv[-1]"
        case '-f'
            echo "REPAINT"
        case '*'
            echo $__buffer
    end
end
function bind; echo "BIND:$argv"; end
function __fish_man_page; echo "MANPAGE"; end
`

// bashStubs shadows the readline builtin the snippet binds through.
const bashStubs = `
bind() { printf 'BIND:%s\n' "$*"; }
`

// stubProgram writes a stand-in for the wrapped program: it answers the
// __explain protocol and nothing else.
func stubProgram(t *testing.T, dir, name string) {
	t.Helper()
	body := "#!/bin/sh\necho \"$2 EXPANDED\"\necho \"help for " + name + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}

// The binding must reach both the emacs and the vi keymaps. It landed in
// whichever one happened to be current when the snippet was sourced, so a user
// whose `set -o vi` came afterwards — or whose plugin manager set the keymap
// from a hook — had no Alt-H at all.
func TestKeyBindingReachesBothKeymaps(t *testing.T) {
	dir := t.TempDir()
	stubProgram(t, dir, "alpha")

	t.Run("bash", func(t *testing.T) {
		snippet := keySnippet(t, dir, "bash", &App{Name: "alpha"})
		out := driveShell(t, "bash", dir, snippet, bashStubs, `echo done`)
		for _, want := range []string{"-m emacs-standard", "-m vi-insert"} {
			if !strings.Contains(out, want) {
				t.Errorf("bash binding does not cover %q:\n%s", want, out)
			}
		}
	})

	t.Run("zsh", func(t *testing.T) {
		snippet := keySnippet(t, dir, "zsh", &App{Name: "alpha"})
		out := driveShell(t, "zsh", dir, snippet, zshStubs, `print -r -- done`)
		for _, want := range []string{"-M emacs", "-M viins"} {
			if !strings.Contains(out, want) {
				t.Errorf("zsh binding does not cover %q:\n%s", want, out)
			}
		}
	})

	t.Run("fish", func(t *testing.T) {
		snippet := keySnippet(t, dir, "fish", &App{Name: "alpha"})
		out := driveShell(t, "fish", dir, snippet, fishStubs, `echo done`)
		for _, want := range []string{"-M default", "-M insert"} {
			if !strings.Contains(out, want) {
				t.Errorf("fish binding does not cover %q:\n%s", want, out)
			}
		}
	})
}

// CLIHELP_NO_KEY_BINDINGS gives back Alt-H — zsh's run-help and fish's man-page
// key — to users who would rather keep it, without giving up completion.
func TestKeyBindingOptOut(t *testing.T) {
	dir := t.TempDir()
	stubProgram(t, dir, "alpha")

	for _, tc := range []struct{ shell, stubs string }{
		{"bash", bashStubs},
		{"zsh", zshStubs},
		{"fish", fishStubs},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			snippet := keySnippet(t, dir, tc.shell, &App{Name: "alpha"})
			set := "export CLIHELP_NO_KEY_BINDINGS=1\n"
			if tc.shell == "fish" {
				set = "set -gx CLIHELP_NO_KEY_BINDINGS 1\n"
			}
			out := driveShell(t, tc.shell, dir, snippet, set+tc.stubs, "echo done")
			if strings.Contains(out, "BIND") {
				t.Errorf("%s bound the key despite the opt-out:\n%s", tc.shell, out)
			}
			if !strings.Contains(out, "done") {
				t.Errorf("%s snippet did not run to completion:\n%s", tc.shell, out)
			}
		})
	}
}
