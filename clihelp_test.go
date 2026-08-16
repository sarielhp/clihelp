package clihelp

import (
	"bytes"
	"os"
	"path/filepath"
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
		"[build](build.md)",
		"[config](config.md)",
		"## Global Flags",
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
		"# build",
		"Compile audio episodes",
		"## Usage",
		"```",
		"podctl build [options] <file>",
		"## Flags",
		"`-o, --output PATH`",
		"`--verbose`",
		"## Examples",
		"`podctl build ep.wav`",
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

	// Title is used as heading; angle brackets are escaped by mdInline
	if !strings.Contains(out, "# config set \\<key> \\<value>") {
		t.Errorf("expected escaped Title as heading, got:\n%s", out)
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

func TestRenderMarkdownRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLIHELP_GEN", "1")

	app := testApp()
	changed, err := RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true on first generation")
	}

	// Check expected files exist
	for _, name := range []string{"index.md", "build.md", "config.md", "config-set.md"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s: %v", name, err)
		}
	}

	// Hash sidecar should exist
	hashPath := filepath.Join(dir, ".clihelp-hash")
	if _, err := os.Stat(hashPath); err != nil {
		t.Errorf("expected .clihelp-hash: %v", err)
	}

	// Re-run with same content but WITHOUT CLIHELP_GEN — should be unchanged
	t.Setenv("CLIHELP_GEN", "")
	changed, err = RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false on unchanged re-run without CLIHELP_GEN")
	}

	// Modify the app tree so hash changes
	app.Version = "2.0.0"
	changed, err = RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true after app modification")
	}

	// index should now contain the new version
	idx, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "2.0.0") {
		t.Errorf("index.md missing updated version\n%s", idx)
	}
}

func TestRenderMarkdownWithoutEnv(t *testing.T) {
	dir := t.TempDir()
	// No CLIHELP_GEN set
	app := testApp()
	changed, err := RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false when CLIHELP_GEN is not set and dir is empty")
	}
	// No .md files should have been created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("unexpected .md file created: %s", e.Name())
		}
	}
}

func TestRenderMarkdownCollectIncludesShortcuts(t *testing.T) {
	app := testApp()
	app.Shortcuts = []Command{
		{Name: "ls", Description: "List running tasks"},
	}
	nodes := collectNodes(app)
	found := false
	for _, n := range nodes {
		if n.cmd.Name == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("collectNodes did not include shortcut command")
	}
}

func TestRenderMarkdownOrphanPruning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLIHELP_GEN", "1")

	app := testApp()
	_, err := RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	// Write an orphan .md file
	orphanPath := filepath.Join(dir, "orphan.md")
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-generate — orphan should be pruned
	_, err = RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("orphan.md was not pruned")
	}
}

func TestRenderMarkdownHashConsistency(t *testing.T) {
	app := testApp()
	pages1, _ := renderMarkdownPages(app)
	pages2, _ := renderMarkdownPages(app)
	h1 := markdownHash(pages1)
	h2 := markdownHash(pages2)
	if h1 != h2 {
		t.Error("markdownHash should be deterministic")
	}
}

func TestRenderInlinePlainText(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "hello world")
	if buf.String() != "hello world" {
		t.Errorf("plain text: got %q", buf.String())
	}
}

func TestRenderInlineBold(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "**bold**")
	want := "\x1b[1mbold\x1b[22m"
	if buf.String() != want {
		t.Errorf("bold: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineItalic(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "*italic*")
	want := "\x1b[3mitalic\x1b[23m"
	if buf.String() != want {
		t.Errorf("italic: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineCode(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "use `--flag` here")
	want := "use \x1b[32m--flag\x1b[39m here"
	if buf.String() != want {
		t.Errorf("code: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineLink(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "see [docs](https://example.com)")
	want := "see \x1b]8;;https://example.com\x1b\\docs\x1b]8;;\x1b\\"
	if buf.String() != want {
		t.Errorf("link: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineStrikethrough(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "~~strike~~")
	want := "\x1b[9mstrike\x1b[29m"
	if buf.String() != want {
		t.Errorf("strikethrough: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineBackslashEscape(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "\\*not italic\\*")
	if buf.String() != "*not italic*" {
		t.Errorf("escape: got %q, want %q", buf.String(), "*not italic*")
	}
}

func TestRenderInlineCodeEscapesMarkdown(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "`**not bold**`")
	want := "\x1b[32m**not bold**\x1b[39m"
	if buf.String() != want {
		t.Errorf("code escapes: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineUnmatchedDelimiter(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "*unclosed")
	if buf.String() != "*unclosed" {
		t.Errorf("unmatched: got %q, want %q", buf.String(), "*unclosed")
	}
}

func TestRenderInlineMixed(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "**bold** and *italic* and `code`")
	got := buf.String()
	if !strings.Contains(got, "\x1b[1mbold\x1b[22m") {
		t.Errorf("mixed missing bold: %q", got)
	}
	if !strings.Contains(got, "\x1b[3mitalic\x1b[23m") {
		t.Errorf("mixed missing italic: %q", got)
	}
	if !strings.Contains(got, "\x1b[32mcode\x1b[39m") {
		t.Errorf("mixed missing code: %q", got)
	}
}

func TestRenderCommandPageUsesTables(t *testing.T) {
	app := testApp()
	nodes := collectNodes(app)
	out := renderCommandPage(app, nodes[0])
	for _, want := range []string{
		"| Flag | Description |",
		"|------|-------------|",
		"| `-o, --output PATH` |",
		"| `--verbose` |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("build page missing table row %q\n%s", want, out)
		}
	}
}

func TestRenderCommandPageNestedUsesTables(t *testing.T) {
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
	for _, want := range []string{
		"| Parameter | Description |",
		"|-----------|-------------|",
		"| `<key>` |",
		"| `<value>` |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("config/set page missing table row %q\n%s", want, out)
		}
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
