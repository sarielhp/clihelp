package clihelp

import (
	"bytes"
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
		{"flag and value", "a [--x V] b", "a [--x\ue000V] b"},
		{"nested", "[--[no-]n v]", "[--[no-]n\ue000v]"},
		{"angle", "<a b> c", "<a\ue000b> c"},
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
