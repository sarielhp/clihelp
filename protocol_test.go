package clihelp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// __clihelp answers in every clihelp program, including one whose author never
// added CompletionCommand() — the capability was always universal, only the
// means of setting it up used to depend on the author opting in.
func bareApp() *App {
	return &App{
		Name: "bare",
		Commands: []Command{
			{Name: "build", Description: "Build it.", Run: func(*Context) error { return nil }},
			{Name: "deploy", Description: "Deploy it.", Run: func(*Context) error { return nil }},
		},
	}
}

func runProto(t *testing.T, app *App, args ...string) *TestResult {
	t.Helper()
	return TestExecute(app, args)
}

func TestClihelpVerbsWithoutCompletionCommand(t *testing.T) {
	app := bareApp()
	if cmd := app.LookupCommand("completion"); cmd != nil {
		t.Fatalf("this app is supposed to have no completion command")
	}

	t.Run("bare invocation lists the verbs", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp")
		res.AssertNoError(t)
		for _, want := range []string{"version", "install", "keys", "wrapper", "bash, zsh, fish"} {
			res.AssertStdoutContains(t, want)
		}
	})

	t.Run("version identifies the library", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp", "version")
		res.AssertNoError(t)
		if got := strings.TrimSpace(res.Stdout); got != "clihelp "+Version {
			t.Errorf("version = %q, want %q", got, "clihelp "+Version)
		}
	})

	t.Run("keys matches the visible generator", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp", "keys", "bash")
		res.AssertNoError(t)
		var want strings.Builder
		if err := GenKeyBindings(bareApp(), "bash", &want); err != nil {
			t.Fatal(err)
		}
		if res.Stdout != want.String() {
			t.Errorf("__clihelp keys disagrees with GenKeyBindings")
		}
	})

	t.Run("unknown verb is an error", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp", "nonesuch")
		res.AssertErrorContains(t, "nonesuch")
	})
}

func TestClihelpInstallPrintsOnlyThePath(t *testing.T) {
	sandboxHome(t)

	res := runProto(t, bareApp(), "__clihelp", "install", "bash")
	res.AssertNoError(t)

	path := strings.TrimSpace(res.Stdout)
	if strings.Count(res.Stdout, "\n") != 1 {
		t.Errorf("stdout should be the path and nothing else, got:\n%s", res.Stdout)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the path on stdout is not a file: %v", err)
	}
	if !strings.Contains(res.Stderr, "Restart your shell") {
		t.Errorf("the human note belongs on stderr, got: %q", res.Stderr)
	}
}

func TestGenWrapperScript(t *testing.T) {
	app := &App{Name: "pod-ctl"}
	var b strings.Builder
	if err := GenWrapperScript(app, "pd", []string{"deploy", "--bucket", "my bucket"}, &b); err != nil {
		t.Fatal(err)
	}
	script := b.String()
	for _, want := range []string{
		"#!/bin/sh",
		"# clihelp-wraps: pod-ctl deploy --bucket 'my bucket'",
		`exec pod-ctl __complete deploy --bucket 'my bucket' "$@"`,
		`exec pod-ctl deploy --bucket 'my bucket' "$@"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("wrapper script is missing %q:\n%s", want, script)
		}
	}

	if err := GenWrapperScript(app, "", nil, &strings.Builder{}); err == nil {
		t.Errorf("a wrapper with no name was accepted")
	}
	if err := GenWrapperScript(nil, "pd", nil, &strings.Builder{}); err == nil {
		t.Errorf("a nil app was accepted")
	}
}

func TestWrapperRegistrationLineGoesToStderr(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	res := runProto(t, bareApp(), "__clihelp", "wrapper", "bd", "build")
	res.AssertNoError(t)

	if !strings.HasPrefix(res.Stdout, "#!/bin/sh") {
		t.Errorf("stdout should be the script alone so it can be redirected to a file:\n%s", res.Stdout)
	}
	if strings.Contains(res.Stdout, "complete -F") {
		t.Errorf("the registration line must not land in the script:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "complete -F _bare_complete bd") {
		t.Errorf("stderr should carry the registration line, got: %q", res.Stderr)
	}
}

// An internal protocol call must not have an installation as a side effect.
func TestProtocolCallsDoNotAutoInstall(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/bash")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "NO_AUTO_COMPLETION", "CLIHELP_NO_AUTO_COMPLETION"} {
		t.Setenv(v, "")
	}
	t.Setenv("TERM", "xterm")

	app := bareApp()
	app.AutoInstallCompletion = true
	var out bytes.Buffer
	app.Stdout = &out
	for _, args := range [][]string{{"__clihelp", "version"}, {"__complete", ""}, {"__explain", "bare build"}} {
		if err := app.ExecuteContext(context.Background(), args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if entries, _ := os.ReadDir(filepath.Join(dataHome, "bash-completion", "completions")); len(entries) > 0 {
		t.Errorf("a protocol call installed a completion script: %v", entries)
	}
}

func TestClihelpUninstall(t *testing.T) {
	sandboxHome(t)

	runProto(t, bareApp(), "__clihelp", "install", "bash").AssertNoError(t)
	res := runProto(t, bareApp(), "__clihelp", "uninstall", "bash")
	res.AssertNoError(t)
	res.AssertStdoutContains(t, "removed")

	again := runProto(t, bareApp(), "__clihelp", "uninstall", "bash")
	again.AssertNoError(t)
	again.AssertStdoutContains(t, "nothing to remove")
}

func TestClihelpInstallWithoutKeys(t *testing.T) {
	home := sandboxHome(t)
	runProto(t, bareApp(), "__clihelp", "install", "--no-keys", "bash").AssertNoError(t)

	body, err := os.ReadFile(filepath.Join(home, ".config", "bare", "shell", "bash"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "_clihelp_explain") {
		t.Errorf("--no-keys still installed the key binding")
	}
}

// The spellings anyone reaches for first must not be taken for arguments:
// "__clihelp wrapper --help" used to generate a wrapper script named "--help",
// and "__clihelp install --help" tried to install for a shell of that name.
func TestClihelpAcceptsTheUsualHelpSpellings(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help", "-H"} {
		t.Run("bare "+arg, func(t *testing.T) {
			res := runProto(t, bareApp(), "__clihelp", arg)
			res.AssertNoError(t)
			for _, want := range []string{"version", "install", "uninstall", "keys", "wrapper"} {
				res.AssertStdoutContains(t, want)
			}
		})
	}

	for _, verb := range []string{"version", "install", "uninstall", "keys", "wrapper"} {
		t.Run(verb+" --help", func(t *testing.T) {
			res := runProto(t, bareApp(), "__clihelp", verb, "--help")
			res.AssertNoError(t)
			res.AssertStdoutContains(t, "__clihelp "+verb)
			if strings.Contains(res.Stdout, "#!/bin/sh") {
				t.Errorf("%s --help generated something instead of explaining itself:\n%s", verb, res.Stdout)
			}
			if strings.Count(res.Stdout, "\n") > 2 {
				t.Errorf("a verb's help should be its own usage, got:\n%s", res.Stdout)
			}
		})
	}

	t.Run("help for an unknown verb still errors", func(t *testing.T) {
		runProto(t, bareApp(), "__clihelp", "nonesuch", "--help").AssertErrorContains(t, "nonesuch")
	})
}

// -H is clihelp's extended-help flag everywhere else, so it asks __clihelp for
// more too: what install would write on this machine, and what is reserved.
func TestClihelpExtendedHelp(t *testing.T) {
	home := sandboxHome(t)
	t.Setenv("SHELL", "/bin/bash")

	concise := runProto(t, bareApp(), "__clihelp", "-h")
	concise.AssertNoError(t)
	extended := runProto(t, bareApp(), "__clihelp", "-H")
	extended.AssertNoError(t)

	if len(extended.Stdout) <= len(concise.Stdout) {
		t.Errorf("-H should say more than -h")
	}
	concise.AssertStdoutContains(t, "-H")
	for _, want := range []string{
		"__complete", "__explain", "__clihelp",
		filepath.Join(home, ".config", "bare", "shell", "bash"),
		filepath.Join(home, ".bashrc"),
		"Nothing runs at shell startup",
	} {
		extended.AssertStdoutContains(t, want)
	}

	// Nothing may be written by asking.
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Errorf("asking for help wrote a startup file")
	}
}

func TestClihelpExtendedHelpWithoutAShell(t *testing.T) {
	sandboxHome(t)
	t.Setenv("SHELL", "")

	res := runProto(t, bareApp(), "__clihelp", "-H")
	res.AssertNoError(t)
	res.AssertStdoutContains(t, "no shell detected")
}
