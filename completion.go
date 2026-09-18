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
	tmpl := fmt.Sprintf(bashCompletionTemplate, name, cleanName, completionScriptVersion)
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
	tmpl := fmt.Sprintf(zshCompletionTemplate, name, cleanName, completionScriptVersion)
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
	tmpl := fmt.Sprintf(fishCompletionTemplate, name, cleanName, completionScriptVersion)
	_, err = io.WriteString(w, tmpl)
	return err
}

// CompletionPath returns the target installation path for the completion script.
func CompletionPath(app *App, shell string) (string, error) {
	if app == nil {
		return "", errors.New("completion path: app is nil")
	}
	shell, err := resolveShell(shell)
	if err != nil {
		return "", err
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

	sh, err := resolveShell("")
	if err != nil {
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
	// A completion script at the older location — installed before this library
	// grew its one-file integration, or by a packager — is kept current, and
	// that is all. This path runs on every ordinary program run, unasked, and
	// bringing a file into existence in someone's home directory is not a
	// decision it gets to make: AutoInstallCompletion is the *author's* choice,
	// while the file lands in the *user's* home. It used to create one here, so
	// running a program for the first time installed something nobody had asked
	// for. Creating is what "completion install" is for.
	//
	// Overwriting is also only safe when this library wrote the file:
	// completionIsCurrent answers "no marker" for a *foreign* script exactly as
	// it does for a stale one, so without the marker check a hand-written
	// completion was read as out of date and replaced by an ordinary command.
	path, err := CompletionPath(a, sh)
	if err != nil {
		return
	}
	if _, statErr := os.Stat(path); statErr != nil {
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
	shell, err = resolveShell(shell)
	if err != nil {
		return "", err
	}

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
	var noKeys, noMan bool
	return Command{
		Name:        "install",
		Description: "Set this program up: tab completion, the Alt-H key binding and the manual page",
		UsageLine:   "completion install [--no-keys] [--no-man] [<shell>]",
		Examples: []Example{
			{Line: "completion install", Description: "Set up the active shell"},
			{Line: "completion install --no-keys zsh", Description: "Set up Zsh completion without the Alt-H binding"},
		},
		Parameters: []Param{
			{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
		},
		Options: []Option{
			Bool(&noKeys, "--no-keys", false, "Install tab completion only, leaving Alt-H alone"),
			Bool(&noMan, "--no-man", false, "Skip the manual page"),
		},
		Notes: []Note{
			{
				Heading: "What It Writes",
				Text:    "One generated file under this application's configuration directory, one permanent line in the shell's startup file that sources it, and a manual page under the user's data directory. The line never changes; the generated file is rewritten whenever the application is upgraded. On fish nothing shared is touched at all, because conf.d is a drop-in directory. Run 'completion uninstall' to remove all of it.",
			},
		},
		Args: MaximumNArgs(1),
		Run: func(ctx *Context) error {
			shell := ""
			if len(ctx.Args) > 0 {
				shell = ctx.Args[0]
			}
			res, err := installProgram(ctx.App, shell, !noKeys, !noMan)
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
			res, err := uninstallProgram(ctx.App, shell)
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
	if res.ManPage != "" {
		fmt.Fprintf(w, "    manual page: %s\n", res.ManPage)
	}
	for _, warning := range res.Warnings {
		fmt.Fprintf(w, "    ! %s\n", warning)
	}
	fmt.Fprintln(w, "Restart your shell to activate it: <Tab> completes, Alt-H explains.")
	fmt.Fprintf(w, "Run '%s uninstall' to undo all of this.\n", setupHint(app))
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
	for _, warning := range res.Warnings {
		fmt.Fprintf(w, "    ! %s\n", warning)
	}
}
