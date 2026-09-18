package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func manApp() *App {
	return &App{
		Name:        "manapp",
		Version:     "2.1.0",
		Description: "**manapp** — Does the [thing](https://example.com/thing).",
		GlobalNote:  "Longer prose.\n\nA second paragraph.",
		GlobalFlags: []Option{{Flags: "-v, --verbose", Description: "Say more"}},
		Commands: []Command{{
			Name:        "build",
			UsageLine:   "manapp build [options] <file>",
			Description: "Build it.",
			Parameters:  []Param{{Name: "<file>", Description: "The file"}},
			Options:     []Option{{Flags: "-o, --output <path>", Description: "Where to write"}},
			Examples:    []Example{{Line: "manapp build a.txt", Description: "Build a.txt"}},
			Notes:       []Note{{Heading: "Caveat", Text: "Mind the gap."}},
			Run:         func(*Context) error { return nil },
		}},
	}
}

func TestGenManPageStructure(t *testing.T) {
	var b strings.Builder
	if err := GenManPage(manApp(), &b); err != nil {
		t.Fatal(err)
	}
	page := b.String()

	for _, want := range []string{
		".\\\" " + manMarker + ": ",
		`.TH "MANAPP" 1 `,
		`"manapp 2.1.0"`, // the version is a quoted .TH argument
		".SH NAME\n",
		".SH SYNOPSIS\n",
		".SH DESCRIPTION\n",
		".SH COMMANDS\n",
		".SS manapp build\n",
		".SH GLOBAL OPTIONS\n",
		".SH EXAMPLES\n",
		".SH NOTES\n",
		".B -v, --verbose\n",
		".I <file>\n",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page is missing %q", want)
		}
	}

	// NAME is "name - summary", with the program's own name trimmed from the
	// summary and no URL crowding the line.
	name := manSectionBody(page, ".SH NAME")
	if !strings.HasPrefix(name, "manapp \\- Does the thing") {
		t.Errorf("NAME line = %q", name)
	}
	if strings.Contains(name, "https://") {
		t.Errorf("the NAME line carries a URL: %q", name)
	}
	// The URL belongs in DESCRIPTION, where there is room for it.
	if !strings.Contains(manSectionBody(page, ".SH DESCRIPTION"), "https://example.com/thing") {
		t.Errorf("DESCRIPTION dropped the link target")
	}

	if strings.Contains(page, "\x1b") {
		t.Errorf("the page contains ANSI escapes")
	}
	if strings.Contains(page, "**") {
		t.Errorf("the page contains unrendered markdown")
	}
}

// manSectionBody returns the first line of a section's body, skipping the
// formatting requests that follow the heading.
func manSectionBody(page, heading string) string {
	rest := page[strings.Index(page, heading+"\n")+len(heading)+1:]
	for _, line := range strings.Split(rest, "\n") {
		if line != ".ad l" && line != ".nh" {
			return line
		}
	}
	return ""
}

func TestManEscaping(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{`C:\temp\x`, `C:\etemp\ex`},
		{".TH is not a heading", `\&.TH is not a heading`},
		{"'quoted at the margin", `\&'quoted at the margin`},
		{"an em\u2014dash", `an em\(emdash`},
		{"plain text", "plain text"},
	} {
		if got := manEscape(tt.in); got != tt.want {
			t.Errorf("manEscape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// And a description that begins at the margin with a request is protected
	// all the way through the generator.
	app := manApp()
	app.Commands[0].Description = ".TH is not a heading here"
	var b strings.Builder
	if err := GenManPage(app, &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `\&.TH is not a heading`) {
		t.Errorf("a line starting with '.' was not protected:\n%s", b.String())
	}
}

// The page has to satisfy the program that will read it.
func TestLiveManRendersThePageWithoutWarnings(t *testing.T) {
	manPath, err := exec.LookPath("man")
	if err != nil {
		t.Skip("man not found, skipping")
	}

	dir := t.TempDir()
	page := filepath.Join(dir, "podctl.1")
	bin := filepath.Join(dir, "podctl")
	if out, err := exec.Command("go", "build", "-o", bin, "./example").CombinedOutput(); err != nil {
		t.Fatalf("failed to build the example CLI: %v\n%s", err, out)
	}
	roff, err := sandboxedCommand(t, bin, "__clihelp", "manpage").Output()
	if err != nil {
		t.Fatalf("generating the page failed: %v", err)
	}
	if err := os.WriteFile(page, roff, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := sandboxedCommand(t, manPath, page)
	cmd.Env = append(cmd.Env, "MANPAGER=cat", "MANWIDTH=80")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("man failed to render the page: %v\n%s", err, stderr.String())
	}
	if warnings := strings.TrimSpace(stderr.String()); warnings != "" {
		t.Errorf("man warned while rendering:\n%s", warnings)
	}
	for _, want := range []string{"PODCTL(1)", "NAME", "SYNOPSIS", "COMMANDS", "podctl build"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the rendered page is missing %q", want)
		}
	}
}

func TestInstallManPage(t *testing.T) {
	home := sandboxHome(t)
	t.Setenv("MANPATH", filepath.Join(home, ".local", "share", "man"))

	path, err := InstallManPage(manApp(), false)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".local", "share", "man", "man1", "manapp.1")
	if path != want {
		t.Errorf("installed to %q, want %q", path, want)
	}
	if !isGeneratedManPage(path) || !manPageIsCurrent(path) {
		t.Errorf("the installed page is not recognized as ours and current")
	}

	// Installing again is a refresh, not a refusal: our own page is not a
	// collision with itself.
	if _, err := InstallManPage(manApp(), false); err != nil {
		t.Errorf("reinstalling refused: %v", err)
	}

	removed, err := UninstallManPage(manApp())
	if err != nil {
		t.Fatal(err)
	}
	if removed != want {
		t.Errorf("uninstall removed %q, want %q", removed, want)
	}
}

func TestInstallManPageRefusesToShadowAnother(t *testing.T) {
	if _, err := exec.LookPath("man"); err != nil {
		t.Skip("man not found, skipping")
	}
	home := sandboxHome(t)

	// A manual page for this program, installed by something else.
	other := filepath.Join(home, "other", "man1")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "manapp.1"), []byte(".TH MANAPP 1\n.SH NAME\nmanapp \\- packaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MANPATH", filepath.Join(home, "other")+":"+filepath.Join(home, ".local", "share", "man"))

	if _, err := InstallManPage(manApp(), false); err == nil {
		t.Errorf("installed over an existing manual page without being forced")
	} else if !strings.Contains(err.Error(), "--force") {
		t.Errorf("the refusal should say how to override it: %v", err)
	}

	if _, err := InstallManPage(manApp(), true); err != nil {
		t.Errorf("--force was refused: %v", err)
	}
}

func TestUninstallLeavesAForeignPageAlone(t *testing.T) {
	home := sandboxHome(t)
	path := filepath.Join(home, ".local", "share", "man", "man1", "manapp.1")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(".TH MANAPP 1\n.SH NAME\nmanapp \\- hand written\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := UninstallManPage(manApp())
	if err != nil {
		t.Fatal(err)
	}
	if removed != "" {
		t.Errorf("removed a page clihelp never generated: %q", removed)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the hand-written page is gone: %v", err)
	}
}

// The same rule as the shell integration: refresh what is installed, never
// install something nobody asked for.
func TestManPageRefreshedButNeverCreated(t *testing.T) {
	home := sandboxHome(t)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("MANPATH", filepath.Join(home, ".local", "share", "man"))
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm")

	app := manApp()
	app.AutoInstallCompletion = true
	path := filepath.Join(home, ".local", "share", "man", "man1", "manapp.1")

	TestExecute(app, []string{"build"}).AssertNoError(t)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("a manual page was installed unasked")
	}

	if _, err := InstallManPage(app, false); err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(path)
	stale := strings.Replace(string(current), manMarker+": 1", manMarker+": 0", 1)
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	TestExecute(app, []string{"build"}).AssertNoError(t)
	if !manPageIsCurrent(path) {
		t.Errorf("a stale generated page was not refreshed")
	}
}

// .TH's arguments were formatted with Go's %q, which emits \" — roff's
// comment-to-end-of-line. A version string carrying a quote therefore truncated
// the footer and silently dropped the section argument, and man says nothing, so
// the live test that asserts empty stderr passed on a corrupted page.
func TestManPageHeaderSurvivesAHostileVersion(t *testing.T) {
	manPath, err := exec.LookPath("man")
	if err != nil {
		t.Skip("man not found, skipping")
	}
	app := manApp()
	app.Version = `1.0"beta\x`

	var b strings.Builder
	if err := GenManPage(app, &b); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "manapp.1")
	if err := os.WriteFile(page, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := sandboxedCommand(t, manPath, page)
	cmd.Env = append(cmd.Env, "MANPAGER=cat", "MANWIDTH=100")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("man failed: %v\n%s", err, stderr.String())
	}
	if w := strings.TrimSpace(stderr.String()); w != "" {
		t.Errorf("man warned:\n%s", w)
	}
	// The fifth .TH argument must survive: when it is eaten, groff falls back to
	// its own default and the page silently loses its section title.
	if !strings.Contains(string(out), "User Commands") {
		t.Errorf("the .TH section argument was lost:\n%s", firstLines(string(out), 3))
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
