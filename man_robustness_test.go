package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// SOURCE_DATE_EPOCH is the reproducible-builds convention. Without it the .TH
// line carried whatever today was, so two builds of the same program produced
// different bytes and a distribution could not reproduce a package.
func TestManPageDateIsReproducible(t *testing.T) {
	app := &App{Name: "myapp", Version: "1.0", Description: "A program."}

	t.Setenv("SOURCE_DATE_EPOCH", "1700000000") // 2023-11-14 UTC
	var first strings.Builder
	if err := GenManPage(app, &first); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.String(), "2023-11-14") {
		t.Errorf("SOURCE_DATE_EPOCH was ignored:\n%s", firstLines(first.String(), 3))
	}

	var second strings.Builder
	if err := GenManPage(app, &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Errorf("two runs with the same SOURCE_DATE_EPOCH produced different pages")
	}

	// Nonsense is ignored rather than fatal, and today's date is used.
	t.Setenv("SOURCE_DATE_EPOCH", "not-a-number")
	var third strings.Builder
	if err := GenManPage(app, &third); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(third.String(), time.Now().Format("2006-01-02")) {
		t.Errorf("an unparsable SOURCE_DATE_EPOCH did not fall back to today:\n%s", firstLines(third.String(), 3))
	}
}

// man prints the path it resolved, so clihelp's own page came back under a
// different spelling — a symlinked $XDG_DATA_HOME is enough, as is /home versus
// /export/home — and the install then refused, reporting a collision with
// itself and telling the user to pass --force to overwrite their own file.
func TestManPageElsewhereIgnoresOurOwnPageUnderAnotherName(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(home, "real", "man1")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(real, "myapp.1")
	if err := os.WriteFile(target, []byte(".TH MYAPP 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(home, "link")
	if err := os.Symlink(filepath.Join(home, "real"), linkDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	viaLink := filepath.Join(linkDir, "man1", "myapp.1")

	// A stand-in man(1) that answers with the other spelling of the same file.
	stub := filepath.Join(home, "bin")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' " + shellQuoteForTest(viaLink) + "\n"
	if err := os.WriteFile(filepath.Join(stub, "man"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))

	app := &App{Name: "myapp"}
	if other, found := manPageElsewhere(app, target); found {
		t.Errorf("our own page, named through a symlink, was reported as a collision at %s", other)
	}

	// A genuinely different page is still reported.
	systemPage := filepath.Join(home, "usr", "man1", "myapp.1")
	writeFixture(t, systemPage, ".TH MYAPP 1\n")
	script = "#!/bin/sh\nprintf '%s\\n' " + shellQuoteForTest(systemPage) + "\n"
	if err := os.WriteFile(filepath.Join(stub, "man"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, found := manPageElsewhere(app, target); !found {
		t.Errorf("a page installed elsewhere was not reported")
	}
}

func shellQuoteForTest(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// The one external command this library runs is bounded, so an unreachable
// MANPATH entry cannot hang a user's program.
func TestManLookupHasADeadline(t *testing.T) {
	if manLookupTimeout <= 0 {
		t.Fatal("man -w runs with no timeout")
	}
	if manLookupTimeout > 10*time.Second {
		t.Errorf("manLookupTimeout is %s, long enough to feel like a hang", manLookupTimeout)
	}
}
