package clihelp

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

func plainColor() *color.Color {
	c := color.New()
	c.DisableColor()
	return c
}

// L1 — a first word that does not fit flushed a line holding only the indent or
// the padded prefix, and a zero-width word left the next one glued to it,
// because both tests used width as a proxy for "is anything on this line yet".
// tree/tree.go already solves this with a boolean.
func TestNoWhitespaceOnlyLines(t *testing.T) {
	var buf bytes.Buffer
	reflow(&buf, plainColor(), 40, 2, "", "https://example.com/an/extremely/long/path/that/never/fits and more")
	requireRendered(t, buf.String())
	for i, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if line != "" && strings.TrimSpace(line) == "" {
			t.Errorf("line %d is %d columns of whitespace and nothing else", i, len(line))
		}
		if strings.HasSuffix(line, " ") {
			t.Errorf("line %d has trailing whitespace: %q", i, line)
		}
	}
}

func TestZeroWidthWordDoesNotGlueTheNext(t *testing.T) {
	var buf bytes.Buffer
	reflow(&buf, plainColor(), 40, 2, "", "\x1b[0m alpha beta")
	if strings.Contains(StripANSI(buf.String()), "alphabeta") {
		t.Errorf("a zero-width word swallowed the following space: %q", StripANSI(buf.String()))
	}
}

// L2 — a whitespace-only line lost the paragraph break, and the blank lines that
// did survive carried the indent as trailing spaces.
func TestWhitespaceOnlyLineIsStillAParagraphBreak(t *testing.T) {
	var withBlank, withSpaces bytes.Buffer
	reflow(&withBlank, plainColor(), 70, 2, "", "para one\n\npara two")
	reflow(&withSpaces, plainColor(), 70, 2, "", "para one\n   \npara two")
	if withBlank.String() != withSpaces.String() {
		t.Errorf("a line of spaces is not treated as a paragraph break:\n%q\n%q",
			withBlank.String(), withSpaces.String())
	}
}

// L3 — the heading was printed from the unfiltered count, so a section whose
// every entry is hidden printed a heading and nothing under it.
func TestNoHeadingWithoutRows(t *testing.T) {
	app := &App{Name: "probe",
		Commands:  []Command{{Name: "c", Description: "C.", Run: func(*Context) error { return nil }}},
		Shortcuts: []Command{{Name: "s", Description: "S.", Hidden: true}},
	}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 70})
	if strings.Contains(StripANSI(buf.String()), "Shortcut Commands:") {
		t.Errorf("a heading was printed with no rows under it:\n%s", StripANSI(buf.String()))
	}
}

// L7 — the prefix column assumes one line. A newline in a Param.Name split it
// across two rows while the column arithmetic was computed on the whole thing —
// the same class as the tree/firstSentence bug already fixed.
func TestPrefixIsAlwaysOneLine(t *testing.T) {
	var buf bytes.Buffer
	reflow(&buf, plainColor(), 58, 20, "two\nlines", "the description")
	first := strings.SplitN(buf.String(), "\n", 2)[0]
	if strings.TrimSpace(StripANSI(first)) == "" {
		t.Errorf("a multi-line prefix produced an empty first row: %q", buf.String())
	}
	if strings.Count(strings.TrimRight(buf.String(), "\n"), "\n") > 1 {
		t.Errorf("a multi-line prefix was not collapsed: %q", buf.String())
	}
}

// M14 — the paren scan ran to the end of the string for every "[", so a
// description full of broken links cost O(n²). A truncated translation
// catalogue entry looks exactly like this.
func TestLinkScanIsNotQuadratic(t *testing.T) {
	measure := func(n int) time.Duration {
		s := strings.Repeat("[a](", n)
		start := time.Now()
		_ = inlineMarkdown(s)
		return time.Since(start)
	}
	small := measure(8000)
	big := measure(32000)
	if big > 8*small+50*time.Millisecond {
		t.Errorf("quadratic: 8000 took %v, 32000 took %v", small, big)
	}
}

// L6 — a tab measured zero columns while a terminal advances to the next stop,
// so the hanging indent and the prefix column desynchronised from what is drawn.
func TestTabsAreMeasuredAsDrawn(t *testing.T) {
	if got := VisualWidth("\t"); got == 0 {
		t.Errorf("a tab measures 0 columns; a terminal advances to the next stop")
	}
	var buf bytes.Buffer
	reflow(&buf, plainColor(), 40, 2, "-\tone", "two three")
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.ContainsRune(line, '\t') {
			t.Errorf("a tab reached the output, where its width is the terminal's guess: %q", line)
		}
	}
}
