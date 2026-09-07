package clihelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupZshCompletion(t *testing.T) (string, func([]string) []string) {
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

	return zshPath, runZshComplete
}

func testZshCommands(t *testing.T, runZshComplete func([]string) []string) {
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
		replies := runZshComplete([]string{"podctl", "deep", ""})
		if len(replies) != 2 {
			t.Errorf("expected 2 subcommands under deep, got: %v", replies)
		}

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

func testZshFlags(t *testing.T, runZshComplete func([]string) []string) {
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
}

func TestLiveZshCompletion(t *testing.T) {
	_, runZshComplete := setupZshCompletion(t)
	testZshCommands(t, runZshComplete)
	testZshFlags(t, runZshComplete)
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
