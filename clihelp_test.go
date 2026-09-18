package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

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

// strip is the library's own stripper, not the third-party one.
//
// stripansi does not understand OSC sequences: given one of this package's
// hyperlinks it does not merely miss it, it eats seven bytes out of the middle
// and leaves the URL visible. Forty assertions here read its output, and
// TestExampleAppNoBareMarkdownAndNoVisibleURLs — whose whole job is proving URLs
// stay hidden — would have reported a visible URL had it used this helper.
func strip(s string) string { return StripANSI(s) }

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
	ran := false

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
					ran = true
					// The binding is asserted here, inside the handler, because
					// that is where an application reads a global flag. Both
					// values were merely logged before, so the flags could have
					// stopped binding to their targets entirely with this test
					// green — it checked only that Execute returned no error.
					if !globalVerbose {
						t.Error("--verbose was given and the bound variable is false")
					}
					if globalQuiet {
						t.Error("--quiet was not given and the bound variable is true")
					}
					return nil
				},
			},
		},
	}

	if err := app.ExecuteContext(context.Background(), []string{"build", "--verbose"}); err != nil {
		t.Errorf("ExecuteContext with global flag should succeed, got error: %v", err)
	}
	if !ran {
		t.Error("the command handler never ran, so nothing above was checked")
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
	res := testExecute(app, []string{"build", "-f", "test.txt"})
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
	resNoTTY := testExecute(app, []string{"export"})
	resNoTTY.AssertErrorContains(t, "required flag(s)")

	// 3. Verify interactive fallback prompts on stderr and constructs tip
	app.InteractiveFallback = true
	// Input 1 for format (text input: "json"), Input 2 for force (select choice 1: "true")
	stdinBuf := bytes.NewBufferString("json\n1\n")
	resTTY := testExecuteWithStdin(app, []string{"export"}, stdinBuf)
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
	res := testExecute(app, []string{"output", "--json", "j.json", "--yaml", "y.yaml"})
	res.AssertErrorContains(t, "mutually exclusive")

	// 2. Test required together fails when one is missing
	res = testExecute(app, []string{"output", "--cert", "c.pem"})
	res.AssertErrorContains(t, "must be used together")

	// 3. Test required together succeeds when both are present
	res = testExecute(app, []string{"output", "--cert", "c.pem", "--key", "k.pem"})
	res.AssertNoError(t)

	// 4. Test RequiredWith fails when dependent flag is missing
	res = testExecute(app, []string{"storage", "--upload", "file.txt"})
	res.AssertErrorContains(t, "flag --bucket is required when using --upload")

	// 5. Test RequiredIf fails when condition matches but flag is missing
	res = testExecute(app, []string{"storage", "--auth-method", "token"})
	res.AssertErrorContains(t, "flag --token is required when auth-method is set to \"token\"")

	// 6. Test RequiredIf succeeds when condition matches and flag is present
	res = testExecute(app, []string{"storage", "--auth-method", "token", "--token", "secret"})
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
	err := Audit(badApp4, AuditOptions{
		AllowPathPermutations: [][]string{
			{"scan", "spam"},
		},
	})
	if err != nil {
		t.Errorf("expected whitelisted permutation to pass audit, got error: %v", err)
	}
}
