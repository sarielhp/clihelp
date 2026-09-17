package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectShellReportsWhatItFound(t *testing.T) {
	for _, tt := range []struct {
		shell string
		want  string
	}{
		{"/bin/bash", "bash"},
		{"/usr/bin/zsh", "zsh"},
		{"/usr/local/bin/fish", "fish"},
		{"/bin/ksh", "ksh"},
		{"/usr/bin/nu", "nu"},
		{"", ""},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			t.Setenv("SHELL", tt.shell)
			if got := detectShell(); got != tt.want {
				t.Errorf("detectShell() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAutoInstallSkipsShellsWithoutAScript(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CLIHELP_NO_AUTO_COMPLETION", "")
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("NO_AUTO_COMPLETION", "")
	t.Setenv("TERM", "xterm")

	app := &App{
		Name:                  "ksh-app",
		AutoInstallCompletion: true,
		Commands:              []Command{{Name: "run", Description: "Run it", Run: func(*Context) error { return nil }}},
	}

	t.Setenv("SHELL", "/bin/ksh")
	TestExecute(app, []string{"run"}).AssertNoError(t)
	if entries, _ := os.ReadDir(filepath.Join(dataHome, "bash-completion", "completions")); len(entries) > 0 {
		t.Errorf("a bash script was installed for a ksh user: %v", entries)
	}

	t.Setenv("SHELL", "/bin/bash")
	TestExecute(app, []string{"run"}).AssertNoError(t)
	if _, err := os.Stat(filepath.Join(dataHome, "bash-completion", "completions", "ksh-app")); err != nil {
		t.Errorf("a bash user got no script: %v", err)
	}
}

func TestCompletionEntryPointsRejectANilApp(t *testing.T) {
	if _, err := CompletionPath(nil, "bash"); err == nil {
		t.Errorf("CompletionPath(nil) returned no error")
	}
	if _, err := InstallCompletion(nil, "bash"); err == nil {
		t.Errorf("InstallCompletion(nil) returned no error")
	}
	if IsCompletionInstalled(nil, "bash") {
		t.Errorf("IsCompletionInstalled(nil) reported an installed script")
	}
}

func TestInstallCompletionWritesCompleteScripts(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	app := &App{Name: "podcli", Commands: []Command{{Name: "run", Description: "Run it"}}}
	for _, shell := range SupportedShells {
		t.Run(shell, func(t *testing.T) {
			path, err := InstallCompletion(app, shell)
			if err != nil {
				t.Fatalf("InstallCompletion(%q): %v", shell, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the installed script: %v", err)
			}
			if !strings.Contains(string(data), "podcli") {
				t.Errorf("the installed %s script looks truncated:\n%s", shell, data)
			}
			if !IsCompletionInstalled(app, shell) {
				t.Errorf("IsCompletionInstalled(%q) = false right after installing", shell)
			}
			// The atomic write must not leave its temporary file behind.
			entries, _ := os.ReadDir(filepath.Dir(path))
			for _, e := range entries {
				if strings.Contains(e.Name(), ".tmp-") {
					t.Errorf("a temporary file was left behind: %s", e.Name())
				}
			}
		})
	}
}

func TestBashScriptUsesTheWordsBashCompletionComputed(t *testing.T) {
	app := &App{Name: "podcli"}
	var b strings.Builder
	if err := GenBashCompletion(app, &b); err != nil {
		t.Fatal(err)
	}
	script := b.String()
	for _, want := range []string{
		`_init_completion -n :=`,
		`"${words[0]}" __complete "${words[@]:1:cword-1}" "$cur"`,
		`__ltrim_colon_completions "$cur"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("bash script is missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, `"${COMP_WORDS[@]:1}"`) {
		t.Errorf("bash script still sends the raw COMP_WORDS:\n%s", script)
	}
}
