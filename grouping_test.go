package clihelp

import (
	"bytes"

	"github.com/fatih/color"
	"strings"
	"testing"
)

// headingOf returns the section heading a row is printed under.
func headingOf(out, row string) string {
	heading := ""
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(line, " ") {
			heading = trimmed
		}
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == row && strings.HasPrefix(line, " ") {
			return heading
		}
	}
	return "<not listed>"
}

// M1 — an entry declaring no group was printed under the previous group's
// heading, and the return to that group printed no heading. The output said
// something false about two commands, and `prev` was never reset because it was
// assigned only inside the branch that prints a heading.
func TestUngroupedEntriesDoNotJoinThePreviousGroup(t *testing.T) {
	app := &App{Name: "probe", Commands: []Command{
		{Name: "one", Group: "Group A", Description: "In A.", Run: func(*Context) error { return nil }},
		{Name: "two", Description: "Ungrouped.", Run: func(*Context) error { return nil }},
		{Name: "three", Group: "Group A", Description: "Also in A.", Run: func(*Context) error { return nil }},
		{Name: "four", Group: "Group B", Description: "In B.", Run: func(*Context) error { return nil }},
	}}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 70})
	out := StripANSI(buf.String())

	if got := headingOf(out, "two"); got == "Group A:" || got == "Group B:" {
		t.Errorf("a command declaring no group is listed under %q:\n%s", got, out)
	}
	for _, name := range []string{"one", "three"} {
		if got := headingOf(out, name); got != "Group A:" {
			t.Errorf("%s declares Group A and is listed under %q:\n%s", name, got, out)
		}
	}
	if got := headingOf(out, "four"); got != "Group B:" {
		t.Errorf("four declares Group B and is listed under %q:\n%s", got, out)
	}
}

// RenderMan calls renderOptionsGrouped with the raw list, unlike renderFlagsPage,
// which normalises first — so the man page had the same defect for flags.
func TestUngroupedFlagsDoNotJoinThePreviousGroup(t *testing.T) {
	app := &App{Name: "probe",
		PersistentOptions: []Option{
			{Flags: "--a", Description: "A.", Group: "Networking"},
			{Flags: "--b", Description: "B."},
			{Flags: "--c", Description: "C.", Group: "Networking"},
		},
		Commands: []Command{{Name: "run", Description: "R.", Run: func(*Context) error { return nil }}},
	}
	var buf bytes.Buffer
	app.renderManPage(Options{Writer: &buf, Width: 70})
	out := StripANSI(buf.String())
	if got := headingOf(out, "--b"); got == "Networking:" {
		t.Errorf("an ungrouped flag is listed under %q:\n%s", got, out)
	}
}

// M2 — buildDefaultUsage counted cmd.Subcommands and ignored SubcommandEntries,
// so Usage promised [args] above a populated Subcommands list. The same
// copy-without-the-preference that SubcommandList's comment says bit doc/.
func TestUsageHonoursSubcommandEntries(t *testing.T) {
	app := &App{Name: "probe", Commands: []Command{{
		Name: "config", Description: "Configure.", Run: func(*Context) error { return nil },
		SubcommandEntries: []Param{{Name: "set", Description: "Set a value."}},
	}}}
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Width: 70}, "config")
	out := StripANSI(buf.String())
	if strings.Contains(out, "Subcommands:") && !strings.Contains(out, "<subcommand>") {
		t.Errorf("Usage promises no subcommand but a Subcommands list follows:\n%s", out)
	}
}

// M4 — Options.Theme replaced App.Theme wholesale, so any per-render theme
// dropped an app-level Separator and with it the whole title block.
func TestOptionsThemeLayersOntoAppTheme(t *testing.T) {
	app := &App{Name: "probe", Theme: &Theme{Separator: true, TitlePrefix: "T: "},
		Commands: []Command{{Name: "c", Title: "c — do it", Description: "C.", Run: func(*Context) error { return nil }}}}

	var withApp, withOpts bytes.Buffer
	app.RenderCommand(Options{Writer: &withApp, Width: 70}, "c")
	app.RenderCommand(Options{Writer: &withOpts, Width: 70, Theme: &Theme{}}, "c")

	if !strings.Contains(StripANSI(withApp.String()), "T: ") {
		t.Fatalf("the fixture is wrong: App.Theme.TitlePrefix did not render:\n%s", withApp.String())
	}
	if !strings.Contains(StripANSI(withOpts.String()), "T: ") {
		t.Errorf("an empty Options.Theme dropped App.Theme.TitlePrefix:\n%s", withOpts.String())
	}
	if withApp.String() != withOpts.String() {
		t.Errorf("an empty Options.Theme changed the output")
	}
}

// M6 — the blank-line decision was made on the markdown source, which is always
// at least as wide as what is drawn, so one link double-spaced the whole list.
func TestGroupedListSpacingIsDecidedOnWhatIsDrawn(t *testing.T) {
	had := color.NoColor
	color.NoColor = false // the claim is about the OSC 8 form, which is narrow
	defer func() { color.NoColor = had }()

	longURL := "https://example.com/" + strings.Repeat("docs/", 14) + "page.html"
	render := func(desc string) string {
		app := &App{Name: "probe", Commands: []Command{
			{Name: "alpha", Description: desc, Run: func(*Context) error { return nil }},
			{Name: "beta", Description: "Short one.", Run: func(*Context) error { return nil }},
		}}
		var buf bytes.Buffer
		app.RenderGlobal(Options{Writer: &buf, Width: 100})
		return StripANSI(buf.String())
	}
	if render("Open [the docs]("+longURL+") for more.") != render("Open the docs for more.") {
		t.Errorf("a link changed the layout of a list whose visible text is identical")
	}
}

// M13 — FirstSentence cut inside a markdown construct, so raw markup reached
// the help.
func TestFirstSentenceKeepsMarkdownIntact(t *testing.T) {
	for _, in := range []string{
		"Try [it](http://x/v1. 2/y) now.",
		"Use `a. b` for that.",
		"Set **a. b** please.",
	} {
		got := FirstSentence(in)
		if strings.Count(got, "`")%2 != 0 || strings.Count(got, "**")%2 != 0 ||
			strings.Count(got, "[") != strings.Count(got, ")") {
			t.Errorf("FirstSentence(%q) = %q — unbalanced markdown reaches the terminal", in, got)
		}
	}
}

// L8 — a backslash escaped anything, so Windows paths and regexes lost
// characters. examples.go's comment records this as why example lines were
// taken out of the inline renderer; descriptions never were.
func TestBackslashOnlyEscapesPunctuation(t *testing.T) {
	for in, want := range map[string]string{
		`C:\temp\x`:        `C:\temp\x`,
		`use \d+ to match`: `use \d+ to match`,
		`\*literal\*`:      `*literal*`,
		`a\\b`:             `a\b`,
	} {
		if got := Inline(in); got != want {
			t.Errorf("Inline(%q) = %q, want %q", in, got, want)
		}
	}
}

// L9 — a link with empty text rendered nothing visible at all.
func TestEmptyLinkTextFallsBackToTheURL(t *testing.T) {
	if got := StripANSI(Inline("[](http://x)")); got == "" {
		t.Errorf("[](url) rendered nothing visible: %q", Inline("[](http://x)"))
	}
}
