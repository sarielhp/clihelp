package clihelp

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// A tab advances to the next eight-column stop, which is what a terminal draws
// and so what every column in this library has to be measured against. Counting
// it as one column put a description one column out for every tab before it.
func TestTabsAdvanceToTheNextStop(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int
	}{
		{"\t", 8},
		{"a\tb", 9},
		{"abcdefg\th", 9},   // seven columns, one more to the stop, then "h"
		{"abcdefgh\ti", 17}, // exactly on a stop: a whole tab follows
		{"\t\t", 16},
		{"ab", 2},
	} {
		if got := VisualWidth(tt.in); got != tt.want {
			t.Errorf("VisualWidth(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// Whether a line may be broken is a fact about what is on it, not about how wide
// it came out: a word of zero display width leaves the two indistinguishable.
// The property that holds either way is that a line goes over its width only
// because one word was wider than the whole line.
func TestAnOverWideLineHoldsOnlyOneWord(t *testing.T) {
	for _, tt := range []struct {
		name          string
		width, indent int
		prefix, text  string
	}{
		{"a zero-width word before a long one", 10, 2, "", "\u200b verylongword12345 tail"},
		{"a zero-width word among short ones", 14, 2, "", "\u200b alpha bravo charlie"},
		{"a long word after a prefix", 12, 8, "--flag", "verylongword12345 tail"},
		{"ordinary prose", 24, 2, "", "alpha bravo charlie delta echo foxtrot golf"},
		{"a list", 20, 2, "", "- alpha bravo charlie delta echo"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reflow(&buf, color.New(color.FgWhite), tt.width, tt.indent, tt.prefix, tt.text)
			requireRendered(t, buf.String())
			for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
				if VisualWidth(line) <= tt.width {
					continue
				}
				if n := len(strings.Fields(StripANSI(line))); n > 1 {
					t.Errorf("a line of %d columns over a width of %d holds %d words, "+
						"so it could have been broken:\n%q",
						VisualWidth(line), tt.width, n, line)
				}
			}
		})
	}
}

// A bullet is a marker followed by a space. Without that rule "*emphasis* text"
// opens with a list marker, and a hanging indent is applied to a paragraph that
// is not a list at all.
func TestAListMarkerNeedsASpaceAfterIt(t *testing.T) {
	for _, tt := range []struct {
		in     string
		want   string
		isList bool
	}{
		{"- item", "- ", true},
		{"  - item", "  - ", true},
		{"* item", "* ", true},
		{"• item", "• ", true},
		{"1. item", "1. ", true},
		{"12.  item", "12.  ", true},
		{"-item", "", false},
		{"*emphasis* and more", "", false},
		{"*bold*", "", false},
		{"1.5 times", "", false},
		{"-", "", false},
	} {
		got, isList := detectListPrefix(tt.in)
		if isList != tt.isList || got != tt.want {
			t.Errorf("detectListPrefix(%q) = (%q, %v), want (%q, %v)",
				tt.in, got, isList, tt.want, tt.isList)
		}
	}
}

// clampIndent narrows the description column on a narrow terminal, and stops.
// Below about six columns the name column is too thin to hold a name, and a
// prefix that no longer fits takes its own line instead — which is the better
// of the two, and is already how a name wider than its column is handled.
func TestTheDescriptionColumnIsNeverClampedToNothing(t *testing.T) {
	for _, tt := range []struct {
		indent, termWidth, minText, want int
	}{
		{20, 0, 20, 20},  // no terminal width known: leave it alone
		{20, 8, 20, 20},  // too narrow to clamp usefully
		{20, 10, 20, 20}, // half is five, so the limit is below the floor
		{20, 40, 20, 20}, // it already fits
		{30, 40, 20, 20}, // clamped to leave the text its columns
		{20, 26, 20, 13}, // half is thirteen, so the difference is split
	} {
		got := clampIndent(tt.indent, tt.termWidth, tt.minText)
		if got != tt.want {
			t.Errorf("clampIndent(%d, %d, %d) = %d, want %d",
				tt.indent, tt.termWidth, tt.minText, got, tt.want)
		}
		if tt.termWidth > 0 && got < 6 {
			t.Errorf("clampIndent(%d, %d, %d) = %d, a column too thin to hold a name",
				tt.indent, tt.termWidth, tt.minText, got)
		}
	}
}

// Help written to a pipe has no terminal to ask, and the width it falls back to
// is what every redirected --help in every program built on this library is laid
// out at.
func TestRedirectedHelpUsesTheFixedWidth(t *testing.T) {
	var probe bytes.Buffer
	if got := (Options{Writer: &probe}).width(); got != 70 {
		t.Errorf("a non-terminal writer laid out at %d columns, want 70", got)
	}

	app := &App{
		Name:        "widthcli",
		Description: strings.Repeat("A sentence that has to be wrapped somewhere. ", 8),
		Commands: []Command{{
			Name:        "build",
			Description: strings.Repeat("Another long description that must wrap. ", 6),
			Run:         func(*Context) error { return nil },
		}},
	}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf})
	for _, line := range strings.Split(buf.String(), "\n") {
		if got := VisualWidth(line); got > 70 && len(strings.Fields(StripANSI(line))) > 1 {
			t.Errorf("a redirected help line ran to %d columns:\n%q", got, line)
		}
	}
}

// App.NoColor is documented as turning off every escape this library emits, for
// every render the application performs — not only the renders whose Options
// happen to repeat it.
func TestAppNoColorReachesEveryRender(t *testing.T) {
	// fatih/color turns itself off when stdout is not a terminal, which is every
	// test run — so with it left alone this asserts nothing. Colour is forced on
	// for the duration, the way nocolor_test.go forces it off.
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := &App{
		Name:        "quiet",
		NoColor:     true,
		Description: "A `code span`, some **bold** and a [link](https://example.com).",
		Commands: []Command{{
			Name:        "build",
			Description: "Build **it**.",
			Options:     []Option{{Flags: "--fast", Description: "Go *fast*."}},
			Run:         func(*Context) error { return nil },
		}},
	}
	for _, tt := range []struct {
		name string
		call func(w io.Writer)
	}{
		{"global", func(w io.Writer) { app.RenderGlobal(Options{Writer: w}) }},
		{"command", func(w io.Writer) { app.RenderCommand(Options{Writer: w}, "build") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.call(&buf)
			if strings.ContainsRune(buf.String(), 0x1b) {
				t.Errorf("App.NoColor was set and the render still emitted escapes:\n%q", buf.String())
			}
		})
	}
}

// A default the author already wrote into the description is not repeated. The
// check exists because "(default: 3) (default: 3)" is what it produced.
func TestADefaultAlreadyInTheTextIsNotRepeated(t *testing.T) {
	for _, tt := range []struct {
		desc, defaultText, want string
	}{
		{"Level to use.", "3", "Level to use. (default: 3)"},
		{"Level to use (default: 3).", "3", "Level to use (default: 3)."},
		{"Level to use [default 3].", "3", "Level to use [default 3]."},
		{"Level to use.", "", "Level to use."},
	} {
		got := decorateOptionDescription(Option{Description: tt.desc, DefaultText: tt.defaultText})
		if got != tt.want {
			t.Errorf("decorateOptionDescription(%q, %q) = %q, want %q",
				tt.desc, tt.defaultText, got, tt.want)
		}
		if strings.Count(got, "default") > 1 {
			t.Errorf("the default was stated twice: %q", got)
		}
	}
}

// The concise tier replaces what did not fit with one line saying so. Output
// that exactly fills the budget did fit, and must be passed through untouched:
// an off-by-one here spends a line of a short terminal's budget announcing that
// nothing was dropped.
func TestOutputExactlyAtTheBudgetIsNotTruncated(t *testing.T) {
	app := &App{Name: "budgetcli"}
	write := func(n int) func(io.Writer) {
		return func(w io.Writer) { _, _ = io.WriteString(w, strings.Repeat("line\n", n)) }
	}
	for _, n := range []int{1, 3, 8} {
		for _, tail := range []string{"", "\n"} {
			var buf bytes.Buffer
			body := strings.Repeat("line\n", n) + tail
			app.budgeted(&buf, Options{Concise: true, ConciseMaxLines: n}, nil,
				func(w io.Writer) { _, _ = io.WriteString(w, body) })
			// Including the blank line a help page ends with: that spacing is
			// deliberate, and going through the truncating path trims it.
			if buf.String() != body {
				t.Errorf("%d lines against a budget of %d were altered:\ngot  %q\nwant %q",
					n, n, buf.String(), body)
			}
		}
	}
	var buf bytes.Buffer
	app.budgeted(&buf, Options{Concise: true, ConciseMaxLines: 3}, nil, write(6))
	if buf.String() == strings.Repeat("line\n", 6) {
		t.Error("output over the budget was passed through unchanged")
	}
}

// Emphasis markers bind to text, not to the space around it: "a ** b" is two
// asterisks the author typed, not the start of bold. Without the rule, prose
// containing a bare "**" or a multiplication sign turned everything up to the
// next one bold.
func TestEmphasisDoesNotBindAcrossSpaces(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"a ** b ** c", "a ** b ** c"},
		{"2 * 3 * 4", "2 * 3 * 4"},
		{"**a **", "**a **"},
		{"** a**", "** a**"},
		{"a **b** c", "a b c"},
		{"a *b* c", "a b c"},
	} {
		if got := StripANSI(renderInlineTo(tt.in, false)); got != tt.want {
			t.Errorf("inlineMarkdown(%q) rendered as %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A terminal shorter than the concise tier's own bound gets one line less than
// its height, so that the shell's next prompt does not scroll the first line
// away. height() returns zero for every writer a test can hand it, so this is
// asserted against the decision rather than through a render.
func TestConciseBudgetShrinksToAShortTerminal(t *testing.T) {
	for _, tt := range []struct {
		configured, height, want int
	}{
		{0, 0, conciseHelpLines},   // not a terminal: the tier's own bound
		{0, 100, conciseHelpLines}, // plenty of room
		{0, 8, 7},                  // shorter than the bound: height less one
		{0, 2, 1},                  // very short, but still a line
		{0, 1, 0},                  // degenerate: budgeted renders through instead
		{5, 8, 5},                  // an explicit budget wins over the height
		{99, 8, 99},                // including one larger than the terminal
		{conciseHelpLines + 3, 0, conciseHelpLines + 3},
	} {
		if got := conciseBudgetFor(tt.configured, tt.height); got != tt.want {
			t.Errorf("conciseBudgetFor(%d, %d) = %d, want %d",
				tt.configured, tt.height, got, tt.want)
		}
	}
}

// A name too wide for its column takes a line of its own, with the description
// beginning on the next line at the column. Padding it in place would push every
// description on that row out of alignment, or run the two together.
//
// This and TestAnUngroupedListGetsNoHeading are the two behaviours that
// example/mail_cli_fake's oracle was the only guard for. Retiring the oracle —
// 499 lines reproducing the renderer to compare against it — cost exactly these
// two mutations out of 38, which is what the survey was run to find out.
func TestAnOverWideNameTakesItsOwnLine(t *testing.T) {
	const width, indent = 60, 12
	var buf bytes.Buffer
	long := "--an-extremely-long-flag-name-far-past-its-column"
	reflow(&buf, color.New(color.FgWhite), width, indent, long,
		"The description belongs on the next line, at the column.")
	lines := strings.Split(strings.TrimRight(StripANSI(buf.String()), "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("the name and its description shared one line:\n%q", buf.String())
	}
	if strings.TrimSpace(lines[0]) != long {
		t.Errorf("the first line is not the name alone: %q", lines[0])
	}
	if strings.Contains(lines[0], "description") {
		t.Errorf("the description was run onto the name's line: %q", lines[0])
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if got := len(line) - len(strings.TrimLeft(line, " ")); got != indent {
			t.Errorf("a continuation line starts at column %d, not the column %d: %q",
				got, indent, line)
		}
	}

	// A name that does fit keeps its description beside it.
	buf.Reset()
	reflow(&buf, color.New(color.FgWhite), width, indent, "-s", "Short.")
	if first := strings.Split(StripANSI(buf.String()), "\n")[0]; !strings.Contains(first, "Short.") {
		t.Errorf("a name that fits was given its own line anyway: %q", first)
	}
}

// Group headings appear only when something is grouped. Giving every entry a
// fallback heading when none is grouped puts a heading above a list that has no
// groups in it — a section title for a section that is the whole list.
func TestAnUngroupedListGetsNoHeading(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   []string
		want []string
	}{
		{"nothing grouped", []string{"", "", ""}, []string{"", "", ""}},
		{"all grouped", []string{"A", "B"}, []string{"A", "B"}},
		{"some grouped", []string{"A", "", "B"}, []string{"A", "Other", "B"}},
		{"one grouped", []string{"", "A", ""}, []string{"Other", "A", "Other"}},
		{"empty", nil, nil},
	} {
		got := normalizeGroups(tt.in, "Other")
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Errorf("%s: normalizeGroups(%v) = %v, want %v", tt.name, tt.in, got, tt.want)
		}
	}

	// And through a render: an application whose commands are all ungrouped
	// shows no heading above them.
	app := &App{
		Name: "plain",
		Commands: []Command{
			{Name: "build", Description: "Build it.", Run: func(*Context) error { return nil }},
			{Name: "clean", Description: "Clean it.", Run: func(*Context) error { return nil }},
		},
	}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 80})
	body := StripANSI(buf.String())
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Other" || trimmed == "Other:" {
			t.Errorf("a fallback group heading appeared above an ungrouped list:\n%s", body)
		}
	}
}

// A name far wider than the column is left out of the column's calculation
// rather than allowed to set it. One 50-column flag among a dozen short ones
// would otherwise push every description half a screen to the right, to line up
// with a name that takes its own line anyway.
//
// The third and last behaviour example/mail_cli_fake's oracle was the only guard
// for.
func TestAnOverWideNameDoesNotWidenTheColumn(t *testing.T) {
	short := []Param{{Name: "-v"}, {Name: "--verbose"}}
	base := colIndent(short)
	if base > defaultMaxColIndent {
		t.Fatalf("a column of short names is already over the cap: %d", base)
	}

	huge := Param{Name: "--an-extremely-long-flag-name-past-any-reasonable-column"}
	withHuge := colIndent(append(append([]Param{}, short...), huge))
	if withHuge != base {
		t.Errorf("one over-wide name moved the column from %d to %d; it should be "+
			"ignored and take its own line instead", base, withHuge)
	}
	if withHuge > defaultMaxColIndent {
		t.Errorf("the column reached %d, past the %d cap", withHuge, defaultMaxColIndent)
	}

	// A list of nothing but over-wide names falls back to the cap rather than to
	// zero, so the descriptions still have a column to start at.
	if only := colIndent([]Param{huge}); only != defaultMaxColIndent {
		t.Errorf("a list of only over-wide names gave a column of %d, want the cap %d",
			only, defaultMaxColIndent)
	}
}
