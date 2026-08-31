package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

func captureOptions(width int) (Options, *bytes.Buffer) {
	var buf bytes.Buffer
	return Options{Writer: &buf, Width: width}, &buf
}

func testApp() *App {
	return &App{
		Name:    "podctl",
		Version: "1.0.0",
		Commands: []Command{
			{
				Name:        "build",
				Description: "Compile audio episodes",
				UsageLine:   "podctl build [options] <file>",
				Options: []Option{
					{Flags: "-o, --output PATH", Description: "Write output to PATH"},
					{Flags: "--verbose", Description: "Enable verbose logging"},
				},
				Examples: []Example{{Line: "podctl build ep.wav"}},
			},
			{
				Name:        "config",
				Description: "Manage configuration",
				UsageLine:   "podctl config <subcommand>",
				Subcommands: []Command{
					{
						Name:        "set",
						Title:       "config set <key> <value>",
						Description: "Set a configuration value",
						UsageLine:   "podctl config set <key> <value>",
						Parameters: []Param{
							{Name: "<key>", Description: "The key to set"},
							{Name: "<value>", Description: "The value to assign"},
						},
					},
				},
			},
		},
	}
}

func strip(s string) string { return stripansi.Strip(s) }

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
							{Flags: "--child-opt <val>", Description: "Child option"},
						},
					},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "parent", "child") {
		t.Fatal("RenderCommand returned false")
	}
	out := strip(buf.String())

	if !strings.Contains(out, "--parent-opt") {
		t.Errorf("child help missing ancestor persistent option --parent-opt\n%s", out)
	}
	if !strings.Contains(out, "--child-opt") {
		t.Errorf("child help missing own option --child-opt\n%s", out)
	}
}

func TestAncestorsForPathWithAliases(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:    "parent",
				Aliases: []string{"p"},
				Subcommands: []Command{
					{
						Name: "child",
					},
				},
			},
		},
	}

	// Using alias should still find ancestors
	ancestors := app.ancestorsForPath("p", "child")
	if len(ancestors) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(ancestors))
	}
	if ancestors[0].Name != "parent" {
		t.Errorf("expected ancestor name 'parent', got %q", ancestors[0].Name)
	}
}

func TestRenderCommandShowsAppPersistentOptions(t *testing.T) {
	app := &App{
		Name: "testapp",
		PersistentOptions: []Option{
			{Flags: "--app-flag <val>", Description: "App-level persistent option"},
		},
		Commands: []Command{
			{
				Name: "cmd",
				Options: []Option{
					{Flags: "--cmd-flag <val>", Description: "Command option"},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "cmd") {
		t.Fatal("RenderCommand returned false")
	}
	out := strip(buf.String())

	if !strings.Contains(out, "--app-flag") {
		t.Errorf("command help missing app-level persistent option --app-flag\n%s", out)
	}
	if !strings.Contains(out, "--cmd-flag") {
		t.Errorf("command help missing own option --cmd-flag\n%s", out)
	}
}

func TestLookupCommand(t *testing.T) {
	app := testApp()
	if c := app.LookupCommand("config", "set"); c == nil || c.Name != "set" {
		t.Fatalf("nested lookup failed: %+v", c)
	}
	if c := app.LookupCommand("config", "set", "time"); c != nil {
		t.Fatalf("expected nil for unknown deep path, got %v", c.Name)
	}
	if c := app.LookupCommand("nope"); c != nil {
		t.Fatalf("expected nil for unknown command")
	}
}

func TestVisualLen(t *testing.T) {
	green := color.New(color.FgGreen).Sprint("podctl")
	if stripansi.Strip(green) != "podctl" {
		t.Fatalf("stripansi mismatch: %q", green)
	}
	if visualLen(green) != 6 {
		t.Fatalf("visualLen of colored %q = %d, want 6", green, visualLen(green))
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

func TestReflowHonorsWidth(t *testing.T) {
	longText := "The quick brown fox jumps over the lazy dog near the riverside cottage daily."
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 20, 2, "", longText)
	max := 0
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if l := len(strip(line)); l > max {
			max = l
		}
	}
	if max > 20 {
		t.Fatalf("reflow exceeded width: max line %d (>20)\n%s", max, buf.String())
	}
}

func TestReflowMultibyteIsRuneAware(t *testing.T) {
	// Each CJK character is 3 bytes but 1 visible char. At width 10, only
	// ~10 visible chars fit per line regardless of byte lengths.
	longText := strings.Repeat("界 ", 30)
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 10, 0, "", longText)
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if n := len([]rune(strip(line))); n > 10 {
			t.Fatalf("line exceeded 10 visible chars (got %d): %q", n, strip(line))
		}
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

func TestReflowNoSeparatorTheme(t *testing.T) {
	o, buf := captureOptions(80)
	o.Theme = &Theme{Separator: false, TitlePrefix: "USAGE: "}
	testApp().RenderCommand(o, "build")
	out := strip(buf.String())
	if strings.Contains(out, "===") {
		t.Errorf("separator rendered despite Theme.Separator=false:\n%q", out)
	}
	if !strings.Contains(out, "USAGE: build") {
		t.Errorf("custom TitlePrefix not applied: got %q, want %q", out, "USAGE: build")
	}
}

func TestRenderCommandReflowAtWidth60(t *testing.T) {
	th := defaultTheme()
	th.Separator = false
	o, buf := captureOptions(60)
	o.Theme = &th

	longDesc := "Compile, encode, and package raw audio source files into fully tagged MP3 podcast episodes with configurable bitrate, loudness normalization, and embedded ID3 metadata tags for distribution across multiple platforms and aggregators like Apple Podcasts, Spotify, and Google Podcasts."

	app := &App{
		Name: "podctl",
		Commands: []Command{
			{
				Name:        "build",
				Description: longDesc,
				UsageLine:   "podctl build [options] <source-file>",
				Options: []Option{
					{Flags: "-o, --output PATH", Description: "Write compiled MP3 output to specified PATH for later distribution and archival storage on cloud platforms"},
					{Flags: "--tags TAGS", Description: "Embed ID3 metadata tags including title, artist, album, episode number, season, and publication date for rich podcast episode descriptions"},
				},
				Notes: []Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use --bitrate 320 for highest quality music episodes or --bitrate 128 for voice-only spoken word content to optimize file size while maintaining acceptable audio fidelity for listeners on mobile connections.",
					},
				},
			},
		},
	}

	app.RenderCommand(o, "build")
	output := strip(buf.String())

	maxLine := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
		if len([]rune(line)) > 60 {
			t.Errorf("line exceeds 60 chars at width 60:\n  len=%d  %q", len([]rune(line)), line)
		}
	}
	t.Logf("width=60: max line = %d chars", maxLine)
}

func TestRenderCommandReflowAtWidth100(t *testing.T) {
	th := defaultTheme()
	th.Separator = false
	o, buf := captureOptions(100)
	o.Theme = &th

	longDesc := "Compile, encode, and package raw audio source files into fully tagged MP3 podcast episodes with configurable bitrate, loudness normalization, and embedded ID3 metadata tags for distribution across multiple platforms and aggregators like Apple Podcasts, Spotify, and Google Podcasts."

	app := &App{
		Name: "podctl",
		Commands: []Command{
			{
				Name:        "build",
				Description: longDesc,
				UsageLine:   "podctl build [options] <source-file>",
				Options: []Option{
					{Flags: "-o, --output PATH", Description: "Write compiled MP3 output to specified PATH for later distribution and archival storage on cloud platforms"},
					{Flags: "--tags TAGS", Description: "Embed ID3 metadata tags including title, artist, album, episode number, season, and publication date for rich podcast episode descriptions"},
				},
				Notes: []Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use --bitrate 320 for highest quality music episodes or --bitrate 128 for voice-only spoken word content to optimize file size while maintaining acceptable audio fidelity for listeners on mobile connections.",
					},
				},
			},
		},
	}

	app.RenderCommand(o, "build")
	output := strip(buf.String())

	// At width 100, description lines (indent=2) wrap at 82, but flag lines
	// (indent ~20) wrap at 100.  No line should exceed the terminal width.
	expectedMax := 100

	maxLine := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
		if len([]rune(line)) > expectedMax {
			t.Errorf("line exceeds %d chars at width 100:\n  len=%d  %q", expectedMax, len([]rune(line)), line)
		}
	}
	t.Logf("width=100: max line = %d chars (expected max %d)", maxLine, expectedMax)
}

func TestReflowWrapWidthRespectsIndentPlus80(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	// At width 200, a section with indent=22 should wrap to 22+80=102 max
	o, buf := captureOptions(200)
	app := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "A short description.",
				Options: []Option{
					{Flags: "--very-long-flag-name-with-many-characters", Description: "This is a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines and test that the wrapping width respects the indent plus 80 rule."},
				},
			},
		},
	}
	app.RenderCommand(o, "cmd")
	raw := stripANSI(buf.String())

	// Flag name is 43 chars, indent = 43 + 4 = 47
	// wrapWidth(47) = min(200, 47+80) = 127
	// Description text (indent=2): wrapWidth(2) = min(200, 82) = 82

	maxLine := 0
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
	}
	if maxLine > 127 {
		t.Errorf("at width 200 with indent 47, max line = %d, expected <= 127", maxLine)
	}
	if maxLine < 100 {
		t.Errorf("at width 200 with indent 47, max line = %d, expected >= 100 (should use wide terminal)", maxLine)
	}

	// At width 60, all lines should wrap to 60 (terminal narrower than indent+80)
	o2, buf2 := captureOptions(60)
	o2.Theme = &Theme{Separator: false}
	app2 := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "A short description that should not wrap at all at this width because it is short.",
			},
		},
	}
	app2.RenderCommand(o2, "cmd")
	raw2 := stripANSI(buf2.String())

	maxLine2 := 0
	for _, line := range strings.Split(strings.TrimSpace(raw2), "\n") {
		if l := len([]rune(line)); l > maxLine2 {
			maxLine2 = l
		}
	}
	if maxLine2 > 60 {
		t.Errorf("at width 60, max line = %d, expected <= 60", maxLine2)
	}

	// Verify description lines (indent=2) are capped at 82 at wide terminal
	o3, buf3 := captureOptions(200)
	app3 := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "This is a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines and test that the wrapping width respects the indent plus 80 rule.",
			},
		},
	}
	app3.RenderCommand(o3, "cmd")
	raw3 := stripANSI(buf3.String())

	maxLine3 := 0
	lines3 := strings.Split(strings.TrimSpace(raw3), "\n")
	for _, line := range lines3 {
		if l := len([]rune(line)); l > maxLine3 {
			maxLine3 = l
		}
	}
	if maxLine3 > 82 {
		t.Errorf("at width 200 with indent 2, max line = %d, expected <= 82", maxLine3)
	}
	// Should have at least one line near 82 (not just a short single line)
	if len(lines3) > 1 && maxLine3 < 60 {
		t.Errorf("at width 200 with indent 2, max line = %d, expected >= 60 (should wrap to ~80)", maxLine3)
	}
}

func TestExampleAppNoBareMarkdownAndNoVisibleURLs(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := exampleApp()
	paths := collectAllPaths(app)

	type urlCheck struct {
		url    string
		prefix []string // path prefix where this URL is hidden (nil = all paths)
	}
	hiddenURLs := []urlCheck{
		{url: "https://example.com/", prefix: nil},
		{url: "https://podctl.example.com/docs/audio", prefix: []string{"build"}},
		{url: "https://podctl.example.com/docs/deploy", prefix: []string{"deploy"}},
	}
	globalOnlyURLs := []string{
		"https://github.com/sarielhp/clihelp",
	}

	// Render every help page with default options (ShowURLs=false)
	for _, path := range paths {
		o, buf := captureOptions(80)
		app.Render(o, path...)
		out := stripANSI(buf.String())

		if strings.Contains(out, "**") {
			t.Errorf("path %v: raw ** found in stripped output", path)
		}

		for _, uc := range hiddenURLs {
			if uc.prefix == nil || hasPrefix(path, uc.prefix) {
				if strings.Contains(out, uc.url) {
					t.Errorf("path %v: hidden URL %q found in stripped output", path, uc.url)
				}
			}
		}
	}

	var docsBuf bytes.Buffer
	app.Stdout = &docsBuf
	if err := app.ExecuteContext(context.Background(), []string{"help", "docs"}); err != nil {
		t.Fatalf("help docs error: %v", err)
	}
	for _, u := range globalOnlyURLs {
		if !strings.Contains(docsBuf.String(), u) {
			t.Errorf("help docs: URL %q should appear in docs output", u)
		}
	}
}

func TestSubcommandNamesAreGreen(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	app := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "parent",
				Description: "Parent command",
				Subcommands: []Command{
					{Name: "child", Description: "A child command"},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	app.RenderCommand(o, "parent")
	raw := buf.String()

	// The subcommand name "child" should be wrapped in green escape codes
	if !strings.Contains(raw, "\x1b[32m") {
		t.Errorf("expected green color escape for subcommand name, got:\n%q", raw)
	}
	// The description following should use body color (white)
	if !strings.Contains(raw, "\x1b[37m") {
		t.Errorf("expected body color for subcommand description, got:\n%q", raw)
	}
	// No raw ** should appear
	if strings.Contains(raw, "**") {
		t.Errorf("raw ** should not appear in output:\n%q", raw)
	}
}

// hasPrefix checks if path has the given prefix (nil prefix matches everything).
func hasPrefix(path, prefix []string) bool {
	if prefix == nil {
		return true
	}
	if len(path) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if path[i] != p {
			return false
		}
	}
	return true
}

// exampleApp builds a replica of example/main.go's podctl app for testing.
func exampleApp() *App {
	return &App{
		Name:        "podctl",
		Description: "[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.",
		Version:     "0.2.9",
		GlobalNote:  "Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)\nRun 'podctl <command> --help' for command-specific options.",
		Commands: []Command{
			{
				Name:        "build",
				Description: "Compile, encode, and package raw audio source files.",
				UsageLine:   "podctl build [options] <source-file> — **build** tool with [docs](https://example.com/build).",
				Notes: []Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use `--bitrate 320` for *highest quality* or `--bitrate 128` for **voice-only** episodes (see [Audio Encoding Guide](https://podctl.example.com/docs/audio)).",
					},
				},
			},
			{
				Name:        "deploy",
				Description: "Publish your compiled podcast RSS feed.",
				UsageLine:   "podctl deploy [options] — **deploy** tool with [docs](https://example.com/deploy).",
				Notes: []Note{
					{
						Heading: "Safety Precaution",
						Text:    "Always test with `--dry-run` before ~~overwriting~~ publishing to **production** (see [Deploy Docs](https://podctl.example.com/docs/deploy)).",
					},
				},
			},
			buildDeepTreeTest(),
		},
	}
}

// levelSuffixesTest maps depth to the two suffixes used for subcommand naming.
var levelSuffixesTest = [][]string{
	2: {"one", "two"},
	3: {"a", "b"},
	4: {"i", "ii"},
}

// buildDeepTreeTest creates the "deep" command with a binary tree of subcommands.
func buildDeepTreeTest() Command {
	return Command{
		Name:        "deep",
		Description: "**deep** — This is the [deep command](https://example.com/deep) at the root.",
		UsageLine:   "podctl deep [options] <subcommand> — This is a **very long usage line** for the [deep command](https://example.com/deep).",
		Subcommands: []Command{
			buildSubTreeTest("alpha", []string{"deep", "alpha"}, 2),
			buildSubTreeTest("beta", []string{"deep", "beta"}, 2),
		},
	}
}

// buildSubTreeTest recursively builds a command node and its binary subcommand tree.
func buildSubTreeTest(name string, path []string, depth int) Command {
	cmd := Command{
		Name:        name,
		Description: fmt.Sprintf("This is the [%s command](https://example.com/%s) at depth %d with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.", name, strings.Join(path, "/"), depth),
		UsageLine:   fmt.Sprintf("podctl %s [options] [arguments...] — This is a very long usage line for the [%s command](https://example.com/%s) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.", strings.Join(path, " "), name, strings.Join(path, "/")),
	}

	if depth < 5 {
		suffixes := levelSuffixesTest[depth]
		child1 := name + "_" + suffixes[0]
		child2 := name + "_" + suffixes[1]
		cmd.Subcommands = []Command{
			buildSubTreeTest(child1, append(path, child1), depth+1),
			buildSubTreeTest(child2, append(path, child2), depth+1),
		}
	}

	return cmd
}

// collectAllPaths returns all command paths in the app, including the empty
// path for global help.
func collectAllPaths(a *App) [][]string {
	var paths [][]string
	paths = append(paths, nil) // global help
	var walk func(cmds []Command, prefix []string)
	walk = func(cmds []Command, prefix []string) {
		for _, c := range cmds {
			path := append(append([]string{}, prefix...), c.Name)
			paths = append(paths, path)
			if len(c.Subcommands) > 0 {
				walk(c.Subcommands, path)
			}
		}
	}
	walk(a.Commands, nil)
	return paths
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

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello world", "hello world"},
		{"sgr bold", "\x1b[1mhello\x1b[22m", "hello"},
		{"sgr color", "\x1b[35;1mhello\x1b[0m", "hello"},
		{"sgr short reset", "\x1b[mhello", "hello"},
		{"osc8 link", "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\", "text"},
		{"mixed", "\x1b[1m**bold**\x1b[22m \x1b]8;;https://x.com\x1b\\link\x1b]8;;\x1b\\", "**bold** link"},
		{"link in sentence", "see \x1b]8;;https://docs.example.com\x1b\\the docs\x1b]8;;\x1b\\ for more", "see the docs for more"},
	}
	for _, tc := range tests {
		got := stripANSI(tc.in)
		if got != tc.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVisualLenWithOSC8Links(t *testing.T) {
	// A word containing an OSC 8 hyperlink should have its visual length
	// computed as only the visible link text, not the full escape sequence.
	link := "\x1b]8;;https://example.com\x1b\\docs\x1b]8;;\x1b\\"
	if n := visualLen(link); n != 4 {
		t.Errorf("visualLen of OSC8 link = %d, want 4 (visible text 'docs', raw=%q)", n, link)
	}

	link2 := "\x1b]8;;https://example.com/very/long/url\x1b\\alpha-two command\x1b]8;;\x1b\\"
	if n := visualLen(link2); n != 17 {
		t.Errorf("visualLen of long OSC8 link = %d, want 17 (visible text 'alpha-two command')", n)
	}
}

func TestUniformWrappingWithLinks(t *testing.T) {
	// Reflow text containing inline links and verify all mid-lines are
	// uniformly filled (none significantly shorter than the target width).
	text := "This is a **bold command** with a [link to docs](https://docs.example.com/very/long/path) and some more text that should wrap nicely across multiple lines without any single line being too short because of the embedded link escape codes that would previously corrupt the visual length calculation."
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 40, 2, "", inline(text))
	plain := stripANSI(buf.String())

	lines := strings.Split(strings.TrimSpace(plain), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 wrapped lines, got %d:\n%s", len(lines), plain)
	}

	// Mid-lines (not first, not last) should all be close to the target width
	maxGap := 0
	for i, line := range lines {
		l := len([]rune(line))
		gap := 40 - l
		if gap > maxGap {
			maxGap = gap
		}
		// First line has indent=2, so it starts at column 2
		if i == 0 && l > 38 {
			continue
		}
		// Last line can be short
		if i == len(lines)-1 {
			continue
		}
		// Mid-lines should be at least 60% of target width
		if gap > 16 {
			t.Errorf("line %d: len=%d (gap=%d), too short at width 40:\n  %q", i, l, gap, line)
		}
	}
	t.Logf("uniform wrapping: max gap from width 40 = %d chars (mid-lines)", maxGap)
}
func TestExecuteHelpUnknown(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:        "build",
				Description: "Build something",
			},
		},
	}

	// This should return an error, not succeed silently
	err := app.ExecuteContext(context.Background(), []string{"help", "nonexistent"})
	if err == nil {
		t.Error("ExecuteContext with help nonexistent should return error, got nil")
	}

	// Test that help with valid command works
	err = app.ExecuteContext(context.Background(), []string{"help", "build"})
	if err != nil {
		t.Errorf("ExecuteContext with help build should succeed, got error: %v", err)
	}
}

func TestExecuteGlobalFlagsBound(t *testing.T) {
	var globalVerbose bool
	var globalQuiet bool

	app := &App{
		Name: "testapp",
		GlobalFlags: []Option{
			Bool(&globalVerbose, "--verbose", false, "Verbose output"),
			Bool(&globalQuiet, "--quiet", false, "Quiet output"),
		},
		Commands: []Command{
			{
				Name: "build",
				Run: func(ctx *Context) error {
					if globalVerbose {
						t.Log("Verbose flag was set")
					}
					if globalQuiet {
						t.Log("Quiet flag was set")
					}
					return nil
				},
			},
		},
	}

	// Test that GlobalFlags are parsed
	err := app.ExecuteContext(context.Background(), []string{"build", "--verbose"})
	if err != nil {
		t.Errorf("ExecuteContext with global flag should succeed, got error: %v", err)
	}
}

func TestOptionDeprecation(t *testing.T) {
	var file string
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:        "build",
				Description: "Build something",
				Options: []Option{
					{
						Flags:       "-f, --file PATH",
						Description: "Input file path",
						Deprecated:  "Use --input instead",
						Binder: func(fs *pflag.FlagSet) error {
							fs.StringVarP(&file, "file", "f", "", "Input file path")
							return nil
						},
					},
				},
			},
		},
	}

	// 1. Verify description rendering includes deprecation text
	o, buf := captureOptions(80)
	app.RenderCommand(o, "build")
	helpText := buf.String()
	if !strings.Contains(helpText, "(deprecated: Use --input instead)") {
		t.Errorf("expect deprecation text in help message, got:\n%q", helpText)
	}

	// 2. Verify warning prints to stderr during run
	res := TestExecute(app, []string{"build", "-f", "test.txt"})
	res.AssertNoError(t)
	res.AssertStderrContains(t, "Warning: flag --file is deprecated: Use --input instead")
}

func TestRequiredFlagConstraints(t *testing.T) {
	var format string
	var force bool
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:        "export",
				Description: "Export data",
				Options: []Option{
					Required(String(&format, "--format <fmt>", "", "Output format")),
					Required(Bool(&force, "--force", false, "Force export")),
				},
				Run: func(ctx *Context) error {
					return nil
				},
			},
		},
	}

	// 1. Verify description rendering includes (required)
	o, buf := captureOptions(80)
	app.RenderCommand(o, "export")
	helpText := buf.String()
	if !strings.Contains(helpText, "Output format (required)") {
		t.Errorf("expect Output format (required) text in help, got:\n%s", helpText)
	}
	if !strings.Contains(helpText, "Force export (required)") {
		t.Errorf("expect Force export (required) text in help, got:\n%s", helpText)
	}

	// 2. Verify missing required flags fail validation without TTY fallback
	resNoTTY := TestExecute(app, []string{"export"})
	resNoTTY.AssertErrorContains(t, "required flag(s)")

	// 3. Verify interactive fallback prompts on stderr and constructs tip
	app.InteractiveFallback = true
	// Input 1 for format (text input: "json"), Input 2 for force (select choice 1: "true")
	stdinBuf := bytes.NewBufferString("json\n1\n")
	resTTY := TestExecuteWithStdin(app, []string{"export"}, stdinBuf)
	resTTY.AssertNoError(t)
	resTTY.AssertStderrContains(t, "Enter value for required flag --format")
	resTTY.AssertStderrContains(t, "select a value for required flag --force")
	resTTY.AssertStderrContains(t, "💡 Tip: Next time, you can run this directly with:")
	resTTY.AssertStderrContains(t, "testapp export --force --format json")

	if format != "json" {
		t.Errorf("expected format to be 'json', got %q", format)
	}
	if !force {
		t.Errorf("expected force to be true, got %v", force)
	}
}

func TestOptionsRelationValidators(t *testing.T) {
	var json, yaml, cert, key, upload, bucket, token, authMethod string
	var commands = []Command{
		{
			Name:        "output",
			Description: "Validate mutually exclusive and required together",
			Options: []Option{
				String(&json, "--json <file>", "", "JSON output"),
				String(&yaml, "--yaml <file>", "", "YAML output"),
				String(&cert, "--cert <file>", "", "Cert path"),
				String(&key, "--key <file>", "", "Key path"),
			},
			OptionsValidator: ValidateOptions(
				MutuallyExclusive("--json", "--yaml"),
				RequiredTogether("--cert", "--key"),
			),
		},
		{
			Name:        "storage",
			Description: "Validate dependent requirements",
			Options: []Option{
				String(&upload, "--upload <file>", "", "Upload file"),
				String(&bucket, "--bucket <name>", "", "Target bucket"),
				String(&token, "--token <val>", "", "Auth token"),
				String(&authMethod, "--auth-method <mode>", "", "Authentication method"),
			},
			OptionsValidator: ValidateOptions(
				RequiredWith("--upload", "--bucket"),
				RequiredIf("--token", "--auth-method=token"),
			),
		},
	}

	app := &App{Name: "testapp", Commands: commands}

	// 1. Test mutually exclusive flags fail
	res := TestExecute(app, []string{"output", "--json", "j.json", "--yaml", "y.yaml"})
	res.AssertErrorContains(t, "mutually exclusive")

	// 2. Test required together fails when one is missing
	res = TestExecute(app, []string{"output", "--cert", "c.pem"})
	res.AssertErrorContains(t, "must be used together")

	// 3. Test required together succeeds when both are present
	res = TestExecute(app, []string{"output", "--cert", "c.pem", "--key", "k.pem"})
	res.AssertNoError(t)

	// 4. Test RequiredWith fails when dependent flag is missing
	res = TestExecute(app, []string{"storage", "--upload", "file.txt"})
	res.AssertErrorContains(t, "flag --bucket is required when using --upload")

	// 5. Test RequiredIf fails when condition matches but flag is missing
	res = TestExecute(app, []string{"storage", "--auth-method", "token"})
	res.AssertErrorContains(t, "flag --token is required when auth-method is set to \"token\"")

	// 6. Test RequiredIf succeeds when condition matches and flag is present
	res = TestExecute(app, []string{"storage", "--auth-method", "token", "--token", "secret"})
	res.AssertNoError(t)
}

func TestAuditHelper(t *testing.T) {
	// 1. Missing Description should fail audit
	badApp1 := &App{
		Commands: []Command{
			{Name: "build"}, // missing description
		},
	}
	if err := Audit(badApp1); err == nil || !strings.Contains(err.Error(), "missing a Description") {
		t.Errorf("expected audit error for missing description, got: %v", err)
	}

	// 2. Duplicate subcommand name should fail audit
	badApp2 := &App{
		Commands: []Command{
			{Name: "build", Description: "Build"},
			{Name: "build", Description: "Duplicate"},
		},
	}
	if err := Audit(badApp2); err == nil || !strings.Contains(err.Error(), "duplicate subcommand name") {
		t.Errorf("expected audit error for duplicate subcommand, got: %v", err)
	}

	// 3. Duplicate flag/option name should fail audit
	badApp3 := &App{
		Commands: []Command{
			{
				Name:        "build",
				Description: "Build",
				Options: []Option{
					{Flags: "-v, --verbose", Description: "v1", Binder: func(fs *pflag.FlagSet) error { return nil }},
					{Flags: "--verbose", Description: "v2", Binder: func(fs *pflag.FlagSet) error { return nil }},
				},
			},
		},
	}
	if err := Audit(badApp3); err == nil || !strings.Contains(err.Error(), "duplicate option") {
		t.Errorf("expected audit error for duplicate option, got: %v", err)
	}

	// 4. Inconsistent path permutations (scan spam vs spam scan) should fail audit
	badApp4 := &App{
		Commands: []Command{
			{
				Name:        "scan",
				Description: "Scan category",
				Subcommands: []Command{
					{Name: "spam", Description: "Scan spam"},
				},
			},
			{
				Name:        "spam",
				Description: "Spam category",
				Subcommands: []Command{
					{Name: "scan", Description: "Spam scan"},
				},
			},
		},
	}
	if err := Audit(badApp4); err == nil || !strings.Contains(err.Error(), "inconsistent path permutation detected") {
		t.Errorf("expected audit error for path permutations, got: %v", err)
	}

	// 5. Whitelisted path permutation should succeed audit
	err := AuditWithOptions(badApp4, AuditOptions{
		AllowPathPermutations: [][]string{
			{"scan", "spam"},
		},
	})
	if err != nil {
		t.Errorf("expected whitelisted permutation to pass audit, got error: %v", err)
	}
}
