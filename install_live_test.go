package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The one-step install, proved in the shell it was installed for.
//
// Every other live test here drives bash, which is the shell whose install path
// is simplest: one rc file, one function, no completion system to initialise.
// zsh and fish are where the mechanism actually differs — ZDOTDIR, a conf.d
// drop-in, and compinit ordering — and until now nothing started either of them
// and looked.

// liveInstall builds the example CLI into a sandboxed home and installs the
// shell integration there, the way a user would. It returns that home.
func liveInstall(t *testing.T, shell string) (home, bin string) {
	t.Helper()
	home = t.TempDir()
	bin = filepath.Join(home, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}
	install := sandboxedCommand(t, bin, "completion", "install", shell)
	install.Env = append(install.Env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"ZDOTDIR=",
		"CLIHELP_NO_AUTO_COMPLETION=1",
	)
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("%s install failed: %v\n%s", shell, err, out)
	}
	return home, bin
}

// liveShell runs script in a shell that reads only what the install put in the
// sandboxed home.
func liveShell(t *testing.T, shell, home, script string, args ...string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not found, skipping", shell)
	}
	cmd := sandboxedCommand(t, path, append(args, "-c", script)...)
	cmd.Env = append(cmd.Env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"ZDOTDIR=",
		"PATH="+home+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the installed %s setup failed: %v\n%s", shell, err, out)
	}
	return string(out)
}

func TestLiveZshOneStepInstall(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not found, skipping")
	}
	home, _ := liveInstall(t, "zsh")

	// The ordinary order: the completion system is up before the rc line runs.
	script := `
autoload -Uz compinit && compinit -u -d "$HOME/.zcompdump"
source "$HOME/.zshrc"
print -r -- "COMPLETION: ${_comps[podctl]:-none}"
print -r -- "WIDGET: ${+widgets[_clihelp_explain]}"
print -r -- "REGISTERED: $_clihelp_apps"
print -r -- "HOOK: ${precmd_functions[*]:-none}"
`
	out := liveShell(t, "zsh", home, script, "-f")

	for _, want := range []string{
		"COMPLETION: _podctl",
		"WIDGET: 1",
		"REGISTERED:  podctl:1  podctl",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q after sourcing only the startup file:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "HOOK: none") {
		t.Errorf("a retry hook was left behind although compinit had already run:\n%s", out)
	}
}

// The order plugin managers and turbo loaders actually produce: the rc line runs
// while compdef does not exist yet. Registration has to survive it.
func TestLiveZshInstallSurvivesDeferredCompinit(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not found, skipping")
	}
	home, _ := liveInstall(t, "zsh")

	script := `
source "$HOME/.zshrc"
print -r -- "BEFORE: ${_comps[podctl]:-none} hook=${precmd_functions[*]:-none}"
autoload -Uz compinit && compinit -u -d "$HOME/.zcompdump"
for f in $precmd_functions; do $f; done
print -r -- "AFTER: ${_comps[podctl]:-none} hook=${precmd_functions[*]:-none}"
`
	out := liveShell(t, "zsh", home, script, "-f")

	if !strings.Contains(out, "BEFORE: none hook=_podctl_deferred_compdef") {
		t.Errorf("sourcing before compinit registered nothing and scheduled no retry:\n%s", out)
	}
	if !strings.Contains(out, "AFTER: _podctl hook=none") {
		t.Errorf("the retry did not register the completion, or did not remove itself:\n%s", out)
	}
}

// fish needs no startup file at all: conf.d is a drop-in directory, so this runs
// a fish with its own configuration and asks what it found there.
func TestLiveFishOneStepInstall(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not found, skipping")
	}
	home, _ := liveInstall(t, "fish")

	if _, err := os.Stat(filepath.Join(home, ".config", "fish", "conf.d", "podctl.fish")); err != nil {
		t.Fatalf("install wrote no conf.d drop-in: %v", err)
	}

	script := `
echo "REGISTERED: $_clihelp_apps"
functions -q __clihelp_explain; and echo "BINDING: yes"; or echo "BINDING: no"
echo "CANDIDATES: "(complete -C "podctl buil")
`
	out := liveShell(t, "fish", home, script)

	for _, want := range []string{
		"REGISTERED: podctl:1 podctl",
		"BINDING: yes",
		"CANDIDATES: build",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q from a fish that read only its own configuration:\n%s", want, out)
		}
	}
}

// Uninstall has to leave the shell as it found it, in every shell.
func TestLiveUninstallLeavesNothingBehind(t *testing.T) {
	for _, shell := range []string{"zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("%s not found, skipping", shell)
			}
			home, bin := liveInstall(t, shell)

			un := sandboxedCommand(t, bin, "completion", "uninstall", shell)
			un.Env = append(un.Env,
				"HOME="+home,
				"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
				"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
				"ZDOTDIR=",
				"CLIHELP_NO_AUTO_COMPLETION=1",
			)
			if out, err := un.CombinedOutput(); err != nil {
				t.Fatalf("uninstall failed: %v\n%s", err, out)
			}

			var script string
			var args []string
			if shell == "zsh" {
				args = []string{"-f"}
				script = `
[ -r "$HOME/.zshrc" ] && source "$HOME/.zshrc"
print -r -- "REGISTERED: ${_clihelp_apps:-none}"
print -r -- "WIDGET: ${+widgets[_clihelp_explain]}"
`
			} else {
				script = `
echo "REGISTERED: "(count $_clihelp_apps)
functions -q __clihelp_explain; and echo "BINDING: yes"; or echo "BINDING: no"
`
			}
			out := liveShell(t, shell, home, script, args...)

			if strings.Contains(out, "podctl") {
				t.Errorf("uninstall left the program registered:\n%s", out)
			}
			if strings.Contains(out, "WIDGET: 1") || strings.Contains(out, "BINDING: yes") {
				t.Errorf("uninstall left the key binding behind:\n%s", out)
			}
			if _, err := os.Stat(filepath.Join(home, ".local", "share", "man", "man1", "podctl.1")); !os.IsNotExist(err) {
				t.Errorf("uninstall left the manual page behind")
			}
		})
	}
}
