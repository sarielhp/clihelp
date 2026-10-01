package clihelp

import (
	"bytes"
	"github.com/mattn/go-runewidth"
	"strings"
	"testing"
)

func TestUsageLineKeepsBracketedGroupsWhole(t *testing.T) {
	const usage = "podctl build [--output PATH] [--bitrate KBPS] [--[no-]normalize] [--tags TAGS] [--loudness LUFS] [--sample-rate HZ] <source-file>"
	tests := []struct {
		name  string
		width int
	}{
		{"width 80", 80}, {"width 60", 60}, {"width 45", 45},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{Name: "podctl", Commands: []Command{{Name: "build", Description: "Build", UsageLine: usage}}}
			var buf bytes.Buffer
			app.RenderCommand(Options{Writer: &buf, Width: tt.width, NoColor: true}, "build")
			page := buf.String()
			var rows []string
			for _, l := range strings.Split(page, "\n") {
				if strings.HasPrefix(l, "Usage:") || strings.HasPrefix(l, "        ") {
					rows = append(rows, strings.TrimSpace(strings.TrimPrefix(l, "Usage:")))
				}
			}
			if len(rows) < 2 {
				t.Fatalf("usage did not wrap at width %d:\n%s", tt.width, page)
			}
			for _, row := range rows {
				if strings.Count(row, "[")-strings.Count(row, "]") != 0 {
					t.Errorf("a row splits a bracketed group: %q\n%s", row, page)
				}
			}
			if got := strings.Join(strings.Fields(strings.Join(rows, " ")), " "); got != usage {
				t.Errorf("wrapping changed the text:\n got %q\nwant %q", got, usage)
			}
			if strings.Contains(page, groupSpace) {
				t.Errorf("the group sentinel leaked into the output:\n%q", page)
			}
		})
	}
}

func TestGlueGroups(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"flag and value", "a [--x V] b", "a [--x" + groupSpace + "V] b"},
		{"nested", "[--[no-]n v]", "[--[no-]n" + groupSpace + "v]"},
		{"angle", "<a b> c", "<a" + groupSpace + "b> c"},
		{"unbalanced opener", "a [b c", "a [b c"},
		{"too long is prose", "[" + strings.Repeat("word ", 12) + "]", "[" + strings.Repeat("word ", 12) + "]"},
		{"no groups", "plain text", "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := glueGroups(tt.in); got != tt.want {
				t.Errorf("glueGroups(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// go-runewidth measures some runes as two columns under East Asian width rules
// (RUNEWIDTH_EASTASIAN=1, or a CJK locale) and as one otherwise. The stand-in
// for a non-breaking space is swapped back to a one-column space after the
// wrap, so it has to be measured as one column under both settings — otherwise
// a glued group costs an extra column per space and lines wrap early, and the
// same help page is laid out differently depending on the user's locale.
func TestUsageWrapDoesNotDependOnEastAsianWidth(t *testing.T) {
	const usage = "x [aa bb cc dd] [ee ff gg hh] [ii jj kk]"
	render := func(eastAsian bool, width int) string {
		was := runewidth.DefaultCondition.EastAsianWidth
		runewidth.DefaultCondition.EastAsianWidth = eastAsian
		defer func() { runewidth.DefaultCondition.EastAsianWidth = was }()
		var buf bytes.Buffer
		writeUsage(&buf, defaultTheme(), Options{NoColor: true}, width, usage)
		return buf.String()
	}
	// Widths where a two-column sentinel moves a break: the groups are 13 and
	// 10 columns, so the cost of each glued space decides where the row ends.
	for _, width := range []int{24, 32, 33} {
		narrow, east := render(false, width), render(true, width)
		if narrow != east {
			t.Errorf("width %d wraps differently under East Asian width rules:\n--- narrow\n%s--- east asian\n%s", width, narrow, east)
		}
		if !strings.Contains(narrow, "\n") {
			t.Fatalf("width %d: the usage line did not wrap, so the test measures nothing:\n%s", width, narrow)
		}
	}
}
