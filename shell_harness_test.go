package clihelp

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Driving the generated shell code without a terminal.
//
// bash's key binding is an ordinary function, so it can be called directly, and
// the tests in explain_shell_test.go do. zsh's widget and fish's binding cannot
// be — but they are ordinary functions too, and in both shells a function
// shadows a builtin. So stubbing `zle`, `bindkey`, `commandline` and `bind` with
// functions that record their arguments drives the real generated code
// headlessly, with no pty and no new dependency. Everything below is what those
// two shells were missing: until now they were only syntax-checked.

// generateSnippet writes a generated artifact into dir and returns its path.
func generateSnippet(t *testing.T, dir, name string, gen func(*App, string, *strings.Builder) error, app *App, shell string) string {
	t.Helper()
	var b strings.Builder
	if err := gen(app, shell, &b); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func keySnippet(t *testing.T, dir, shell string, app *App) string {
	t.Helper()
	return generateSnippet(t, dir, "keys."+shell, func(a *App, s string, b *strings.Builder) error {
		return GenKeyBindings(a, s, b)
	}, app, shell)
}

// driveShell runs body in shell with the snippet sourced and the interactive
// builtins stubbed, and returns everything it printed.
func driveShell(t *testing.T, shell, dir, snippet, stubs, body string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not found, skipping", shell)
	}

	script := stubs + "\nsource " + snippet + "\n" + body + "\n"

	var args []string
	switch shell {
	case "bash":
		args = []string{"--norc", "--noprofile", "-c", script}
	case "fish":
		args = []string{"--no-config", "-c", script}
	default:
		args = []string{"-f", "-c", script}
	}
	cmd := sandboxedCommand(t, path, args...)
	cmd.Env = append(cmd.Env, "HOME="+dir, "PATH="+dir+":"+os.Getenv("PATH"), "LINES=24", "COLUMNS=78")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", shell, err, out)
	}
	return string(out)
}

// zshStubs shadows the ZLE builtins a widget uses, recording each call.
const zshStubs = `
zle() { print -r -- "ZLE:$*" }
bindkey() { print -r -- "BINDKEY:$*" }
`

// fishStubs shadows fish's interactive builtins. commandline answers the buffer
// it is given and records a replacement.
const fishStubs = `
set -g __buffer ""
function commandline
    if test (count $argv) -eq 0
        echo $__buffer
        return
    end
    switch $argv[1]
        case '-r' '--replace'
            set -g __buffer $argv[-1]
            echo "SET:$argv[-1]"
        case '-f'
            echo "REPAINT"
        case '*'
            echo $__buffer
    end
end
function bind; echo "BIND:$argv"; end
function __fish_man_page; echo "MANPAGE"; end
`

// bashStubs shadows the readline builtin the snippet binds through.
const bashStubs = `
bind() { printf 'BIND:%s\n' "$*"; }
`

// stubProgram writes a stand-in for the wrapped program: it answers the
// __explain protocol and nothing else.
func stubProgram(t *testing.T, dir, name string) {
	t.Helper()
	body := "#!/bin/sh\necho \"$2 EXPANDED\"\necho \"help for " + name + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}

// The binding must reach both the emacs and the vi keymaps. It landed in
// whichever one happened to be current when the snippet was sourced, so a user
// whose `set -o vi` came afterwards — or whose plugin manager set the keymap
// from a hook — had no Alt-H at all.
func TestKeyBindingReachesBothKeymaps(t *testing.T) {
	dir := t.TempDir()
	stubProgram(t, dir, "alpha")

	t.Run("bash", func(t *testing.T) {
		snippet := keySnippet(t, dir, "bash", &App{Name: "alpha"})
		out := driveShell(t, "bash", dir, snippet, bashStubs, `echo done`)
		for _, want := range []string{"-m emacs-standard", "-m vi-insert"} {
			if !strings.Contains(out, want) {
				t.Errorf("bash binding does not cover %q:\n%s", want, out)
			}
		}
	})

	t.Run("zsh", func(t *testing.T) {
		snippet := keySnippet(t, dir, "zsh", &App{Name: "alpha"})
		out := driveShell(t, "zsh", dir, snippet, zshStubs, `print -r -- done`)
		for _, want := range []string{"-M emacs", "-M viins"} {
			if !strings.Contains(out, want) {
				t.Errorf("zsh binding does not cover %q:\n%s", want, out)
			}
		}
	})

	t.Run("fish", func(t *testing.T) {
		snippet := keySnippet(t, dir, "fish", &App{Name: "alpha"})
		out := driveShell(t, "fish", dir, snippet, fishStubs, `echo done`)
		for _, want := range []string{"-M default", "-M insert"} {
			if !strings.Contains(out, want) {
				t.Errorf("fish binding does not cover %q:\n%s", want, out)
			}
		}
	})
}

// CLIHELP_NO_KEY_BINDINGS gives back Alt-H — zsh's run-help and fish's man-page
// key — to users who would rather keep it, without giving up completion.
func TestKeyBindingOptOut(t *testing.T) {
	dir := t.TempDir()
	stubProgram(t, dir, "alpha")

	for _, tc := range []struct{ shell, stubs string }{
		{"bash", bashStubs},
		{"zsh", zshStubs},
		{"fish", fishStubs},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			snippet := keySnippet(t, dir, tc.shell, &App{Name: "alpha"})
			set := "export CLIHELP_NO_KEY_BINDINGS=1\n"
			if tc.shell == "fish" {
				set = "set -gx CLIHELP_NO_KEY_BINDINGS 1\n"
			}
			out := driveShell(t, tc.shell, dir, snippet, set+tc.stubs, "echo done")
			if strings.Contains(out, "BIND") {
				t.Errorf("%s bound the key despite the opt-out:\n%s", tc.shell, out)
			}
			if !strings.Contains(out, "done") {
				t.Errorf("%s snippet did not run to completion:\n%s", tc.shell, out)
			}
		})
	}
}

// zshCompletionStubs shadows what the script reaches for at registration time.
// autoload is stubbed too, so `autoload -Uz add-zsh-hook` cannot replace the
// add-zsh-hook stub with the real one.
const zshCompletionStubs = `
autoload() { return 0 }
add-zsh-hook() { print -r -- "HOOK:$*" }
`

func completionSnippet(t *testing.T, dir, shell string, app *App) string {
	t.Helper()
	gen := map[string]func(*App, io.Writer) error{
		"bash": GenBashCompletion,
		"zsh":  GenZshCompletion,
		"fish": GenFishCompletion,
	}[shell]
	if gen == nil {
		t.Fatalf("no generator for %q", shell)
	}
	return generateSnippet(t, dir, "completion."+shell, func(a *App, _ string, b *strings.Builder) error {
		return gen(a, b)
	}, app, shell)
}

// Sourcing the zsh script before compinit has run left completion silently
// unregistered: compdef did not exist yet, so neither branch fired and there was
// no error anywhere. Plugin managers that defer compinit, and rc files that
// source clihelp's bootstrap near the top, both land here.
func TestZshCompletionRegistersAfterDeferredCompinit(t *testing.T) {
	dir := t.TempDir()
	snippet := completionSnippet(t, dir, "zsh", &App{Name: "alpha"})

	// compdef is deliberately absent while the script is sourced, then defined,
	// then the registered hook is fired the way the first prompt would fire it.
	body := `
compdef() { print -r -- "COMPDEF:$*" }
if (( ${+functions[_alpha_deferred_compdef]} )); then
    _alpha_deferred_compdef
fi
print -r -- done`
	out := driveShell(t, "zsh", dir, snippet, zshCompletionStubs, body)

	if !strings.Contains(out, "HOOK:precmd _alpha_deferred_compdef") {
		t.Errorf("no precmd retry was registered:\n%s", out)
	}
	if !strings.Contains(out, "COMPDEF:_alpha alpha") {
		t.Errorf("the retry never registered the completion:\n%s", out)
	}
	if !strings.Contains(out, "HOOK:-d precmd _alpha_deferred_compdef") {
		t.Errorf("the retry did not remove its own hook:\n%s", out)
	}
}

// The ordinary case must not grow a hook: when compinit has already run the
// script registers immediately and leaves nothing behind on precmd.
func TestZshCompletionRegistersImmediately(t *testing.T) {
	dir := t.TempDir()
	snippet := completionSnippet(t, dir, "zsh", &App{Name: "alpha"})
	stubs := zshCompletionStubs + "\ncompdef() { print -r -- \"COMPDEF:$*\" }\n"
	out := driveShell(t, "zsh", dir, snippet, stubs, `print -r -- done`)

	if !strings.Contains(out, "COMPDEF:_alpha alpha") {
		t.Errorf("completion was not registered:\n%s", out)
	}
	if strings.Contains(out, "HOOK:") {
		t.Errorf("a precmd hook was installed although compdef was available:\n%s", out)
	}
}

// stubCompleter writes a stand-in program that answers __complete with fixed
// lines, so a completion script can be driven without building anything.
func stubCompleter(t *testing.T, dir, name, output string) {
	t.Helper()
	body := "#!/bin/sh\nprintf '%s' " + shellSingleQuote(output) + "\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Every shell must fall back to filenames when the program offers no
// candidates: that is what the bash script's "complete -o default" does, and
// without it an argument that is a path cannot be completed at all. zsh offered
// nothing and fish's "-f" forbade files outright.
func TestCompletionFallsBackToFiles(t *testing.T) {
	t.Run("zsh", func(t *testing.T) {
		dir := t.TempDir()
		stubCompleter(t, dir, "alpha", "")
		snippet := completionSnippet(t, dir, "zsh", &App{Name: "alpha"})
		stubs := zshCompletionStubs + `
compdef() { : }
_files() { print -r -- "FILES" }
compadd() { print -r -- "COMPADD:$*" }
_describe() { print -r -- "DESCRIBE:$*" }
`
		body := "local -a words; words=(alpha sub); local CURRENT=2; _alpha"
		out := driveShell(t, "zsh", dir, snippet, stubs, body)
		if !strings.Contains(out, "FILES") {
			t.Errorf("zsh offered no filename fallback:\n%s", out)
		}
	})

	t.Run("fish", func(t *testing.T) {
		dir := t.TempDir()
		stubCompleter(t, dir, "alpha", "")
		snippet := completionSnippet(t, dir, "fish", &App{Name: "alpha"})
		stubs := `
function complete; echo "COMPLETE:$argv"; end
function commandline; echo alpha; end
`
		body := `
if functions -q __fish_alpha_needs_files
    __fish_alpha_needs_files; and echo "NEEDS-FILES"
end`
		out := driveShell(t, "fish", dir, snippet, stubs, body)
		if !strings.Contains(out, "NEEDS-FILES") {
			t.Errorf("fish has no filename fallback for an empty candidate list:\n%s", out)
		}
	})
}

// ...and must not offer files on top of the candidates the program gave, which
// is what dropping fish's -f outright would have done.
func TestCompletionKeepsCandidatesFileFree(t *testing.T) {
	t.Run("zsh", func(t *testing.T) {
		dir := t.TempDir()
		stubCompleter(t, dir, "alpha", "build\tcompile it\n")
		snippet := completionSnippet(t, dir, "zsh", &App{Name: "alpha"})
		stubs := zshCompletionStubs + `
compdef() { : }
_files() { print -r -- "FILES" }
compadd() { print -r -- "COMPADD:$*" }
_describe() { print -r -- "DESCRIBE:$*" }
`
		body := "local -a words; words=(alpha sub); local CURRENT=2; _alpha"
		out := driveShell(t, "zsh", dir, snippet, stubs, body)
		if strings.Contains(out, "FILES") {
			t.Errorf("zsh mixed filenames into a non-empty candidate list:\n%s", out)
		}
		if !strings.Contains(out, "DESCRIBE:") {
			t.Errorf("zsh dropped the described candidates:\n%s", out)
		}
	})

	t.Run("fish", func(t *testing.T) {
		dir := t.TempDir()
		stubCompleter(t, dir, "alpha", "build\n")
		snippet := completionSnippet(t, dir, "fish", &App{Name: "alpha"})
		stubs := `
function complete; echo "COMPLETE:$argv"; end
function commandline; echo alpha; end
`
		body := "__fish_alpha_needs_files; and echo \"NEEDS-FILES\"\ntrue"
		out := driveShell(t, "fish", dir, snippet, stubs, body)
		if strings.Contains(out, "NEEDS-FILES") {
			t.Errorf("fish asked for files although the program gave candidates:\n%s", out)
		}
	})
}

// A single candidate that is the empty string is still a candidate list. zsh's
// "[ -n $completions ]" read the array as one joined string, so it decided there
// was nothing to add — the same bug that would misjudge the files fallback.
func TestZshCountsCandidatesNotTheirText(t *testing.T) {
	dir := t.TempDir()
	stubCompleter(t, dir, "alpha", "\n\n")
	snippet := completionSnippet(t, dir, "zsh", &App{Name: "alpha"})
	stubs := zshCompletionStubs + `
compdef() { : }
_files() { print -r -- "FILES" }
compadd() { print -r -- "COMPADD:$*" }
_describe() { print -r -- "DESCRIBE:$*" }
`
	body := "local -a words; words=(alpha sub); local CURRENT=2; _alpha"
	out := driveShell(t, "zsh", dir, snippet, stubs, body)
	if !strings.Contains(out, "FILES") {
		t.Errorf("blank-only output should count as no candidates:\n%s", out)
	}
}

// callExplain drives the shared Alt-H dispatcher against a given registry and
// command line, in whichever shell, and returns what it printed.
func callExplain(t *testing.T, shell, dir, registry, line string) string {
	t.Helper()
	snippet := keySnippet(t, dir, shell, &App{Name: "alpha"})
	var stubs, body string
	switch shell {
	case "bash":
		stubs = bashStubs
		body = "_clihelp_apps=" + shellSingleQuote(registry) + "\nREADLINE_LINE=" +
			shellSingleQuote(line) + "\nREADLINE_POINT=0\n_clihelp_explain\nprintf 'LINE:%s\\n' \"$READLINE_LINE\""
	case "zsh":
		stubs = zshStubs
		body = "_clihelp_apps=" + shellSingleQuote(registry) + "\nBUFFER=" +
			shellSingleQuote(line) + "\nCURSOR=0\n_clihelp_explain\nprint -r -- \"LINE:$BUFFER\""
	case "fish":
		stubs = fishStubs
		body = "set -g _clihelp_apps (string split ' ' -- " + shellSingleQuote(strings.TrimSpace(registry)) + ")\n" +
			"set -g __buffer " + shellSingleQuote(line) + "\n__clihelp_explain\necho \"LINE:$__buffer\"\ntrue"
	}
	return driveShell(t, shell, dir, snippet, stubs, body)
}

// The registry is a wire format shared by every clihelp program in the shell,
// including ones built against a different library version. Entries carry the
// protocol they speak, so a dispatcher can tell what is on the other end before
// calling it — and a bare name, written by a program that predates the suffix,
// still means protocol 1.
func TestDispatcherReadsTheRegistryProtocol(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			dir := t.TempDir()
			stubProgram(t, dir, "beta")

			t.Run("versioned entry", func(t *testing.T) {
				out := callExplain(t, shell, dir, " beta:1 ", "beta run")
				if !strings.Contains(out, "EXPANDED") {
					t.Errorf("a versioned registry entry was not honoured:\n%s", out)
				}
			})

			t.Run("bare entry", func(t *testing.T) {
				out := callExplain(t, shell, dir, " beta ", "beta run")
				if !strings.Contains(out, "EXPANDED") {
					t.Errorf("an entry from an older program was not honoured:\n%s", out)
				}
			})

			t.Run("protocol this dispatcher does not speak", func(t *testing.T) {
				out := callExplain(t, shell, dir, " beta:99 ", "beta run")
				if strings.Contains(out, "EXPANDED") {
					t.Errorf("an unknown protocol was called anyway:\n%s", out)
				}
			})

			t.Run("unregistered", func(t *testing.T) {
				out := callExplain(t, shell, dir, " gamma:1 ", "beta run")
				if strings.Contains(out, "EXPANDED") {
					t.Errorf("an unregistered program was called:\n%s", out)
				}
			})
		})
	}
}

// Registration writes both forms for one release, so a dispatcher installed by
// an already-present clihelp program — which knows only bare names — keeps
// working while every new dispatcher reads the protocol.
func TestKeyBindingRegistersBothForms(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			var b strings.Builder
			if err := GenKeyBindings(&App{Name: "alpha"}, shell, &b); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"alpha:" + strconv.Itoa(explainProtocolVersion), "alpha"} {
				if !strings.Contains(b.String(), want) {
					t.Errorf("%s snippet never registers %q:\n%s", shell, want, b.String())
				}
			}
		})
	}
}

// A command line is one string. fish read it with `(commandline)`, which splits
// a multi-line buffer into one element per line and then passed those as
// separate arguments, so everything after the first line was dropped — the
// continuation lines of exactly the long command a user would press Alt-H on.
// bash and zsh have always passed the whole buffer as one argument.
func TestFishExplainPassesTheWholeBuffer(t *testing.T) {
	dir := t.TempDir()
	// The stub reports what it was actually handed, with newlines made visible.
	body := "#!/bin/sh\nprintf 'expanded\\n'\nprintf 'argc=%s\\n' \"$#\"\n" +
		"printf 'line=%s\\n' \"$(printf '%s' \"$2\" | tr '\\n' '|')\"\n"
	if err := os.WriteFile(filepath.Join(dir, "beta"), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	snippet := keySnippet(t, dir, "fish", &App{Name: "alpha"})
	drive := fishStubs + `
function commandline
    if test (count $argv) -eq 0
        printf '%s\n' "beta run \\" "  --flag"
        return
    end
    switch $argv[1]
        case '-r' '--replace'
            echo "SET:$argv[-1]"
        case '*'
            echo REPAINT
    end
end
`
	out := driveShell(t, "fish", dir, snippet, drive,
		"set -g _clihelp_apps beta:1\n__clihelp_explain\ntrue")

	if !strings.Contains(out, "argc=2") {
		t.Errorf("the buffer was split across several arguments:\n%s", out)
	}
	if !strings.Contains(out, "--flag") {
		t.Errorf("everything after the first line was lost:\n%s", out)
	}
}

// bash puts a COMPREPLY entry on the command line verbatim, so a candidate
// containing a space was re-parsed as two arguments the moment it was inserted.
// zsh and fish quote theirs.
func TestBashCompletionQuotesCandidates(t *testing.T) {
	dir := t.TempDir()
	stubCompleter(t, dir, "alpha", "prod east\nstaging\n")
	snippet := completionSnippet(t, dir, "bash", &App{Name: "alpha"})

	body := `
COMP_WORDS=(alpha deploy "")
COMP_CWORD=2
_alpha_complete
printf 'REPLY:%s\n' "${COMPREPLY[@]}"`
	out := driveShell(t, "bash", dir, snippet, bashStubs, body)

	if !strings.Contains(out, `REPLY:prod\ east`) {
		t.Errorf("a candidate with a space was not quoted for insertion:\n%s", out)
	}
	if !strings.Contains(out, "REPLY:staging") {
		t.Errorf("an ordinary candidate was mangled:\n%s", out)
	}
}

// The wrapper reconstructs the wrapped command line by stripping its own name
// off the front. It cut at the first space, so a tab between the name and the
// arguments matched nothing, emptied the remainder, and explained the bare
// command as though the user had typed no arguments at all.
func TestWrapperSplitsOnAnyBlank(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found, skipping")
	}
	dir := t.TempDir()
	// The wrapped program reports the line it was asked to explain.
	body := "#!/bin/sh\nprintf 'expanded\\n'\nprintf 'got=[%s]\\n' \"$2\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pod"), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	var script strings.Builder
	if err := GenWrapperScript(&App{Name: "pod"}, "pd", []string{"deploy"}, &script); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(dir, "pd")
	if err := os.WriteFile(wrapper, []byte(script.String()), 0o700); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct{ name, line, want string }{
		{"space", "pd --stage=prod", "pod deploy --stage=prod"},
		{"tab", "pd\t--stage=prod", "pod deploy --stage=prod"},
		{"several blanks", "pd   --stage=prod", "pod deploy --stage=prod"},
		{"no arguments", "pd", "pod deploy"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := sandboxedCommand(t, shPath, wrapper, "__explain", tt.line)
			cmd.Env = append(cmd.Env, "PATH="+dir+":"+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("the wrapper failed: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "got=["+tt.want+"]") {
				t.Errorf("wanted the wrapped line %q:\n%s", tt.want, out)
			}
		})
	}
}
