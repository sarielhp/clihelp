package clihelp

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// Two-column listings are measured in display columns, but the padding was done
// by rune count, so a wide (CJK) name pushed its description out of line with
// every other row — the 2026-09-17 review's formatPrefix finding.
func TestTwoColumnListingAlignsWideNames(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{Name: "build", Description: "Build it"},
			{Name: "検索", Description: "Search it"},
			{Name: "deploy", Description: "Deploy it"},
		},
	}
	o, buf := captureOptions(100)
	app.RenderGlobal(o)

	columns := map[string]int{}
	for _, line := range strings.Split(strip(buf.String()), "\n") {
		for _, cmd := range app.Commands {
			prefix := "  " + cmd.Name + " "
			if strings.HasPrefix(line, prefix) {
				idx := strings.Index(line, cmd.Description)
				if idx < 0 {
					t.Fatalf("description of %q not on its row: %q", cmd.Name, line)
				}
				columns[cmd.Name] = runewidth.StringWidth(line[:idx])
			}
		}
	}
	if len(columns) != len(app.Commands) {
		t.Fatalf("not every command was listed: %v", columns)
	}
	want := columns["build"]
	for name, col := range columns {
		if col != want {
			t.Errorf("description column of %q starts at %d, want %d (same as every other row)", name, col, want)
		}
	}
}

func TestCommandWithoutDescriptionStillAppears(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{Name: "build", Description: "Build it"},
			{Name: "quiet"},
			{Name: "zero", Description: "\u200b"},
		},
	}
	o, buf := captureOptions(80)
	app.RenderGlobal(o)
	out := strip(buf.String())
	for _, name := range []string{"build", "quiet", "zero"} {
		if !strings.Contains(out, name) {
			t.Errorf("command %q vanished from the listing:\n%s", name, out)
		}
	}
}

func TestStandardFlagRowsSurviveSimilarFlags(t *testing.T) {
	var verbose bool
	var host string
	app := &App{
		Name:    "app",
		Version: "1.2.3",
		GlobalFlags: []Option{
			Bool(&verbose, "-v, --verbose", false, "Verbose output"),
			String(&host, "--host <h>", "", "Host to use"),
		},
		Commands: []Command{{Name: "run", Description: "Run it"}},
	}
	o, buf := captureOptions(90)
	app.renderFlagsPage(o)
	out := strip(buf.String())
	for _, want := range []string{"--version", "--help"} {
		if !strings.Contains(out, want) {
			t.Errorf("the %s row was suppressed by a flag that merely contains it:\n%s", want, out)
		}
	}
}

func TestInlineKeepsUnterminatedEmphasis(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"**", "**"},
		{"a ** b", "a ** b"},
		{"2 * 3 * 4", "2 * 3 * 4"},
		{"**bold**", "bold"},
		{"*italic*", "italic"},
	} {
		if got := strip(inlineMarkdown(tt.in)); got != tt.want {
			t.Errorf("inlineMarkdown(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
