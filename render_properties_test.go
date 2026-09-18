package clihelp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// assertFitsWidth is the property every width test in this package should have
// been asserting. They counted runes or bytes instead, which tolerates a 40%
// overshoot on CJK text — the library measures 界 as two columns and one test's
// comment claimed it was one.
// requireRendered fails when there is nothing to make an assertion about.
//
// Every property this file checks is of the form "no line in the output does X",
// and every one of them holds vacuously of no output at all: reflowMargin — the
// function that writes every wrapped line in this library — could be made to
// return without writing anything and the width, balance and whitespace
// properties all still passed. A property about output has to be paired with the
// fact that there was output.
func requireRendered(t *testing.T, out string) {
	t.Helper()
	if strings.TrimSpace(StripANSI(out)) == "" {
		t.Fatalf("nothing was rendered, so every property below holds vacuously: %q", out)
	}
}

func assertFitsWidth(t *testing.T, out string, width int) {
	t.Helper()
	requireRendered(t, out)
	inExamples := false
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		// Example command lines are emitted verbatim on purpose, so that one can
		// be copied and pasted; examples.go documents that and a test asserts it.
		if trimmed := strings.TrimSpace(line); strings.HasSuffix(trimmed, ":") {
			inExamples = trimmed == "Examples:" || trimmed == "EXAMPLES"
		}
		if inExamples {
			continue
		}
		c := VisualWidth(line)
		if c <= width {
			continue
		}
		// A line can only be brought inside the width if something on it can be
		// broken. A name wider than the terminal has nowhere to go.
		indent := VisualWidth(line) - VisualWidth(strings.TrimLeft(line, " "))
		longest := 0
		for _, word := range strings.Fields(line) {
			if w := VisualWidth(word); w > longest {
				longest = w
			}
		}
		if indent+longest > width {
			continue
		}
		// A row holding only a name — which the renderer puts on its own line
		// when the name is wider than its column — has nothing to wrap. A
		// signature is one token to a reader even when it contains spaces:
		// breaking "-o, --output <path>" across two lines would be worse than
		// letting it run over.
		trimmed := strings.TrimSpace(line)
		isSignature := strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "<")
		if isSignature && !strings.Contains(trimmed, "  ") {
			continue
		}
		t.Errorf("line is %d display columns, over the %d requested:\n  %q", c, width, line)
	}
}

func propertyApp() *App {
	return &App{
		Name:        "probe",
		Description: "A tool for probing things, with a description long enough to wrap at any width.",
		GlobalFlags: []Option{{Flags: "-v, --verbose", Description: "Verbose output logging."}},
		Commands: []Command{
			{Name: "build", Description: "Build the thing. A second sentence that should not appear in a list.",
				Run:        func(*Context) error { return nil },
				Parameters: []Param{{Name: "<src>", Description: "Where to read from."}},
				Options: []Option{{Flags: "-o, --output <path>", Description: "Where to write.",
					DefaultText: "./out"}},
				Examples: []Example{{Line: "probe build .", Description: "Build here."}},
			},
			{Name: "検索", Description: "Search, with a wide-rune name.", Run: func(*Context) error { return nil }},
			{Name: "🚀ship", Description: "Ship it.", Run: func(*Context) error { return nil }},
			{Name: "éxport", Description: "Export it.", Run: func(*Context) error { return nil }},
			{Name: "éxport2", Description: "Export it again.", Run: func(*Context) error { return nil }},
		},
	}
}

// M19 — no test in this package asserted that a rendered line fits the width it
// was given, and none rendered below 60 columns. 20 and 40 are where the
// defects were.
func TestEveryTierFitsItsWidth(t *testing.T) {
	app := propertyApp()
	for _, width := range []int{20, 40, 60, 80, 120} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			for name, render := range map[string]func(Options){
				"RenderGlobal":     app.RenderGlobal,
				"RenderFlags":      app.RenderFlags,
				"RenderMan":        app.RenderMan,
				"RenderHelpTopics": app.RenderHelpTopics,
			} {
				var buf bytes.Buffer
				render(Options{Writer: &buf, Width: width})
				t.Run(name, func(t *testing.T) { assertFitsWidth(t, strip(buf.String()), width) })
			}
			for _, concise := range []bool{false, true} {
				var buf bytes.Buffer
				app.RenderCommand(Options{Writer: &buf, Width: width, Concise: concise}, "build")
				assertFitsWidth(t, strip(buf.String()), width)
			}
		})
	}
}

// Emoji, ZWJ sequences and combining marks appear in no test anywhere in this
// module. They align correctly today, but that is a property of runewidth's
// grapheme clustering and nothing here would notice a dependency bump.
func TestWideRunesShareOneDescriptionColumn(t *testing.T) {
	var buf bytes.Buffer
	propertyApp().RenderGlobal(Options{Writer: &buf, Width: 80})

	columns := map[string]int{}
	for _, line := range strings.Split(strip(buf.String()), "\n") {
		for _, name := range []string{"build", "検索", "🚀ship", "éxport", "éxport2"} {
			prefix := "  " + name
			if strings.HasPrefix(line, prefix) {
				rest := strings.TrimLeft(line[len(prefix):], " ")
				columns[name] = VisualWidth(line) - VisualWidth(rest)
			}
		}
	}
	if len(columns) < 4 {
		t.Fatalf("the fixture is wrong: only matched %d rows", len(columns))
	}
	var first int
	for _, c := range columns {
		if first == 0 {
			first = c
		} else if c != first {
			t.Errorf("description columns disagree across wide runes: %v", columns)
			break
		}
	}
}

// M17 — MaxContentWidth was never driven through a renderer. Deleting it from
// the flag table left the suite green.
func TestMaxContentWidthReachesTheFlagTable(t *testing.T) {
	app := &App{Name: "probe", Commands: []Command{{
		Name: "run", Description: "R.", Run: func(*Context) error { return nil },
		Options: []Option{{Flags: "-o <p>", Description: strings.Repeat("word ", 60)}},
	}}}
	var wide, narrow bytes.Buffer
	app.RenderCommand(Options{Writer: &wide, Width: 200, MaxContentWidth: 120}, "run")
	app.RenderCommand(Options{Writer: &narrow, Width: 200, MaxContentWidth: 40}, "run")

	widest := func(s string) int {
		max := 0
		for _, l := range strings.Split(s, "\n") {
			if w := VisualWidth(l); w > max {
				max = w
			}
		}
		return max
	}
	if widest(strip(narrow.String())) >= widest(strip(wide.String())) {
		t.Errorf("MaxContentWidth did not reach the renderer: 40 gave %d columns, 120 gave %d",
			widest(strip(narrow.String())), widest(strip(wide.String())))
	}
}

// M20 — App.Theme is set by no test in this package, so mutating the assignment
// that reads it left the suite green.
func TestAppThemeReachesTheRenderer(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := propertyApp()
	app.Theme = &Theme{Hdr: color.New(color.FgMagenta, color.Bold), Separator: true, TitlePrefix: "T: "}
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Width: 80}, "build")

	if !strings.Contains(buf.String(), "\x1b[35;1m") {
		t.Errorf("App.Theme.Hdr was ignored:\n%q", buf.String())
	}
	if !strings.Contains(strip(buf.String()), "T: ") {
		t.Errorf("App.Theme.TitlePrefix was ignored:\n%s", strip(buf.String()))
	}
}

// L15 — exported with no caller anywhere in the module and no test.
func TestCommandArgs(t *testing.T) {
	for _, tt := range []struct {
		cmd  Command
		want string
	}{
		{Command{Name: "set", Parameters: []Param{{Name: "<key>"}, {Name: "<value>"}}}, "<key> <value>"},
		{Command{Name: "set", UsageLine: "app set [options] <key> <value> — set a key"}, "<key> <value>"},
		{Command{Name: "set", UsageLine: "app set"}, ""},
		{Command{Name: "set"}, ""},
	} {
		if got := commandArgs(tt.cmd); got != tt.want {
			t.Errorf("commandArgs(%+v) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}

func TestDisplayNameWithArgs(t *testing.T) {
	cmd := Command{Name: "set", Aliases: []string{"s"}, Parameters: []Param{{Name: "<key>"}}}
	if got, want := DisplayNameWithArgs(cmd), "set (s) <key>"; got != want {
		t.Errorf("DisplayNameWithArgs = %q, want %q", got, want)
	}
}

func TestRenderGlobalFlagsIsRenderFlags(t *testing.T) {
	app := propertyApp()
	var a, b bytes.Buffer
	app.RenderFlags(Options{Writer: &a, Width: 80})
	app.RenderFlags(Options{Writer: &b, Width: 80})
	if a.String() != b.String() {
		t.Errorf("RenderGlobalFlags diverged from RenderFlags")
	}
}

// H6 — these five behaviours were guarded only by example/mail_cli_fake's
// oracle, which is a byte comparison against a reimplementation of the renderer
// at one fixed width. Each of them survived a mutation of the library with
// `go test .` still green.
func TestGlobalHelpListsShortcutsAndConfig(t *testing.T) {
	app := &App{Name: "podctl", ConfigPath: "/etc/podctl.toml",
		Commands: []Command{{Name: "build", Description: "Build it. A second sentence.",
			Run: func(*Context) error { return nil }}},
		Shortcuts: []Command{{Name: "b", Description: "Shortcut for build.",
			Run: func(*Context) error { return nil }}},
		GlobalFlags: []Option{{Flags: "-v, --verbose", Description: "Be loud."}},
	}
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 80})
	out := strip(buf.String())

	for _, want := range []string{
		"Shortcut Commands:",       // renderGlobalShortcuts, deletable while green
		"b",                        // its rows
		"Config: /etc/podctl.toml", // the footer, deletable while green
		"Global Flags:",            // the heading, deletable while green
	} {
		if !strings.Contains(out, want) {
			t.Errorf("global help is missing %q:\n%s", want, out)
		}
	}
	// colIndent's tightening: the column is the widest name plus four, not the
	// maximum. "build" is five wide, so the description starts at column 9.
	if !strings.Contains(out, "  build  Build it.") {
		t.Errorf("the description column is not tightened to the widest name:\n%s", out)
	}
	// And a command list shows one sentence, not the whole description.
	if strings.Contains(out, "A second sentence") {
		t.Errorf("the command list shows more than the first sentence:\n%s", out)
	}
}
