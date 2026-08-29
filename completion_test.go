package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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

func TestLiveBashCompletion(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found on system, skipping live bash completion test")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "podctl")

	// Build the example demo application binary
	buildCmd := exec.Command("go", "build", "-o", binPath, "./example")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build example CLI binary: %v, output: %s", err, string(out))
	}

	// Generate bash completion script using the binary itself
	genCmd := exec.Command(binPath, "completion", "bash")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate bash completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "podctl.bash")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write completion script: %v", err)
	}

	runBashComplete := func(words []string, cword int) []string {
		var quotedWords []string
		for _, w := range words {
			quotedWords = append(quotedWords, fmt.Sprintf("%q", w))
		}
		wordsArray := strings.Join(quotedWords, " ")

		bashScript := fmt.Sprintf(`
source %q
COMP_WORDS=(%s)
COMP_CWORD=%d
_podctl_complete
for r in "${COMPREPLY[@]}"; do
    echo "$r"
done
`, scriptPath, wordsArray, cword)

		cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", bashScript)
		// Add bin directory to PATH so COMP_WORDS[0] can invoke podctl
		cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bash completion execution failed: %v, output: %s", err, string(out))
		}

		var lines []string
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		return lines
	}

	t.Run("RootCommands", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", ""}, 1)
		expectedCmds := []string{"build", "config", "deploy", "status", "completion", "deep"}
		for _, exp := range expectedCmds {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected root command %q in bash completions, got: %v", exp, replies)
			}
		}
	})

	t.Run("RootCommandPrefix", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "bu"}, 1)
		if len(replies) != 1 || replies[0] != "build" {
			t.Errorf("expected ['build'], got: %v", replies)
		}
	})

	t.Run("SubcommandFlagsLong", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "build", "--"}, 2)
		expectedFlags := []string{"--output", "--bitrate", "--normalize", "--no-normalize", "--tags", "--verbose", "--silent"}
		for _, expected := range expectedFlags {
			found := false
			for _, r := range replies {
				if r == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected flag %q in replies %v", expected, replies)
			}
		}
	})

	t.Run("SubcommandFlagsShort", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "build", "-"}, 2)
		expectedShort := []string{"-o", "-b", "-v", "-s"}
		for _, exp := range expectedShort {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected short flag %q in bash replies %v", exp, replies)
			}
		}
	})

	t.Run("NestedSubcommand", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "config", ""}, 2)
		expected := []string{"get", "set"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected nested subcommand %q in bash replies %v", exp, replies)
			}
		}
	})

	t.Run("DeepNestedSubcommand", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "config", "set", ""}, 3)
		expected := []string{"space"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected deep nested subcommand %q in bash replies %v", exp, replies)
			}
		}
	})

	t.Run("DeepBinaryTreeHierarchy", func(t *testing.T) {
		// Level 1 -> Level 2
		replies := runBashComplete([]string{"podctl", "deep", ""}, 2)
		if len(replies) != 2 {
			t.Errorf("expected 2 subcommands under deep, got: %v", replies)
		}

		// Level 2 -> Level 3
		replies = runBashComplete([]string{"podctl", "deep", "alpha", ""}, 3)
		expectedL3 := []string{"alpha_one", "alpha_two"}
		for _, exp := range expectedL3 {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha, got: %v", exp, replies)
			}
		}

		// Level 3 -> Level 4
		replies = runBashComplete([]string{"podctl", "deep", "alpha", "alpha_one", ""}, 4)
		expectedL4 := []string{"alpha_one_a", "alpha_one_b"}
		for _, exp := range expectedL4 {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one, got: %v", exp, replies)
			}
		}

		// Level 4 -> Level 5 (leaf nodes)
		replies = runBashComplete([]string{"podctl", "deep", "alpha", "alpha_one", "alpha_one_a", ""}, 5)
		expectedL5 := []string{"alpha_one_a_i", "alpha_one_a_ii"}
		for _, exp := range expectedL5 {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one alpha_one_a, got: %v", exp, replies)
			}
		}
	})

	t.Run("EnumFlagValues", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "deploy", "-S", ""}, 3)
		expected := []string{"staging", "production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected enum value %q in bash replies %v", exp, replies)
			}
		}
	})

	t.Run("EnumFlagPrefix", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "deploy", "-S", "s"}, 3)
		if len(replies) != 1 || replies[0] != "staging" {
			t.Errorf("expected ['staging'], got: %v", replies)
		}
	})

	t.Run("EqualsFormFlagValues", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "deploy", "--stage="}, 2)
		expected := []string{"--stage=staging", "--stage=production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected equals flag %q in bash replies %v", exp, replies)
			}
		}
	})

	t.Run("CompletionSubcommands", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "completion", ""}, 2)
		expected := []string{"bash", "zsh", "fish", "install"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected completion subcommand %q in bash replies %v", exp, replies)
			}
		}
	})
}

func TestLiveBashDynamicCallback(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found on system, skipping live bash completion test")
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	binPath := filepath.Join(tmpDir, "dyncli")

	code := `package main

import (
	"os"
	"strings"
	"github.com/sarielhp/clihelp"
)

func main() {
	app := &clihelp.App{
		Name: "dyncli",
		Commands: []clihelp.Command{
			{
				Name: "play",
				Options: []clihelp.Option{
					{
						Flags: "-p, --podcast <id>",
						Description: "Podcast name",
						Complete: func(toComplete string) []string {
							podcasts := []string{"history", "hardfork", "huberman"}
							var res []string
							for _, p := range podcasts {
								if strings.HasPrefix(p, toComplete) {
									res = append(res, p)
								}
							}
							return res
						},
					},
				},
			},
			clihelp.CompletionCommand(),
		},
	}
	_ = app.Execute(os.Args[1:])
}
`
	if err := os.WriteFile(srcPath, []byte(code), 0600); err != nil {
		t.Fatalf("failed to write test code: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build dynamic CLI binary: %v, output: %s", err, string(out))
	}

	genCmd := exec.Command(binPath, "completion", "bash")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate bash completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "dyncli.bash")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write bash completion script: %v", err)
	}

	bashScript := fmt.Sprintf(`
source %q
COMP_WORDS=("dyncli" "play" "-p" "h")
COMP_CWORD=3
_dyncli_complete
for r in "${COMPREPLY[@]}"; do
    echo "$r"
done
`, scriptPath)

	cmd := exec.Command(bashPath, "--norc", "--noprofile", "-c", bashScript)
	cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash completion failed: %v, output: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 completions for dynamic bash test, got: %v", lines)
	}
}

func TestLiveZshCompletion(t *testing.T) {
	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not found on system, skipping live zsh completion test")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "podctl")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./example")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build example CLI binary: %v, output: %s", err, string(out))
	}

	genCmd := exec.Command(binPath, "completion", "zsh")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate zsh completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "podctl.zsh")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write zsh script: %v", err)
	}

	runZshComplete := func(words []string) []string {
		var quotedWords []string
		for _, w := range words {
			quotedWords = append(quotedWords, fmt.Sprintf("%q", w))
		}
		wordsArray := strings.Join(quotedWords, " ")

		zshScript := fmt.Sprintf(`
# Mock zsh completion hooks for testing output captures
_describe() {
    shift 2
    shift
    while (( $# > 0 )); do
        local -a items
        items=("${(@P)1}")
        for item in "${items[@]}"; do
            print -r -- "$item"
        done
        shift
    done
}
compadd() {
    while (( $# > 0 )); do
        if [[ "$1" == "-a" ]]; then
            shift
            local -a items
            items=("${(@P)1}")
            for item in "${items[@]}"; do
                print -r -- "$item"
            done
        fi
        shift
    done
}

words=(%s)
CURRENT=${#words[@]}
source %q
`, wordsArray, scriptPath)

		cmd := exec.Command(zshPath, "-f", "-c", zshScript)
		cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("zsh completion failed: %v, output: %s", err, string(out))
		}

		var lines []string
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		return lines
	}

	t.Run("ZshRootCommands", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", ""})
		expectedCmds := []string{"build", "config", "deploy", "status", "completion", "deep"}
		for _, exp := range expectedCmds {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected root command %q in zsh completions, got: %v", exp, replies)
			}
		}
	})

	t.Run("ZshRootCommandPrefix", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "bu"})
		foundBuild := false
		for _, r := range replies {
			if strings.HasPrefix(r, "build:") || r == "build" {
				foundBuild = true
				break
			}
		}
		if !foundBuild {
			t.Errorf("expected 'build' command in zsh completions, got: %v", replies)
		}
	})

	t.Run("ZshFlagsLong", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "build", "--"})
		expectedFlags := []string{"--output", "--bitrate", "--normalize", "--no-normalize", "--tags", "--verbose", "--silent"}
		for _, expected := range expectedFlags {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, expected+":") || r == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected flag %q in zsh replies %v", expected, replies)
			}
		}
	})

	t.Run("ZshFlagsShort", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "build", "-"})
		expectedShort := []string{"-o", "-b", "-v", "-s"}
		for _, exp := range expectedShort {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected short flag %q in zsh replies %v", exp, replies)
			}
		}
	})

	t.Run("ZshNestedSubcommand", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "config", ""})
		expected := []string{"get", "set"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected nested subcommand %q in zsh replies %v", exp, replies)
			}
		}
	})

	t.Run("ZshDeepNestedSubcommand", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "config", "set", ""})
		expected := []string{"space"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected deep nested subcommand %q in zsh replies %v", exp, replies)
			}
		}
	})

	t.Run("ZshDeepBinaryTreeHierarchy", func(t *testing.T) {
		// Level 1 -> Level 2
		replies := runZshComplete([]string{"podctl", "deep", ""})
		if len(replies) != 2 {
			t.Errorf("expected 2 subcommands under deep, got: %v", replies)
		}

		// Level 2 -> Level 3
		replies = runZshComplete([]string{"podctl", "deep", "alpha", ""})
		expectedL3 := []string{"alpha_one", "alpha_two"}
		for _, exp := range expectedL3 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha, got: %v", exp, replies)
			}
		}

		// Level 3 -> Level 4
		replies = runZshComplete([]string{"podctl", "deep", "alpha", "alpha_one", ""})
		expectedL4 := []string{"alpha_one_a", "alpha_one_b"}
		for _, exp := range expectedL4 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one, got: %v", exp, replies)
			}
		}

		// Level 4 -> Level 5 (leaf nodes)
		replies = runZshComplete([]string{"podctl", "deep", "alpha", "alpha_one", "alpha_one_a", ""})
		expectedL5 := []string{"alpha_one_a_i", "alpha_one_a_ii"}
		for _, exp := range expectedL5 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one alpha_one_a, got: %v", exp, replies)
			}
		}
	})

	t.Run("ZshEnumFlagValues", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "deploy", "-S", ""})
		expected := []string{"staging", "production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp || strings.HasPrefix(r, exp+":") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected enum value %q in zsh replies %v", exp, replies)
			}
		}
	})

	t.Run("ZshEnumFlagPrefix", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "deploy", "-S", "s"})
		foundStaging := false
		for _, r := range replies {
			if r == "staging" || strings.HasPrefix(r, "staging:") {
				foundStaging = true
				break
			}
		}
		if !foundStaging {
			t.Errorf("expected 'staging' in zsh replies, got: %v", replies)
		}
	})

	t.Run("ZshEqualsFormFlagValues", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "deploy", "--stage="})
		expected := []string{"--stage=staging", "--stage=production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected equals flag %q in zsh replies %v", exp, replies)
			}
		}
	})

	t.Run("ZshCompletionSubcommands", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "completion", ""})
		expected := []string{"bash", "zsh", "fish", "install"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+":") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected completion subcommand %q in zsh replies %v", exp, replies)
			}
		}
	})
}

func TestLiveZshDynamicCallback(t *testing.T) {
	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not found on system, skipping live zsh completion test")
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	binPath := filepath.Join(tmpDir, "dyncli")

	code := `package main

import (
	"os"
	"strings"
	"github.com/sarielhp/clihelp"
)

func main() {
	app := &clihelp.App{
		Name: "dyncli",
		Commands: []clihelp.Command{
			{
				Name: "play",
				Options: []clihelp.Option{
					{
						Flags: "-p, --podcast <id>",
						Description: "Podcast name",
						Complete: func(toComplete string) []string {
							podcasts := []string{"history\tDan Snow", "hardfork\tTech News", "huberman\tHealth"}
							var res []string
							for _, p := range podcasts {
								if strings.HasPrefix(p, toComplete) {
									res = append(res, p)
								}
							}
							return res
						},
					},
				},
			},
			clihelp.CompletionCommand(),
		},
	}
	_ = app.Execute(os.Args[1:])
}
`
	if err := os.WriteFile(srcPath, []byte(code), 0600); err != nil {
		t.Fatalf("failed to write test code: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build dynamic CLI binary: %v, output: %s", err, string(out))
	}

	genCmd := exec.Command(binPath, "completion", "zsh")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate zsh completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "dyncli.zsh")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write zsh completion script: %v", err)
	}

	zshScript := fmt.Sprintf(`
_describe() {
    shift 2
    shift
    while (( $# > 0 )); do
        local -a items
        items=("${(@P)1}")
        for item in "${items[@]}"; do
            print -r -- "$item"
        done
        shift
    done
}
compadd() {
    while (( $# > 0 )); do
        if [[ "$1" == "-a" ]]; then
            shift
            local -a items
            items=("${(@P)1}")
            for item in "${items[@]}"; do
                print -r -- "$item"
            done
        fi
        shift
    done
}

words=("dyncli" "play" "-p" "h")
source %q
`, scriptPath)

	cmd := exec.Command(zshPath, "-f", "-c", zshScript)
	cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh completion failed: %v, output: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 completions for dynamic zsh test, got %d: %v", len(lines), lines)
	}
	for _, expectedPrefix := range []string{"history:", "hardfork:", "huberman:"} {
		found := false
		for _, l := range lines {
			if strings.HasPrefix(l, expectedPrefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected zsh completion starting with %q, got: %v", expectedPrefix, lines)
		}
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

func TestLiveFishCompletion(t *testing.T) {
	fishPath, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish not found on system, skipping live fish completion test")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "podctl")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./example")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build example CLI binary: %v, output: %s", err, string(out))
	}

	genCmd := exec.Command(binPath, "completion", "fish")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate fish completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "podctl.fish")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write fish completion script: %v", err)
	}

	runFishComplete := func(commandLine string) []string {
		fishScript := fmt.Sprintf(`
source %q
complete -C %q
`, scriptPath, commandLine)

		cmd := exec.Command(fishPath, "--no-config", "-c", fishScript)
		cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fish completion failed for %q: %v, output: %s", commandLine, err, string(out))
		}

		var lines []string
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		return lines
	}

	t.Run("RootCommands", func(t *testing.T) {
		replies := runFishComplete("podctl ")
		expectedCmds := []string{"build", "config", "deploy", "status", "completion", "deep"}
		for _, exp := range expectedCmds {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected root command %q in fish completions, got: %v", exp, replies)
			}
		}
	})

	t.Run("RootCommandPrefix", func(t *testing.T) {
		replies := runFishComplete("podctl bu")
		if len(replies) != 1 || !strings.HasPrefix(replies[0], "build\t") {
			t.Errorf("expected ['build\\t...'], got: %v", replies)
		}
	})

	t.Run("SubcommandFlagsLong", func(t *testing.T) {
		replies := runFishComplete("podctl build --")
		expectedFlags := []string{"--output", "--bitrate", "--normalize", "--no-normalize", "--tags", "--verbose", "--silent"}
		for _, expected := range expectedFlags {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, expected+"\t") || r == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected flag %q in fish replies %v", expected, replies)
			}
		}
	})

	t.Run("SubcommandFlagsShort", func(t *testing.T) {
		replies := runFishComplete("podctl build -")
		expectedShort := []string{"-o", "-b", "-v", "-s"}
		for _, exp := range expectedShort {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected short flag %q in fish replies %v", exp, replies)
			}
		}
	})

	t.Run("NestedSubcommand", func(t *testing.T) {
		replies := runFishComplete("podctl config ")
		expected := []string{"get", "set"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected nested subcommand %q in fish replies %v", exp, replies)
			}
		}
	})

	t.Run("DeepNestedSubcommand", func(t *testing.T) {
		replies := runFishComplete("podctl config set ")
		expected := []string{"space"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected deep nested subcommand %q in fish replies %v", exp, replies)
			}
		}
	})

	t.Run("DeepBinaryTreeHierarchy", func(t *testing.T) {
		// Level 1 -> Level 2
		replies := runFishComplete("podctl deep ")
		if len(replies) != 2 {
			t.Errorf("expected 2 subcommands under deep, got: %v", replies)
		}

		// Level 2 -> Level 3
		replies = runFishComplete("podctl deep alpha ")
		expectedL3 := []string{"alpha_one", "alpha_two"}
		for _, exp := range expectedL3 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha, got: %v", exp, replies)
			}
		}

		// Level 3 -> Level 4
		replies = runFishComplete("podctl deep alpha alpha_one ")
		expectedL4 := []string{"alpha_one_a", "alpha_one_b"}
		for _, exp := range expectedL4 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one, got: %v", exp, replies)
			}
		}

		// Level 4 -> Level 5 (leaf nodes)
		replies = runFishComplete("podctl deep alpha alpha_one alpha_one_a ")
		expectedL5 := []string{"alpha_one_a_i", "alpha_one_a_ii"}
		for _, exp := range expectedL5 {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q under deep alpha alpha_one alpha_one_a, got: %v", exp, replies)
			}
		}
	})

	t.Run("EnumFlagValues", func(t *testing.T) {
		replies := runFishComplete("podctl deploy -S ")
		expected := []string{"staging", "production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if r == exp || strings.HasPrefix(r, exp+"\t") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected enum value %q in fish replies %v", exp, replies)
			}
		}
	})

	t.Run("EnumFlagPrefix", func(t *testing.T) {
		replies := runFishComplete("podctl deploy -S s")
		if len(replies) != 1 || replies[0] != "staging" {
			t.Errorf("expected ['staging'], got: %v", replies)
		}
	})

	t.Run("EqualsFormFlagValues", func(t *testing.T) {
		replies := runFishComplete("podctl deploy --stage=")
		expected := []string{"--stage=staging", "--stage=production"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected equals flag %q in fish replies %v", exp, replies)
			}
		}
	})

	t.Run("CompletionSubcommands", func(t *testing.T) {
		replies := runFishComplete("podctl completion ")
		expected := []string{"bash", "zsh", "fish", "install"}
		for _, exp := range expected {
			found := false
			for _, r := range replies {
				if strings.HasPrefix(r, exp+"\t") || r == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected completion subcommand %q in fish replies %v", exp, replies)
			}
		}
	})
}

func TestLiveFishDynamicCallback(t *testing.T) {
	fishPath, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish not found on system, skipping live fish completion test")
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	binPath := filepath.Join(tmpDir, "dyncli")

	code := `package main

import (
	"os"
	"strings"
	"github.com/sarielhp/clihelp"
)

func main() {
	var podcast string
	app := &clihelp.App{
		Name: "dyncli",
		Commands: []clihelp.Command{
			{
				Name: "play",
				Options: []clihelp.Option{
					{
						Flags: "-p, --podcast <id>",
						Description: "Podcast name",
						Complete: func(toComplete string) []string {
							podcasts := []string{"history\tDan Snow History", "hardfork\tTech News", "huberman\tHealth"}
							var res []string
							for _, p := range podcasts {
								if strings.HasPrefix(p, toComplete) {
									res = append(res, p)
								}
							}
							return res
						},
					},
				},
			},
			clihelp.CompletionCommand(),
		},
	}
	_ = podcast
	_ = app.Execute(os.Args[1:])
}
`
	if err := os.WriteFile(srcPath, []byte(code), 0600); err != nil {
		t.Fatalf("failed to write test code: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build dynamic CLI binary: %v, output: %s", err, string(out))
	}

	genCmd := exec.Command(binPath, "completion", "fish")
	scriptBytes, err := genCmd.Output()
	if err != nil {
		t.Fatalf("failed to generate fish completion script: %v", err)
	}

	scriptPath := filepath.Join(tmpDir, "dyncli.fish")
	if err := os.WriteFile(scriptPath, scriptBytes, 0600); err != nil {
		t.Fatalf("failed to write fish completion script: %v", err)
	}

	fishScript := fmt.Sprintf(`
source %q
complete -C "dyncli play -p h"
`, scriptPath)

	cmd := exec.Command(fishPath, "--no-config", "-c", fishScript)
	cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish completion failed: %v, output: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 completions for 'dyncli play -p h', got %d: %v", len(lines), lines)
	}
	for _, expectedPrefix := range []string{"history\t", "hardfork\t", "huberman\t"} {
		found := false
		for _, l := range lines {
			if strings.HasPrefix(l, expectedPrefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected completion starting with %q, got: %v", expectedPrefix, lines)
		}
	}
}
