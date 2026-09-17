package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The startup-file line is shell source. Go's %q is a Go literal, not a shell
// word: it emits double quotes, inside which $( ), backticks and $VAR are still
// live. Anything that reaches the path — App.Name, $HOME, $XDG_CONFIG_HOME —
// therefore became executable content in the user's ~/.bashrc, permanently.
func TestBootstrapLineIsShellQuoted(t *testing.T) {
	for _, tt := range []struct {
		name  string
		shell string
	}{{"bash", "bash"}, {"zsh", "zsh"}, {"fish", "fish"}} {
		t.Run(tt.name, func(t *testing.T) {
			home := sandboxHome(t)
			// A config directory whose name would expand if it were not quoted.
			hostile := filepath.Join(home, "cfg $(touch "+filepath.Join(home, "EXECUTED")+")")
			t.Setenv("XDG_CONFIG_HOME", hostile)

			target := filepath.Join(hostile, "myapp", "shell", tt.shell)
			block := bootstrapBlock(installApp(), tt.shell, target)

			if strings.Contains(block, `"`+target+`"`) {
				t.Errorf("the path is in double quotes, where command substitution still runs:\n%s", block)
			}
			if !strings.Contains(block, "'") {
				t.Errorf("the path is not single-quoted:\n%s", block)
			}
		})
	}
}

// And end to end: sourcing the rc a real install produced must not run anything.
func TestSourcingTheInstalledBlockExecutesNothing(t *testing.T) {
	home := sandboxHome(t)
	marker := filepath.Join(home, "EXECUTED")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg $(touch "+marker+")"))

	if _, err := InstallShellIntegration(installApp(), "bash", true); err != nil {
		t.Fatal(err)
	}
	rc, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rc), "$(touch") && !strings.Contains(string(rc), "'") {
		t.Errorf("unquoted command substitution written into the startup file:\n%s", rc)
	}
	runShell(t, "bash", filepath.Join(home, ".bashrc"))
	if _, err := os.Stat(marker); err == nil {
		t.Errorf("sourcing the installed ~/.bashrc executed a command substitution")
	}
}

// App.Name becomes a file name, a shell function name, an rc-file marker and a
// registry key. A display fallback is not an identity.
func TestGeneratorsRejectANameTheyCannotSafelyEmit(t *testing.T) {
	for _, name := range []string{"", "my app", "pd$(id)", "pd`id`", "pd;id", "-pd", "..", "a/b", "pd\nid"} {
		t.Run(strings.ReplaceAll(name, "\n", "\\n"), func(t *testing.T) {
			app := &App{Name: name}
			var b strings.Builder
			if err := GenKeyBindings(app, "bash", &b); err == nil {
				t.Errorf("GenKeyBindings accepted %q and emitted:\n%s", name, b.String())
			}
			if _, err := IntegrationPath(app, "bash"); err == nil {
				t.Errorf("IntegrationPath accepted %q", name)
			}
			if _, err := ManPagePath(app); err == nil {
				t.Errorf("ManPagePath accepted %q", name)
			}
		})
	}

	// And an ordinary name still works everywhere.
	for _, name := range []string{"podctl", "pod-ctl", "pod_ctl", "pod.ctl", "p9"} {
		app := &App{Name: name}
		var b strings.Builder
		if err := GenKeyBindings(app, "bash", &b); err != nil {
			t.Errorf("GenKeyBindings rejected the ordinary name %q: %v", name, err)
		}
		if _, err := IntegrationPath(app, "bash"); err != nil {
			t.Errorf("IntegrationPath rejected the ordinary name %q: %v", name, err)
		}
	}
}

func runShell(t *testing.T, shell, rc string) {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not found", shell)
	}
	cmd := exec.Command(path, "--norc", "--noprofile", "-c", "source "+rc)
	cmd.Env = append(os.Environ(), "HOME="+filepath.Dir(rc))
	_ = cmd.Run()
}
