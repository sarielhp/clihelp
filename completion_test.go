package clihelp

import (
	"bytes"
	"context"
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
			path, err := InstallCompletion(app, tt.shell)
			if err != nil {
				t.Fatalf("InstallCompletion(%q) failed: %v", tt.shell, err)
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
		path, err := InstallCompletion(app, "")
		if err != nil {
			t.Fatalf("InstallCompletion(\"\") with SHELL=zsh failed: %v", err)
		}
		if filepath.Base(path) != "_testcli" {
			t.Errorf("expected zsh installation, got: %s", path)
		}
	})

	t.Run("UnsupportedShell", func(t *testing.T) {
		_, err := InstallCompletion(app, "unknown_shell")
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
	expected := []string{"bash", "zsh", "fish", "install"}
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

	var outBuf bytes.Buffer
	app := &App{
		Name:     "myapp",
		Stdout:   &outBuf,
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
	if !strings.Contains(outBuf.String(), "Autocompletion installed to:") {
		t.Errorf("expected installation confirmation, got: %s", outBuf.String())
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

	installedPath, err := InstallCompletion(app, "zsh")
	if err != nil {
		t.Fatalf("InstallCompletion error: %v", err)
	}
	if installedPath != expectedZsh {
		t.Errorf("InstallCompletion path = %q, want %q", installedPath, expectedZsh)
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
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "xterm-256color")

	ran := false
	app := &App{
		Name:                  "autocli",
		AutoInstallCompletion: true,
		Run: func(ctx *Context) error {
			ran = true
			return nil
		},
	}

	expectedPath := filepath.Join(tmpDir, "share", "zsh", "site-functions", "_autocli")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Fatalf("completion file should not exist yet")
	}

	if err := app.ExecuteContext(context.Background(), []string{}); err != nil {
		t.Fatalf("app.ExecuteContext failed: %v", err)
	}

	if !ran {
		t.Fatalf("expected app Run handler to execute")
	}

	info, err := os.Stat(expectedPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected completion file to be auto-installed at %q, err: %v", expectedPath, err)
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
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_AUTO_COMPLETION", "1")

	app := &App{
		Name:                  "autocli",
		AutoInstallCompletion: true,
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
