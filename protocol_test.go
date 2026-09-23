package clihelp

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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

func runProto(t *testing.T, app *App, args ...string) *testResult {
	t.Helper()
	return testExecute(app, args)
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
		// The program and the wrapped command line are quoted once, into shell
		// variables, so that nothing below re-parses them.
		"__clihelp_app=pod-ctl",
		"__clihelp_target=",
		`exec "$__clihelp_app" __complete deploy --bucket 'my bucket' "$@"`,
		`exec "$__clihelp_app" deploy --bucket 'my bucket' "$@"`,
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
	app.AutoRefreshIntegration = true
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
	res.AssertStderrContains(t, "removed") // the report is for a human
	res.AssertStdoutContains(t, "/")       // the paths are for a script

	again := runProto(t, bareApp(), "__clihelp", "uninstall", "bash")
	again.AssertNoError(t)
	again.AssertStderrContains(t, "nothing to remove")
	if strings.TrimSpace(again.Stdout) != "" {
		t.Errorf("nothing was removed, so stdout should be empty: %q", again.Stdout)
	}
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

func TestClihelpManPageVerb(t *testing.T) {
	sandboxHome(t)

	t.Run("prints roff to stdout", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp", "manpage")
		res.AssertNoError(t)
		if !strings.HasPrefix(res.Stdout, ".\\\" ") {
			t.Errorf("stdout should be the page alone, so it can be redirected:\n%s", res.Stdout[:80])
		}
		res.AssertStdoutContains(t, `.TH "BARE" 1`)
	})

	t.Run("install prints only the path", func(t *testing.T) {
		res := runProto(t, bareApp(), "__clihelp", "manpage", "--install")
		res.AssertNoError(t)
		if strings.Count(res.Stdout, "\n") != 1 {
			t.Errorf("stdout should be the path alone:\n%s", res.Stdout)
		}
		if _, err := os.Stat(strings.TrimSpace(res.Stdout)); err != nil {
			t.Errorf("the path on stdout is not a file: %v", err)
		}
		if !strings.Contains(res.Stderr, "Alt-H") {
			t.Errorf("the note for a human belongs on stderr: %q", res.Stderr)
		}
	})

	t.Run("uninstall", func(t *testing.T) {
		first := runProto(t, bareApp(), "__clihelp", "manpage", "--uninstall")
		first.AssertStdoutContains(t, "/") // the removed path, for a script
		first.AssertStderrContains(t, "removed")
		runProto(t, bareApp(), "__clihelp", "manpage", "--uninstall").AssertStderrContains(t, "no generated manual page")
	})

	t.Run("an unknown option is an error", func(t *testing.T) {
		runProto(t, bareApp(), "__clihelp", "manpage", "--nonesuch").AssertErrorContains(t, "--nonesuch")
	})
}

func TestManPageCommandIsOptional(t *testing.T) {
	sandboxHome(t)
	app := bareApp()
	app.Commands = append(app.Commands, ManPageCommand())

	res := testExecute(app, []string{"manpage"})
	res.AssertNoError(t)
	res.AssertStdoutContains(t, `.TH "BARE" 1`)

	// And the same thing is reachable without the author adding it.
	runProto(t, bareApp(), "__clihelp", "manpage").AssertStdoutContains(t, `.TH "BARE" 1`)
}

// A flag is a flag wherever it appears, an unknown token is an error, and a
// surplus positional is an error — the same grammar the visible commands get
// from pflag. "--no-keys" silently ignored after the shell name meant the user
// declined a global key binding and got one anyway.
func TestClihelpVerbsParseArgumentsInAnyOrder(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		keys    bool
		wantErr string
	}{
		{"flag first", []string{"--no-keys", "bash"}, false, ""},
		{"shell first", []string{"bash", "--no-keys"}, false, ""},
		{"no flag", []string{"bash"}, true, ""},
		{"flag alone", []string{"--no-keys"}, false, ""},
		{"unknown option", []string{"bash", "--nonesuch"}, false, "--nonesuch"},
		{"two shells", []string{"bash", "zsh"}, false, "zsh"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := sandboxHome(t)
			t.Setenv("SHELL", "/bin/bash")
			res := runProto(t, bareApp(), append([]string{"__clihelp", "install"}, tt.args...)...)
			if tt.wantErr != "" {
				res.AssertErrorContains(t, tt.wantErr)
				return
			}
			res.AssertNoError(t)
			body, err := os.ReadFile(filepath.Join(home, ".config", "bare", "shell", "bash"))
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(body), "_clihelp_explain"); got != tt.keys {
				t.Errorf("key binding installed = %v, want %v", got, tt.keys)
			}
		})
	}
}

func TestClihelpVerbsRejectContradictoryFlags(t *testing.T) {
	sandboxHome(t)
	runProto(t, bareApp(), "__clihelp", "manpage", "--install", "--uninstall").
		AssertErrorContains(t, "mutually exclusive")
	runProto(t, bareApp(), "__clihelp", "manpage", "--force").
		AssertErrorContains(t, "--force")
	runProto(t, bareApp(), "__clihelp", "uninstall", "bash", "zsh").
		AssertErrorContains(t, "zsh")
}

// A wrapper is put on $PATH and forgotten, so it must pass its arguments through
// unchanged. escapeShellArg's single quotes were being interpolated into a
// double-quoted string, where they are inert and $( ) still runs; and its
// deny-list missed the glob characters, so "a[1]" matched a file in the cwd.
func TestGeneratedWrapperPassesArgumentsThrough(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "EXECUTED")
	// A file that a glob would match, in the directory the wrapper runs from.
	if err := os.WriteFile(filepath.Join(dir, "a1"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The wrapped program: report exactly what it received.
	stub := "#!/bin/sh\nfor a in \"$@\"; do printf 'ARG[%s]\\n' \"$a\"; done\n"
	if err := os.WriteFile(filepath.Join(dir, "pod"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}

	args := []string{"deploy", "--msg=$(touch " + marker + ")", "a[1]", "it's", "a b"}
	var script strings.Builder
	if err := GenWrapperScript(&App{Name: "pod"}, "pd", args, &script); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(dir, "pd")
	if err := os.WriteFile(wrapper, []byte(script.String()), 0o700); err != nil {
		t.Fatal(err)
	}

	if out, err := sandboxedCommand(t, shPath, "-n", wrapper).CombinedOutput(); err != nil {
		t.Fatalf("the generated wrapper is not valid sh: %v\n%s", err, out)
	}

	cmd := sandboxedCommand(t, wrapper, "extra")
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, "PATH="+dir+":"+os.Getenv("PATH"))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running the wrapper failed: %v", err)
	}
	got := string(out)
	for _, want := range args {
		if !strings.Contains(got, "ARG["+want+"]") {
			t.Errorf("argument %q did not survive:\n%s", want, got)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Errorf("the wrapper executed a command substitution from its own arguments")
	}
}

// The wrapper's __explain branch answers the Alt-H protocol, but nothing ever
// added the wrapper's name to the dispatcher's registry — so the branch was
// unreachable from a keystroke, and the documentation described behaviour that
// could not occur.
func TestWrapperRegistrationRegistersWithTheDispatcher(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	res := runProto(t, bareApp(), "__clihelp", "wrapper", "pd", "build")
	res.AssertNoError(t)
	if !strings.Contains(res.Stderr, "_clihelp_apps") {
		t.Errorf("the registration advice does not add the wrapper to the Alt-H registry:\n%s", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "complete -F") {
		t.Errorf("the completion registration went missing:\n%s", res.Stderr)
	}
}

func TestWrapperRejectsANameItCannotSafelyEmit(t *testing.T) {
	for _, name := range []string{"", "-w", "a/b", "..", "w;id", "w$(id)", "w\nid"} {
		var b strings.Builder
		if err := GenWrapperScript(&App{Name: "pod"}, name, nil, &b); err == nil {
			t.Errorf("GenWrapperScript accepted the name %q", name)
		}
	}
}

// Stdout is what a script captures: a path, a generated script, a version — one
// item per line, nothing else. Everything addressed to a human goes to stderr,
// and a visible command obeys the same rule as its __clihelp twin. Half the
// surface broke this, so a packager could not tell which half they were on.
func TestStdoutCarriesOnlyMachineReadableOutput(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"install", []string{"__clihelp", "install", "bash"}},
		{"uninstall", []string{"__clihelp", "uninstall", "bash"}},
		{"manpage --install", []string{"__clihelp", "manpage", "--install"}},
		{"manpage --uninstall", []string{"__clihelp", "manpage", "--uninstall"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sandboxHome(t)
			t.Setenv("SHELL", "/bin/bash")
			// Install first so the uninstall cases have something to remove.
			if strings.Contains(tt.name, "uninstall") {
				pre := []string{"__clihelp", "install", "bash"}
				if strings.Contains(tt.name, "manpage") {
					pre = []string{"__clihelp", "manpage", "--install"}
				}
				runProto(t, bareApp(), pre...).AssertNoError(t)
			}

			res := runProto(t, bareApp(), tt.args...)
			res.AssertNoError(t)
			requireRendered(t, res.Stdout)
			for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
				if line == "" {
					continue
				}
				if !filepath.IsAbs(line) {
					t.Errorf("stdout carries something a script cannot use: %q", line)
				}
			}
		})
	}
}

func TestAppDisableSetup(t *testing.T) {
	app := bareApp()
	app.DisableSetup = true

	t.Run("suppresses __clihelp setup protocol", func(t *testing.T) {
		res := runProto(t, app, "__clihelp")
		if res.Error == nil {
			t.Errorf("__clihelp was handled when DisableSetup is true")
		}
		if strings.Contains(res.Stdout, "shell integration for this program") {
			t.Errorf("stdout printed setup verbs when DisableSetup is true: %s", res.Stdout)
		}
	})

	t.Run("preserves __complete protocol", func(t *testing.T) {
		res := runProto(t, app, "__complete", "b")
		res.AssertNoError(t)
		res.AssertStdoutContains(t, "build")
	})

	t.Run("preserves __explain protocol", func(t *testing.T) {
		res := runProto(t, app, "__explain", "bare build")
		res.AssertNoError(t)
		res.AssertStdoutContains(t, "bare build")
	})
}

func TestClihelpWrapperFromFlag(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	dir := t.TempDir()
	shim := filepath.Join(dir, "my-shim")
	scriptContent := "#!/bin/bash\nbare build -v \"$@\"\n"
	if err := os.WriteFile(shim, []byte(scriptContent), 0o755); err != nil {
		t.Fatal(err)
	}

	app := bareApp()

	t.Run("auto name and extracted arguments", func(t *testing.T) {
		res := runProto(t, app, "__clihelp", "wrapper", "--from", shim)
		res.AssertNoError(t)
		res.AssertStdoutContains(t, "# clihelp-wraps: bare build -v")
		res.AssertStdoutContains(t, `__clihelp_target='bare build -v'`)
		res.AssertStderrContains(t, "complete -F _bare_complete my-shim")
	})

	t.Run("override wrapper name", func(t *testing.T) {
		res := runProto(t, app, "__clihelp", "wrapper", "--from", shim, "custom-name")
		res.AssertNoError(t)
		res.AssertStdoutContains(t, `__clihelp_target='bare build -v'`)
		res.AssertStderrContains(t, "complete -F _bare_complete custom-name")
	})

	t.Run("rejects contradictory positional arguments", func(t *testing.T) {
		res := runProto(t, app, "__clihelp", "wrapper", "--from", shim, "custom-name", "extra-arg")
		if res.Error == nil {
			t.Errorf("expected error when extra arguments passed with --from")
		}
	})

	t.Run("missing --from argument", func(t *testing.T) {
		res := runProto(t, app, "__clihelp", "wrapper", "--from")
		if res.Error == nil {
			t.Errorf("expected error when --from is missing value")
		}
	})
}
