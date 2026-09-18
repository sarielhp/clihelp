package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One command, one interruption. Setting a program up used to mean "install"
// followed by "manpage --install", two calls the user had to know about, so the
// man page — which is also what makes Alt-H answer natively in zsh and fish —
// was the half most people would never get.
func TestInstallAlsoInstallsTheManPage(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	res, err := installProgram(app, "bash", true, true)
	if err != nil {
		t.Fatal(err)
	}

	page := filepath.Join(home, ".local", "share", "man", "man1", "myapp.1")
	if res.ManPage != page {
		t.Errorf("InstallResult.ManPage = %q, want %q", res.ManPage, page)
	}
	body, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("install left no manual page: %v", err)
	}
	if !strings.Contains(string(body), ".TH") {
		t.Errorf("the installed page is not roff:\n%s", body)
	}
	if res.Integration == "" {
		t.Errorf("the shell integration was skipped")
	}
}

// ...and uninstall has to undo all of it, because that is what its report
// promises. A man page left behind after "uninstall" is the user's problem
// forever, since nothing tells them it is there.
func TestUninstallAlsoRemovesTheManPage(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	if _, err := installProgram(app, "bash", true, true); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(home, ".local", "share", "man", "man1", "myapp.1")
	if _, err := os.Stat(page); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	res, err := uninstallProgram(app, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(page); !os.IsNotExist(err) {
		t.Errorf("uninstall left the manual page behind")
	}
	var found bool
	for _, p := range res.Removed {
		if p == page {
			found = true
		}
	}
	if !found {
		t.Errorf("the removed manual page is not in the report: %v", res.Removed)
	}
}

// The man page is the secondary half: a machine where one is already installed
// system-wide must still get its shell integration, with the collision reported
// rather than raised.
func TestInstallSurvivesAManPageItCannotWrite(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	// A page of someone else's, where ours would go, is the collision clihelp
	// refuses to overwrite without --force.
	elsewhere := filepath.Join(home, "other", "man1")
	writeFixture(t, filepath.Join(elsewhere, "myapp.1"), ".TH MYAPP 1\n")
	t.Setenv("MANPATH", elsewhere+"/..")

	res, err := installProgram(app, "bash", true, true)
	if err != nil {
		t.Fatalf("a man-page collision failed the whole installation: %v", err)
	}
	if res.Integration == "" {
		t.Errorf("the shell integration was skipped")
	}
	if res.ManPage != "" {
		t.Errorf("a colliding page was reported as installed: %q", res.ManPage)
	}
}

// --no-man opts out, the way --no-keys does.
func TestInstallWithoutTheManPage(t *testing.T) {
	home := sandboxHome(t)
	app := installApp()

	res, err := installProgram(app, "bash", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.ManPage != "" {
		t.Errorf("--no-man still installed a page: %q", res.ManPage)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "man", "man1", "myapp.1")); !os.IsNotExist(err) {
		t.Errorf("--no-man wrote a manual page anyway")
	}
}

// The removal hint has to name a command that exists. "completion uninstall"
// only exists when the author mounted CompletionCommand(), which is the exact
// case __clihelp was added for — and one copy of the wrong hint goes into the
// user's startup file permanently.
func TestRemovalHintNamesACommandThatExists(t *testing.T) {
	bare := &App{Name: "myapp", Commands: []Command{{Name: "build", Run: func(*Context) error { return nil }}}}
	mounted := &App{Name: "myapp", Commands: []Command{CompletionCommand()}}

	if got := setupHint(bare); got != "myapp __clihelp" {
		t.Errorf("an app without CompletionCommand() is told %q", got)
	}
	if got := setupHint(mounted); got != "myapp completion" {
		t.Errorf("an app with CompletionCommand() is told %q", got)
	}

	for _, tt := range []struct {
		app  *App
		want string
	}{
		{bare, "__clihelp uninstall"},
		{mounted, "completion uninstall"},
	} {
		block := bootstrapBlock(tt.app, "bash", "/tmp/x")
		if !strings.Contains(block, tt.want) {
			t.Errorf("the startup-file block does not mention %q:\n%s", tt.want, block)
		}
	}
}
