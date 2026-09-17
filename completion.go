package clihelp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-runewidth"
)

// SupportedShells lists available shell autocompletion formats.
var SupportedShells = []string{"bash", "zsh", "fish"}

// completionScriptVersion marks the generated scripts. It is raised whenever a
// template changes in a way that already-installed scripts must pick up, so that
// the auto-install path rewrites them instead of leaving an old script in place.
const completionScriptVersion = 2

// sanitizeCompletionField makes a string safe to put in one field of a
// completion record. The protocol is line-oriented with a tab between candidate
// and description, so an embedded newline or tab would forge a record; control
// characters are dropped because the candidates are printed to a terminal.
func sanitizeCompletionField(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 0x20 || r == 0x7f:
			return -1
		}
		return r
	}, s)
}

// emitCandidate writes one completion record: candidate, tab, description.
func emitCandidate(w io.Writer, candidate, description string) {
	fmt.Fprintf(w, "%s\t%s\n", sanitizeCompletionField(candidate), completionDescription(description))
}

// completionDescriptionWidth caps a candidate's description. zsh and fish print
// it beside the candidate in a menu that is as wide as the terminal; a
// paragraph there pushes the candidates apart and wraps over several rows.
const completionDescriptionWidth = 72

// completionDescription reduces a Description to the one short line a shell menu
// can show: inline markdown rendered away, whitespace collapsed, the first
// sentence only, and a cap in display columns. Sending the field raw put
// "**deep** — … [deep command](https://example.com/deep) …" into fish's menu,
// markup and all.
func completionDescription(s string) string {
	flat := strings.Join(strings.Fields(stripANSI(inline(s))), " ")
	return runewidth.Truncate(sanitizeCompletionField(firstSentence(flat)), completionDescriptionWidth, "…")
}

func (a *App) completeRootCommands(w io.Writer) {
	seen := map[string]bool{}
	emit := func(name, desc string) {
		if seen[name] {
			return
		}
		seen[name] = true
		emitCandidate(w, name, desc)
	}
	for _, cmd := range a.Commands {
		if !cmd.Hidden {
			emit(cmd.Name, cmd.Description)
		}
	}
	for _, s := range a.Shortcuts {
		if !s.Hidden {
			emit(s.Name, s.Description)
		}
	}
}

func completePrevFlagValue(w io.Writer, activeOptions []Option, prevWord, toComplete string) bool {
	if !strings.HasPrefix(prevWord, "-") {
		return false
	}
	for _, opt := range activeOptions {
		if opt.Complete == nil {
			continue
		}
		spec := parseFlagSpec(opt.Flags)
		matched := false
		for _, long := range spec.longNames {
			if "--"+long == prevWord {
				matched = true
				break
			}
		}
		for _, short := range spec.shortNames {
			if "-"+short == prevWord {
				matched = true
				break
			}
		}
		if matched {
			for _, res := range opt.Complete(toComplete) {
				// A callback may return "value\tdescription"; the first tab is
				// the field boundary, any later one is not.
				cand, desc, _ := strings.Cut(res, "\t")
				emitCandidate(w, cand, desc)
			}
			return true
		}
	}
	return false
}

func flagMatchesName(spec flagSpec, namePart string) bool {
	for _, long := range spec.longNames {
		if "--"+long == namePart {
			return true
		}
	}
	for _, short := range spec.shortNames {
		if "-"+short == namePart {
			return true
		}
	}
	return false
}

func completeFlagInlineValue(w io.Writer, activeOptions []Option, toComplete string, eq int) {
	namePart := toComplete[:eq]
	for _, opt := range activeOptions {
		if opt.Hidden {
			continue
		}
		spec := parseFlagSpec(opt.Flags)
		if !flagMatchesName(spec, namePart) {
			continue
		}
		if opt.Complete != nil {
			for _, res := range opt.Complete(toComplete[eq+1:]) {
				value, _, _ := strings.Cut(res, "\t")
				emitCandidate(w, namePart+"="+value, opt.Description)
			}
		} else {
			emitCandidate(w, namePart+"=", opt.Description)
		}
	}
}

func completeFlags(w io.Writer, activeOptions []Option, toComplete string) {
	if eq := strings.Index(toComplete, "="); eq >= 0 {
		completeFlagInlineValue(w, activeOptions, toComplete, eq)
		return
	}
	for _, opt := range activeOptions {
		if opt.Hidden {
			continue
		}
		spec := parseFlagSpec(opt.Flags)
		for _, long := range spec.longNames {
			flagName := "--" + long
			if strings.HasPrefix(flagName, toComplete) {
				emitCandidate(w, flagName, opt.Description)
			}
		}
		for _, short := range spec.shortNames {
			flagName := "-" + short
			if strings.HasPrefix(flagName, toComplete) {
				emitCandidate(w, flagName, opt.Description)
			}
		}
	}
}

func (a *App) completeSubcommands(w io.Writer, currentCmd *Command, toComplete string) {
	subcommands := a.Commands
	if currentCmd != nil {
		subcommands = currentCmd.Subcommands
	}

	matches := filterCommandsByPrefix(subcommands, toComplete)
	for _, cmd := range matches {
		emitCandidate(w, cmd.Name, cmd.Description)
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, toComplete) {
				emitCandidate(w, alias, cmd.Description)
			}
		}
	}

	if currentCmd == nil {
		seen := map[string]bool{}
		for _, sub := range a.Commands {
			if !sub.Hidden {
				seen[sub.Name] = true
			}
		}
		for _, s := range a.Shortcuts {
			if s.Hidden || seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			if strings.HasPrefix(s.Name, toComplete) {
				emitCandidate(w, s.Name, s.Description)
			}
		}
	}
}

func (a *App) handleComplete(_ context.Context, args []string) error {
	w := a.stdout()
	if len(args) == 0 {
		a.completeRootCommands(w)
		return nil
	}

	toComplete := args[len(args)-1]
	prevWord := ""
	if len(args) > 1 {
		prevWord = args[len(args)-2]
	}

	res, err := a.resolveCommand(args[:len(args)-1])
	if err != nil {
		return err
	}
	currentCmd := res.cmd

	activeOptions := a.collectOptions(res.path, currentCmd)

	if completePrevFlagValue(w, activeOptions, prevWord, toComplete) {
		return nil
	}

	if strings.HasPrefix(toComplete, "-") {
		completeFlags(w, activeOptions, toComplete)
		return nil
	}

	a.completeSubcommands(w, currentCmd, toComplete)
	return nil
}

// GenBashCompletion writes a Bash tab-completion script to w.
func GenBashCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`# bash completion for %[1]s
# clihelp-completion-version: %[3]d
_%[2]s_complete() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        # -n := keeps "--flag=value" and "host:port" as single words, which is
        # what the completion protocol is given below.
        _init_completion -n := || return
    else
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
        cur="${words[cword]}"
        prev="${words[cword-1]}"
    fi

    # Send the words bash-completion computed, not the raw COMP_WORDS: those are
    # split at every character of COMP_WORDBREAKS, so "--unit=" arrived as three
    # words and a colon-bearing argument as three more.
    local out
    out=$( "${words[0]}" __complete "${words[@]:1:cword-1}" "$cur" 2>/dev/null ) || return

    # Candidates are data: add them literally. compgen -W would expand them,
    # running any command substitution a candidate happens to contain.
    COMPREPLY=()
    local line cand
    while IFS= read -r line; do
        [[ -z $line ]] && continue
        cand="${line%%%%	*}"
        [[ $cand == "$cur"* ]] && COMPREPLY+=("$cand")
    done <<< "$out"

    if declare -F __ltrim_colon_completions >/dev/null 2>&1; then
        __ltrim_colon_completions "$cur"
    fi
}
complete -o default -F _%[2]s_complete %[1]s
`, name, cleanName, completionScriptVersion)
	_, err := io.WriteString(w, tmpl)
	return err
}

// GenZshCompletion writes a Zsh tab-completion script to w.
func GenZshCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`#compdef %[1]s
# clihelp-completion-version: %[3]d

_%[2]s() {
    local -a completions
    local -a completions_with_descriptions
    local line

    local -a words_to_pass
    if (( CURRENT > 1 )); then
        words_to_pass=("${(@)words[2,CURRENT]}")
    elif (( ${#words[@]} > 1 )); then
        words_to_pass=("${(@)words[2,-1]}")
    elif (( ${#@} > 0 )); then
        words_to_pass=("$@")
    fi

    local binary_cmd="${words[1]:-%[1]s}"
    local output
    if declare -f _call_program >/dev/null 2>&1; then
        output=(${(f)"$(_call_program %[1]s ${binary_cmd} __complete "${words_to_pass[@]}")"})
    else
        output=(${(f)"$(${binary_cmd} __complete "${words_to_pass[@]}")"})
    fi

    for line in "${output[@]}"; do
        if [[ -z "$line" ]]; then
            continue
        fi
        if [[ "$line" == *$'\t'* ]]; then
            local cand="${line%%%%	*}"
            local desc="${line#*	}"
            cand="${cand//:/\\:}"
            desc="${desc//:/\\:}"
            completions_with_descriptions+=("${cand}:${desc}")
        else
            completions+=("${line//:/\\:}")
        fi
    done

    if [ -n "$completions_with_descriptions" ]; then
        _describe -t commands '%[1]s' completions_with_descriptions
    fi
    if [ -n "$completions" ]; then
        compadd -a completions
    fi
}

# Autoloaded from $fpath this file *is* the completion function and has to call
# it; sourced from a startup file it must only register itself, because calling
# the function outside a completion context prints "can only be called from
# completion function" at every shell start. $funcstack[1] tells the two apart:
# the function's name when autoloaded, this file's path when sourced.
if [ "$funcstack[1]" = "_%[2]s" ]; then
    _%[2]s "$@"
elif type compdef >/dev/null 2>&1; then
    compdef _%[2]s %[1]s
fi
`, name, cleanName, completionScriptVersion)
	_, err := io.WriteString(w, tmpl)
	return err
}

// GenFishCompletion writes a Fish tab-completion script to w.
func GenFishCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`# fish completion for %[1]s
# clihelp-completion-version: %[3]d
function __fish_%[2]s_complete
    set -l cmd (commandline -opc) (commandline -ct)
    test (count $cmd) -gt 1; and set -e cmd[1]
    %[1]s __complete $cmd
end

complete -c %[1]s -f -a '(__fish_%[2]s_complete)'
`, name, cleanName, completionScriptVersion)
	_, err := io.WriteString(w, tmpl)
	return err
}

// CompletionPath returns the target installation path for the completion script.
func CompletionPath(app *App, shell string) (string, error) {
	if app == nil {
		return "", errors.New("completion path: app is nil")
	}
	if shell == "" {
		shell = detectShell()
	}
	shell = strings.ToLower(strings.TrimSpace(shell))
	if shell == "" {
		return "", errors.New("cannot detect the active shell: $SHELL is not set; name one of " + strings.Join(SupportedShells, ", "))
	}

	appName := appName(app)

	var targetDir, fileName string
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user home directory: %w", err)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}

	switch shell {
	case "bash":
		targetDir = filepath.Join(dataHome, "bash-completion", "completions")
		fileName = appName
	case "zsh":
		targetDir = filepath.Join(dataHome, "zsh", "site-functions")
		fileName = "_" + appName
	case "fish":
		targetDir = filepath.Join(configHome, "fish", "completions")
		fileName = appName + ".fish"
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(SupportedShells, ", "))
	}

	return filepath.Join(targetDir, fileName), nil
}

// IsCompletionInstalled checks if the shell completion script is already installed
// in the user's standard XDG directory for the given shell (or detected active shell).
func IsCompletionInstalled(app *App, shell string) bool {
	if app == nil {
		return false
	}
	path, err := CompletionPath(app, shell)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// maybeAutoInstallCompletion checks and installs completions silently if enabled.
func (a *App) maybeAutoInstallCompletion(args []string) {
	if a == nil || !a.AutoInstallCompletion {
		return
	}
	// Never run during an internal protocol call — a setup call must not have an
	// install as a side effect — or in CI/non-interactive test runs.
	if len(args) > 0 && strings.HasPrefix(args[0], "__") {
		return
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("TERM") == "dumb" || os.Getenv("NO_AUTO_COMPLETION") != "" || os.Getenv("CLIHELP_NO_AUTO_COMPLETION") != "" {
		return
	}
	sh := detectShell()
	if !isSupportedShell(sh) {
		return // no script exists for this shell; writing a bash one would be a lie
	}
	if !IsCompletionInstalled(a, sh) || !completionIsCurrent(a, sh) {
		_, _ = InstallCompletion(a, sh)
	}
}

// completionIsCurrent reports whether the installed script was generated by this
// version of the templates. IsCompletionInstalled keeps its documented meaning
// ("a script exists"); staleness is checked only where a rewrite is safe.
func completionIsCurrent(app *App, shell string) bool {
	path, err := CompletionPath(app, shell)
	if err != nil {
		return true // nothing we could install anyway
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	head := make([]byte, 256)
	n, _ := f.Read(head)
	marker := fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion)
	return strings.Contains(string(head[:n]), marker)
}

// InstallCompletion installs the shell completion script for the given app and shell.
// If shell is empty, it detects the active shell via the SHELL environment variable.
// Returns the absolute file path where the completion script was written.
func InstallCompletion(app *App, shell string) (string, error) {
	targetPath, err := CompletionPath(app, shell)
	if err != nil {
		return "", err
	}
	if shell == "" {
		shell = detectShell()
	}
	shell = strings.ToLower(strings.TrimSpace(shell))

	var script bytes.Buffer
	switch shell {
	case "bash":
		err = GenBashCompletion(app, &script)
	case "zsh":
		err = GenZshCompletion(app, &script)
	case "fish":
		err = GenFishCompletion(app, &script)
	}
	if err != nil {
		return "", fmt.Errorf("failed to generate %s completion: %w", shell, err)
	}

	targetDir := filepath.Dir(targetPath)
	if err = os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %q: %w", targetDir, err)
	}
	if err = writeFileAtomically(targetPath, script.Bytes()); err != nil {
		return "", fmt.Errorf("failed to write completion file %q: %w", targetPath, err)
	}

	return targetPath, nil
}

// writeFileAtomically writes data to a temporary file beside path and renames it
// over path, so an interrupted or failing write cannot leave a half-written
// script where a working one used to be. Close is checked, because it is where a
// buffered write reports a full disk.
func writeFileAtomically(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op once the rename has succeeded

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// detectShell names the shell from $SHELL, or returns "" when $SHELL is unset.
// It does not translate an unknown shell into "bash": a dash, ksh or nushell
// user was silently given a bash script in their home directory, which their
// shell cannot read and which nothing ever removes.
func detectShell() string {
	sh := os.Getenv("SHELL")
	if strings.TrimSpace(sh) == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(sh))
}

// isSupportedShell reports whether a completion script exists for shell.
func isSupportedShell(shell string) bool {
	for _, s := range SupportedShells {
		if s == shell {
			return true
		}
	}
	return false
}

// CompletionCommand returns a standard clihelp.Command providing 'bash', 'zsh', 'fish', and 'install' subcommands.
func CompletionCommand() Command {
	return Command{
		Name:        "completion",
		Description: "Generate or install shell tab-completion scripts",
		UsageLine:   "completion <subcommand>",
		Examples: []Example{
			{Line: "completion zsh", Description: "Generate Zsh tab-completion script"},
			{Line: "completion install", Description: "Install tab-completions for the active shell"},
		},
		Notes: []Note{
			{
				Heading: "Shell Tip",
				Text:    "Tip: <Tab> to complete, Ctrl-D to list choices. Run 'completion keys' and source the result from your shell's rc file to bind Alt-H, which expands the command line and shows the help for the command it names.",
			},
		},
		Subcommands: append(completionGenerateSubcommands(),
			completionKeysSubcommand(),
			completionInstallSubcommand(),
		),
	}
}

// completionGenerateSubcommands returns the per-shell script generators.
func completionGenerateSubcommands() []Command {
	return []Command{
		{
			Name:        "bash",
			Description: "Generate Bash tab-completion script",
			UsageLine:   "completion bash",
			Args:        NoArgs,
			Run: func(ctx *Context) error {
				return GenBashCompletion(ctx.App, ctx.Stdout)
			},
		},
		{
			Name:        "zsh",
			Description: "Generate Zsh tab-completion script",
			UsageLine:   "completion zsh",
			Args:        NoArgs,
			Run: func(ctx *Context) error {
				return GenZshCompletion(ctx.App, ctx.Stdout)
			},
		},
		{
			Name:        "fish",
			Description: "Generate Fish tab-completion script",
			UsageLine:   "completion fish",
			Args:        NoArgs,
			Run: func(ctx *Context) error {
				return GenFishCompletion(ctx.App, ctx.Stdout)
			},
		},
	}
}

// completionKeysSubcommand prints the shell snippet that binds Alt-H.
func completionKeysSubcommand() Command {
	return Command{
		Name:        "keys",
		Description: "Print shell key bindings (Alt-H expands the command line and explains it)",
		UsageLine:   "completion keys [<shell>]",
		Examples: []Example{
			{Line: "completion keys bash", Description: "Print the Bash key bindings"},
		},
		Parameters: []Param{
			{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
		},
		Notes: []Note{
			{
				Heading: "Why This Is Separate",
				Text:    "Every shell loads a completion script lazily, on the first completion of the command, so a key binding written there would not exist until <Tab> had already been pressed once. Source this from your shell's rc file instead; the first line of the output says how.",
			},
		},
		Args: MaximumNArgs(1),
		Run: func(ctx *Context) error {
			shell := ""
			if len(ctx.Args) > 0 {
				shell = ctx.Args[0]
			}
			return GenKeyBindings(ctx.App, shell, ctx.Stdout)
		},
	}
}

// completionInstallSubcommand installs the completion script for a shell.
func completionInstallSubcommand() Command {
	return Command{
		Name:        "install",
		Description: "Install tab-completion script to standard user directory",
		UsageLine:   "completion install [<shell>]",
		Examples: []Example{
			{Line: "completion install zsh", Description: "Install completions to ~/.local/share/zsh/site-functions"},
		},
		Parameters: []Param{
			{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
		},
		Args: MaximumNArgs(1),
		Run: func(ctx *Context) error {
			shell := ""
			if len(ctx.Args) > 0 {
				shell = ctx.Args[0]
			}
			path, err := InstallCompletion(ctx.App, shell)
			if err != nil {
				return err
			}
			fmt.Fprintf(ctx.Stdout, "✓ Autocompletion installed to: %s\n", path)
			if shell == "zsh" || (shell == "" && detectShell() == "zsh") {
				fmt.Fprintln(ctx.Stdout, "Note: If not already configured, ensure the directory is in your Zsh $fpath in ~/.zshrc:")
				fmt.Fprintln(ctx.Stdout, "    fpath=(~/.local/share/zsh/site-functions $fpath)")
			}
			fmt.Fprintf(ctx.Stdout, "Tip: <Tab> to complete, Ctrl-D to list choices.\n")
			fmt.Fprintf(ctx.Stdout, "     For Alt-H (expand the command line and show its help), add this to your shell's rc file:\n")
			fmt.Fprintf(ctx.Stdout, "         eval \"$(%s completion keys)\"\n", appName(ctx.App))
			fmt.Fprintln(ctx.Stdout, "Restart your shell or open a new terminal session to activate.")
			return nil
		},
	}
}
