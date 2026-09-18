package clihelp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func budgetApp() *App {
	cmd := Command{
		Name: "sync", Description: "Synchronize local state with the remote service.",
		Run:        func(*Context) error { return nil },
		Parameters: []Param{{Name: "<src>", Description: "Source."}, {Name: "<dst>", Description: "Destination."}},
		Notes:      []Note{{Heading: "Notes", Text: "Something worth knowing."}},
	}
	for i := 0; i < 30; i++ {
		cmd.Options = append(cmd.Options, Option{
			Flags:       fmt.Sprintf("--opt-%02d", i),
			Description: "Option number with a description long enough to occupy most of one line.",
		})
	}
	for i := 0; i < 6; i++ {
		cmd.Examples = append(cmd.Examples, Example{Line: "probe sync a b", Description: "An example."})
	}
	app := &App{Name: "probe", Commands: []Command{cmd}}
	for i := 0; i < 12; i++ {
		app.Commands = append(app.Commands, Command{
			Name: fmt.Sprintf("cmd%02d", i), Description: "Another command with a reasonably long description.",
			Run: func(*Context) error { return nil },
		})
	}
	return app
}

func lineCount(s string) int {
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

// H4 — README.md, llms.txt, docs/comparison-with-cobra.md and
// docs/flags-and-options.md all promise that -h stays within 24 lines. Nothing
// enforced it: the concise tier only suppressed LongDescription and Notes, so a
// command with many flags rendered 92 lines, byte-identical to --help.
func TestConciseHelpFitsItsBudget(t *testing.T) {
	app := budgetApp()
	for _, width := range []int{40, 60, 80, 100, 120} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			var buf bytes.Buffer
			app.RenderCommand(Options{Writer: &buf, Width: width, Concise: true}, "sync")
			if n := lineCount(buf.String()); n > conciseHelpLines {
				t.Errorf("concise -h is %d lines at width %d, over the documented %d",
					n, width, conciseHelpLines)
			}
		})
	}
}

// RenderGlobal ignored o.Concise entirely, so `probe -h` and `probe --help`
// were byte-identical at the root.
func TestConciseGlobalHelpFitsItsBudget(t *testing.T) {
	app := budgetApp()
	var concise, extended bytes.Buffer
	app.RenderGlobal(Options{Writer: &concise, Width: 80, Concise: true})
	app.RenderGlobal(Options{Writer: &extended, Width: 80})

	if n := lineCount(concise.String()); n > conciseHelpLines {
		t.Errorf("concise global help is %d lines, over the documented %d", n, conciseHelpLines)
	}
	if concise.String() == extended.String() && lineCount(extended.String()) > conciseHelpLines {
		t.Errorf("-h and --help are byte-identical at the root")
	}
}

// Truncating must say so, and say where the rest is — silently dropping a
// command's flags would be worse than a long page.
func TestTruncatedHelpSaysWhereTheRestIs(t *testing.T) {
	var buf bytes.Buffer
	budgetApp().RenderCommand(Options{Writer: &buf, Width: 80, Concise: true}, "sync")
	out := StripANSI(buf.String())
	if !strings.Contains(out, "more lines") {
		t.Errorf("truncated help does not say how much was dropped:\n%s", out)
	}
	if !strings.Contains(out, "probe help sync") {
		t.Errorf("truncated help does not name the command that shows the rest:\n%s", out)
	}
}

// Help that already fits is not touched, and never gains a note.
func TestShortHelpIsNotTruncated(t *testing.T) {
	app := &App{Name: "probe", Commands: []Command{
		{Name: "go", Description: "Do the thing.", Run: func(*Context) error { return nil }},
	}}
	var concise, plain bytes.Buffer
	app.RenderCommand(Options{Writer: &concise, Width: 80, Concise: true}, "go")
	app.RenderCommand(Options{Writer: &plain, Width: 80}, "go")
	if strings.Contains(concise.String(), "more lines") {
		t.Errorf("short help gained a truncation note:\n%s", concise.String())
	}
	if concise.String() != plain.String() {
		t.Errorf("short help differs between the tiers:\n%q\n%q", concise.String(), plain.String())
	}
}

// The budget is a default, not a law: an author who wants the old behaviour
// asks for it.
func TestConciseMaxLinesOverridesTheBudget(t *testing.T) {
	var buf bytes.Buffer
	budgetApp().RenderCommand(Options{Writer: &buf, Width: 80, Concise: true, ConciseMaxLines: -1}, "sync")
	if n := lineCount(buf.String()); n <= conciseHelpLines {
		t.Errorf("ConciseMaxLines=-1 still truncated to %d lines", n)
	}
	if strings.Contains(StripANSI(buf.String()), "more lines") {
		t.Errorf("ConciseMaxLines=-1 still added a truncation note")
	}
}
