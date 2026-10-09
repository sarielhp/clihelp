package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A wrapper on $PATH, then one install, then a real shell completing through
// it — in each shell. TestInstallRegistersWrappersFoundOnPath checks the text
// the install writes; this checks that the text works, which is the only thing
// the user sees.
//
// The preset arguments start with a flag, as "mt" runs "mail_cli -2 tui": a
// flag before the command is what an earlier parser dropped, and what a
// completion that ignored the presets would get wrong.

// liveInstallWithWrapper builds the example CLI into a sandboxed home, writes
// the wrapper "pd" for "podctl -v deploy" beside it, and installs for shell
// with that directory on $PATH, so that the install finds the wrapper.
func liveInstallWithWrapper(t *testing.T, shell string) string {
	t.Helper()
	if _, err := exec.LookPath(shell); err != nil {
		skipMissingToolf(t, "%s not found, skipping", shell)
	}
	home := t.TempDir()
	bin := filepath.Join(home, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}
	script, err := sandboxedCommand(t, bin, "__clihelp", "wrapper", "pd", "-v", "deploy").Output()
	if err != nil {
		t.Fatalf("generating the wrapper failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "pd"), script, 0o755); err != nil {
		t.Fatal(err)
	}
	install := sandboxedCommand(t, bin, "completion", "install", shell)
	install.Env = append(install.Env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"PATH="+home+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("%s install failed: %v\n%s", shell, err, out)
	}
	return home
}

// wantLines fails for each wanted line out does not contain.
func wantLines(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
}

func TestLiveWrapperCompletesAfterInstall(t *testing.T) {
	t.Run("bash", func(t *testing.T) {
		home := liveInstallWithWrapper(t, "bash")
		// The function is the one the install registered for pd, not one the
		// test names: a missing registration must fail here.
		script := `
source "$HOME/.bashrc"
fn=$(complete -p pd 2>/dev/null | sed -n 's/.*-F \([^ ]*\) .*/\1/p')
echo "FUNCTION: ${fn:-none}"
COMP_WORDS=(pd '--b'); COMP_CWORD=1
[ -n "$fn" ] && "$fn"
echo "CANDIDATES: ${COMPREPLY[*]}"
echo "REGISTERED: $_clihelp_apps"
`
		out := liveShell(t, "bash", home, script, "--norc", "--noprofile")
		wantLines(t, out, "FUNCTION: _podctl_complete", "CANDIDATES: --bucket", "pd:1")
	})

	t.Run("zsh", func(t *testing.T) {
		home := liveInstallWithWrapper(t, "zsh")
		// zsh's completion builtins only work inside a completion widget, so
		// the two that report candidates print them instead, as in the zsh
		// harness; everything else is the installed code.
		script := `
autoload -Uz compinit && compinit -u -d "$HOME/.zcompdump"
source "$HOME/.zshrc"
print -r -- "FUNCTION: ${_comps[pd]:-none}"
_describe() { shift 3; local n; for n in "$@"; do print -r -- "CANDIDATE: ${${(P)n}[@]}"; done }
compadd() { [[ $1 == -a ]] && print -r -- "CANDIDATE: ${${(P)2}[@]}"; return 0 }
_files() { : }
words=(pd --b); CURRENT=2
[[ -n ${_comps[pd]} ]] && ${_comps[pd]}
print -r -- "REGISTERED: $_clihelp_apps"
`
		out := liveShell(t, "zsh", home, script, "-f")
		wantLines(t, out, "FUNCTION: _podctl", "CANDIDATE: --bucket", "pd:1")
	})

	t.Run("fish", func(t *testing.T) {
		home := liveInstallWithWrapper(t, "fish")
		script := `
echo "CANDIDATES: "(complete -C "pd --b")
echo "REGISTERED: $_clihelp_apps"
`
		out := liveShell(t, "fish", home, script)
		wantLines(t, out, "CANDIDATES: --bucket", "pd:1")
	})
}
