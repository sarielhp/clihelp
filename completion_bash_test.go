package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupBashCompletion(t *testing.T) (string, func([]string, int) []string) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found on system, skipping live bash completion test")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "podctl")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./example")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build example CLI binary: %v, output: %s", err, string(out))
	}

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

	return bashPath, runBashComplete
}

func testBashCommands(t *testing.T, runBashComplete func([]string, int) []string) {
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
		replies := runBashComplete([]string{"podctl", "deep", ""}, 2)
		if len(replies) != 2 {
			t.Errorf("expected 2 subcommands under deep, got: %v", replies)
		}

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

func testBashFlags(t *testing.T, runBashComplete func([]string, int) []string) {
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
}

func TestLiveBashCompletion(t *testing.T) {
	_, runBashComplete := setupBashCompletion(t)
	testBashCommands(t, runBashComplete)
	testBashFlags(t, runBashComplete)
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
