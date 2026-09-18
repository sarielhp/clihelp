package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

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
	app.Shortcuts = []clihelp.Command{
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

// RenderMarkdown owns the pages it writes, not the directory: a page of an
// earlier pass that no longer corresponds to a command is pruned, while a file
// the generator never wrote is left alone. Deleting every .md it did not write
// destroyed a directory that already held documentation, on the first run.
func TestRenderMarkdownPrunesItsOwnPagesOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLIHELP_GEN", "1")

	app := testApp()
	if _, err := RenderMarkdown(app, MarkdownOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "build.md")); err != nil {
		t.Fatalf("expected build.md from the first pass: %v", err)
	}

	handPath := filepath.Join(dir, "handwritten.md")
	if err := os.WriteFile(handPath, []byte("hand written"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Drop the command whose page was generated above.
	var kept []clihelp.Command
	for _, c := range app.Commands {
		if c.Name != "build" {
			kept = append(kept, c)
		}
	}
	app.Commands = kept

	if _, err := RenderMarkdown(app, MarkdownOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "build.md")); !os.IsNotExist(err) {
		t.Error("build.md was not pruned after its command was removed")
	}
	if _, err := os.Stat(handPath); err != nil {
		t.Errorf("a file the generator never wrote was deleted: %v", err)
	}
}

// The hash records what was generated, not what survived: a page deleted by hand
// left the hash matching and the page missing for good.
func TestRenderMarkdownRestoresADeletedPage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLIHELP_GEN", "1")

	app := testApp()
	if _, err := RenderMarkdown(app, MarkdownOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}

	page := filepath.Join(dir, "build.md")
	if err := os.Remove(page); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CLIHELP_GEN", "")
	changed, err := RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected a regeneration pass after a page was deleted")
	}
	if _, err := os.Stat(page); err != nil {
		t.Errorf("build.md was not restored: %v", err)
	}
}

func TestMarkdownTablesEscapePipes(t *testing.T) {
	app := &clihelp.App{
		Name: "app",
		Commands: []clihelp.Command{{
			Name:        "run",
			Description: "Run it",
			Parameters: []clihelp.Param{
				{Name: "a|b", Description: "either a | b"},
			},
			Subcommands: []clihelp.Command{
				{Name: "now", Description: "run a | b now"},
			},
		}},
		GlobalFlags: []clihelp.Option{
			{Flags: "--mode <a|b>", Description: "one of a | b"},
		},
	}
	pages, err := renderMarkdownPages(app)
	if err != nil {
		t.Fatal(err)
	}
	for name, page := range pages {
		for _, line := range strings.Split(page, "\n") {
			if !strings.HasPrefix(line, "| ") {
				continue
			}
			if n := strings.Count(line, "|") - strings.Count(line, "\\|"); n != 3 {
				t.Errorf("%s: table row has %d unescaped pipes, want 3:\n%s", name, n, line)
			}
		}
	}
}

func TestRenderNavPage(t *testing.T) {
	app := testApp()
	out := renderNav(app)

	for _, want := range []string{
		"# podctl — Navigation",
		"## Commands",
		"- [build](build.md)",
		"- [config](config.md)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("nav page missing %q\n%s", want, out)
		}
	}
}

func TestRenderMarkdownIncludesNavPage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLIHELP_GEN", "1")

	app := testApp()
	_, err := RenderMarkdown(app, MarkdownOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	navPath := filepath.Join(dir, "nav.md")
	if _, err := os.Stat(navPath); err != nil {
		t.Errorf("nav.md not generated: %v", err)
	}

	data, err := os.ReadFile(navPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[build](build.md)") {
		t.Errorf("nav.md missing build link: %s", data)
	}
}

func TestRenderCommandPageHasFooter(t *testing.T) {
	app := testApp()
	nodes := collectNodes(app)
	out := renderCommandPage(app, nodes[0])

	if !strings.Contains(out, "[↑ podctl](index.md)") {
		t.Errorf("command page missing parent link: %s", out)
	}
	if !strings.Contains(out, "[nav](nav.md)") {
		t.Errorf("command page missing nav link: %s", out)
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

func TestRenderMarkdownSkipsHiddenCommands(t *testing.T) {
	app := testApp()
	app.Commands = append(app.Commands, clihelp.Command{Name: "secret", Hidden: true})
	pages, err := renderMarkdownPages(app)
	if err != nil {
		t.Fatal(err)
	}
	for name := range pages {
		if name == "secret.md" {
			t.Error("hidden command produced a markdown page")
		}
	}
	if strings.Contains(pages["index.md"], "secret") {
		t.Error("hidden command appeared in index.md")
	}
	if strings.Contains(pages["nav.md"], "secret") {
		t.Error("hidden command appeared in nav.md")
	}
}

func TestRenderMarkdownSlugCollisionErrors(t *testing.T) {
	app := &clihelp.App{
		Name: "collide",
		Commands: []clihelp.Command{
			{Name: "set-up"},
			{Name: "set up"},
		},
	}
	_, err := renderMarkdownPages(app)
	if err == nil {
		t.Fatal("expected slug collision error, got nil")
	}
}

func TestPageHeaderQuotesFrontMatter(t *testing.T) {
	got := pageHeader(pageMeta{title: `weird: "title" with 'quote'`, parent: `p: 'q'`})
	for _, want := range []string{"title: 'weird: \"title\" with ''quote'''", "parent: 'p: ''q'''"} {
		if !strings.Contains(got, want) {
			t.Errorf("pageHeader missing quoted field %q, got:\n%s", want, got)
		}
	}
}

func testApp() *clihelp.App {
	return &clihelp.App{
		Name:    "podctl",
		Version: "1.0.0",
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile audio episodes",
				UsageLine:   "podctl build [options] <file>",
				Options: []clihelp.Option{
					{Flags: "-o, --output PATH", Description: "Write output to PATH"},
					{Flags: "--verbose", Description: "Enable verbose logging"},
				},
				Examples: []clihelp.Example{{Line: "podctl build ep.wav"}},
			},
			{
				Name:        "config",
				Description: "Manage configuration",
				UsageLine:   "podctl config <subcommand>",
				Subcommands: []clihelp.Command{
					{
						Name:        "set",
						Title:       "config set <key> <value>",
						Description: "Set a configuration value",
						UsageLine:   "podctl config set <key> <value>",
						Parameters: []clihelp.Param{
							{Name: "<key>", Description: "The key to set"},
							{Name: "<value>", Description: "The value to assign"},
						},
					},
				},
			},
		},
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
		{"a`b", "``a`b``"},
		{"`quoted`", "`` `quoted` ``"},
		{"two\nlines", "`two lines`"},
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
	app.GlobalFlags = []clihelp.Option{{Flags: "--verbose", Description: "Verbose output"}}
	app.ConfigPath = "/etc/podctl.yaml"
	out := renderIndex(app)

	for _, want := range []string{
		"# podctl",
		"A podcast distribution toolkit.",
		"## Commands",
		// A "Command -> clihelp.Command" rename, applied when doc became a
		// subpackage, reached two Markdown table headers inside string literals
		// and this golden string with them — so the published docs grew a column
		// called "clihelp.Command" and the test agreed.
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

func TestRenderCommandPageLongDescriptionAndRawNotes(t *testing.T) {
	app := &clihelp.App{
		Name: "testcli",
	}
	node := cmdNode{
		path: []string{"deploy"},
		cmd: clihelp.Command{
			Name:            "deploy",
			Description:     "Short description",
			LongDescription: "Full Markdown documentation for deploy command.",
			Notes: []clihelp.Note{
				{
					Heading: "Configuration Example",
					Text:    "host: localhost\nport: 8080",
					Raw:     true,
				},
			},
		},
	}

	out := renderCommandPage(app, node)

	if !strings.Contains(out, "Full Markdown documentation for deploy command.") {
		t.Errorf("expected LongDescription in Markdown page, got:\n%s", out)
	}
	if strings.Contains(out, "Short description") {
		t.Errorf("Short description should be replaced by LongDescription, got:\n%s", out)
	}
	if !strings.Contains(out, "## Configuration Example") {
		t.Errorf("expected heading in Markdown page, got:\n%s", out)
	}
	expectedCodeBlock := "```\nhost: localhost\nport: 8080\n```"
	if !strings.Contains(out, expectedCodeBlock) {
		t.Errorf("expected raw note wrapped in fenced code block, got:\n%s", out)
	}
}

// The page's subcommand list and the terminal help's have to be the same list.
// They were not: clihelp prefers an explicit SubcommandEntries over walking the
// Subcommands tree — that field exists to document subcommands the tree does not
// carry — and the copy of that helper made when this became a subpackage lost
// the preference, so entries that had no Command behind them vanished from the
// docs while `--help` went on showing them.
func TestSubcommandTableMatchesTheTerminalHelp(t *testing.T) {
	cmd := clihelp.Command{
		Name:        "whitelist",
		Description: "Manage the whitelist.",
		UsageLine:   "app whitelist <subcommand>",
		SubcommandEntries: []clihelp.Param{
			{Name: "add <email>", Description: "Add an address."},
			{Name: "del <email>", Description: "Remove an address."},
			{Name: "list", Description: "List every address."},
		},
		Subcommands: []clihelp.Command{
			{Name: "add", Description: "Add an address.", Run: func(*clihelp.Context) error { return nil }},
			{Name: "del", Description: "Remove an address.", Run: func(*clihelp.Context) error { return nil }},
		},
	}
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{cmd}}
	page := renderCommandPage(app, cmdNode{path: []string{"whitelist"}, cmd: cmd})

	// Every documented entry appears, including the one with no Command behind it.
	// The angle brackets are escaped for markdown, as they are everywhere else.
	for _, want := range []string{`add \<email>`, `del \<email>`, "| list |"} {
		if !strings.Contains(page, want) {
			t.Errorf("the subcommand table is missing %q:\n%s", want, page)
		}
	}
	// The two that do have pages are still linked, by their command word.
	for _, want := range []string{"(whitelist-add.md)", "(whitelist-del.md)"} {
		if !strings.Contains(page, want) {
			t.Errorf("the subcommand table lost the link %q:\n%s", want, page)
		}
	}
}
