package clihelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// M7 — every theme colour vanishes when fatih/color decides output is not a
// terminal, but the inline renderer had its own hardcoded escapes and kept
// emitting them. A redirected help file was not plain text.
func TestNoColorSuppressesInlineEscapes(t *testing.T) {
	had := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = had }()

	app := &App{Name: "demo", Commands: []Command{{
		Name:        "sync",
		Description: "Runs `sync --all`; see [the docs](https://example.com/d); **always** check *first*.",
		Run:         func(*Context) error { return nil },
	}}}
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Width: 100}, "sync")
	if strings.ContainsRune(buf.String(), '\x1b') {
		t.Errorf("escapes emitted with color.NoColor set:\n%q", buf.String())
	}
	// The label is what the author wrote for a human to read; a bare URL in help
	// output is exactly what OSC 8 exists to avoid, and
	// TestExampleAppNoBareMarkdownAndNoVisibleURLs says so. A manual page asks
	// for the spelled-out form explicitly, and only man.go does.
	if strings.Contains(buf.String(), "https://example.com/d") {
		t.Errorf("a bare URL leaked into plain-text help:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "the docs") {
		t.Errorf("the link label was lost:\n%s", buf.String())
	}
}

// ...and an application can decline colour for one render without touching the
// process-wide switch, which is what --no-color needs and could not have.
func TestOptionsNoColorIsPerRender(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := &App{Name: "demo", Commands: []Command{{
		Name: "sync", Description: "Runs **it** now.", Run: func(*Context) error { return nil },
	}}}

	var coloured, plain bytes.Buffer
	app.RenderCommand(Options{Writer: &coloured, Width: 80}, "sync")
	app.RenderCommand(Options{Writer: &plain, Width: 80, NoColor: true}, "sync")

	if !strings.ContainsRune(coloured.String(), '\x1b') {
		t.Fatalf("the fixture is wrong: no colour was emitted at all")
	}
	if strings.ContainsRune(plain.String(), '\x1b') {
		t.Errorf("Options.NoColor still emitted escapes:\n%q", plain.String())
	}
	if StripANSI(coloured.String()) == plain.String() {
		return // identical once stripped, which is the point
	}
	t.Errorf("NoColor changed more than the colour:\n%q\n%q", StripANSI(coloured.String()), plain.String())
}
