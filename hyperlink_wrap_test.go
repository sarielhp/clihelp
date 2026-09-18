package clihelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// openOSC counts OSC 8 openers on a line that have no terminator.
func openOSC(line string) int {
	closers := strings.Count(line, osc8+oscEnd)
	return strings.Count(line, osc8) - closers - closers
}

func assertEveryLineIsBalanced(t *testing.T, out string) {
	t.Helper()
	requireRendered(t, out)
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if n := openOSC(line); n != 0 {
			t.Errorf("line %d leaves %d hyperlink(s) open: %q", i, n, line)
		}
	}
}

// H2 — a hyperlink whose text wraps put the opener on one line and the
// terminator on another. fatih/color closes each line with an SGR reset, which
// does not close a hyperlink, so any consumer that truncates by line drops the
// terminator and the terminal hyperlinks everything it prints afterwards —
// including the shell prompt, after the program has exited.
func TestWrappedHyperlinkIsClosedOnEveryLine(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := &App{
		Name:        "demo",
		Description: "Read [the complete operations manual for this command](https://example.com/m) before you begin.",
	}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 40})
	assertEveryLineIsBalanced(t, buf.String())
}

// The same thing through a note, where the author hard-wrapped the link across
// two source lines — the ordinary case, not a hostile one.
func TestHardWrappedHyperlinkIsClosed(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	var buf bytes.Buffer
	renderNoteContent(&buf, defaultTheme(), Options{}, 70, 2,
		Note{Heading: "Reference", Text: "Read the [reference](https://example.com/very/long/path/to/the/reference) first, it matters."})
	assertEveryLineIsBalanced(t, buf.String())
}

// The consequence that made this high severity: Alt-H truncates by line.
func TestExplainNeverLeavesAHyperlinkOpen(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := &App{Name: "demo", Commands: []Command{{
		Name: "sync",
		Description: "Synchronises everything. Before you run this read " +
			"[the complete operations manual for this command](https://example.com/manual) " +
			"and then read it a second time because it matters.",
		Run: func(*Context) error { return nil },
	}}}
	var buf bytes.Buffer
	app.Explain(&buf, "demo sync", 40, 6)
	if n := openOSC(buf.String()); n != 0 {
		t.Errorf("Explain left %d hyperlink(s) open:\n%q", n, buf.String())
	}
}

// A link that fits on one line must still be one link, not a reopened pair.
func TestUnwrappedHyperlinkIsUntouched(t *testing.T) {
	had := color.NoColor
	color.NoColor = false // this is a claim about the escape, so emit escapes
	defer func() { color.NoColor = had }()

	out := inline("see [docs](https://example.com/d) now")
	var buf bytes.Buffer
	reflow(&buf, defaultTheme().Body, 70, 0, "", out)
	if got := strings.Count(buf.String(), osc8); got != 2 {
		t.Errorf("a link that fits was rewritten: %d OSC introducers, want 2:\n%q", got, buf.String())
	}
	if !strings.Contains(StripANSI(buf.String()), "see docs now") {
		t.Errorf("visible text changed: %q", StripANSI(buf.String()))
	}
}

// M11 — the stripper could not match an OSC containing a newline, an
// unterminated OSC, or a CSI with private markers or sub-parameters. The first
// two are sequences clihelp's own output could contain.
func TestStripANSICoversWhatWeCanEmit(t *testing.T) {
	for _, tt := range []struct{ name, in, want string }{
		{"osc with a newline in the payload", "a" + osc8 + "http://x/a\nb" + oscEnd + "t" + osc8 + oscEnd + "z", "atz"},
		{"unterminated osc", "a" + osc8 + "http://x", "a"},
		{"csi private marker", "a\x1b[?25lb", "ab"},
		{"csi colon sub-parameters", "a\x1b[38:2::1:2:3mb\x1b[39m", "ab"},
		{"ordinary sgr", "a\x1b[31mb\x1b[0m", "ab"},
		{"truecolour sgr", "a\x1b[38;2;1;2;3mb\x1b[0;22;23m", "ab"},
		{"complete osc 8", "a" + osc8 + "http://x" + oscEnd + "b" + osc8 + oscEnd, "ab"},
		{"osc terminated by BEL", "a\x1b]0;title\x07b", "ab"},
	} {
		if got := StripANSI(tt.in); got != tt.want {
			t.Errorf("%s: StripANSI(%q) = %q, want %q (width %d)", tt.name, tt.in, got, tt.want, VisualWidth(tt.in))
		}
	}
}
