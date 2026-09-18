package clihelp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// zsh reads $ZDOTDIR/.zshrc when ZDOTDIR is set, and ~/.zshrc otherwise. Getting
// that wrong puts the sourcing line in a file the user's shell never reads, so
// the install reports success and nothing works. Nothing asserted it.
func TestZshHonoursZDOTDIR(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	t.Run("unset", func(t *testing.T) {
		t.Setenv("ZDOTDIR", "")
		path, owned, err := startupFile(app, "zsh")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, ".zshrc"); path != want {
			t.Errorf("startup file = %q, want %q", path, want)
		}
		if owned {
			t.Error("a shared .zshrc is not ours to own")
		}
	})

	t.Run("set", func(t *testing.T) {
		zdot := filepath.Join(home, "zdot")
		t.Setenv("ZDOTDIR", zdot)
		path, _, err := startupFile(app, "zsh")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(zdot, ".zshrc"); path != want {
			t.Errorf("startup file = %q, want %q — the block would go where zsh never looks", path, want)
		}
	})

	t.Run("an install lands in the ZDOTDIR file", func(t *testing.T) {
		zdot := filepath.Join(home, "zdot2")
		if err := os.MkdirAll(zdot, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ZDOTDIR", zdot)
		if _, err := installShellIntegration(app, "zsh", true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(zdot, ".zshrc")); err != nil {
			t.Errorf("nothing was written to $ZDOTDIR/.zshrc: %v", err)
		}
		if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
			t.Error("the block went to ~/.zshrc although ZDOTDIR is set")
		}
	})
}

// The automatic refresh preserves what the user installed: an integration
// written with --no-keys must not gain the Alt-H binding when it is refreshed.
// integrationHasKeys is the only thing carrying that decision forward, and it
// could be made to answer "yes" always with the suite green.
func TestIntegrationHasKeysReadsTheFile(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	for _, keys := range []bool{true, false} {
		if _, err := installShellIntegration(app, "bash", keys); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(home, ".config", "myapp", "shell", "bash")
		if got := integrationHasKeys(path); got != keys {
			t.Errorf("installed with keys=%v, integrationHasKeys reports %v", keys, got)
		}
	}

	// And a file that does not exist has no keys, rather than defaulting to yes.
	if integrationHasKeys(filepath.Join(home, "nothing-here")) {
		t.Error("a missing integration file reports that it has keys")
	}
}

// An explicit uninstall stays uninstalled: the marker is what tells the
// automatic path not to put anything back.
func TestUninstalledMarkerIsHonoured(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	if _, err := installShellIntegration(app, "bash", true); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".config", "myapp", "shell", "bash")
	if _, err := uninstallShellIntegration(app, "bash"); err != nil {
		t.Fatal(err)
	}
	marker := uninstalledMarker(target)
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("uninstall left no marker at %s: %v", marker, err)
	}
	if filepath.Dir(marker) != filepath.Dir(target) {
		t.Errorf("the marker %q does not sit beside the integration file %q", marker, target)
	}

	// The automatic path must leave it alone.
	t.Setenv("SHELL", "/bin/bash")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm")
	app.AutoRefreshIntegration = true
	TestExecute(app, []string{"build"}).AssertNoError(t)
	if _, err := os.Stat(target); err == nil {
		t.Error("the automatic path restored an integration the user had removed")
	}
}

// upsertBlock's second return value is what the report tells the user: whether
// the startup file was edited or was already sourcing the integration.
func TestUpsertBlockReportsWhetherItChanged(t *testing.T) {
	const begin, end = "# >>> x >>>", "# <<< x <<<"
	block := begin + "\nsource it\n" + end + "\n"

	for _, tt := range []struct {
		name        string
		contents    string
		wantChanged bool
	}{
		{"an empty file gains the block", "", true},
		{"a file without the block gains it", "export PATH=/x\n", true},
		{"an identical block is not a change", "export PATH=/x\n\n" + block, false},
		{"a stale block is a change", "export PATH=/x\n\n" + begin + "\nold\n" + end + "\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, changed, err := upsertBlock(tt.contents, block, begin, end)
			if err != nil {
				t.Fatal(err)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if !strings.Contains(out, "source it") {
				t.Errorf("the block is not in the result:\n%q", out)
			}
			if changed == (out == tt.contents) {
				t.Errorf("changed = %v but the contents %s", changed,
					map[bool]string{true: "are identical", false: "differ"}[out == tt.contents])
			}
		})
	}
}

// The manual page is refreshed when the templates change, which is what
// manPageVersion is for.
func TestManPageIsCurrentReadsTheVersion(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	path, err := installManPage(app, true)
	if err != nil {
		t.Fatal(err)
	}
	if !manPageIsCurrent(path) {
		t.Fatal("a freshly written page is not current")
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	current := fmt.Sprintf("%s: %d", manMarker, manPageVersion)
	stale := strings.Replace(string(body), current, fmt.Sprintf("%s: 0", manMarker), 1)
	if stale == string(body) {
		t.Fatalf("the fixture is wrong: no version marker in the page")
	}
	writeFixture(t, path, stale)
	if manPageIsCurrent(path) {
		t.Error("a page stamped with an older version is reported as current, so an upgrade would never refresh it")
	}
	_ = home
}

// The lookup of a competing manual page runs another program, and it runs inside
// the user's. The deadline is what stops an unreachable MANPATH entry hanging it.
func TestManLookupDeadlineIsApplied(t *testing.T) {
	home := sandboxHome(t)
	stub := filepath.Join(home, "bin")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A man(1) that never returns.
	if err := os.WriteFile(filepath.Join(stub, "man"), []byte("#!/bin/sh\nsleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))

	start := time.Now()
	_, _ = manPageElsewhere(installApp(), filepath.Join(home, "myapp.1"))
	if elapsed := time.Since(start); elapsed > manLookupTimeout+2*time.Second {
		t.Errorf("the lookup ran for %v against a %v deadline", elapsed, manLookupTimeout)
	}
}
