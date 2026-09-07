package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupFishCompletion(t *testing.T) (string, func(string) []string) {
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

	return fishPath, runFishComplete
}

func testFishCommands(t *testing.T, runFishComplete func(string) []string) {
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

func testFishFlags(t *testing.T, runFishComplete func(string) []string) {
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
}

func TestLiveFishCompletion(t *testing.T) {
	_, runFishComplete := setupFishCompletion(t)
	testFishCommands(t, runFishComplete)
	testFishFlags(t, runFishComplete)
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
