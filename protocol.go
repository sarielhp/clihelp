package clihelp

import (
	"fmt"
	"io"
	"os"
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
		{"install", "[--no-keys] [--no-man] [<shell>]", "set this program up: completion, the Alt-H binding and the manual page"},
		{"uninstall", "[<shell>]", "remove what install wrote"},
		{"keys", "[<shell>]", "print the key bindings, for inspection or manual setup"},
		{"wrapper", "[--from <path>] <name> [<args>...]", "print a wrapper script for this program, with arguments"},
		{"manpage", "[--install|--uninstall] [--force]", "print a roff manual page, or install it for man(1)"},
	}
}

// isHelpRequest recognizes the spellings anyone reaches for first. Without it,
// "__clihelp --help" was an unknown verb and "__clihelp wrapper --help"
// cheerfully generated a wrapper script named "--help".
//
// "-H" is clihelp's own extended-help flag and asks for more, as it does
// everywhere else in the library. It is accepted whatever App.ExtendedHelpFlag
// says, because that field governs the application's flags, and __clihelp is the
// library's surface rather than the application's.
func isHelpRequest(arg string) bool {
	switch arg {
	case "-h", "--help", "help", "-H":
		return true
	}
	return false
}

func isExtendedHelpRequest(arg string) bool { return arg == "-H" }

// handleClihelpCommand serves "<app> __clihelp <verb> [args...]".
func (a *App) handleClihelpCommand(args []string) error {
	verb, rest := "", []string(nil)
	if len(args) > 0 {
		verb, rest = args[0], args[1:]
	}
	if verb == "" || isHelpRequest(verb) {
		a.printClihelpVerbs(a.stdout(), isExtendedHelpRequest(verb))
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
		shells, err := parseVerbArgs("uninstall", rest, nil, 1)
		if err != nil {
			return err
		}
		res, err := uninstallProgram(a, firstArg(shells))
		if err != nil {
			return err
		}
		// Paths to stdout, prose to stderr: see the stream rule in AGENTS.md.
		for _, path := range res.Removed {
			fmt.Fprintln(a.stdout(), path)
		}
		reportUninstall(a.stderr(), res)
		return nil
	case "keys":
		shells, err := parseVerbArgs("keys", rest, nil, 1)
		if err != nil {
			return err
		}
		return GenKeyBindings(a, firstArg(shells), a.stdout())
	case "wrapper":
		return a.clihelpWrapper(rest)
	case "manpage":
		return a.clihelpManPage(rest)
	default:
		return fmt.Errorf("unknown %s verb %q (try %s with no arguments)", protoClihelp, verb, protoClihelp)
	}
}

// printVerbHelp prints one verb's usage line.
func (a *App) printVerbHelp(w io.Writer, verb string) error {
	o := Options{Theme: a.Theme}.withApp(a)
	th := o.theme(a)
	for _, v := range clihelpVerbs() {
		if v.name != verb {
			continue
		}
		usage := strings.TrimSpace(fmt.Sprintf("%s %s", th.Subcommand.Sprint(v.name), th.Flag.Sprint(v.args)))
		fmt.Fprintf(w, "%s %s %s\n    %s\n", th.Accent.Sprint(appName(a)), th.Flag.Sprint(protoClihelp), usage, th.Body.Sprint(v.about))
		return nil
	}
	return fmt.Errorf("unknown %s verb %q (try %s with no arguments)", protoClihelp, verb, protoClihelp)
}

// parseVerbArgs splits a verb's arguments into the flags it accepts and its
// positionals, in any order, and rejects anything it was not told about.
//
// Each verb used to parse for itself: install looked at args[0] only, manpage
// looped over everything, and the rest silently discarded surplus positionals.
// So "install bash --no-keys" installed the key binding the user had just
// declined, and "uninstall bash zsh" said nothing about zsh. The visible
// commands get this grammar from pflag; this is the same grammar for the hidden
// spelling of the same verbs.
func parseVerbArgs(verb string, args []string, flags map[string]*bool, maxPositional int) ([]string, error) {
	var positional []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			seen, ok := flags[arg]
			if !ok {
				return nil, fmt.Errorf("unknown option %q for %s %s", arg, protoClihelp, verb)
			}
			*seen = true
			continue
		}
		positional = append(positional, arg)
	}
	if len(positional) > maxPositional {
		return nil, fmt.Errorf("%s %s accepts at most %d argument(s), got %q", protoClihelp, verb, maxPositional, strings.Join(positional, " "))
	}
	return positional, nil
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func (a *App) printClihelpVerbs(w io.Writer, extended bool) {
	o := Options{Writer: w, Theme: a.Theme}.withApp(a)
	th := o.theme(a)
	name := appName(a)

	fmt.Fprintf(w, "%s %s — %s (clihelp %s)\n\n",
		th.Accent.Sprint(name),
		th.Flag.Sprint(protoClihelp),
		th.Body.Sprint("shell integration for this program, built with"),
		th.ExampleComment.Sprint(Version),
	)

	fmt.Fprintf(w, "%s\n  %s %s %s %s\n\n",
		th.Hdr.Sprint("Usage:"),
		th.Accent.Sprint(name),
		th.Flag.Sprint(protoClihelp),
		th.Subcommand.Sprint("<verb>"),
		th.Flag.Sprint("[options...]"),
	)

	fmt.Fprintf(w, "%s\n", th.Hdr.Sprint("Verbs:"))
	var params []Param
	for _, v := range clihelpVerbs() {
		disp := v.name
		if v.args != "" {
			disp += " " + v.args
		}
		params = append(params, Param{
			Name:        disp,
			Description: v.about,
		})
	}

	termWidth := o.width()
	indent := colIndentFor(params, termWidth, minTextColumns)

	for _, v := range clihelpVerbs() {
		verbDisplay := th.Subcommand.Sprint(v.name)
		if v.args != "" {
			verbDisplay += " " + th.Flag.Sprint(v.args)
		}
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, verbDisplay, v.about)
	}

	fmt.Fprintf(w, "\n%s\n", th.Hdr.Sprint("Shells:"))
	activeShell := detectShell()
	if activeShell != "" {
		fmt.Fprintf(w, "  Detected:  %s %s\n",
			th.Subcommand.Sprint(activeShell),
			th.ExampleComment.Sprint("(active)"),
		)
	}
	fmt.Fprintf(w, "  Supported: %s\n", th.Body.Sprint(strings.Join(supportedShells, ", ")))

	fmt.Fprintf(w, "\n%s\n", th.Hdr.Sprint("Examples:"))
	examples := []struct{ cmd, comment string }{
		{name + " " + protoClihelp + " install", "# set up completion, Alt-H, and man page"},
		{name + " " + protoClihelp + " wrapper pd deploy > ~/bin/pd", "# generate wrapper script with preset args"},
		{name + " " + protoClihelp + " wrapper --from ~/bin/mt", "# inspect existing script and generate wrapper"},
		{name + " " + protoClihelp + " manpage --install", "# install manual page for man(1)"},
	}
	for _, ex := range examples {
		fmt.Fprintf(w, "  %-46s %s\n",
			th.Subcommand.Sprint(ex.cmd),
			th.ExampleComment.Sprint(ex.comment),
		)
	}

	if !extended {
		fmt.Fprintf(w, "\nRun '%s %s -H' for what install writes and which names are reserved.\n", name, protoClihelp)
		return
	}
	a.printClihelpDetail(w, th)
}

// printClihelpDetail answers the two questions a hidden command cannot answer
// through the ordinary help system: what it would write on this machine, and
// what it has reserved.
func (a *App) printClihelpDetail(w io.Writer, th Theme) {
	fmt.Fprintf(w, "\n%s\n", th.Hdr.Sprint("Reserved argument names — an application must not declare commands with them:"))
	fmt.Fprintf(w, "  %-12s %s\n", th.Flag.Sprint(protoComplete), th.Body.Sprint("the completion protocol, called by the generated shell scripts"))
	fmt.Fprintf(w, "  %-12s %s\n", th.Flag.Sprint(protoExplain), th.Body.Sprint("the Alt-H protocol, called by the generated key bindings"))
	fmt.Fprintf(w, "  %-12s %s\n", th.Flag.Sprint(protoClihelp), th.Body.Sprint("these verbs"))

	shell, shellErr := resolveShell("")
	fmt.Fprintf(w, "\n%s", th.Hdr.Sprint("What 'install' would write here"))
	if shellErr != nil {
		fmt.Fprintf(w, " (no shell detected; name one on the command line):\n")
		return
	}
	fmt.Fprintf(w, ", for %s:\n", th.Subcommand.Sprint(shell))
	if path, err := IntegrationPath(a, shell); err == nil {
		fmt.Fprintf(w, "  %s  %s\n", th.Flag.Sprint("generated:"), th.Body.Sprint(path))
	}
	if path, owned, err := startupFile(a, shell); err == nil {
		what := "a marked block in"
		if owned {
			what = "a drop-in file:"
		}
		fmt.Fprintf(w, "  %s %s %s\n", th.Flag.Sprint("sourced by:"), what, th.Body.Sprint(path))
	}
	fmt.Fprintf(w, "%s\n", th.ExampleComment.Sprint("Nothing runs at shell startup: the line is a file test and a source, and an\nupgrade rewrites the generated file rather than your configuration."))
}

// clihelpInstall writes the generated file's path to stdout, alone, so that a
// script can capture it; everything for a human goes to stderr.
func (a *App) clihelpInstall(args []string) error {
	var noKeys, noMan bool
	rest, err := parseVerbArgs("install", args, map[string]*bool{"--no-keys": &noKeys, "--no-man": &noMan}, 1)
	if err != nil {
		return err
	}
	res, err := installProgram(a, firstArg(rest), !noKeys, !noMan)
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
	var fromPath string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--from":
			if i+1 >= len(args) {
				return fmt.Errorf("%s wrapper: --from requires a path or command argument", protoClihelp)
			}
			i++
			fromPath = args[i]
		case strings.HasPrefix(arg, "--from="):
			fromPath = strings.TrimPrefix(arg, "--from=")
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown option %q for %s wrapper", arg, protoClihelp)
		default:
			positional = append(positional, arg)
		}
	}

	return executeWrapperGen(a, fromPath, positional, a.stdout(), a.stderr())
}

func executeWrapperGen(app *App, fromPath string, positional []string, stdout, stderr io.Writer) error {
	var name string
	var wrapped []string

	if fromPath != "" {
		targetPath, defaultName, err := resolveWrapperTarget(fromPath)
		if err != nil {
			return err
		}
		f, err := os.Open(targetPath)
		if err != nil {
			return fmt.Errorf("failed to open wrapper script %q: %w", targetPath, err)
		}
		defer f.Close()

		extracted, err := extractWrapperArgs(f, appName(app))
		if err != nil {
			return err
		}
		wrapped = extracted

		if len(positional) > 0 {
			name = positional[0]
			if len(positional) > 1 {
				return fmt.Errorf("%s wrapper: cannot specify preset arguments when using --from", protoClihelp)
			}
		} else {
			name = defaultName
		}
	} else {
		if len(positional) == 0 {
			return fmt.Errorf("usage: %s wrapper [--from <path>] <name> [<args>...]", protoClihelp)
		}
		name, wrapped = positional[0], positional[1:]
	}

	if err := GenWrapperScript(app, name, wrapped, stdout); err != nil {
		return err
	}
	printWrapperRegistration(stderr, app, name, wrapped)
	return nil
}

// clihelpManPage prints the manual page, or installs it. Printing goes to stdout
// alone so a packager can redirect it into their build.
func (a *App) clihelpManPage(args []string) error {
	var install, uninstall, force bool
	if _, err := parseVerbArgs("manpage", args, map[string]*bool{
		"--install": &install, "--uninstall": &uninstall, "--force": &force,
	}, 0); err != nil {
		return err
	}
	return a.manPageAction(a.stdout(), a.stderr(), install, uninstall, force)
}

const wrapperTemplate = `#!/bin/sh
# clihelp-wraps: {{target}}
#
# Wrapper for {{app}}, generated by "{{app}} {{proto}} wrapper {{name}}".
# Shell completion and the Alt-H explanation work through it because it answers
# clihelp's protocol calls on behalf of the program it wraps.

# Quoted once, here, so that nothing below interpolates shell text into a
# context where it would be re-parsed.
__clihelp_app={{app_q}}
__clihelp_target={{target_q}}

case $1 in
    __complete)
        shift
        exec "$__clihelp_app" __complete {{args}} "$@"
        ;;
    __explain)
        # The first line of the answer is the command line to put back on the
        # prompt, and it is echoed back unchanged: rewriting "{{name}} ..." to
        # the wrapped form would replace what was deliberately typed short.
        printf '%s\n' "$2"
        line=$2
        while [ "${line#[[:space:]]}" != "$line" ]; do line=${line#[[:space:]]}; done
        # Split on the first run of blanks, not on a space: "${line#* }" found
        # no match when the separator was a tab, which emptied rest and dropped
        # every argument the user had typed.
        first=${line%%[[:space:]]*}
        rest=${line#"$first"}
        while [ "${rest#[[:space:]]}" != "$rest" ]; do rest=${rest#[[:space:]]}; done
        "$__clihelp_app" __explain "$__clihelp_target${rest:+ }$rest" | tail -n +2
        exit 0
        ;;
esac

exec "$__clihelp_app" {{args}} "$@"
`

// GenWrapperScript writes a wrapper script that invokes the application with
// args prepended, and forwards clihelp's protocol calls so that completion and
// Alt-H keep working through it.
//
// A wrapper is usually written by someone who is not the application's author,
// after it was built, which is why it carries its own "clihelp-wraps:" marker
// rather than the application declaring a list of wrappers it cannot know.
func GenWrapperScript(app *App, name string, args []string, w io.Writer) error {
	appN, err := safeAppName(app)
	if err != nil {
		return err
	}
	// A wrapper name becomes a command name and a shell token in the registration
	// line the user is told to paste, so it is held to the same rule.
	if err := safeWrapperName(name); err != nil {
		return err
	}

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, escapeShellArg(arg))
	}
	joined := strings.Join(quoted, " ")

	target := appN
	if joined != "" {
		target += " " + joined
	}

	script := strings.NewReplacer(
		"{{app}}", appN,
		"{{app_q}}", escapeShellArg(appN),
		"{{name}}", name,
		"{{args}}", joined,
		"{{target}}", target,
		"{{target_q}}", escapeShellArg(target),
		"{{proto}}", protoClihelp,
	).Replace(wrapperTemplate)
	_, err = io.WriteString(w, script)
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

	// Two registrations, because there are two mechanisms: the shell's completion
	// table, and the Alt-H dispatcher's registry. Without the second the wrapper's
	// own __explain branch is unreachable from a keystroke, which is what made the
	// documented behaviour impossible.
	// The registry entry carries the protocol the wrapper's __explain speaks;
	// see the key-binding snippets for the format.
	entry := fmt.Sprintf("%s:%d", name, explainProtocolVersion)
	lines := map[string]string{
		"bash": fmt.Sprintf("complete -F _%s_complete %s\n#   _clihelp_apps=\"${_clihelp_apps:-} %s \"", fn, name, entry),
		"zsh":  fmt.Sprintf("compdef _%s %s\n#   _clihelp_apps=\"${_clihelp_apps:-} %s \"", fn, name, entry),
		"fish": fmt.Sprintf("complete -c %s --wraps %s\n#   contains -- %s $_clihelp_apps; or set -g _clihelp_apps $_clihelp_apps %s", name, escapeShellArg(target), entry, entry),
	}

	fmt.Fprintf(w, "\n# Put %s somewhere on your PATH, then register it with your shell,\n", name)
	fmt.Fprintf(w, "# after the completion script for %s has been loaded:\n", appName(app))
	shell := detectShell()
	if line, ok := lines[shell]; ok {
		fmt.Fprintf(w, "#   %s\n", line)
		return
	}
	for _, s := range supportedShells {
		fmt.Fprintf(w, "#   %-5s %s\n", s+":", lines[s])
	}
}
