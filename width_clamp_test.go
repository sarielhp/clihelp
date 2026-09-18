package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

// M15 — colIndent never looked at the available width, so a narrow terminal was
// left with almost nothing for the text: a 13-column flag name at width 20 left
// about three columns, and every word got its own line. Explain feeds the
// shell's real $COLUMNS, so a narrow pane reaches this.
func TestNarrowTerminalKeepsAUsableTextColumn(t *testing.T) {
	app := &App{Name: "probe",
		GlobalFlags: []Option{{Flags: "-v, --verbose", Description: "Verbose output logging enabled"}},
		Commands:    []Command{{Name: "c", Description: "C.", Run: func(*Context) error { return nil }}},
	}
	params := []Param{{Name: "-v, --verbose"}}
	for _, width := range []int{20, 24, 30, 40, 80} {
		indent := colIndentFor(params, width, minTextColumns)
		text := width - indent
		want := min(minTextColumns, width/2)
		if text < want {
			t.Errorf("width %d: the description column is %d wide, want at least %d",
				width, text, want)
		}
	}

	// And the whole render still fits the width it was given.
	for _, width := range []int{20, 40, 80} {
		var buf bytes.Buffer
		app.RenderGlobal(Options{Writer: &buf, Width: width})
		for _, line := range strings.Split(StripANSI(buf.String()), "\n") {
			if w := VisualWidth(line); w > width && len(strings.Fields(line)) > 1 {
				t.Errorf("width %d: a line is %d columns: %q", width, w, line)
			}
		}
	}
}

// L4 — the title was wrapped to maxContent *after* a two-column indent while
// the rule under it was drawn at maxContent exactly, so on a wide terminal the
// title overhung its own separator.
func TestTitleFitsInsideItsSeparator(t *testing.T) {
	app := &App{Name: "probe", Theme: &Theme{Separator: true},
		Commands: []Command{{Name: "c", Title: strings.Repeat("ab ", 40), Description: "x",
			Run: func(*Context) error { return nil }}}}
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Width: 200}, "c")

	rule := 0
	for _, line := range strings.Split(StripANSI(buf.String()), "\n") {
		if trimmed := strings.TrimSpace(line); len(trimmed) > 10 && strings.Trim(trimmed, "-=─") == "" {
			rule = VisualWidth(trimmed)
			break
		}
	}
	if rule == 0 {
		t.Fatal("the fixture is wrong: no separator rule was drawn")
	}
	for _, line := range strings.Split(StripANSI(buf.String()), "\n") {
		if w := VisualWidth(line); w > rule && strings.Contains(line, "ab") {
			t.Errorf("a title line is %d columns inside a %d-column rule: %q", w, rule, line)
		}
	}
}

// L5 — VisualWidth was not additive over a whitespace join for invalid UTF-8,
// and reflowWords accumulates per-word widths, so a stray byte pushed a line
// past the requested width.
func TestVisualWidthIsAdditiveOverAJoin(t *testing.T) {
	parts := []string{"00000\xf60", "000000000000000", "0"}
	sum := 0
	for _, p := range parts {
		sum += VisualWidth(p)
	}
	sum += len(parts) - 1 // the joining spaces
	if whole := VisualWidth(strings.Join(parts, " ")); whole != sum {
		t.Errorf("VisualWidth is not additive: parts sum to %d, the join measures %d", sum, whole)
	}
}
