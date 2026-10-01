package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

// The concise page (-h) is a prompt, held to a few lines. A blank line between
// every command row — added as soon as one row wraps — spent that budget on
// nothing: at 40–60 columns the root page listed four to six of seven commands.
// The hanging indent already separates wrapped rows; the extended page keeps its
// blank lines.
func TestConciseListingsHaveNoBlankSeparators(t *testing.T) {
	app := &App{Name: "podctl", Description: "A tool.", Commands: []Command{
		{Name: "build", Description: "Compile, encode and package raw audio into MP3 podcast episodes"},
		{Name: "serve", Description: "Serve feeds"},
		{Name: "config", Description: "View, inspect, set and manage application configuration settings"},
		{Name: "deploy", Description: "Deploy"},
	}}
	rows := func(o Options) (blank int, listed int) {
		var buf bytes.Buffer
		o.Writer = &buf
		app.RenderGlobal(o)
		in := false
		for _, l := range strings.Split(buf.String(), "\n") {
			switch {
			case l == "Commands:":
				in = true
			case in && strings.TrimSpace(l) == "":
				blank++
				in = false
			case in && l != "" && l[0] == ' ' && !strings.HasPrefix(l, "      "):
				listed++
			}
		}
		return
	}
	// "blank" counts the one blank that ends the section; a blank between rows
	// would have ended it early, leaving fewer rows counted than there are commands.
	for _, width := range []int{40, 60} {
		if _, listed := rows(Options{Width: width, NoColor: true, Concise: true, ConciseMaxLines: 40}); listed < 4 {
			t.Errorf("concise page at width %d lists only %d of 4 command rows before a blank line", width, listed)
		}
	}
	if _, listed := rows(Options{Width: 60, NoColor: true}); listed >= 4 {
		t.Errorf("the extended page lost its blank separators between wrapped rows (%d rows before the first blank)", listed)
	}
}

// The line that says how to get the rest is the one a truncated page cannot
// lose. At narrow widths it was cut through its own text, so the command it
// names ended mid-word ("run 'podctl help buil…") and the page's only pointer
// to the rest was unusable.
func TestTruncationNoteKeepsTheCommandItNames(t *testing.T) {
	var examples []Example
	for _, l := range []string{"podctl build a.wav", "podctl build b.wav", "podctl build c.wav", "podctl build d.wav"} {
		examples = append(examples, Example{Line: l, Description: "Build one of the episodes from its raw audio file"})
	}
	app := &App{Name: "podctl", Commands: []Command{{Name: "build", Description: "Build the episodes", Examples: examples}}}
	for _, width := range []int{30, 40, 60, 120} {
		t.Run(strings.Repeat("w", 1)+string(rune('0'+width/10%10))+string(rune('0'+width%10)), func(t *testing.T) {
			var buf bytes.Buffer
			app.RenderCommand(Options{Writer: &buf, Width: width, NoColor: true, Concise: true, ConciseMaxLines: 8}, "build")
			lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
			if len(lines) > 8 {
				t.Fatalf("%d lines, over the budget of 8:\n%s", len(lines), buf.String())
			}
			last := lines[len(lines)-1]
			if !strings.HasPrefix(last, "…") {
				t.Fatalf("the page was not truncated, so the test measures nothing:\n%s", buf.String())
			}
			if !strings.Contains(last, "podctl help build") {
				t.Errorf("width %d: the note does not keep the command it names: %q", width, last)
			}
			if visualLen(last) > width {
				t.Errorf("width %d: the note is %d columns wide: %q", width, visualLen(last), last)
			}
		})
	}
}
