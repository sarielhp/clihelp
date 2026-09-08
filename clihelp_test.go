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
