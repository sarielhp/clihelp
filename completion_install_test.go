package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
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
		Name:                   "ksh-app",
		AutoRefreshIntegration: true,
		Commands:               []Command{{Name: "run", Description: "Run it", Run: func(*Context) error { return nil }}},
	}

	// The refresh is what the shell gate has to hold back, since the auto path
	// never creates anything: a stale bash script must stay untouched for a ksh
	// user and be repaired for a bash one.
	script := filepath.Join(dataHome, "bash-completion", "completions", "ksh-app")
	var body bytes.Buffer
	if err := GenBashCompletion(app, &body); err != nil {
		t.Fatal(err)
	}
	marker := fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)
	stale := strings.Replace(body.String(), marker, "clihelp-completion-version: 0", 1)
	writeFixture(t, script, stale)

	t.Setenv("SHELL", "/bin/ksh")
	TestExecute(app, []string{"run"}).AssertNoError(t)
	if got, _ := os.ReadFile(script); strings.Contains(string(got), marker) {
		t.Errorf("a ksh user's run refreshed a bash script")
	}

	t.Setenv("SHELL", "/bin/bash")
	TestExecute(app, []string{"run"}).AssertNoError(t)
	if got, _ := os.ReadFile(script); !strings.Contains(string(got), marker) {
		t.Errorf("a bash user's stale script was not refreshed:\n%s", got)
	}
}

func TestCompletionEntryPointsRejectANilApp(t *testing.T) {
	if _, err := completionScriptPath(nil, "bash"); err == nil {
		t.Errorf("completionScriptPath(nil) returned no error")
	}
	if _, err := installCompletion(nil, "bash"); err == nil {
		t.Errorf("installCompletion(nil) returned no error")
	}
	if isCompletionInstalled(nil, "bash") {
		t.Errorf("isCompletionInstalled(nil) reported an installed script")
	}
}

func TestInstallCompletionWritesCompleteScripts(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	app := &App{Name: "podcli", Commands: []Command{{Name: "run", Description: "Run it"}}}
	for _, shell := range supportedShells {
		t.Run(shell, func(t *testing.T) {
			path, err := installCompletion(app, shell)
			if err != nil {
				t.Fatalf("installCompletion(%q): %v", shell, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading the installed script: %v", err)
			}
			if !strings.Contains(string(data), "podcli") {
				t.Errorf("the installed %s script looks truncated:\n%s", shell, data)
			}
			if !isCompletionInstalled(app, shell) {
				t.Errorf("isCompletionInstalled(%q) = false right after installing", shell)
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

func TestCompletionDescriptionsAreOneShortPlainLine(t *testing.T) {
	long := "Compile, encode, and package raw audio into MP3 episodes. Supports configurable bitrate, loudness normalization, and embedded ID3 tags."
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{"first sentence only", long, "Compile, encode, and package raw audio into MP3 episodes."},
		{"markdown is rendered away", "**deep** — the [deep command](https://example.com/deep) here.", "deep — the deep command here."},
		{"code spans", "Set the `--out` path.", "Set the --out path."},
		{"newlines collapse", "First line\nsecond line", "First line second line"},
		{"short text is untouched", "Run it", "Run it"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := completionDescription(tt.in); got != tt.want {
				t.Errorf("completionDescription(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	t.Run("width is capped", func(t *testing.T) {
		wide := strings.Repeat("超", 80) // two display columns per rune
		got := completionDescription(wide)
		if w := runewidth.StringWidth(got); w > completionDescriptionWidth {
			t.Errorf("description is %d columns wide, over the %d cap", w, completionDescriptionWidth)
		}
		if !strings.HasSuffix(got, "…") {
			t.Errorf("a truncated description should say so: %q", got)
		}
	})
}

func TestCompletionProtocolEmitsShortDescriptions(t *testing.T) {
	var out bytes.Buffer
	app := &App{
		Name:   "podcli",
		Stdout: &out,
		Commands: []Command{{
			Name:        "build",
			Description: "**Compile** audio into [MP3](https://example.com/mp3) episodes. Supports bitrate, normalization, and ID3 tags for distribution everywhere.",
		}},
	}
	if err := app.ExecuteContext(context.Background(), []string{"__complete"}); err != nil {
		t.Fatal(err)
	}
	for _, record := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		_, desc, found := strings.Cut(record, "\t")
		if !found {
			t.Fatalf("record %q has no description field", record)
		}
		if strings.ContainsAny(desc, "*[]`") {
			t.Errorf("markdown leaked into a completion description: %q", desc)
		}
		if runewidth.StringWidth(desc) > completionDescriptionWidth {
			t.Errorf("description is %d columns wide: %q", runewidth.StringWidth(desc), desc)
		}
	}
}

// A script that carries no version marker is always stale, so the auto-installer
// rewrote it on every single invocation of the program. Only the bash template
// had one.
func TestEveryGeneratedScriptCarriesItsVersion(t *testing.T) {
	app := &App{Name: "podcli"}
	marker := fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)
	for _, shell := range supportedShells {
		t.Run(shell, func(t *testing.T) {
			var b strings.Builder
			var err error
			switch shell {
			case "bash":
				err = GenBashCompletion(app, &b)
			case "zsh":
				err = GenZshCompletion(app, &b)
			case "fish":
				err = GenFishCompletion(app, &b)
			}
			if err != nil {
				t.Fatal(err)
			}
			head := b.String()
			if len(head) > 256 {
				head = head[:256] // completionIsCurrent looks no further
			}
			if !strings.Contains(head, marker) {
				t.Errorf("%s script has no version marker in its first 256 bytes:\n%s", shell, head)
			}
		})
	}
}

func TestInstalledScriptIsRecognizedAsCurrent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	app := &App{Name: "podcli"}
	for _, shell := range supportedShells {
		t.Run(shell, func(t *testing.T) {
			path, err := installCompletion(app, shell)
			if err != nil {
				t.Fatal(err)
			}
			if !completionIsCurrent(app, shell) {
				t.Errorf("a freshly installed %s script is reported stale, so it would be rewritten on every run", shell)
			}
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			// A second auto-install pass must leave the file alone.
			if !isCompletionInstalled(app, shell) || !completionIsCurrent(app, shell) {
				t.Fatalf("%s script not recognized after installing", shell)
			}
			after, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !before.ModTime().Equal(after.ModTime()) {
				t.Errorf("the %s script was rewritten by a check that should not write", shell)
			}
		})
	}
}
