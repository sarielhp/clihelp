package clihelp

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// sandboxedCommand builds a command whose environment cannot reach the real home
// directory. Several tests here run the example binary, which sets
// AutoRefreshIntegration, so an inherited environment means the test suite
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

// homeFingerprint records the content of every file this library can write to in
// a home directory, keyed by path.
//
// It hashes the bytes rather than recording size and modification time. Those
// are cheaper, and they were what this guard used until an unexplained failure
// during a release: ~/.bashrc was reported as modified while its content was
// byte for byte what it had been, with none of clihelp's markers in it. The
// suite shares a machine with the user's editors, shells and other tools, and
// any of them touching a watched file inside the twenty seconds the suite runs
// failed the build and sent the next reader hunting for a bug in this library.
// What this guard protects is the user's content, and content is what it
// compares. A modification time that moves with the bytes unchanged is reported
// by fingerprintDiff as a note rather than a failure.
func homeFingerprint(home string) map[string]string {
	out := map[string]string{}
	record := func(path string, info os.FileInfo) {
		sum := "unreadable"
		if data, err := os.ReadFile(path); err == nil {
			sum = fmt.Sprintf("%x", sha256.Sum256(data))
		}
		out[path] = fmt.Sprintf("%s mtime=%d", sum, info.ModTime().UnixNano())
	}
	for _, p := range clihelpOwnedHomePaths(home) {
		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			record(path, info)
			return nil
		})
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			record(p, info)
		}
	}
	return out
}

// contentOf strips the modification time from a fingerprint entry, leaving the
// hash of the bytes.
func contentOf(entry string) string {
	if i := strings.Index(entry, " mtime="); i >= 0 {
		return entry[:i]
	}
	return entry
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

	// Sandbox the whole package, not just the tests that remember to. The
	// fingerprint below is the alarm; this is the lock. A test that reaches a
	// write path without calling sandboxHome used to land in the developer's own
	// home directory, and the alarm only told us afterwards — once, from inside a
	// release, which is a poor moment to find out. Tests that need a home of
	// their own still call sandboxHome, which layers on top of this.
	shared, mkErr := os.MkdirTemp("", "clihelp-package-home")
	if mkErr == nil {
		for k, v := range map[string]string{
			"HOME":            shared,
			"XDG_CONFIG_HOME": filepath.Join(shared, ".config"),
			"XDG_DATA_HOME":   filepath.Join(shared, ".local", "share"),
			"ZDOTDIR":         "",
		} {
			if setErr := os.Setenv(k, v); setErr != nil {
				panic(setErr)
			}
		}
	}

	code := m.Run()

	// os.Exit below skips deferred calls, so the shared home is removed here.
	if shared != "" {
		_ = os.RemoveAll(shared)
	}

	changed, touched := fingerprintDiff(before, homeFingerprint(home))
	if len(touched) > 0 {
		fmt.Fprintf(os.Stderr, "\nnote: a watched file was rewritten with identical content, "+
			"which this suite cannot have done:\n  %s\n", strings.Join(touched, "\n  "))
	}
	if len(changed) > 0 {
		fmt.Fprintf(os.Stderr, "\nthe test suite modified the real home directory:\n  %s\n",
			strings.Join(changed, "\n  "))
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// fingerprintDiff reports what changed, and separately what was merely touched.
// The first list fails the suite; the second is printed so that a file another
// process on this machine rewrote with identical bytes is visible without being
// mistaken for this library writing into someone's home directory.
func fingerprintDiff(before, after map[string]string) (changed, touched []string) {
	for path, sig := range after {
		old, ok := before[path]
		switch {
		case !ok:
			changed = append(changed, "created: "+path)
		case contentOf(old) != contentOf(sig):
			changed = append(changed, "modified: "+path)
		case old != sig:
			touched = append(touched, "touched, content unchanged: "+path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, "removed: "+path)
		}
	}
	sort.Strings(changed)
	sort.Strings(touched)
	return changed, touched
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

	if changed, touched := fingerprintDiff(before, homeFingerprint(home)); len(changed)+len(touched) != 0 {
		t.Errorf("an unchanged home reported a difference: %v %v", changed, touched)
	}

	// A rewrite that leaves the bytes alone is what another process on this
	// machine looks like, and it is what failed a release with nothing wrong.
	// It is reported, and it does not fail the suite.
	if err := os.WriteFile(watched, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(watched, time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	changed, touched := fingerprintDiff(before, homeFingerprint(home))
	if len(changed) != 0 {
		t.Errorf("a rewrite with identical content was reported as a modification: %v", changed)
	}
	if len(touched) != 1 || !strings.HasPrefix(touched[0], "touched, content unchanged:") {
		t.Errorf("a rewrite with identical content was not noted: %v", touched)
	}

	if err := os.WriteFile(watched, []byte("edited by a test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := fingerprintDiff(before, homeFingerprint(home)); len(changed) != 1 || !strings.HasPrefix(changed[0], "modified:") {
		t.Errorf("an edited startup file was not detected: %v", changed)
	}

	created := filepath.Join(home, ".local", "share", "bash-completion", "completions", "myapp")
	writeFixture(t, created, "installed by a test\n")
	changed, _ = fingerprintDiff(before, homeFingerprint(home))
	if len(changed) != 2 {
		t.Errorf("a newly installed completion script was not detected: %v", changed)
	}
}
