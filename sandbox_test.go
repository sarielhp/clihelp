package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// sandboxedCommand builds a command whose environment cannot reach the real home
// directory. Several tests here run the example binary, which sets
// AutoInstallCompletion, so an inherited environment means the test suite
// installs into the developer's home — which is exactly what happened before
// this helper existed.
func sandboxedCommand(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"ZDOTDIR=",
		"MANPATH="+filepath.Join(home, ".local", "share", "man"),
		"CLIHELP_NO_AUTO_COMPLETION=1",
	)
	return cmd
}

// clihelpOwnedHomePaths are every path this library can write to in a home
// directory. The guard below watches all of them.
func clihelpOwnedHomePaths(home string) []string {
	return []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config", "fish", "conf.d"),
		filepath.Join(home, ".local", "share", "bash-completion", "completions"),
		filepath.Join(home, ".local", "share", "zsh", "site-functions"),
		filepath.Join(home, ".config", "fish", "completions"),
		filepath.Join(home, ".local", "share", "man", "man1"),
	}
}

func homeFingerprint(home string) map[string]string {
	out := map[string]string{}
	for _, p := range clihelpOwnedHomePaths(home) {
		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			out[path] = fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
			return nil
		})
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			out[p] = fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
		}
	}
	return out
}

// TestMain fails the suite if it changed anything in the real home directory.
//
// Sandboxing is a per-test discipline, and a per-test discipline has already
// failed once in this repository: a block ended up in the author's own ~/.bashrc.
// This guard is applied once and covers every test, including the ones that run
// the example binary in a subprocess.
func TestMain(m *testing.M) {
	home, err := os.UserHomeDir()
	if err != nil {
		os.Exit(m.Run())
	}
	before := homeFingerprint(home)
	code := m.Run()
	if diff := fingerprintDiff(before, homeFingerprint(home)); len(diff) > 0 {
		fmt.Fprintf(os.Stderr, "\nthe test suite modified the real home directory:\n  %s\n",
			strings.Join(diff, "\n  "))
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func fingerprintDiff(before, after map[string]string) []string {
	var diff []string
	for path, sig := range after {
		if old, ok := before[path]; !ok {
			diff = append(diff, "created: "+path)
		} else if old != sig {
			diff = append(diff, "modified: "+path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			diff = append(diff, "removed: "+path)
		}
	}
	sort.Strings(diff)
	return diff
}

// The guard's own logic is tested against a temporary directory: proving it by
// letting something write to the real home would be the very thing it exists to
// prevent.
func TestHomeFingerprintDetectsChanges(t *testing.T) {
	home := t.TempDir()
	watched := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(watched, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := homeFingerprint(home)

	if diff := fingerprintDiff(before, homeFingerprint(home)); len(diff) != 0 {
		t.Errorf("an unchanged home reported a difference: %v", diff)
	}

	if err := os.WriteFile(watched, []byte("edited by a test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if diff := fingerprintDiff(before, homeFingerprint(home)); len(diff) != 1 || !strings.HasPrefix(diff[0], "modified:") {
		t.Errorf("an edited startup file was not detected: %v", diff)
	}

	created := filepath.Join(home, ".local", "share", "bash-completion", "completions", "myapp")
	writeFixture(t, created, "installed by a test\n")
	diff := fingerprintDiff(before, homeFingerprint(home))
	if len(diff) != 2 {
		t.Errorf("a newly installed completion script was not detected: %v", diff)
	}
}
