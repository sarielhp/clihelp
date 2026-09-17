package clihelp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

// PrintError prints a formatted error message to the App's stderr with colored prefix.
func (a *App) PrintError(err error) {
	if err == nil {
		return
	}
	errColor := color.New(color.FgRed, color.Bold)
	errColor.Fprintf(a.stderr(), "Error: ")
	fmt.Fprintln(a.stderr(), err.Error())
}

// Execute runs the application using os.Args[1:] and context.Background().
func (a *App) Execute(args []string) error {
	return a.ExecuteContext(context.Background(), args)
}

func (a *App) checkTopLevelVersion(args []string) (bool, error) {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		if !a.hasCommandNamed("version") {
			if a.Version == "" {
				return false, fmt.Errorf("%s: no version is set for this application", appName(a))
			}
			if a.Name != "" {
				fmt.Fprintf(a.stdout(), "%s %s\n", a.Name, a.Version)
			} else {
				fmt.Fprintln(a.stdout(), a.Version)
			}
			return true, nil
		}
	}
	return false, nil
}

type helpFlags struct {
	concise  bool
	extended bool
}

func (h *helpFlags) requested() bool {
	return h.concise || h.extended
}

// helpFlagNames lists the built-in help flags as they are written on the command
// line. bindHelpFlags registers exactly these, and leadingFlagArity recognizes
// exactly these, so the two stay in step.
func (a *App) helpFlagNames() []string {
	names := []string{"-h", "--help-concise", "--help"}
	if a.ExtendedHelpFlag {
		names = append(names, "-H")
	}
	return names
}

// bindHelpFlags registers the built-in help flags on fs and returns the targets
// they write to.
func (a *App) bindHelpFlags(fs *pflag.FlagSet, cmdName string) *helpFlags {
	var h helpFlags
	fs.BoolVarP(&h.concise, "help-concise", "h", false, "Concise help for "+cmdName)
	_ = fs.MarkHidden("help-concise")

	if a.ExtendedHelpFlag {
		fs.BoolVarP(&h.extended, "help", "H", false, "Extended help for "+cmdName)
	} else {
		fs.BoolVar(&h.extended, "help", false, "Extended help for "+cmdName)
	}
	_ = fs.MarkHidden("help")
	return &h
}

func (a *App) setupFlagSet(targetCmd *Command, ancestors []*Command) (*pflag.FlagSet, *helpFlags, error) {
	cmdName := a.Name
	if targetCmd != nil {
		cmdName = targetCmd.Name
	}

	fs := pflag.NewFlagSet(cmdName, pflag.ContinueOnError)
	fs.SetOutput(a.stderr())

	h := a.bindHelpFlags(fs, cmdName)

	if err := bindAndMark(fs, a.PersistentOptions); err != nil {
		return nil, nil, err
	}
	if err := bindAndMark(fs, a.GlobalFlags); err != nil {
		return nil, nil, err
	}
	for _, anc := range ancestors {
		if err := bindAndMark(fs, anc.PersistentOptions); err != nil {
			return nil, nil, err
		}
	}
	if targetCmd != nil {
		if err := bindAndMark(fs, targetCmd.PersistentOptions); err != nil {
			return nil, nil, err
		}
		if err := bindAndMark(fs, targetCmd.Options); err != nil {
			return nil, nil, err
		}
	}
	return fs, h, nil
}

func (a *App) collectAllActiveOptions(targetCmd *Command, ancestors []*Command) []Option {
	var allOptions []Option
	allOptions = append(allOptions, a.PersistentOptions...)
	allOptions = append(allOptions, a.GlobalFlags...)
	for _, anc := range ancestors {
		allOptions = append(allOptions, anc.PersistentOptions...)
	}
	if targetCmd != nil {
		allOptions = append(allOptions, targetCmd.PersistentOptions...)
		allOptions = append(allOptions, targetCmd.Options...)
	}
	return allOptions
}

func (a *App) validateParsedFlags(fs *pflag.FlagSet, allOptions []Option, targetCmd *Command, path []string) error {
	missing := getMissingRequiredFlags(fs, allOptions)
	prompted := false
	if len(missing) > 0 {
		isTTY := false
		if f, ok := a.stdout().(*os.File); ok && (int(f.Fd()) == 1 || int(f.Fd()) == 2) {
			isTTY = true
		}
		if a.Stdout != nil || a.Stderr != nil {
			isTTY = true
		}

		if a.InteractiveFallback && isTTY {
			if err := promptForMissing(fs, missing, a.stdin(), a.stderr()); err != nil {
				return err
			}
			prompted = true
		} else {
			var names []string
			for _, m := range missing {
				names = append(names, `"`+strings.TrimPrefix(m.Name, "flag-")+`"`)
			}
			return fmt.Errorf("required flag(s) %s not set", strings.Join(names, ", "))
		}
	}

	checkDeprecatedFlags(fs, allOptions, a.stderr())

	if targetCmd != nil && targetCmd.OptionsValidator != nil {
		if err := targetCmd.OptionsValidator(fs); err != nil {
			return err
		}
	}

	if prompted {
		cmdStr := constructCommand(a, path, fs, fs.Args())
		fmt.Fprintf(a.stderr(), "\n💡 Tip: Next time, you can run this directly with:\n   %s\n\n", cmdStr)
	}
	return nil
}

func (a *App) runLifecycle(ctx context.Context, targetCmd *Command, path []string, cmdArgs, args []string) error {
	cliCtx := &Context{
		Context: ctx,
		App:     a,
		Command: targetCmd,
		Args:    cmdArgs,
		RawArgs: args,
		Stdout:  a.stdout(),
		Stderr:  a.stderr(),
	}

	if a.BeforeRun != nil {
		if err := a.BeforeRun(cliCtx); err != nil {
			return err
		}
	}

	if targetCmd != nil && targetCmd.PreRun != nil {
		if err := targetCmd.PreRun(cliCtx); err != nil {
			return err
		}
	}

	if targetCmd != nil && targetCmd.Run != nil {
		if err := targetCmd.Run(cliCtx); err != nil {
			return err
		}
	} else if targetCmd == nil && a.Run != nil {
		if err := a.Run(cliCtx); err != nil {
			return err
		}
	} else {
		o := Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager}
		if len(path) == 0 {
			a.RenderGlobal(o)
		} else {
			a.RenderCommand(o, path...)
		}
		return nil
	}

	if targetCmd != nil && targetCmd.PostRun != nil {
		if err := targetCmd.PostRun(cliCtx); err != nil {
			return err
		}
	}

	if a.AfterRun != nil {
		if err := a.AfterRun(cliCtx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteContext runs the application using the given context and argument slice.
func (a *App) ExecuteContext(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "__complete" {
		return a.handleComplete(ctx, args[1:])
	}

	a.maybeAutoInstallCompletion(args)

	if handled, err := a.checkTopLevelVersion(args); handled || err != nil {
		return err
	}

	res, err := a.resolveCommand(args)
	if err != nil {
		return err
	}
	if res.isHelp {
		_, helpErr := a.handleHelpInvocation(res.helpPath)
		return helpErr
	}
	targetCmd, ancestors, path, remaining := res.cmd, res.ancestors, res.path, res.remaining

	if targetCmd == nil && len(path) == 0 && len(remaining) == 0 && a.Run == nil {
		a.RenderGlobal(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return nil
	}

	fs, helpFlags, err := a.setupFlagSet(targetCmd, ancestors)
	if err != nil {
		return err
	}

	if parseErr := fs.Parse(remaining); parseErr != nil {
		return parseErr
	}

	if helpFlags.requested() {
		isExtended := helpFlags.extended
		o := Options{
			Writer:   a.stdout(),
			Theme:    a.Theme,
			Pager:    a.Pager,
			Concise:  !isExtended,
			Extended: isExtended,
		}
		if len(path) == 0 {
			a.RenderGlobal(o)
		} else {
			a.RenderCommand(o, path...)
		}
		return nil
	}

	allOptions := a.collectAllActiveOptions(targetCmd, ancestors)
	if err := a.validateParsedFlags(fs, allOptions, targetCmd, path); err != nil {
		return err
	}

	if targetCmd != nil && targetCmd.Args != nil {
		if err := targetCmd.Args(fs.Args()); err != nil {
			return err
		}
	}

	return a.runLifecycle(ctx, targetCmd, path, fs.Args(), args)
}

func (a *App) hasCommandNamed(name string) bool {
	cmd, _ := findCommand(a.Commands, name)
	return cmd != nil
}

// filterCommandsByPrefix returns all non-hidden commands whose Name or any
// Alias starts with the given prefix.
func filterCommandsByPrefix(cmds []Command, prefix string) []*Command {
	var result []*Command
	for i := range cmds {
		cmd := &cmds[i]
		if cmd.Hidden {
			continue
		}
		if strings.HasPrefix(cmd.Name, prefix) {
			result = append(result, cmd)
			continue
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				result = append(result, cmd)
				break
			}
		}
	}
	return result
}

func isHelpToken(arg string, cmds []Command, abbrev bool) bool {
	cmd, _ := findCommand(cmds, arg)
	if cmd != nil {
		return false
	}
	if arg == "help" || arg == "h" {
		return true
	}
	if abbrev && strings.HasPrefix("help", arg) {
		if len(filterCommandsByPrefix(cmds, arg)) == 0 {
			return true
		}
	}
	return false
}

func (a *App) lookupCommandPath(path []string) (*Command, []string) {
	if len(path) == 0 {
		return nil, nil
	}
	currentSlice := a.Commands
	var found *Command
	var resolvedPath []string
	for idx, p := range path {
		cmd, _ := findCommand(currentSlice, p)
		if cmd == nil && idx == 0 && len(a.Shortcuts) > 0 {
			cmd, _ = findCommand(a.Shortcuts, p)
		}
		if cmd == nil && a.AbbrevCommands {
			matches := filterCommandsByPrefix(currentSlice, p)
			if len(matches) == 1 {
				cmd = matches[0]
			} else if idx == 0 && len(a.Shortcuts) > 0 {
				shortcutMatches := filterCommandsByPrefix(a.Shortcuts, p)
				if len(shortcutMatches) == 1 {
					cmd = shortcutMatches[0]
				}
			}
		}
		if cmd == nil {
			return nil, nil
		}
		found = cmd
		resolvedPath = append(resolvedPath, cmd.Name)
		currentSlice = cmd.Subcommands
	}
	return found, resolvedPath
}

func (a *App) handleRootHelpTopic(topic string) (bool, error) {
	switch {
	case topic == "flags" || topic == "options" || topic == "opts" || topic == "flag" || (a.AbbrevCommands && (strings.HasPrefix("flags", topic) || strings.HasPrefix("options", topic))):
		a.RenderFlags(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case topic == "man" || topic == "all" || topic == "full" || topic == "manual" || (a.AbbrevCommands && (strings.HasPrefix("man", topic) || strings.HasPrefix("manual", topic))):
		a.RenderMan(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case topic == "topics" || topic == "help" || (a.AbbrevCommands && strings.HasPrefix("topics", topic)):
		a.RenderHelpTopics(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case topic == "v" || topic == "-v" || topic == "version" || topic == "--version" || (a.AbbrevCommands && strings.HasPrefix("version", topic)):
		if a.Version == "" {
			return false, fmt.Errorf("%s: no version is set for this application", appName(a))
		}
		if a.Name != "" {
			fmt.Fprintf(a.stdout(), "%s %s\n", a.Name, a.Version)
		} else {
			fmt.Fprintln(a.stdout(), a.Version)
		}
		return true, nil
	case topic == "d" || topic == "-d" || topic == "docs" || topic == "doc" || topic == "more" || (a.AbbrevCommands && (strings.HasPrefix("docs", topic) || strings.HasPrefix("more", topic))):
		th := a.Theme
		if th == nil {
			t := defaultTheme()
			th = &t
		}
		o := Options{Writer: a.stdout(), Theme: th, Pager: a.Pager}
		if a.GlobalNote != "" {
			reflow(a.stdout(), th.Body, wrapWidth(o.width(), 0, o.maxContent()), 0, "", inline(a.GlobalNote))
		} else if a.Description != "" {
			reflow(a.stdout(), th.Body, wrapWidth(o.width(), 0, o.maxContent()), 0, "", inline(a.Description))
		} else {
			fmt.Fprintf(a.stdout(), "No extended documentation available for %s.\n", appName(a))
		}
		return true, nil
	}
	return false, fmt.Errorf("unknown help topic %q", topic)
}

func (a *App) handleHelpInvocation(helpPath []string) (bool, error) {
	o := Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager, Extended: true}
	if len(helpPath) == 0 {
		a.RenderGlobal(o)
		return true, nil
	}
	if helpCmd, resolvedPath := a.lookupCommandPath(helpPath); helpCmd != nil {
		a.RenderCommand(o, resolvedPath...)
		return true, nil
	}
	if len(helpPath) == 1 {
		return a.handleRootHelpTopic(helpPath[0])
	}
	return false, fmt.Errorf("unknown help topic %q", helpPath[0])
}

func matchAbbrevCommand(currentCommands []Command, arg string) (*Command, error) {
	matches := filterCommandsByPrefix(currentCommands, arg)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, cmd := range matches {
			names[i] = cmd.Name
		}
		var buf strings.Builder
		buf.WriteString(fmt.Sprintf("command %q is ambiguous. Did you mean one of these?\n", arg))
		for _, name := range names {
			buf.WriteString(fmt.Sprintf("  %s\n", name))
		}
		return nil, errors.New(buf.String())
	}
	return nil, nil
}

func (a *App) findSubcommandPaths(target string) []string {
	var matches []string
	_ = a.Walk(func(path []string, cmd *Command) error {
		if cmd.Hidden || len(path) <= 1 {
			return nil
		}
		if strings.EqualFold(cmd.Name, target) {
			matches = append(matches, strings.Join(path, " "))
			return nil
		}
		for _, alias := range cmd.Aliases {
			if strings.EqualFold(alias, target) {
				matches = append(matches, strings.Join(path, " "))
				return nil
			}
		}
		return nil
	})
	return matches
}

func formatSubcommandSuggestions(arg, parentName string, suggestions []string) error {
	if len(suggestions) == 1 {
		return fmt.Errorf("unknown command %q for %q. Did you mean %q?", arg, parentName, suggestions[0])
	}
	const maxDisplay = 5
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("unknown command %q for %q. Did you mean one of these?\n", arg, parentName))
	limit := len(suggestions)
	if limit > maxDisplay {
		limit = maxDisplay
	}
	for i := 0; i < limit; i++ {
		buf.WriteString(fmt.Sprintf("  %s\n", suggestions[i]))
	}
	if len(suggestions) > maxDisplay {
		buf.WriteString(fmt.Sprintf("  (... and %d more)\n", len(suggestions)-maxDisplay))
	}
	return errors.New(strings.TrimRight(buf.String(), "\n"))
}

func (a *App) checkUnknownCommand(currentCmd *Command, currentCommands []Command, arg string) error {
	if len(currentCommands) > 0 && ((currentCmd == nil && a.Run == nil) || (currentCmd != nil && currentCmd.Run == nil)) {
		parentName := a.Name
		if currentCmd != nil {
			parentName = currentCmd.Name
		}
		if suggestions := a.findSubcommandPaths(arg); len(suggestions) > 0 {
			return formatSubcommandSuggestions(arg, parentName, suggestions)
		}
		if suggestion := suggestCommand(arg, currentCommands); suggestion != "" {
			return fmt.Errorf("unknown command %q for %q. Did you mean %q?", arg, parentName, suggestion)
		}
		return fmt.Errorf("unknown command %q for %q", arg, parentName)
	}
	return nil
}

// resolution is the outcome of matching an argument list against the command
// tree.
type resolution struct {
	cmd       *Command   // deepest command matched, nil when none was
	ancestors []*Command // commands matched above cmd, outermost first
	path      []string   // names of the matched commands, in order
	indices   []int      // position in args of each matched command token
	remaining []string   // arguments left for pflag: skipped flags, then the rest
	isHelp    bool       // the arguments name a help invocation, not a command
	helpPath  []string   // for a help invocation, what help was asked for
}

// flagTakesValue reports whether opt consumes the argument after it. The typed
// constructors record this; for an Option assembled by hand the flag spec is the
// only evidence, and a placeholder is what distinguishes "--out <file>" from a
// switch.
func flagTakesValue(opt Option, spec flagSpec) bool {
	switch opt.arity {
	case arityValue:
		return true
	case arityFlag:
		return false
	}
	return spec.placeholder != "" && !spec.isToggle
}

// addFlagArity records every name under which opts can be written, mirroring the
// names bindHelper registers.
func addFlagArity(arity map[string]bool, opts []Option) {
	for _, opt := range opts {
		spec := parseFlagSpec(opt.Flags)
		takesValue := flagTakesValue(opt, spec)
		for _, long := range spec.longNames {
			arity["--"+long] = takesValue
		}
		for _, short := range spec.shortNames {
			arity["-"+short] = takesValue
		}
		if len(spec.longNames) == 0 && len(spec.shortNames) > 0 {
			arity["--flag-"+spec.shortNames[0]] = takesValue
		}
	}
}

// leadingFlagArity maps every flag that may legally appear before the next
// command name to whether it consumes the argument after it: the app's
// persistent and global flags, the built-in help flags, and the persistent flags
// of the commands resolved so far. A command's own options are deliberately
// absent, since they cannot precede the command they belong to.
//
// The map is derived from the flag specs rather than from a bound pflag.FlagSet,
// because binding writes each declared default through the consumer's own
// pointer: a probe built by binding reset a running program's own variables
// every time its arguments were resolved.
func (a *App) leadingFlagArity(resolved []*Command) map[string]bool {
	arity := make(map[string]bool)
	for _, name := range a.helpFlagNames() {
		arity[name] = false
	}
	addFlagArity(arity, a.PersistentOptions)
	addFlagArity(arity, a.GlobalFlags)
	for _, cmd := range resolved {
		addFlagArity(arity, cmd.PersistentOptions)
	}
	return arity
}

// scanLeadingFlag reports how many arguments the flag at args[0] occupies and
// whether it is a flag known to fs. It mirrors pflag's own parsing rules so that
// command resolution skips exactly what pflag will later consume: "--flag value"
// and "-f value" take two arguments, while "--flag=value", "-fvalue", "-f=value"
// and flags that take no value take one. A flag whose value is missing entirely
// is reported as one argument, leaving pflag to produce the error.
func scanLeadingFlag(arity map[string]bool, args []string) (int, bool) {
	arg := args[0]
	if len(arg) < 2 {
		return 0, false // a bare "-" is a positional argument
	}
	if !strings.HasPrefix(arg, "--") {
		return scanShorthandCluster(arity, args)
	}
	name := arg[2:]
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		_, known := arity["--"+name[:eq]]
		return 1, known
	}
	takesValue, known := arity["--"+name] // an empty name, from "--", is never known
	if !known {
		return 0, false
	}
	if !takesValue || len(args) == 1 {
		return 1, true
	}
	return 2, true
}

// scanShorthandCluster scans a run of shorthand flags such as -v, -vAx or -A=x.
// Valueless flags are consumed one letter at a time; the first flag that takes a
// value ends the cluster, taking the rest of the token or the next argument.
func scanShorthandCluster(arity map[string]bool, args []string) (int, bool) {
	for shorts := args[0][1:]; shorts != ""; shorts = shorts[1:] {
		takesValue, known := arity["-"+shorts[:1]]
		if !known {
			return 0, false
		}
		if len(shorts) > 2 && shorts[1] == '=' {
			return 1, true // -f=value
		}
		if !takesValue {
			continue // takes no value; the cluster may go on
		}
		if len(shorts) > 1 || len(args) == 1 {
			return 1, true // -fvalue, or a value that is missing
		}
		return 2, true // -f value
	}
	return 1, true
}

// matchCommand returns the command named by arg, using prefix matching when
// AbbrevCommands is enabled and an exact match fails. A nil command and a nil
// error mean arg names no command.
func (a *App) matchCommand(cmds []Command, arg string) (*Command, error) {
	if matched, _ := findCommand(cmds, arg); matched != nil {
		return matched, nil
	}
	if a.AbbrevCommands {
		return matchAbbrevCommand(cmds, arg)
	}
	return nil, nil
}

// resolveCommandPath resolves a path of command names, using prefix matching when
// AbbrevCommands is enabled and exact match fails. Global and persistent flags
// may be interleaved with the command names: a recognized flag (and its value)
// is skipped and handed back in remaining, so that "app --global cmd" resolves
// cmd just as "app cmd --global" does. Anything else beginning with "-",
// including "--" and an unrecognized flag, ends resolution.
func (a *App) resolveCommandPath(args []string, currentCommands []Command) (resolution, error) {
	var res resolution
	var leading []string
	var resolved []*Command
	probe := a.leadingFlagArity(nil)
	idx := 0

	for idx < len(args) {
		arg := args[idx]

		// Resolution only reports that help was asked for; rendering it here
		// would make every caller that merely resolves — completion, example
		// colorization, example validation — emit a help page as a side effect.
		if isHelpToken(arg, currentCommands, a.AbbrevCommands) {
			return resolution{
				isHelp:   true,
				helpPath: append(append([]string{}, res.path...), args[idx+1:]...),
			}, nil
		}

		if strings.HasPrefix(arg, "-") {
			count, known := scanLeadingFlag(probe, args[idx:])
			if !known {
				break
			}
			leading = append(leading, args[idx:idx+count]...)
			idx += count
			continue
		}

		matched, err := a.matchCommand(currentCommands, arg)
		if err != nil {
			return resolution{}, err
		}
		if matched == nil {
			if err := a.checkUnknownCommand(res.cmd, currentCommands, arg); err != nil {
				return resolution{}, err
			}
			break
		}

		if res.cmd != nil {
			res.ancestors = append(res.ancestors, res.cmd)
		}
		res.cmd = matched
		res.path = append(res.path, matched.Name)
		res.indices = append(res.indices, idx)
		currentCommands = matched.Subcommands
		resolved = append(resolved, matched)
		probe = a.leadingFlagArity(resolved)
		idx++
	}

	res.remaining = append(leading, args[idx:]...)
	return res, nil
}

func (a *App) resolveCommand(args []string) (resolution, error) {
	currentCommands := a.Commands
	return a.resolveCommandPath(args, currentCommands)
}

func suggestCommand(typed string, available []Command) string {
	bestDist := 3
	bestName := ""
	for _, cmd := range available {
		d := levenshtein(typed, cmd.Name)
		if d < bestDist {
			bestDist = d
			bestName = cmd.Name
		}
		for _, alias := range cmd.Aliases {
			da := levenshtein(typed, alias)
			if da < bestDist {
				bestDist = da
				bestName = cmd.Name
			}
		}
	}
	return bestName
}

func levenshtein(a, b string) int {
	aRunes := []rune(strings.ToLower(a))
	bRunes := []rune(strings.ToLower(b))
	la := len(aRunes)
	lb := len(bRunes)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	v0 := make([]int, lb+1)
	v1 := make([]int, lb+1)
	for i := 0; i <= lb; i++ {
		v0[i] = i
	}
	for i := 0; i < la; i++ {
		v1[0] = i + 1
		for j := 0; j < lb; j++ {
			cost := 0
			if aRunes[i] != bRunes[j] {
				cost = 1
			}
			v1[j+1] = min3(v1[j]+1, v0[j+1]+1, v0[j]+cost)
		}
		for j := 0; j <= lb; j++ {
			v0[j] = v1[j]
		}
	}
	return v0[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func bindAndMark(fs *pflag.FlagSet, opts []Option) error {
	for _, opt := range opts {
		if opt.Binder != nil {
			if err := opt.Binder(fs); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkDeprecatedFlags warns once per deprecated option that was used, under any
// of its spellings. The flag the user wrote may be a hidden alias, so the notice
// names the option's primary spelling.
func checkDeprecatedFlags(fs *pflag.FlagSet, opts []Option, stderr io.Writer) {
	warned := make(map[string]bool)
	fs.Visit(func(f *pflag.Flag) {
		group := optionGroup(f)
		if warned[group] {
			return
		}
		for _, opt := range opts {
			if opt.Deprecated == "" || parseFlagSpec(opt.Flags).primaryFlagName() != group {
				continue
			}
			warned[group] = true
			fmt.Fprintf(stderr, "Warning: flag --%s is deprecated: %s\n", strings.TrimPrefix(group, "flag-"), opt.Deprecated)
			return
		}
	})
}
