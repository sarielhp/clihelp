package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellCompletionEmptyArgsIncludesShortcuts(t *testing.T) {
	var outBuf bytes.Buffer
	app := &App{
		Name:   "podcli",
		Stdout: &outBuf,
		Commands: []Command{
			{Name: "scan", Description: "Scan podcasts"},
		},
		Shortcuts: []Command{
			{Name: "quick", Description: "Quick action"},
		},
	}

	err := app.ExecuteContext(context.Background(), []string{"__complete"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "quick\tQuick action") {
		t.Errorf("expected shortcut in empty args completion, got: %q", outBuf.String())
	}
}

func TestShellCompletionProtocol(t *testing.T) {
	var outBuf bytes.Buffer
	var podcastVal string
	var fillVal bool

	app := &App{
		Name:   "podcli",
		Stdout: &outBuf,
		Commands: []Command{
			{
				Name:        "scan",
				Aliases:     []string{"rescan"},
				Description: "Scan podcasts",
				Options: []Option{
					{
						Flags:       "-p, --podcast <id>",
						Description: "Podcast ID",
						Complete: func(toComplete string) []string {
							candidates := []string{"pod1\tHistory podcast", "pod2\tTech podcast"}
							var res []string
							for _, c := range candidates {
								if strings.HasPrefix(c, toComplete) {
									res = append(res, c)
								}
							}
							return res
						},
						Binder: String(&podcastVal, "-p, --podcast <id>", "", "Podcast ID").Binder,
					},
					Bool(&fillVal, "-f, --fill", false, "Fill gaps"),
				},
			},
			{
				Name:        "download",
				Description: "Download episodes",
			},
		},
		Shortcuts: []Command{
			{
				Name:        "quick",
				Description: "Quick scan and download",
			},
		},
	}

	// 1. Root command completion
	outBuf.Reset()
	err := app.ExecuteContext(context.Background(), []string{"__complete", "sc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "scan\tScan podcasts") {
		t.Errorf("expected scan suggestion, got: %q", outBuf.String())
	}

	// 2. Subcommand flag completion
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"__complete", "scan", "--"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "--podcast\tPodcast ID") || !strings.Contains(outBuf.String(), "--fill\tFill gaps") {
		t.Errorf("expected flag suggestions, got: %q", outBuf.String())
	}

	// 3. Dynamic flag value completion
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"__complete", "scan", "-p", "pod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "pod1\tHistory podcast") || !strings.Contains(outBuf.String(), "pod2\tTech podcast") {
		t.Errorf("expected dynamic value completions, got: %q", outBuf.String())
	}

	// 4. Shortcut command completion
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"__complete", "qu"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "quick\tQuick scan and download") {
		t.Errorf("expected shortcut suggestion, got: %q", outBuf.String())
	}
}

func TestShellCompletionGenerators(t *testing.T) {
	app := &App{Name: "my-app"}

	var bashBuf bytes.Buffer
	if err := GenBashCompletion(app, &bashBuf); err != nil {
		t.Fatalf("GenBashCompletion error: %v", err)
	}
	if !strings.Contains(bashBuf.String(), "_my_app_complete") || !strings.Contains(bashBuf.String(), "complete -o default -F _my_app_complete my-app") {
		t.Errorf("invalid bash completion script: %s", bashBuf.String())
	}

	var zshBuf bytes.Buffer
	if err := GenZshCompletion(app, &zshBuf); err != nil {
		t.Fatalf("GenZshCompletion error: %v", err)
	}
	if !strings.Contains(zshBuf.String(), "#compdef my-app") || !strings.Contains(zshBuf.String(), "_my_app") {
		t.Errorf("invalid zsh completion script: %s", zshBuf.String())
	}

	var fishBuf bytes.Buffer
	if err := GenFishCompletion(app, &fishBuf); err != nil {
		t.Fatalf("GenFishCompletion error: %v", err)
	}
	if !strings.Contains(fishBuf.String(), "function __fish_my_app_complete") || !strings.Contains(fishBuf.String(), "complete -c my-app -f -a '(__fish_my_app_complete)'") {
		t.Errorf("invalid fish completion script: %s", fishBuf.String())
	}
}

func TestInstallCompletion(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, "config"))

	app := &App{Name: "testcli"}

	tests := []struct {
		shell       string
		expectedSub string
		wantFile    string
	}{
		{"bash", "bash-completion/completions", "testcli"},
		{"zsh", "zsh/site-functions", "_testcli"},
		{"fish", "fish/completions", "testcli.fish"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			path, err := installCompletion(app, tt.shell)
			if err != nil {
				t.Fatalf("installCompletion(%q) failed: %v", tt.shell, err)
			}
			if !strings.Contains(path, tt.expectedSub) || filepath.Base(path) != tt.wantFile {
				t.Errorf("unexpected path %q, want filename %q in %q", path, tt.wantFile, tt.expectedSub)
			}
			if _, err := os.Stat(path); err != nil {
				t.Errorf("installed file does not exist: %v", err)
			}
		})
	}

	t.Run("DefaultShellDetection", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/zsh")
		path, err := installCompletion(app, "")
		if err != nil {
			t.Fatalf("installCompletion(\"\") with SHELL=zsh failed: %v", err)
		}
		if filepath.Base(path) != "_testcli" {
			t.Errorf("expected zsh installation, got: %s", path)
		}
	})

	t.Run("UnsupportedShell", func(t *testing.T) {
		_, err := installCompletion(app, "unknown_shell")
		if err == nil {
			t.Fatal("expected error for unsupported shell, got nil")
		}
	})
}

func TestCompletionCommand(t *testing.T) {
	cmd := CompletionCommand()
	if cmd.Name != "completion" {
		t.Errorf("expected command name 'completion', got %q", cmd.Name)
	}

	subNames := make([]string, len(cmd.Subcommands))
	for i, sub := range cmd.Subcommands {
		subNames[i] = sub.Name
	}
	expected := []string{"bash", "zsh", "fish", "keys", "install", "uninstall"}
	if len(subNames) != len(expected) {
		t.Fatalf("expected subcommands %v, got %v", expected, subNames)
	}
	for i, exp := range expected {
		if subNames[i] != exp {
			t.Errorf("expected subcommand %d to be %q, got %q", i, exp, subNames[i])
		}
	}

	// Test executing completion subcommands on an app
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, "data"))

	var outBuf, errBuf bytes.Buffer
	app := &App{
		Name:     "myapp",
		Stdout:   &outBuf,
		Stderr:   &errBuf,
		Commands: []Command{cmd},
	}

	if err := app.ExecuteContext(context.Background(), []string{"completion", "bash"}); err != nil {
		t.Fatalf("completion bash failed: %v", err)
	}
	if !strings.Contains(outBuf.String(), "_myapp_complete") {
		t.Errorf("expected bash script in stdout, got: %s", outBuf.String())
	}

	outBuf.Reset()
	if err := app.ExecuteContext(context.Background(), []string{"completion", "install", "bash"}); err != nil {
		t.Fatalf("completion install bash failed: %v", err)
	}
	// The human report goes to stderr; stdout carries the generated file's path
	// so a script can capture it.
	if !strings.Contains(errBuf.String(), "shell integration installed") {
		t.Errorf("expected the installation report on stderr, got: %s", errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "shell") {
		t.Errorf("expected the generated path on stdout, got: %s", outBuf.String())
	}
}

func TestShellCompletionEqualsForm(t *testing.T) {
	var outBuf bytes.Buffer
	var unit string
	app := &App{
		Name:   "deploy",
		Stdout: &outBuf,
		Commands: []Command{
			{
				Name: "go",
				Options: []Option{
					Enum(&unit, "--unit <size>", []string{"MB", "GB"}, "MB", "size unit"),
				},
			},
		},
	}
	err := app.ExecuteContext(context.Background(), []string{"__complete", "go", "--unit="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := outBuf.String()
	for _, want := range []string{"--unit=MB", "--unit=GB"} {
		if !strings.Contains(out, want) {
			t.Errorf("equals-form completion missing %q, got: %q", want, out)
		}
	}
}

func TestShellCompletionDedupeShortcuts(t *testing.T) {
	var outBuf bytes.Buffer
	app := &App{
		Name:   "dedupe",
		Stdout: &outBuf,
		Commands: []Command{
			{Name: "scan", Description: "Scan"},
		},
		Shortcuts: []Command{
			{Name: "scan", Description: "Duplicated shortcut"},
		},
	}
	if err := app.ExecuteContext(context.Background(), []string{"__complete"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(outBuf.String(), "scan\t"); got != 1 {
		t.Errorf("expected exactly one 'scan' completion, got %d:\n%s", got, outBuf.String())
	}
}

func TestShellCompletionPropagatesResolveError(t *testing.T) {
	var outBuf bytes.Buffer
	app := &App{
		Name:   "errcli",
		Stdout: &outBuf,
		Commands: []Command{
			{Name: "parent", Subcommands: []Command{{Name: "child"}}},
		},
	}
	err := app.ExecuteContext(context.Background(), []string{"__complete", "parent", "bogus", "x"})
	if err == nil {
		t.Fatal("expected resolve error to propagate from handleComplete, got nil")
	}
}

func TestCompletionPathAndIsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	app := &App{Name: "mytool"}

	zshPath, err := CompletionPath(app, "zsh")
	if err != nil {
		t.Fatalf("CompletionPath(zsh) error: %v", err)
	}
	expectedZsh := filepath.Join(tmpDir, "share", "zsh", "site-functions", "_mytool")
	if zshPath != expectedZsh {
		t.Errorf("got %q, want %q", zshPath, expectedZsh)
	}

	fishPath, err := CompletionPath(app, "fish")
	if err != nil {
		t.Fatalf("CompletionPath(fish) error: %v", err)
	}
	expectedFish := filepath.Join(tmpDir, "config", "fish", "completions", "mytool.fish")
	if fishPath != expectedFish {
		t.Errorf("got %q, want %q", fishPath, expectedFish)
	}

	bashPath, err := CompletionPath(app, "bash")
	if err != nil {
		t.Fatalf("CompletionPath(bash) error: %v", err)
	}
	expectedBash := filepath.Join(tmpDir, "share", "bash-completion", "completions", "mytool")
	if bashPath != expectedBash {
		t.Errorf("got %q, want %q", bashPath, expectedBash)
	}

	if IsCompletionInstalled(app, "zsh") {
		t.Errorf("expected IsCompletionInstalled to be false before install")
	}

	installedPath, err := installCompletion(app, "zsh")
	if err != nil {
		t.Fatalf("installCompletion error: %v", err)
	}
	if installedPath != expectedZsh {
		t.Errorf("installCompletion path = %q, want %q", installedPath, expectedZsh)
	}

	if !IsCompletionInstalled(app, "zsh") {
		t.Errorf("expected IsCompletionInstalled to be true after install")
	}
}

func TestAutoInstallCompletionOnExecute(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("SHELL", "/bin/zsh")
	// All four, as install_test.go, man_test.go, protocol_test.go and
	// autopath_test.go already do. This file had an incomplete copy, so the suite
	// failed for anyone with NO_AUTO_COMPLETION set — including anyone following
	// this repository's own review guardrails.
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm-256color")

	ran := false
	app := &App{
		Name:                   "autocli",
		AutoRefreshIntegration: true,
		Run: func(ctx *Context) error {
			ran = true
			return nil
		},
	}

	// The auto path refreshes; it does not create. An installed script that a
	// library upgrade has left stale is what it exists for, so that is what this
	// test sets up. (TestAutoInstallNeverCreatesACompletionScript covers the
	// other half: with nothing installed, an ordinary run writes nothing.)
	expectedPath := filepath.Join(tmpDir, "share", "zsh", "site-functions", "_autocli")
	var script bytes.Buffer
	if err := GenZshCompletion(app, &script); err != nil {
		t.Fatal(err)
	}
	marker := fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)
	stale := strings.Replace(script.String(), marker, "clihelp-completion-version: 0", 1)
	writeFixture(t, expectedPath, stale)

	if err := app.ExecuteContext(context.Background(), []string{}); err != nil {
		t.Fatalf("app.ExecuteContext failed: %v", err)
	}

	if !ran {
		t.Fatalf("expected app Run handler to execute")
	}

	body, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("the installed script is gone: %v", err)
	}
	if !strings.Contains(string(body), marker) {
		t.Fatalf("a stale script was not refreshed at %q:\n%s", expectedPath, body)
	}

	if !IsCompletionInstalled(app, "zsh") {
		t.Errorf("expected IsCompletionInstalled to be true")
	}
}

func TestAutoInstallCompletionKillSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("SHELL", "/bin/zsh")
	// All four, as install_test.go, man_test.go, protocol_test.go and
	// autopath_test.go already do. This file had an incomplete copy, so the suite
	// failed for anyone with NO_AUTO_COMPLETION set — including anyone following
	// this repository's own review guardrails.
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_AUTO_COMPLETION", "1")

	app := &App{
		Name:                   "autocli",
		AutoRefreshIntegration: true,
		Run: func(ctx *Context) error {
			return nil
		},
	}

	expectedPath := filepath.Join(tmpDir, "share", "zsh", "site-functions", "_autocli")
	if err := app.ExecuteContext(context.Background(), []string{}); err != nil {
		t.Fatalf("app.ExecuteContext failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("expected completion file NOT to be installed when NO_AUTO_COMPLETION=1 is set")
	}
}

// The unattended refresh happens when the author asked for it and not otherwise.
// It may never act on its own: an ordinary program run writing into a home
// directory nobody pointed it at is the defect this whole path was narrowed to
// avoid.
func TestAutoRefreshHappensOnlyWhenAsked(t *testing.T) {
	for _, tt := range []struct {
		name        string
		set         func(*App)
		wantRefresh bool
	}{
		{"asked for", func(a *App) { a.AutoRefreshIntegration = true }, true},
		{"not asked for", func(a *App) {}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
			t.Setenv("SHELL", "/bin/zsh")
			for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
				t.Setenv(v, "")
			}
			t.Setenv("TERM", "xterm-256color")

			app := &App{Name: "autocli", Run: func(ctx *Context) error { return nil }}
			tt.set(app)

			path := filepath.Join(tmpDir, "share", "zsh", "site-functions", "_autocli")
			var script bytes.Buffer
			if err := GenZshCompletion(app, &script); err != nil {
				t.Fatal(err)
			}
			marker := fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)
			stale := strings.Replace(script.String(), marker, "clihelp-completion-version: 0", 1)
			writeFixture(t, path, stale)

			if err := app.ExecuteContext(context.Background(), []string{}); err != nil {
				t.Fatalf("ExecuteContext failed: %v", err)
			}

			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("the installed script is gone: %v", err)
			}
			if refreshed := strings.Contains(string(body), marker); refreshed != tt.wantRefresh {
				t.Errorf("the stale script was refreshed = %v, want %v", refreshed, tt.wantRefresh)
			}
		})
	}
}

func TestPositionalIndex(t *testing.T) {
	arity := map[string]bool{
		"--verbose":    false,
		"--out":        true,
		"-o":           true,
		"-v":           false,
		"--opt":        false, // Optional flag
		"--hidden-val": true,
	}

	tests := []struct {
		name           string
		remaining      []string
		wantIndex      int
		wantTerminator bool
	}{
		{"empty", nil, 0, false},
		{"switch consumes nothing", []string{"--verbose"}, 0, false},
		{"value flag consumes two", []string{"--out", "file"}, 0, false},
		{"inline value is one word", []string{"--out=file"}, 0, false},
		{"shorthand with value", []string{"-o", "file"}, 0, false},
		{"cluster, value on last short", []string{"-vo", "file"}, 0, false},
		{"Optional flag never consumes next word", []string{"--opt", "x"}, 1, false},
		{"hidden flags are in the map", []string{"--hidden-val", "x"}, 0, false},
		{"one positional typed", []string{"alpha"}, 1, false},
		{"flag after a positional", []string{"alpha", "--verbose"}, 1, false},
		{"bare dash is positional", []string{"-"}, 1, false},
		{"terminator alone", []string{"--"}, 0, true},
		{"flag after terminator is positional", []string{"--", "--verbose"}, 1, true},
		{"unknown flag counted as word", []string{"--unknown", "alpha"}, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotTerm := positionalIndex(arity, tt.remaining)
			if gotIndex != tt.wantIndex || gotTerm != tt.wantTerminator {
				t.Errorf("positionalIndex(%v) = (%d, %v), want (%d, %v)",
					tt.remaining, gotIndex, gotTerm, tt.wantIndex, tt.wantTerminator)
			}
		})
	}
}

func TestPositionalArityRawOptionsAndNilCmd(t *testing.T) {
	app := &App{
		Name: "testapp",
		GlobalFlags: []Option{
			{Flags: "-g, --global <val>", Description: "global", arity: arityValue},
		},
		Commands: []Command{
			{
				Name: "sub",
				Options: []Option{
					{Flags: "--local <val>", Description: "local", arity: arityValue},
					{Flags: "--hidden-local <tok>", Description: "hidden", arity: arityValue, Hidden: true},
				},
			},
		},
	}

	resRoot := resolution{}
	arityRoot := app.positionalArity(resRoot, nil)
	if !arityRoot["--global"] {
		t.Errorf("expected arityRoot to contain --global")
	}

	cmd := &app.Commands[0]
	resSub := resolution{cmd: cmd}
	aritySub := app.positionalArity(resSub, cmd)
	if !aritySub["--local"] {
		t.Errorf("expected aritySub to contain --local")
	}
	if !aritySub["--hidden-local"] {
		t.Errorf("expected aritySub to contain --hidden-local (hidden options must be preserved)")
	}
}

func TestFlagValueGuard(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "mycli",
		Stdout: &out,
		Commands: []Command{
			{
				Name: "scan",
				Options: []Option{
					{Flags: "--output <file>", Description: "Output file", arity: arityValue},
					{
						Flags:       "-v, --verbose",
						Description: "Verbose",
					},
					{
						Flags:       "-o, --out <format>",
						Description: "Format",
						arity:       arityValue,
						Complete: func(toComplete string) []string {
							return []string{"json\tJSON format", "yaml\tYAML format"}
						},
					},
					{
						Flags:       "-f <file>",
						Description: "File with inline",
						arity:       arityValue,
					},
				},
				Parameters: []Param{
					{
						Name: "<prefix>",
						Complete: func(toComplete string) []string {
							return []string{"%inbox", "%spam"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	t.Run("flag with nil Complete emits nothing", func(t *testing.T) {
		out.Reset()
		err := app.ExecuteContext(context.Background(), []string{"__complete", "scan", "--output", ""})
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() != 0 {
			t.Errorf("expected empty output for flag without Complete callback, got: %q", out.String())
		}
	})

	t.Run("shorthand cluster -vo reaches -o callback", func(t *testing.T) {
		out.Reset()
		err := app.ExecuteContext(context.Background(), []string{"__complete", "scan", "-vo", ""})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "json\tJSON format") {
			t.Errorf("expected -o callback to be invoked for -vo cluster, got: %q", out.String())
		}
	})

	t.Run("shorthand with inline value -fo does not fire guard", func(t *testing.T) {
		out.Reset()
		err := app.ExecuteContext(context.Background(), []string{"__complete", "scan", "-fo", ""})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "%inbox") {
			t.Errorf("expected positional completion after -fo inline value, got: %q", out.String())
		}
	})
}

func TestPositionalCompletionSlotsAndVariadic(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "mycli",
		Stdout: &out,
		Commands: []Command{
			{
				Name: "copy",
				Parameters: []Param{
					{
						Name: "<src>",
						Complete: func(toComplete string) []string {
							return []string{"src1", "src2"}
						},
					},
					{
						Name: "<dest>",
						Complete: func(toComplete string) []string {
							return []string{"dst1", "dst2"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
			{
				Name: "upload",
				Parameters: []Param{
					{
						Name: "<target>",
						Complete: func(toComplete string) []string {
							return []string{"bucketA"}
						},
					},
					{
						Name:     "<files...>",
						Variadic: true,
						Complete: func(toComplete string) []string {
							return []string{"f1", "f2"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
			{
				Name: "flags",
				Options: []Option{
					{Flags: "--mode <m>", Description: "Mode", arity: arityValue},
					{Flags: "--secret <key>", Description: "Secret", arity: arityValue, Hidden: true},
					Optional(Option{Flags: "--opt [val]", Description: "Optional"}, "default"),
				},
				Parameters: []Param{
					{
						Name: "<param>",
						Complete: func(toComplete string) []string {
							return []string{"slot0"}
						},
					},
					{
						Name: "<param2>",
						Complete: func(toComplete string) []string {
							return []string{"slot1"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	t.Run("slot 0 and slot 1 complete independently", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "copy", ""})
		if !strings.Contains(out.String(), "src1") || strings.Contains(out.String(), "dst1") {
			t.Errorf("slot 0 mismatch: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "copy", "src1", ""})
		if !strings.Contains(out.String(), "dst1") || strings.Contains(out.String(), "src1") {
			t.Errorf("slot 1 mismatch: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "copy", "src1", "dst1", ""})
		if out.Len() != 0 {
			t.Errorf("slot 2 should not complete on non-variadic command: %q", out.String())
		}
	})

	t.Run("variadic parameter keeps completing", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "upload", "bucketA", ""})
		if !strings.Contains(out.String(), "f1") {
			t.Errorf("slot 1 mismatch: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "upload", "bucketA", "f1", ""})
		if !strings.Contains(out.String(), "f1") {
			t.Errorf("slot 2 variadic mismatch: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "upload", "bucketA", "f1", "f2", ""})
		if !strings.Contains(out.String(), "f1") {
			t.Errorf("slot 3 variadic mismatch: %q", out.String())
		}
	})

	t.Run("flags before positionals do not shift slot", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "flags", "--mode", "fast", ""})
		if !strings.Contains(out.String(), "slot0") {
			t.Errorf("expected slot 0 after value flag, got: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "flags", "--secret", "xyz", ""})
		if !strings.Contains(out.String(), "slot0") {
			t.Errorf("expected slot 0 after hidden value flag, got: %q", out.String())
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "flags", "--opt", "x", ""})
		if !strings.Contains(out.String(), "slot1") {
			t.Errorf("expected slot 1 after Optional flag and positional arg, got: %q", out.String())
		}
	})
}

func TestTerminatorGatingAndSubcommands(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "demo",
		Stdout: &out,
		Commands: []Command{
			{
				Name: "unspam",
				Options: []Option{
					{Flags: "--output <f>", arity: arityValue},
				},
				Subcommands: []Command{
					{Name: "folder", Description: "Folder subcommand"},
				},
				Parameters: []Param{
					{
						Name: "<msg_id>",
						Complete: func(toComplete string) []string {
							return []string{"msg1", "msg2"}
						},
					},
					{
						Name: "<dest>",
						Complete: func(toComplete string) []string {
							return []string{"destA", "destB"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	t.Run("slot 0 offers subcommands and positionals", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "unspam", ""})
		res := out.String()
		if !strings.Contains(res, "folder\tFolder subcommand") {
			t.Errorf("expected subcommand at slot 0: %q", res)
		}
		if !strings.Contains(res, "msg1") {
			t.Errorf("expected positional at slot 0: %q", res)
		}
	})

	t.Run("slot 1 does not offer subcommands", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "unspam", "msg1", ""})
		res := out.String()
		if strings.Contains(res, "folder") {
			t.Errorf("subcommand must not appear at slot 1: %q", res)
		}
		if !strings.Contains(res, "destA") {
			t.Errorf("expected positional slot 1: %q", res)
		}
	})

	t.Run("terminator gates guard and completes positionals", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "unspam", "--", "--output", ""})
		res := out.String()
		if !strings.Contains(res, "destA") {
			t.Errorf("expected positional slot 1 after -- --output, got: %q", res)
		}
	})

	t.Run("flags and subcommands suppressed after terminator", func(t *testing.T) {
		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "unspam", "--", "-"})
		res := out.String()
		if strings.Contains(res, "--output") {
			t.Errorf("flags must not be offered after --: %q", res)
		}

		out.Reset()
		_ = app.ExecuteContext(context.Background(), []string{"__complete", "unspam", "--", ""})
		res = out.String()
		if strings.Contains(res, "folder\tFolder subcommand") {
			t.Errorf("subcommands must not be offered after --: %q", res)
		}
	})
}

func TestPositionalCandidateSanitization(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "demo",
		Stdout: &out,
		Commands: []Command{
			{
				Name: "test",
				Parameters: []Param{
					{
						Name:        "<item>",
						Description: "Item parameter description",
						Complete: func(toComplete string) []string {
							return []string{
								"simple",
								"tabbed\tFirst tab\tsecond tab",
								"multi\nline\tDescription",
							}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	_ = app.ExecuteContext(context.Background(), []string{"__complete", "test", ""})
	res := out.String()

	if !strings.Contains(res, "simple\t\n") {
		t.Errorf("expected bare candidate to have empty description: %q", res)
	}
	if !strings.Contains(res, "tabbed\tFirst tab second tab\n") {
		t.Errorf("expected tab split on first tab: %q", res)
	}
	if strings.Contains(res, "\nline") {
		t.Errorf("newline in candidate should be sanitized: %q", res)
	}
}

func TestRootCompletionDoesNotPanic(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "mycli",
		Stdout: &out,
		GlobalFlags: []Option{
			{Flags: "--global", Description: "Global switch"},
		},
		Commands: []Command{
			{Name: "run", Description: "Run command"},
		},
	}

	for _, args := range [][]string{
		{"__complete", ""},
		{"__complete", "-"},
		{"__complete", "--"},
		{"__complete", "--global", ""},
	} {
		out.Reset()
		if err := app.ExecuteContext(context.Background(), args); err != nil {
			t.Fatalf("args %v failed: %v", args, err)
		}
	}
}
