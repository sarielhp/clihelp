package clihelp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestRenderCommandShowsAncestorPersistentOptions(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name: "parent",
				PersistentOptions: []Option{
					{Flags: "--parent-opt <val>", Description: "Parent persistent option"},
				},
				Subcommands: []Command{
					{
						Name: "child",
						Options: []Option{
							{Flags: "--child-opt <val>", Description: "Child local option"},
						},
						Subcommands: []Command{
							{
								Name: "grandchild",
								Options: []Option{
									{Flags: "--grandchild-opt <val>", Description: "Grandchild local option"},
								},
							},
						},
					},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "parent", "child", "grandchild") {
		t.Fatal("RenderCommand failed")
	}

	out := buf.String()
	if !strings.Contains(out, "--parent-opt") {
		t.Errorf("expected parent persistent option in grandchild help, got:\n%s", out)
	}
	if strings.Contains(out, "--child-opt") {
		t.Errorf("child local option should not appear in grandchild help, got:\n%s", out)
	}
	if !strings.Contains(out, "--grandchild-opt") {
		t.Errorf("expected grandchild local option in grandchild help, got:\n%s", out)
	}
}

func TestAncestorsForPathWithAliases(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:    "parent",
				Aliases: []string{"p"},
				PersistentOptions: []Option{
					{Flags: "--parent-opt <val>", Description: "Parent persistent option"},
				},
				Subcommands: []Command{
					{
						Name:    "child",
						Aliases: []string{"c"},
						Subcommands: []Command{
							{
								Name: "grandchild",
							},
						},
					},
				},
			},
		},
	}

	ancestors := app.ancestorsForPath("p", "c", "grandchild")
	if len(ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].Name != "parent" || ancestors[1].Name != "child" {
		t.Errorf("expected [parent, child], got [%s, %s]", ancestors[0].Name, ancestors[1].Name)
	}
}

func TestRenderCommandShowsAppPersistentOptions(t *testing.T) {
	app := &App{
		Name: "testapp",
		PersistentOptions: []Option{
			{Flags: "--root-opt <val>", Description: "Root persistent option"},
		},
		Commands: []Command{
			{
				Name: "cmd",
				Options: []Option{
					{Flags: "--cmd-opt <val>", Description: "Command local option"},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "cmd") {
		t.Fatal("RenderCommand failed")
	}

	out := buf.String()
	if !strings.Contains(out, "--root-opt") {
		t.Errorf("expected root persistent option in command help, got:\n%s", out)
	}
	if !strings.Contains(out, "--cmd-opt") {
		t.Errorf("expected command local option in command help, got:\n%s", out)
	}
}

func TestLookupCommand(t *testing.T) {
	app := testApp()
	if cmd := app.LookupCommand("build"); cmd == nil || cmd.Name != "build" {
		t.Errorf("LookupCommand(build) failed: %v", cmd)
	}
	if cmd := app.LookupCommand("config", "set"); cmd == nil || cmd.Name != "set" {
		t.Errorf("LookupCommand(config, set) failed: %v", cmd)
	}
	if cmd := app.LookupCommand("missing"); cmd != nil {
		t.Errorf("LookupCommand(missing) expected nil, got %v", cmd)
	}
	if cmd := app.LookupCommand(); cmd != nil {
		t.Errorf("LookupCommand() expected nil, got %v", cmd)
	}
}

func TestRenderGlobal(t *testing.T) {
	o, buf := captureOptions(80)
	app := testApp()
	app.RenderGlobal(o)
	out := strip(buf.String())

	for _, want := range []string{"Usage:  podctl", "build", "Compile audio episodes", "config", "Manage configuration"} {
		if !strings.Contains(out, want) {
			t.Errorf("global output missing %q\n%q", want, out)
		}
	}
}

func TestRenderCommand(t *testing.T) {
	o, buf := captureOptions(80)
	app := testApp()
	if !app.RenderCommand(o, "build") {
		t.Fatal("RenderCommand(\"build\") returned false")
	}
	out := strip(buf.String())

	for _, want := range []string{
		"Usage:  podctl build", "Compile audio episodes",
		"Flags:", "-o, --output PATH", "Write output to PATH",
		"Examples:", "podctl build ep.wav",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("command output missing %q\n%q", want, out)
		}
	}
}

func TestRenderNestedCommand(t *testing.T) {
	o, buf := captureOptions(80)
	app := testApp()
	if !app.RenderCommand(o, "config", "set") {
		t.Fatal("RenderCommand(\"config\",\"set\") returned false")
	}
	out := strip(buf.String())
	for _, want := range []string{"config set <key> <value>", "Parameters:", "<key>", "The value to assign"} {
		if !strings.Contains(out, want) {
			t.Errorf("nested output missing %q", want)
		}
	}
}

func TestRenderCommandNotFound(t *testing.T) {
	o, _ := captureOptions(80)
	if testApp().RenderCommand(o, "wat") {
		t.Fatal("RenderCommand for unknown command should return false")
	}
}

func TestRenderDispatch(t *testing.T) {
	app := testApp()
	_, gbuf := captureOptions(80)
	o := Options{Writer: gbuf, Width: 80}
	app.Render(o)

	o2, cbuf := captureOptions(80)
	app.Render(o2, "build")
	if !strings.Contains(strip(cbuf.String()), "podctl build") {
		t.Errorf("Render with path should render the command page")
	}
}

func TestRenderGlobalDescriptionAndNote(t *testing.T) {
	app := testApp()
	app.Description = "A podcast distribution toolkit."
	app.GlobalNote = "See docs for account setup."
	o, buf := captureOptions(80)
	app.RenderGlobal(o)
	out := strip(buf.String())
	if !strings.Contains(out, "A podcast distribution toolkit.") {
		t.Errorf("global output missing App.Description")
	}
	// GlobalNote is now in help docs/more
	var docsBuf bytes.Buffer
	app.Stdout = &docsBuf
	if err := app.ExecuteContext(context.Background(), []string{"help", "docs"}); err != nil {
		t.Fatalf("help docs error: %v", err)
	}
	if !strings.Contains(docsBuf.String(), "See docs for account setup.") {
		t.Errorf("help docs missing App.GlobalNote")
	}
	// Ensure defaults keep the classic layout when fields are unset.
	o2, buf2 := captureOptions(80)
	testApp().RenderGlobal(o2)
	if strings.Contains(strip(buf2.String()), "podcast distribution") {
		t.Errorf("unexpected description rendered when unset")
	}
}

func TestRenderExampleDescription(t *testing.T) {
	app := testApp()
	app.Commands[0].Examples[0].Description = "Compiles a single episode."
	o, buf := captureOptions(80)
	app.RenderCommand(o, "build")
	out := strip(buf.String())
	if !strings.Contains(out, "Compiles a single episode.") {
		t.Errorf("Example.Description not rendered:\n%q", out)
	}
}

func TestRenderCustomThemeColors(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	o, buf := captureOptions(80)
	o.Theme = &Theme{
		Hdr:         color.New(color.FgMagenta, color.Bold),
		Body:        color.New(color.FgGreen),
		Accent:      color.New(color.FgBlue, color.Bold),
		Separator:   true,
		TitlePrefix: "Detailed Usage: ",
	}
	testApp().RenderCommand(o, "build")
	raw := buf.String()
	if !strings.Contains(raw, "\x1b[35;1m") {
		t.Errorf("expect bold magenta header codes, got:\n%q", raw)
	}
}

func TestRenderFlagColoring(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	// Test default flag coloring (Cyan)
	{
		o, buf := captureOptions(80)
		testApp().RenderCommand(o, "build")
		raw := buf.String()
		cyanVal := color.New(color.FgCyan).Sprint("X")
		cyanSeq := cyanVal[:strings.Index(cyanVal, "X")]
		if !strings.Contains(raw, cyanSeq+"  -o, --output PATH") {
			t.Errorf("expect default cyan flags styling, got:\n%q", raw)
		}
	}

	// Test custom flag coloring (Red)
	{
		o, buf := captureOptions(80)
		o.Theme = &Theme{
			Flag: color.New(color.FgRed),
		}
		testApp().RenderCommand(o, "build")
		raw := buf.String()
		redVal := color.New(color.FgRed).Sprint("X")
		redSeq := redVal[:strings.Index(redVal, "X")]
		if !strings.Contains(raw, redSeq+"  -o, --output PATH") {
			t.Errorf("expect custom red flags styling, got:\n%q", raw)
		}
	}
}

func TestTieredHelpConciseAndExtended(t *testing.T) {
	app := &App{
		Name: "testcli",
		Commands: []Command{
			{
				Name:            "build",
				Description:     "Short build description.",
				LongDescription: "Comprehensive long build description that covers all aspects of the build pipeline in depth.",
				Options: []Option{
					{Flags: "-o, --output <path>", Description: "Output artifact path"},
				},
				Examples: []Example{
					{Line: "testcli build -o dist/app", Description: "Build application binary"},
				},
				Notes: []Note{
					{Heading: "Notes", Text: "Detailed notes about caching and compiler flags."},
				},
			},
			{
				Name:        "simple",
				Description: "A simple command with no notes or long description.",
			},
		},
	}

	t.Run("concise help suppresses notes and shows footer hint", func(t *testing.T) {
		o, buf := captureOptions(80)
		o.Concise = true
		app.RenderCommand(o, "build")
		out := strip(buf.String())

		if !strings.Contains(out, "Short build description.") {
			t.Errorf("expected Description in concise help, got:\n%s", out)
		}
		if strings.Contains(out, "Comprehensive long build description") {
			t.Errorf("LongDescription should NOT appear in concise help, got:\n%s", out)
		}
		if strings.Contains(out, "Detailed notes about caching") {
			t.Errorf("Notes should be suppressed in concise help, got:\n%s", out)
		}
		if !strings.Contains(out, "Run 'testcli help build' (or --help)") || !strings.Contains(out, "extended documentation and") {
			t.Errorf("expected footer hint in concise help, got:\n%s", out)
		}
	})

	t.Run("concise help with ExtendedHelpFlag includes -H in footer hint", func(t *testing.T) {
		appWithH := *app
		appWithH.ExtendedHelpFlag = true
		o, buf := captureOptions(80)
		o.Concise = true
		appWithH.RenderCommand(o, "build")
		out := strip(buf.String())

		if !strings.Contains(out, "Run 'testcli help build' (or --help / -H)") || !strings.Contains(out, "extended documentation and") {
			t.Errorf("expected footer hint with -H, got:\n%s", out)
		}
	})

	t.Run("concise help on command without notes or long description renders normally", func(t *testing.T) {
		o, buf := captureOptions(80)
		o.Concise = true
		app.RenderCommand(o, "simple")
		out := strip(buf.String())

		if !strings.Contains(out, "A simple command with no notes or long description.") {
			t.Errorf("expected description, got:\n%s", out)
		}
		if strings.Contains(out, "extended documentation and examples") {
			t.Errorf("simple command should NOT have footer hint, got:\n%s", out)
		}
	})

	t.Run("extended help displays LongDescription Notes and Examples", func(t *testing.T) {
		o, buf := captureOptions(80)
		o.Extended = true
		app.RenderCommand(o, "build")
		out := strip(buf.String())

		if !strings.Contains(out, "Comprehensive long build description") {
			t.Errorf("expected LongDescription in extended help, got:\n%s", out)
		}
		if strings.Contains(out, "Short build description.") {
			t.Errorf("Short Description should be replaced by LongDescription, got:\n%s", out)
		}
		if !strings.Contains(out, "Detailed notes about caching") {
			t.Errorf("expected Notes in extended help, got:\n%s", out)
		}
		if !strings.Contains(out, "testcli build -o dist/app") {
			t.Errorf("expected Examples in extended help, got:\n%s", out)
		}
		if strings.Contains(out, "for extended documentation and examples") {
			t.Errorf("extended help should NOT have footer hint, got:\n%s", out)
		}
	})
}
