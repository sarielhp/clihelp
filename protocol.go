package clihelp

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Argument names beginning with "__" are reserved by clihelp; an application
// must not declare commands with these names.
//
//	__complete   the completion protocol, called by the generated shell scripts
//	__explain    the Alt-H protocol, called by the generated key bindings
//	__clihelp    the setup verbs below
//
// The first two are a frozen boundary: installed scripts call them by name.
// Everything a human or a setup script invokes goes under __clihelp instead, so
// the reserved surface stays one word however many verbs are added.
//
// __clihelp answers in every clihelp program, whether or not its author added
// CompletionCommand() to the command tree. The capability was always universal —
// ExecuteContext serves __complete before it ever looks at the command tree —
// while the means of setting it up used to depend on the author opting in. That
// left a program that completes perfectly with no way to be asked to install its
// own completion, which is precisely the case a dotfiles script or a packager
// runs into.
//
// It is hidden, not secret: it is absent from help and completion output because
// no human needs it in the way, and it never acts unless it is invoked.
const (
	protoComplete = "__complete"
	protoExplain  = "__explain"
	protoClihelp  = "__clihelp"
)

// clihelpVerb is one entry of the __clihelp surface. The list is the single
// source for both the summary and a verb's own usage line.
type clihelpVerb struct {
	name  string
	args  string
	about string
}

func clihelpVerbs() []clihelpVerb {
	return []clihelpVerb{
		{"version", "", "report the clihelp version this program was built with"},
		{"install", "[--no-keys] [<shell>]", "set up the shell: tab completion and the Alt-H binding"},
		{"uninstall", "[<shell>]", "remove what install wrote"},
		{"keys", "[<shell>]", "print the key bindings, for inspection or manual setup"},
		{"wrapper", "<name> [<args>...]", "print a wrapper script for this program, with arguments"},
	}
}

// isHelpRequest recognizes the spellings anyone reaches for first. Without it,
// "__clihelp --help" was an unknown verb and "__clihelp wrapper --help"
// cheerfully generated a wrapper script named "--help".
func isHelpRequest(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	}
	return false
}

// handleClihelpCommand serves "<app> __clihelp <verb> [args...]".
func (a *App) handleClihelpCommand(args []string) error {
	verb, rest := "", []string(nil)
	if len(args) > 0 {
		verb, rest = args[0], args[1:]
	}
	if verb == "" || isHelpRequest(verb) {
		a.printClihelpVerbs(a.stdout())
		return nil
	}
	if len(rest) > 0 && isHelpRequest(rest[0]) {
		return a.printVerbHelp(a.stdout(), verb)
	}

	switch verb {
	case "version":
		fmt.Fprintf(a.stdout(), "clihelp %s\n", Version)
		return nil
	case "install":
		return a.clihelpInstall(rest)
	case "uninstall":
		res, err := UninstallShellIntegration(a, firstArg(rest))
		if err != nil {
			return err
		}
		reportUninstall(a.stdout(), res)
		return nil
	case "keys":
		return GenKeyBindings(a, firstArg(rest), a.stdout())
	case "wrapper":
		return a.clihelpWrapper(rest)
	default:
		return fmt.Errorf("unknown %s verb %q (try %s with no arguments)", protoClihelp, verb, protoClihelp)
	}
}

// printVerbHelp prints one verb's usage line.
func (a *App) printVerbHelp(w io.Writer, verb string) error {
	for _, v := range clihelpVerbs() {
		if v.name != verb {
			continue
		}
		fmt.Fprintf(w, "%s %s %s %s\n    %s\n", appName(a), protoClihelp, v.name, v.args, v.about)
		return nil
	}
	return fmt.Errorf("unknown %s verb %q (try %s with no arguments)", protoClihelp, verb, protoClihelp)
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func (a *App) printClihelpVerbs(w io.Writer) {
	fmt.Fprintf(w, "%s %s — shell integration for this program, built with clihelp %s\n\n", appName(a), protoClihelp, Version)
	width := 0
	for _, v := range clihelpVerbs() {
		if n := len(v.name) + len(v.args) + 1; n > width {
			width = n
		}
	}
	for _, v := range clihelpVerbs() {
		fmt.Fprintf(w, "  %s %-*s  %s\n", protoClihelp, width, strings.TrimSpace(v.name+" "+v.args), v.about)
	}
	fmt.Fprintf(w, "\nSupported shells: %s\n", strings.Join(SupportedShells, ", "))
}

// clihelpInstall writes the generated file's path to stdout, alone, so that a
// script can capture it; everything for a human goes to stderr.
func (a *App) clihelpInstall(args []string) error {
	keys := true
	if len(args) > 0 && args[0] == "--no-keys" {
		keys, args = false, args[1:]
	}
	res, err := InstallShellIntegration(a, firstArg(args), keys)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.stdout(), res.Integration)
	reportInstall(a.stderr(), a, res)
	return nil
}

// clihelpWrapper writes the wrapper script to stdout, alone, so that it can be
// redirected straight into a file; the line that registers it with the shell
// goes to stderr.
func (a *App) clihelpWrapper(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s wrapper <name> [<args>...]", protoClihelp)
	}
	name, wrapped := args[0], args[1:]
	if err := GenWrapperScript(a, name, wrapped, a.stdout()); err != nil {
		return err
	}
	printWrapperRegistration(a.stderr(), a, name, wrapped)
	return nil
}

const wrapperTemplate = `#!/bin/sh
# clihelp-wraps: {{target}}
#
# Wrapper for {{app}}, generated by "{{app}} {{proto}} wrapper {{name}}".
# Shell completion and the Alt-H explanation work through it because it answers
# clihelp's protocol calls on behalf of the program it wraps.

case $1 in
    __complete)
        shift
        exec {{app}} __complete {{args}} "$@"
        ;;
    __explain)
        # The first line of the answer is the command line to put back on the
        # prompt, and it is echoed back unchanged: rewriting "{{name}} ..." to
        # "{{target}} ..." would replace what was deliberately typed short.
        printf '%s\n' "$2"
        rest=${2#* }
        [ "$rest" = "$2" ] && rest=
        {{app}} __explain "{{target}} $rest" | tail -n +2
        exit 0
        ;;
esac

exec {{app}} {{args}} "$@"
`

// GenWrapperScript writes a wrapper script that invokes the application with
// args prepended, and forwards clihelp's protocol calls so that completion and
// Alt-H keep working through it.
//
// A wrapper is usually written by someone who is not the application's author,
// after it was built, which is why it carries its own "clihelp-wraps:" marker
// rather than the application declaring a list of wrappers it cannot know.
func GenWrapperScript(app *App, name string, args []string, w io.Writer) error {
	if app == nil {
		return errors.New("wrapper: app is nil")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("wrapper: a name is required")
	}

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, escapeShellArg(arg))
	}
	joined := strings.Join(quoted, " ")

	target := appName(app)
	if joined != "" {
		target += " " + joined
	}

	script := strings.NewReplacer(
		"{{app}}", appName(app),
		"{{name}}", name,
		"{{args}}", joined,
		"{{target}}", target,
		"{{proto}}", protoClihelp,
	).Replace(wrapperTemplate)
	_, err := io.WriteString(w, script)
	return err
}

// printWrapperRegistration prints the one line that tells a shell to complete
// name the way it completes the wrapped program. No shell will call a completion
// function for a name it was never told about, so this line is unavoidable —
// it is the same shape as git's __git_complete.
func printWrapperRegistration(w io.Writer, app *App, name string, args []string) {
	fn := strings.ReplaceAll(appName(app), "-", "_")
	target := appName(app)
	if len(args) > 0 {
		target += " " + strings.Join(args, " ")
	}

	lines := map[string]string{
		"bash": fmt.Sprintf("complete -F _%s_complete %s", fn, name),
		"zsh":  fmt.Sprintf("compdef _%s %s", fn, name),
		"fish": fmt.Sprintf("complete -c %s --wraps %s", name, escapeShellArg(target)),
	}

	fmt.Fprintf(w, "\n# Put %s somewhere on your PATH, then register it with your shell,\n", name)
	fmt.Fprintf(w, "# after the completion script for %s has been loaded:\n", appName(app))
	shell := detectShell()
	if line, ok := lines[shell]; ok {
		fmt.Fprintf(w, "#   %s\n", line)
		return
	}
	for _, s := range SupportedShells {
		fmt.Fprintf(w, "#   %-5s %s\n", s+":", lines[s])
	}
}
