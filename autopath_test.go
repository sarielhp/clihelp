package clihelp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// An ordinary program run may refresh a file clihelp generated, and may not edit
// a startup file or create anything the user did not ask for. The auto path runs
// inside every invocation of every program built with the library, so it is the
// most conservative code here and needs the tightest test.
func TestAutoPathNeverEditsAStartupFile(t *testing.T) {
	home := sandboxHome(t)
	autoEnv(t)

	app := installApp()
	app.AutoRefreshIntegration = true
	if _, err := installShellIntegration(app, "bash", true); err != nil {
		t.Fatal(err)
	}

	rc := filepath.Join(home, ".bashrc")
	userOnly := "export EDITOR=vi\n"
	if err := os.WriteFile(rc, []byte(userOnly), 0o600); err != nil {
		t.Fatal(err)
	}
	makeIntegrationStale(t, filepath.Join(home, ".config", "myapp", "shell", "bash"))

	testExecute(app, []string{"build"}).AssertNoError(t)

	if got, _ := os.ReadFile(rc); string(got) != userOnly {
		t.Errorf("an ordinary program run edited the startup file:\n%s", got)
	}
	if !integrationIsCurrent(filepath.Join(home, ".config", "myapp", "shell", "bash")) {
		t.Errorf("the refresh did not bring the generated file up to date")
	}
}

func TestAutoPathDoesNotCreateAStartupFile(t *testing.T) {
	home := sandboxHome(t)
	autoEnv(t)
	t.Setenv("SHELL", "/bin/zsh")

	app := installApp()
	app.AutoRefreshIntegration = true
	if _, err := installShellIntegration(app, "zsh", true); err != nil {
		t.Fatal(err)
	}
	// The user adopts ZDOTDIR after installing, so the startup file the auto path
	// would compute is one that does not exist.
	zdot := filepath.Join(home, ".config", "zsh")
	t.Setenv("ZDOTDIR", zdot)
	makeIntegrationStale(t, filepath.Join(home, ".config", "myapp", "shell", "zsh"))

	testExecute(app, []string{"build"}).AssertNoError(t)

	if _, err := os.Stat(filepath.Join(zdot, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf("an ordinary program run created a startup file")
	}
}

// Uninstall has to stay uninstalled: the auto path must not put back an artifact
// the user removed, in this or any other directory.
func TestUninstallSurvivesAnOrdinaryRun(t *testing.T) {
	home := sandboxHome(t)
	autoEnv(t)

	app := installApp()
	app.AutoRefreshIntegration = true
	if _, err := installShellIntegration(app, "bash", true); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstallShellIntegration(app, "bash"); err != nil {
		t.Fatal(err)
	}
	after := homeTree(t, home)

	testExecute(app, []string{"build"}).AssertNoError(t)

	if got := homeTree(t, home); strings.Join(got, ",") != strings.Join(after, ",") {
		t.Errorf("an ordinary run resurrected artifacts after uninstall:\nafter uninstall: %v\nafter one run:   %v", after, got)
	}
}

func autoEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("TERM", "xterm")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
}

func makeIntegrationStale(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(body), integrationMarkerFn+": "+integrationVersion(), integrationMarkerFn+": 0.0", 1)
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
}

func homeTree(t *testing.T, home string) []string {
	t.Helper()
	var out []string
	_ = filepath.Walk(home, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			out = append(out, strings.TrimPrefix(p, home))
		}
		return nil
	})
	sort.Strings(out)
	return out
}
