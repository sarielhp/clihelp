package clihelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
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

	for _, want := range []string{"Usage of podctl:", "build", "Compile audio episodes", "config", "Manage configuration", "Version:", "1.0.0"} {
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
		"Detailed Usage: build", "Description:", "Compile audio episodes",
		"Usage:", "podctl build [options] <file>",
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
	for _, want := range []string{"Detailed Usage: config set <key> <value>", "Parameters:", "<key>", "The value to assign"} {
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
	if !strings.Contains(strip(cbuf.String()), "Detailed Usage: build") {
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
	if !strings.Contains(out, "See docs for account setup.") {
		t.Errorf("global output missing App.GlobalNote")
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
		t.Errorf("custom TitlePrefix not applied")
	}
}

func TestMarkdownSlug(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"build", "build"},
		{"config", "config"},
		{"config set", "config-set"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"mixedCase", "mixedcase"},
		{"special!@#chars", "special-chars"},
		{"a__b__c", "a-b-c"},
		{"---d---", "d"},
		{"", ""},
	}
	for _, tc := range tests {
		got := markdownSlug(tc.in)
		if got != tc.want {
			t.Errorf("markdownSlug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMarkdownRelFile(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{"build"}, "build.md"},
		{[]string{"config", "set"}, "config-set.md"},
		{[]string{"config", "set", "time"}, "config-set-time.md"},
		{[]string{"my cmd"}, "my-cmd.md"},
		{nil, "index.md"},
		{[]string{}, "index.md"},
	}
	for _, tc := range tests {
		got := markdownRelFile(tc.path)
		if got != tc.want {
			t.Errorf("markdownRelFile(%v) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestMdInline(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain text", "plain text"},
		{"*italic*", `\*italic\*`},
		{"_underscore_", `\_underscore\_`},
		{"`backtick`", "\\`backtick\\`"},
		{"[bracket]", `\[bracket\]`},
		{"<tag>", `\<tag>`},
		{"back\\slash", `back\\slash`},
		{"mixed * _ ` [ ] < >", "mixed \\* \\_ \\` \\[ \\] \\< >"},
	}
	for _, tc := range tests {
		got := mdInline(tc.in)
		if got != tc.want {
			t.Errorf("mdInline(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMdCode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"foo", "`foo`"},
		{"-o, --output", "`-o, --output`"},
		{"a`b", "`a\\`b`"},
	}
	for _, tc := range tests {
		got := mdCode(tc.in)
		if got != tc.want {
			t.Errorf("mdCode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderIndex(t *testing.T) {
	app := testApp()
	app.Description = "A podcast distribution toolkit."
	app.GlobalNote = "See docs."
	app.GlobalFlags = []Option{{Flags: "--verbose", Description: "Verbose output"}}
	app.ConfigPath = "/etc/podctl.yaml"
	out := renderIndex(app)

	for _, want := range []string{
		"# podctl",
		"A podcast distribution toolkit.",
		"## Commands",
		"| Command | Description |",
		"| [build](build.md) |",
		"| [config](config.md) |",
		"## Global Flags",
		"| Flag | Description |",
		"`--verbose`",
		"## Version",
		"1.0.0",
		"## Config file location",
		"`/etc/podctl.yaml`",
		"## About",
		"See docs.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderIndex missing %q\n%s", want, out)
		}
	}
}

func TestRenderCommandPage(t *testing.T) {
	app := testApp()
	nodes := collectNodes(app)
	if len(nodes) == 0 {
		t.Fatal("collectNodes returned empty")
	}
	// First node should be "build"
	out := renderCommandPage(app, nodes[0])
	for _, want := range []string{
		"# podctl build",
		"Compile audio episodes",
		"## Usage",
		"```",
		"podctl build [options] <file>",
		"## Flags",
		"`-o, --output PATH`",
		"`--verbose`",
		"## Examples",
		"`podctl build ep.wav`",
		"[↑ podctl](index.md)",
		"[nav](nav.md)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderCommandPage(build) missing %q\n%s", want, out)
		}
	}
}

func TestRenderCommandPageNested(t *testing.T) {
	app := testApp()
	nodes := collectNodes(app)
	var setNode cmdNode
	for _, n := range nodes {
		if len(n.path) == 2 && n.path[1] == "set" {
			setNode = n
			break
		}
	}
	if setNode.cmd.Name == "" {
		t.Fatal("did not find config/set node")
	}
	out := renderCommandPage(app, setNode)

	// Title is now full path: app name + command path
	if !strings.Contains(out, "# podctl config set") {
		t.Errorf("expected full-path title, got:\n%s", out)
	}
	for _, want := range []string{
		"## Parameters",
		"`<key>`",
		"`<value>`",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderCommandPage(config/set) missing %q", want)
		}
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

	// wrapW is capped at min(width, 80)
	expectedMax := 80

	maxLine := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
		if len([]rune(line)) > expectedMax {
			t.Errorf("line exceeds %d chars at width 100:\n  len=%d  %q", expectedMax, len([]rune(line)), line)
		}
	}
	t.Logf("width=100: max line = %d chars", maxLine)
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
