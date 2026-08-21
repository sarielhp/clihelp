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
	if !strings.Contains(fishBuf.String(), "__my_app_complete") || !strings.Contains(fishBuf.String(), "complete -c my-app") {
		t.Errorf("invalid fish completion script: %s", fishBuf.String())
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

	// Test Case 1: Root command completion for "podctl bu"
	t.Run("RootCommandPrefix", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "bu"}, 1)
		if len(replies) != 1 || replies[0] != "build" {
			t.Errorf("expected ['build'], got: %v", replies)
		}
	})

	// Test Case 2: Subcommand flag completion for "podctl build --"
	t.Run("SubcommandFlagPrefix", func(t *testing.T) {
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

	// Test Case 3: Nested subcommand completion for "podctl config s"
	t.Run("NestedSubcommand", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "config", "s"}, 2)
		if len(replies) != 1 || replies[0] != "set" {
			t.Errorf("expected ['set'], got: %v", replies)
		}
	})

	// Test Case 4: Enum flag completion for "podctl deploy -S s"
	t.Run("EnumFlagValues", func(t *testing.T) {
		replies := runBashComplete([]string{"podctl", "deploy", "-S", "s"}, 3)
		if len(replies) != 1 || replies[0] != "staging" {
			t.Errorf("expected ['staging'], got: %v", replies)
		}
	})
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
		foundBuild := false
		for _, r := range replies {
			if strings.HasPrefix(r, "build:") {
				foundBuild = true
				break
			}
		}
		if !foundBuild {
			t.Errorf("expected 'build' command with description in zsh completions, got: %v", replies)
		}
	})

	t.Run("ZshFlags", func(t *testing.T) {
		replies := runZshComplete([]string{"podctl", "build", "--"})
		foundOutput := false
		for _, r := range replies {
			if strings.HasPrefix(r, "--output:") {
				foundOutput = true
				break
			}
		}
		if !foundOutput {
			t.Errorf("expected '--output' flag with description in zsh completions, got: %v", replies)
		}
	})
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
		t.Fatalf("failed to write fish script: %v", err)
	}

	fishScript := fmt.Sprintf(`
source %q
complete -C "podctl b"
`, scriptPath)

	cmd := exec.Command(fishPath, "--no-config", "-c", fishScript)
	cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish completion failed: %v, output: %s", err, string(out))
	}

	if !strings.Contains(string(out), "build") {
		t.Errorf("expected 'build' in fish completions, got: %s", string(out))
	}
}
