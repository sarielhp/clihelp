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

// supportedShells lists available shell autocompletion formats.

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
// The colour form, then stripped: it is the visible text that belongs in a
// completion menu. The plain form would be the wrong choice here — under
// NoColor it spells nothing out, but asking for it couples this to whether
// the shell happens to be a terminal.
func completionDescription(s string) string {
	flat := strings.Join(strings.Fields(stripANSI(renderInlineTo(s, false))), " ")
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

// effectiveFlagToken reduces a shorthand cluster to the flag that actually takes
// the value: in "-vo <val>" that is "-o". Only meaningful once scanLeadingFlag has
// reported that the word consumes the next argument, which guarantees every short
// before the last takes no value.
func effectiveFlagToken(prevWord string) string {
	if strings.HasPrefix(prevWord, "--") || !strings.HasPrefix(prevWord, "-") || len(prevWord) <= 2 {
		return prevWord
	}
	return "-" + prevWord[len(prevWord)-1:]
}

// positionalArity maps every flag legal at this command to whether it consumes
// the following argument, hidden flags included. It is leadingFlagArity plus the
// command's own Options: those cannot precede the command, but they are exactly
// the flags that precede its positionals.
func (a *App) positionalArity(res resolution, cmd *Command) map[string]bool {
	resolved := append([]*Command{}, res.ancestors...)
	if cmd != nil {
		resolved = append(resolved, cmd)
	}
	arity := a.leadingFlagArity(resolved)
	if cmd != nil {
		addFlagArity(arity, cmd.Options)
	}
	return arity
}

func positionalIndex(arity map[string]bool, remaining []string) (n int, afterTerminator bool) {
	for i := 0; i < len(remaining); {
		arg := remaining[i]
		if arg == "--" {
			return n + len(remaining) - i - 1, true
		}
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			if count, known := scanLeadingFlag(arity, remaining[i:]); known {
				i += count
				continue
			}
		}
		n++
		i++
	}
	return n, false
}

// paramAt returns the Param occupying slot n, or nil. Past the end only a
// Variadic final parameter answers; Audit enforces that Variadic appears
// nowhere else, so the last entry is the only one that can be unbounded.
func paramAt(params []Param, n int) *Param {
	if n < 0 || len(params) == 0 {
		return nil
	}
	if n < len(params) {
		return &params[n]
	}
	if last := &params[len(params)-1]; last.Variadic {
		return last
	}
	return nil
}

func completePositional(w io.Writer, cmd *Command, n int, toComplete string) {
	if cmd == nil {
		return
	}
	p := paramAt(cmd.Parameters, n)
	if p == nil || p.Complete == nil {
		return
	}
	for _, res := range p.Complete(toComplete) {
		cand, desc, _ := strings.Cut(res, "\t")
		emitCandidate(w, cand, desc)
	}
}

func (a *App) handleComplete(ctx context.Context, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
	arity := a.positionalArity(res, currentCmd)
	n, afterTerminator := positionalIndex(arity, res.remaining)

	if !afterTerminator {
		// A value-taking flag owns the next word. Whether or not the flag has a
		// callback, nothing else may answer for it -- see plan §3.4.
		if count, known := scanLeadingFlag(arity, []string{prevWord, toComplete}); known && count == 2 {
			completePrevFlagValue(w, activeOptions, effectiveFlagToken(prevWord), toComplete)
			return nil
		}
		if strings.HasPrefix(toComplete, "-") {
			completeFlags(w, activeOptions, toComplete)
			return nil
		}
		if n == 0 {
			a.completeSubcommands(w, currentCmd, toComplete) // a subcommand only ever sits at slot 0
		}
	}
	completePositional(w, currentCmd, n, toComplete)
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
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(supportedShells, ", "))
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

// installCompletion installs the shell completion script for the given app and shell.
// If shell is empty, it detects the active shell via the SHELL environment variable.
// Returns the absolute file path where the completion script was written.
func installCompletion(app *App, shell string) (string, error) {
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
