package clihelp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// sandboxHome points every path the installer computes at a temporary directory:
// $HOME for the startup files, XDG for everything else.
func sandboxHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("ZDOTDIR", "")
	return home
}

func installApp() *App {
	return &App{
		Name:     "myapp",
		Commands: []Command{{Name: "build", Description: "Build it", Run: func(*Context) error { return nil }}},
	}
}

func TestInstallShellIntegrationWritesOneFileAndOneLine(t *testing.T) {
	for _, tt := range []struct {
		shell   string
		startup string
		owned   bool // clihelp writes a whole drop-in file rather than a block
	}{
		{"bash", ".bashrc", false},
		{"zsh", ".zshrc", false},
		{"fish", ".config/fish/conf.d/myapp.fish", true},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			home := sandboxHome(t)
			res, err := InstallShellIntegration(installApp(), tt.shell, true)
			if err != nil {
				t.Fatal(err)
			}

			integration := filepath.Join(home, ".config", "myapp", "shell", tt.shell)
			if res.Integration != integration {
				t.Errorf("integration file = %q, want %q", res.Integration, integration)
			}
			body, err := os.ReadFile(integration)
			if err != nil {
				t.Fatalf("integration file not written: %v", err)
			}
			completionSymbol := "_myapp_complete"
			if tt.shell == "zsh" {
				completionSymbol = "_myapp()"
			} else if tt.shell == "fish" {
				completionSymbol = "__fish_myapp_complete"
			}
			for _, want := range []string{integrationMarkerFn, completionSymbol, "_clihelp_explain"} {
				if !strings.Contains(string(body), want) {
					t.Errorf("%s integration is missing %q", tt.shell, want)
				}
			}

			startup := filepath.Join(home, tt.startup)
			if res.Startup != startup {
				t.Errorf("startup file = %q, want %q", res.Startup, startup)
			}
			line, err := os.ReadFile(startup)
			if err != nil {
				t.Fatalf("startup file not written: %v", err)
			}
			if !strings.Contains(string(line), integration) {
				t.Errorf("the startup file does not source the integration:\n%s", line)
			}
			// The bootstrap must not run the program at shell start.
			if strings.Contains(string(line), "$(") || strings.Contains(string(line), "`") {
				t.Errorf("the startup line executes something:\n%s", line)
			}
			if !tt.owned && !strings.Contains(string(line), ">>> myapp shell integration (clihelp) >>>") {
				t.Errorf("the block is not marked:\n%s", line)
			}
		})
	}
}

func TestInstallLeavesTheRestOfTheStartupFileAlone(t *testing.T) {
	home := sandboxHome(t)
	rc := filepath.Join(home, ".bashrc")
	original := "export EDITOR=vi\nalias ll='ls -l'\n"
	if err := os.WriteFile(rc, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := InstallShellIntegration(installApp(), "bash", true)
	if err != nil {
		t.Fatal(err)
	}
	if !first.StartupEdit {
		t.Errorf("the first install should have edited the startup file")
	}
	after, _ := os.ReadFile(rc)
	if !strings.HasPrefix(string(after), original) {
		t.Errorf("the existing contents were not preserved:\n%s", after)
	}
	if mode, _ := os.Stat(rc); mode.Mode().Perm() != 0o600 {
		t.Errorf("the startup file's mode changed to %v", mode.Mode().Perm())
	}

	// Installing again must not add a second block.
	second, err := InstallShellIntegration(installApp(), "bash", true)
	if err != nil {
		t.Fatal(err)
	}
	if second.StartupEdit {
		t.Errorf("a repeated install edited the startup file again")
	}
	again, _ := os.ReadFile(rc)
	if n := strings.Count(string(again), ">>> myapp shell integration (clihelp) >>>"); n != 1 {
		t.Errorf("the startup file carries %d blocks, want 1", n)
	}

	// And uninstalling must leave exactly what was there before.
	if _, err := UninstallShellIntegration(installApp(), "bash"); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(rc)
	if string(restored) != original {
		t.Errorf("uninstall did not restore the startup file:\n%q\nwant:\n%q", restored, original)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "myapp", "shell", "bash")); !os.IsNotExist(err) {
		t.Errorf("uninstall left the generated file behind")
	}
}

func TestInstallSupersedesTheOldScriptLocation(t *testing.T) {
	home := sandboxHome(t)

	// A script installed the old way, into the shell's own directory.
	legacy, err := InstallCompletion(installApp(), "bash")
	if err != nil {
		t.Fatal(err)
	}
	// And a file in the same directory that clihelp did not write.
	foreign := filepath.Join(filepath.Dir(legacy), "someone-else")
	if err := os.WriteFile(foreign, []byte("# not ours\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := InstallShellIntegration(installApp(), "bash", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("the superseded script was left in place, so the shell would load two copies")
	}
	if len(res.Removed) != 1 || res.Removed[0] != legacy {
		t.Errorf("Removed = %v, want [%s]", res.Removed, legacy)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("a file clihelp never wrote was removed: %v", err)
	}
	_ = home
}

func TestInstallWithoutKeys(t *testing.T) {
	home := sandboxHome(t)
	if _, err := InstallShellIntegration(installApp(), "bash", false); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(home, ".config", "myapp", "shell", "bash"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "_clihelp_explain") {
		t.Errorf("--no-keys still installed the key binding:\n%s", body)
	}
	if !strings.Contains(string(body), "_myapp_complete") {
		t.Errorf("--no-keys dropped the completion as well")
	}
}

// Auto-install keeps an existing integration current, and never creates one or
// edits a startup file on its own.
func TestAutoInstallRefreshesButNeverCreates(t *testing.T) {
	home := sandboxHome(t)
	t.Setenv("SHELL", "/bin/bash")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm")

	app := installApp()
	app.AutoRefreshIntegration = true

	TestExecute(app, []string{"build"}).AssertNoError(t)
	if _, err := os.Stat(filepath.Join(home, ".config", "myapp", "shell", "bash")); !os.IsNotExist(err) {
		t.Errorf("auto-install created a shell integration nobody asked for")
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Errorf("auto-install created a startup file")
	}

	// Now install for real, and staleness must be repaired on the next run.
	if _, err := InstallShellIntegration(app, "bash", true); err != nil {
		t.Fatal(err)
	}
	// Age the installed file the way a clihelp upgrade would: same content, an
	// older version marker.
	path := filepath.Join(home, ".config", "myapp", "shell", "bash")
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(current), integrationMarkerFn+": "+integrationVersion(), integrationMarkerFn+": 0.0", 1)
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	TestExecute(app, []string{"build"}).AssertNoError(t)
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), integrationVersion()) {
		t.Errorf("a stale integration was not refreshed:\n%s", body)
	}
	if !strings.Contains(string(body), "_clihelp_explain") {
		t.Errorf("the refresh dropped the key bindings the user had installed")
	}
}

// The unattended path may keep a file the user installed current; it may not
// bring one into existence. This is the harder half of that rule, and the half
// the code got wrong: when no shell integration was present it fell through to
// the older completion-script location and *created* a script there, so an
// ordinary command run put a file in the user's home that nobody had asked for.
func TestAutoInstallNeverCreatesACompletionScript(t *testing.T) {
	home := sandboxHome(t)
	t.Setenv("SHELL", "/bin/bash")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm")

	app := installApp()
	app.AutoRefreshIntegration = true

	script, err := CompletionPath(app, "bash")
	if err != nil {
		t.Fatal(err)
	}

	TestExecute(app, []string{"build"}).AssertNoError(t)
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Errorf("an ordinary run created %s", script)
	}
	if entries, err := os.ReadDir(home); err == nil {
		for _, e := range entries {
			if e.Name() != ".config" {
				t.Errorf("an ordinary run created ~/%s", e.Name())
			}
		}
	}

	// A script the user installed, gone stale the way a library upgrade makes it
	// stale, is still repaired — the refresh half of the rule.
	var body bytes.Buffer
	if err := GenBashCompletion(app, &body); err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(body.String(),
		fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion),
		"clihelp-completion-version: 0", 1)
	writeFixture(t, script, stale)

	TestExecute(app, []string{"build"}).AssertNoError(t)
	got, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)) {
		t.Errorf("a stale script the user had installed was not refreshed:\n%s", got)
	}
}

// Two programs installing at once contend for one startup file. The write is
// atomic, so the file is never torn — but a lock-free read-modify-write loses
// whole blocks, and every caller reported success. Counting is the only evidence
// that settles this; a clean -race run says nothing about a race through the
// filesystem.
func TestConcurrentInstallsKeepEveryBlock(t *testing.T) {
	home := sandboxHome(t)
	const n = 8

	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			app := &App{
				Name:     fmt.Sprintf("app%d", i),
				Commands: []Command{{Name: "build", Description: "Build it", Run: func(*Context) error { return nil }}},
			}
			_, errs[i] = InstallShellIntegration(app, "bash", true)
		}(i)
	}
	wg.Wait()

	rc, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	missing := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Errorf("install %d failed: %v", i, errs[i])
		}
		if !strings.Contains(string(rc), fmt.Sprintf(">>> app%d shell integration", i)) {
			missing++
		}
	}
	if missing > 0 {
		t.Errorf("%d of %d blocks were lost, and every install reported success:\n%s", missing, n, rc)
	}
}
