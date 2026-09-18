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
const completionScriptVersion = 4

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
	name, err := safeAppName(app)
	if err != nil {
		return err
	}
	cleanName := shellFuncName(name)
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
	_, err = io.WriteString(w, tmpl)
	return err
}

// GenZshCompletion writes a Zsh tab-completion script to w.
func GenZshCompletion(app *App, w io.Writer) error {
	name, err := safeAppName(app)
	if err != nil {
		return err
	}
	cleanName := shellFuncName(name)
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
        # ${(q)...} is essential: _call_program ends in "eval ... $argv[2,-1]",
        # so anything spliced in unquoted is re-parsed as shell code, and $words
        # holds the command line the user has typed verbatim. Without the quoting
        # flag, a line containing $(...) executes when Tab is pressed.
        output=(${(f)"$(_call_program %[1]s ${(q)binary_cmd} __complete ${(q)words_to_pass[@]})"})
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
else
    # compinit has not run yet. Sourcing this file before it — a plugin manager
    # that defers compinit, or an rc file that sources clihelp's bootstrap near
    # the top — used to leave completion unregistered with nothing printed to
    # say so. Retry from the first prompt, then take the hook back out.
    _%[2]s_deferred_compdef() {
        type compdef >/dev/null 2>&1 || return
        compdef _%[2]s %[1]s
        add-zsh-hook -d precmd _%[2]s_deferred_compdef
        unfunction _%[2]s_deferred_compdef
    }
    autoload -Uz add-zsh-hook 2>/dev/null &&
        add-zsh-hook precmd _%[2]s_deferred_compdef
fi
`, name, cleanName, completionScriptVersion)
	_, err = io.WriteString(w, tmpl)
	return err
}

// GenFishCompletion writes a Fish tab-completion script to w.
func GenFishCompletion(app *App, w io.Writer) error {
	name, err := safeAppName(app)
	if err != nil {
		return err
	}
	cleanName := shellFuncName(name)
	tmpl := fmt.Sprintf(`# fish completion for %[1]s
# clihelp-completion-version: %[3]d
function __fish_%[2]s_complete
    set -l cmd (commandline -opc) (commandline -ct)
    test (count $cmd) -gt 1; and set -e cmd[1]
    %[1]s __complete $cmd
end

complete -c %[1]s -f -a '(__fish_%[2]s_complete)'
`, name, cleanName, completionScriptVersion)
	_, err = io.WriteString(w, tmpl)
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

	appName, err := safeAppName(app)
	if err != nil {
		return "", err
	}

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
	a.refreshManPage()

	sh := detectShell()
	if !isSupportedShell(sh) {
		return // no script exists for this shell; writing a bash one would be a lie
	}
	// A shell integration the user installed is kept current, and never created
	// here: a key binding appearing in someone's shell because they happened to
	// run an unrelated command would be an overreach, and editing a startup file
	// unasked doubly so.
	if path, err := IntegrationPath(a, sh); err == nil {
		if _, statErr := os.Stat(uninstalledMarker(path)); statErr == nil {
			return // the user removed it on purpose; putting anything back is not ours to do
		}
		if _, statErr := os.Stat(path); statErr == nil {
			if !integrationIsCurrent(path) {
				// Refresh only: an ordinary program run rewrites the generated
				// file and never touches a startup file.
				autoDebug(a, refreshShellIntegration(a, sh, integrationHasKeys(path)))
			}
			return
		}
	}
	// Creating a script where there is none is what AutoInstallCompletion
	// documents. Replacing one is only safe when this library wrote it:
	// completionIsCurrent answers "no marker" for a *foreign* script exactly as it
	// does for a stale one, so without this check a hand-written completion was
	// read as out of date and overwritten by an ordinary command.
	path, err := CompletionPath(a, sh)
	if err != nil {
		return
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		_, installErr := InstallCompletion(a, sh)
		autoDebug(a, installErr)
		return
	}
	if isGeneratedCompletionScript(path) && !completionIsCurrent(a, sh) {
		_, installErr := InstallCompletion(a, sh)
		autoDebug(a, installErr)
	}
}

// autoDebug surfaces an error the auto path would otherwise swallow, when
// CLIHELP_DEBUG is set.
//
// Discarding these is the right default — the user ran a command of their own,
// not an installer — but without an escape hatch a half-finished install is
// undiagnosable: "completion stopped working after the upgrade", no error, no
// log, and no way to ask.
func autoDebug(a *App, err error) {
	if err == nil || os.Getenv("CLIHELP_DEBUG") == "" {
		return
	}
	fmt.Fprintf(a.stderr(), "clihelp: %v\n", err)
}

// isGeneratedCompletionScript reports whether this library wrote the script at
// path. The delete path in install.go has always asked this question before
// removing a file; the write path did not.
func isGeneratedCompletionScript(path string) bool {
	return fileHasMarker(path, "clihelp-completion-version:", markerWindow)
}

// completionIsCurrent reports whether the installed script was generated by this
// version of the templates. IsCompletionInstalled keeps its documented meaning
// ("a script exists"); staleness is checked only where a rewrite is safe.
func completionIsCurrent(app *App, shell string) bool {
	path, err := CompletionPath(app, shell)
	if err != nil {
		return true // nothing we could install anyway
	}
	return fileHasMarker(path, fmt.Sprintf("clihelp-completion-version: %d", completionScriptVersion), markerWindow)
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
	if err = writeFileAtomically(targetPath, script.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("failed to write completion file %q: %w", targetPath, err)
	}

	return targetPath, nil
}

// writeFileAtomically writes data to a temporary file beside path and renames it
// into place.
//
// The contract, stated so it can be relied on: the replacement is atomic, so a
// reader sees either the old file or the new one; a symlink is followed rather
// than replaced; the file's permission bits are preserved by the caller passing
// them in; the data and the directory entry are flushed before it returns; and a
// clean error return leaves nothing behind. Ownership, ACLs and hard links are
// *not* preserved — a rename installs a new inode — and an interrupt between the
// write and the rename can leave one .tmp-* sibling.
func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if refuseForeignOwner(path) {
		return fmt.Errorf("refusing to write %q as root: it belongs to another user", path)
	}

	// Write through a symlink rather than over it. rename(2) replaces the link
	// itself, and ~/.zshrc is routinely a link into a dotfiles repo: replacing it
	// detaches the repo without a word, and git status stays clean. Resolving
	// also keeps the temp file beside the real file, so the rename stays within
	// one filesystem.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

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
	// Durability, not just visibility: the rename can reach disk before the data
	// blocks do, and the file at the other end of this is the user's shell
	// startup file. A machine that loses power mid-install should not bring back
	// an empty ~/.bashrc.
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	preserveOwner(path, tmp)
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		dir.Close()
	}
	return nil
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
			completionUninstallSubcommand(),
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

// completionInstallSubcommand sets up the shell integration for a shell.
func completionInstallSubcommand() Command {
	var noKeys bool
	return Command{
		Name:        "install",
		Description: "Install tab completion and the Alt-H key binding for a shell",
		UsageLine:   "completion install [--no-keys] [<shell>]",
		Examples: []Example{
			{Line: "completion install", Description: "Set up the active shell"},
			{Line: "completion install --no-keys zsh", Description: "Set up Zsh completion without the Alt-H binding"},
		},
		Parameters: []Param{
			{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
		},
		Options: []Option{
			Bool(&noKeys, "--no-keys", false, "Install tab completion only, leaving Alt-H alone"),
		},
		Notes: []Note{
			{
				Heading: "What It Writes",
				Text:    "One generated file under this application's configuration directory, and one permanent line in the shell's startup file that sources it. The line never changes; the generated file is rewritten whenever the application is upgraded. On fish nothing shared is touched at all, because conf.d is a drop-in directory. Run 'completion uninstall' to remove both.",
			},
		},
		Args: MaximumNArgs(1),
		Run: func(ctx *Context) error {
			shell := ""
			if len(ctx.Args) > 0 {
				shell = ctx.Args[0]
			}
			res, err := InstallShellIntegration(ctx.App, shell, !noKeys)
			if err != nil {
				return err
			}
			// The same stream rule as the __clihelp twin: the generated file's
			// path is the machine-readable answer, the report is for a human.
			fmt.Fprintln(ctx.Stdout, res.Integration)
			reportInstall(ctx.Stderr, ctx.App, res)
			return nil
		},
	}
}

// completionUninstallSubcommand removes what install wrote.
func completionUninstallSubcommand() Command {
	return Command{
		Name:        "uninstall",
		Description: "Remove the installed tab completion and key binding",
		UsageLine:   "completion uninstall [<shell>]",
		Parameters: []Param{
			{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
		},
		Args: MaximumNArgs(1),
		Run: func(ctx *Context) error {
			shell := ""
			if len(ctx.Args) > 0 {
				shell = ctx.Args[0]
			}
			res, err := UninstallShellIntegration(ctx.App, shell)
			if err != nil {
				return err
			}
			for _, path := range res.Removed {
				fmt.Fprintln(ctx.Stdout, path)
			}
			reportUninstall(ctx.Stderr, res)
			return nil
		},
	}
}

// reportInstall names every file the installation touched.
func reportInstall(w io.Writer, app *App, res InstallResult) {
	fmt.Fprintf(w, "\u2713 %s shell integration installed\n", res.Shell)
	fmt.Fprintf(w, "    generated:  %s\n", res.Integration)
	if res.Startup != "" {
		state := "already sourced by"
		if res.StartupEdit {
			state = "added the line that sources it to"
		}
		fmt.Fprintf(w, "    %s %s\n", state, res.Startup)
	}
	for _, path := range res.Removed {
		fmt.Fprintf(w, "    superseded, removed: %s\n", path)
	}
	fmt.Fprintln(w, "Restart your shell to activate it: <Tab> completes, Alt-H explains.")
	fmt.Fprintf(w, "Run '%s completion uninstall' to undo all of this.\n", appName(app))
}

func reportUninstall(w io.Writer, res InstallResult) {
	if len(res.Removed) == 0 && !res.StartupEdit {
		fmt.Fprintf(w, "nothing to remove for %s\n", res.Shell)
		return
	}
	fmt.Fprintf(w, "\u2713 %s shell integration removed\n", res.Shell)
	for _, path := range res.Removed {
		fmt.Fprintf(w, "    removed: %s\n", path)
	}
	if res.StartupEdit && res.Startup != "" {
		fmt.Fprintf(w, "    edited:  %s\n", res.Startup)
	}
}
