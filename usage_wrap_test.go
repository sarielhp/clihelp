package clihelp

import (
	"bytes"
	"fmt"
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

// Usage lines are drawn by three renderers — the root page, a command page and
// `help flags` — and the grouping fix was only exercised on one of them; reverting
// either of the other two passed the whole suite.
func TestEveryUsageRendererKeepsGroupsWhole(t *testing.T) {
	const usage = "podctl [--output PATH] [--bitrate KBPS] [--[no-]normalize] [--tags TAGS] [--loudness LUFS] <source-file>"
	app := &App{
		Name:      "podctl",
		UsageLine: usage,
		Commands:  []Command{{Name: "build", Description: "Build", UsageLine: usage}},
	}
	renderers := map[string]func(*bytes.Buffer, int){
		"root page":    func(b *bytes.Buffer, w int) { app.RenderGlobal(Options{Writer: b, Width: w, NoColor: true}) },
		"command page": func(b *bytes.Buffer, w int) { app.RenderCommand(Options{Writer: b, Width: w, NoColor: true}, "build") },
		"help flags":   func(b *bytes.Buffer, w int) { app.renderFlagsPage(Options{Writer: b, Width: w, NoColor: true}) },
	}
	for name, render := range renderers {
		for _, width := range []int{45, 60, 120} {
			t.Run(fmt.Sprintf("%s/%d", name, width), func(t *testing.T) {
				var buf bytes.Buffer
				render(&buf, width)
				var rows []string
				inUsage := false
				for _, l := range strings.Split(buf.String(), "\n") {
					switch {
					case strings.HasPrefix(l, "Usage:"):
						inUsage = true
						rows = append(rows, l)
					case inUsage && strings.HasPrefix(l, "        "):
						rows = append(rows, l)
					default:
						inUsage = false
					}
				}
				if len(rows) == 0 {
					t.Fatalf("no usage line on the %s:\n%s", name, buf.String())
				}
				for _, row := range rows {
					if strings.Count(row, "[")-strings.Count(row, "]") != 0 {
						t.Errorf("a row splits a bracketed group: %q", row)
					}
					if limit := wrapWidth(width, 8, 80); visualLen(row) > limit {
						t.Errorf("usage row is %d columns, past the %d it was wrapped to: %q", visualLen(row), limit, row)
					}
				}
			})
		}
	}
}

func TestGlueGroupsBoundary(t *testing.T) {
	// A group is glued when its closing bracket is at most maxGluedGroup runes
	// past its opening one.
	group := func(span int) string { return "[a " + strings.Repeat("x", span-3) + "]" }
	tests := []struct {
		name string
		span int
		glue bool
	}{
		{"exactly at the limit", maxGluedGroup, true},
		{"one past the limit", maxGluedGroup + 1, false},
		{"well inside", 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := group(tt.span)
			got := glueGroups(in)
			if glued := strings.Contains(got, groupSpace); glued != tt.glue {
				t.Errorf("glueGroups of a %d-rune group: glued = %v, want %v", tt.span, glued, tt.glue)
			}
		})
	}
}
