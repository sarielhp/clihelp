package clihelp

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

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

// isHelpToken reports whether arg asks for help at this level rather than naming
// something.
//
// "help" is universal wherever it can be: a node with subcommands can always be
// asked to explain itself, which is the git-like convention. Its abbreviations
// are not, and neither is the bare "h" — a node with no subcommands takes
// positional arguments, and swallowing them there made the word unreachable. A
// leaf command could not be handed "help", "hel" or "h" at all, with no escape,
// because this runs before the "--" branch.
//
// The old shape also reserved "h" unconditionally, ahead of the guard two lines
// below, so with AbbrevCommands on and a command "hello" the shorter
// abbreviation rendered help while the longer one ran the command.
func isHelpToken(arg string, cmds []Command, abbrev bool, atRoot bool) bool {
	if arg == "" {
		return false // an empty argument is a prefix of everything, and names nothing
	}
	cmd, _ := findCommand(cmds, arg)
	if cmd != nil {
		return false
	}
	// "help" itself is universal wherever there is something to explain: the
	// application's own page at the root, and a group's page below it.
	if arg == "help" && (atRoot || len(cmds) > 0) {
		return true
	}
	if !atRoot && len(cmds) == 0 {
		// A leaf takes positional arguments and has nothing below it to
		// document, so the word belongs to the command. "app echo --help" and
		// "app help echo" both still work; swallowing the argument bought a
		// third way to reach the same page at the price of a word an echo- or
		// grep-shaped command could never be given.
		return false
	}
	if len(filterCommandsByPrefix(cmds, arg)) > 0 {
		return false // it names something here, and that is more specific
	}
	return arg == "h" || (abbrev && strings.HasPrefix("help", arg))
}

func (a *App) lookupCommandPath(path []string) (*Command, []string) {
	if len(path) == 0 {
		return nil, nil
	}
	currentSlice := a.Commands
	var found *Command
	var resolvedPath []string
	for _, p := range path {
		// The same resolver execution uses, so that "help X" and "X" can never
		// name different commands.
		cmd, err := a.matchCommandOrShortcut(currentSlice, p, found == nil)
		if err != nil || cmd == nil {
			return nil, nil
		}
		found = cmd
		resolvedPath = append(resolvedPath, cmd.Name)
		currentSlice = cmd.Subcommands
	}
	return found, resolvedPath
}

// isExamplesTopic reports whether topic names the examples page.
func (a *App) isExamplesTopic(topic string) bool {
	return topic == "examples" || topic == "example" || topic == "eg" ||
		(a.AbbrevCommands && topic != "" && strings.HasPrefix("examples", topic))
}

func (a *App) handleRootHelpTopic(topic string) (bool, error) {
	if topic == "" {
		// Every topic below is matched by prefix, and "" is a prefix of all of
		// them, so an unset shell variable used to render the flags reference —
		// but only when AbbrevCommands happened to be on.
		return false, fmt.Errorf("unknown help topic %q", topic)
	}
	switch {
	case topic == "flags" || topic == "options" || topic == "opts" || topic == "flag" || (a.AbbrevCommands && (strings.HasPrefix("flags", topic) || strings.HasPrefix("options", topic))):
		a.renderFlagsPage(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case topic == "man" || topic == "all" || topic == "full" || topic == "manual" || (a.AbbrevCommands && (strings.HasPrefix("man", topic) || strings.HasPrefix("manual", topic))):
		a.renderManPage(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case a.isExamplesTopic(topic):
		a.renderExamplesPage(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return true, nil
	case topic == "topics" || topic == "help" || (a.AbbrevCommands && strings.HasPrefix("topics", topic)):
		a.renderTopicsPage(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
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
	// "help examples <command>" narrows the page to one command and its
	// subcommands. A command of the author's own named "examples" still wins,
	// because the lookup above ran first.
	if a.isExamplesTopic(helpPath[0]) {
		a.renderExamplesPage(o, helpPath[1:]...)
		return true, nil
	}
	if len(helpPath) == 1 {
		return a.handleRootHelpTopic(helpPath[0])
	}
	// Naming only the first element pointed at whichever word came first, even
	// when it resolved and a later one did not.
	return false, fmt.Errorf("unknown help topic %q", strings.Join(helpPath, " "))
}

func matchAbbrevCommand(currentCommands []Command, arg string) (*Command, error) {
	if arg == "" {
		return nil, nil // every name starts with it, so it abbreviates nothing
	}
	matches := filterCommandsByPrefix(currentCommands, arg)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		// Name the spelling the prefix matched. Listing cmd.Name for a command
		// matched through an alias printed a name the user's own prefix does
		// not match, which reads as a non sequitur.
		names := make([]string, len(matches))
		for i, cmd := range matches {
			names[i] = matchedSpelling(cmd, arg)
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

// findSubcommandPaths names the deep commands called target, for the "did you
// mean" message.
//
// Hidden ancestry counts. The check used to be on the visited node alone, so a
// visible subcommand under a hidden parent was offered with the hidden parent's
// name spelled out — which is the exact invocation, for the commands authors
// hide precisely because they are deprecated, internal or destructive. Walk is
// pre-order, so a hidden parent is always recorded before its children.
// matchedSpelling is the name or alias of cmd that arg is a prefix of, for a
// message about arg. The command's own name is the fallback.
func matchedSpelling(cmd *Command, arg string) string {
	if strings.HasPrefix(cmd.Name, arg) {
		return cmd.Name
	}
	for _, alias := range cmd.Aliases {
		if strings.HasPrefix(alias, arg) {
			return fmt.Sprintf("%s (%s)", alias, cmd.Name)
		}
	}
	return cmd.Name
}

func (a *App) findSubcommandPaths(target string) []string {
	var matches []string
	var hidden []string
	_ = a.Walk(func(path []string, cmd *Command) error {
		joined := strings.Join(path, " ")
		for _, prefix := range hidden {
			if strings.HasPrefix(joined, prefix+" ") {
				return nil // an ancestor is hidden, so this is not reachable either
			}
		}
		if cmd.Hidden {
			hidden = append(hidden, joined)
			return nil
		}
		if len(path) <= 1 {
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

// takesPositionals reports whether the enclosing command can accept arg as one
// of its own positional arguments, in which case calling it an unknown command
// would be wrong: "demo scan inbox" is scan's argument, not a misspelled
// subcommand.
//
// The question used to be answered by "does it have a handler", which is a
// weaker thing to ask. A command declares what it takes — Args: NoArgs is the
// commonest validator in the applications built on this library — and that
// declaration was read nowhere. So a command with a handler and Args: NoArgs
// reported "unknown arguments: [bogus]" where it had everything it needed to
// say "unknown command "bogus". Did you mean …". At the root it was worse:
// App had no Args field at all, so handling a bare invocation and rejecting
// typos could not be asked for separately.
func takesPositionals(a *App, currentCmd *Command) bool {
	if currentCmd != nil {
		if currentCmd.Run == nil {
			return false // a grouping command: the word can only be a subcommand
		}
		return acceptsPositionals(currentCmd.Args)
	}
	if a.Run == nil {
		return false // nothing would receive them
	}
	if a.Args != nil {
		return acceptsPositionals(a.Args)
	}
	// Undeclared at the root: an application with commands is taken to have no
	// positional arguments of its own. Reaching here at all means the word
	// matched no command, and a word that is not one of an application's
	// commands is far likelier to be a typo than an argument.
	return false
}

// acceptsPositionals reports whether a validator admits any positional argument
// at all. An undeclared or unknowable arity is assumed to, which is what the
// library did before it could ask.
func acceptsPositionals(v ArgsValidator) bool {
	if v == nil {
		return true
	}
	_, max := v.Arity()
	return max != 0
}

func (a *App) checkUnknownCommand(currentCmd *Command, path []string, currentCommands []Command, arg string) error {
	// Shortcuts are top-level commands too. Leaving them out meant an
	// application whose verbs all live in Shortcuts had no commands at this
	// level by this test's reckoning, so the check never ran at all: any typo
	// printed the global help and exited 0. The copy matters — appending to the
	// caller's slice could write into App.Commands' spare capacity.
	if currentCmd == nil && len(a.Shortcuts) > 0 {
		currentCommands = append(append([]Command{}, currentCommands...), a.Shortcuts...)
	}
	if len(currentCommands) > 0 && !takesPositionals(a, currentCmd) {
		parentName := appName(a)
		if currentCmd != nil {
			parentName = currentCmd.Name
		}
		if suggestions := a.findSubcommandPaths(arg); len(suggestions) > 0 {
			return formatSubcommandSuggestions(arg, parentName, suggestions)
		}
		if suggestion := suggestCommand(arg, currentCommands); suggestion != "" {
			return fmt.Errorf("unknown command %q for %q. Did you mean %q?", arg, parentName, suggestion)
		}
		// With nothing close enough to suggest, the user is left holding a
		// rejection and no way forward. A real application built on this library
		// added that sentence itself; it belongs here.
		return fmt.Errorf("unknown command %q for %q. Run %q for a list of commands",
			arg, parentName, strings.Join(append([]string{appName(a)}, path...), " ")+" -h")
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
		primaryLong, _ := spec.primaryNames()
		if primaryLong == "" && len(spec.shortNames) > 0 {
			arity["--flag-"+spec.shortNames[0]] = takesValue
		}
		// bindToggle derives a negative spelling for the toggle's base name and
		// for every positive long name, while parseFlagSpec reports only the
		// names the spec string spells out. So "--cache" bound a "--no-cache"
		// nothing here named, and resolution stopped at it: the command after it
		// became a positional and the program printed its help and exited 0.
		if opt.toggle {
			base := spec.toggleBase()
			if base != "" {
				primaryLong = base
			}
			// The base name is negated whether or not the spec spelled it out:
			// toggleBase falls back to the first long name, but bindToggle binds
			// "--no-"+base regardless, and a name bound but not named here is a
			// name resolution stops at.
			for _, long := range append([]string{base}, spec.longNames...) {
				if long != "" && !strings.HasPrefix(long, "no-") {
					arity["--no-"+long] = takesValue
				}
			}
		}
		// Each shorthand after the first is registered under a synthetic long
		// name, because pflag offers no way to attach a second shorthand.
		for i := 1; i < len(spec.shortNames); i++ {
			arity["--"+aliasFlagName(primaryLong, spec.shortNames[i])] = takesValue
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
	addFlagArity(arity, a.Options)
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

// matchCommandOrShortcut matches arg against the commands available at this
// depth and, at the root, against App.Shortcuts as well. Shortcuts are top-level
// commands shown under their own heading.
//
// The order is exact before abbreviated, and commands before shortcuts within
// each. It has to be one order, used by every caller: this function and
// lookupCommandPath used to try the four strategies in different sequences, so
// "app dep" ran the command "deploy" while "app help dep" documented the
// shortcut "dep" — the page a user reads described something other than what
// runs.
//
// An ambiguity among real commands is not a shortcut's to break. That fell out
// of the old shape: the error was computed and then discarded whenever a
// shortcut happened to share the prefix, so with commands "deploy" and
// "destroy" and a shortcut "dance", typing "d" ran "dance".
func (a *App) matchCommandOrShortcut(cmds []Command, arg string, atRoot bool) (*Command, error) {
	shortcuts := atRoot && len(a.Shortcuts) > 0

	if exact, _ := findCommand(cmds, arg); exact != nil {
		return exact, nil
	}
	if shortcuts {
		if exact, _ := findCommand(a.Shortcuts, arg); exact != nil {
			return exact, nil
		}
	}
	if !a.AbbrevCommands {
		return nil, nil
	}
	matched, err := matchAbbrevCommand(cmds, arg)
	if matched != nil || err != nil || !shortcuts {
		return matched, err
	}
	return matchAbbrevCommand(a.Shortcuts, arg)
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
		if isHelpToken(arg, currentCommands, a.AbbrevCommands, res.cmd == nil) {
			return resolution{
				isHelp: true,
				helpPath: append(append([]string{}, res.path...),
					helpTopicWords(probe, args[idx+1:], len(res.path) == 0)...),
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

		matched, err := a.matchCommandOrShortcut(currentCommands, arg, res.cmd == nil)
		if err != nil {
			return res, err
		}
		if matched == nil {
			if err := a.checkUnknownCommand(res.cmd, res.path, currentCommands, arg); err != nil {
				return res, err
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

// helpTopicWords keeps the words of a help invocation that can name a command or
// a topic, dropping flags, their values and the "--" terminator.
//
// The raw tail used to become the path, so a flag written after "help" turned
// the whole thing into "unknown help topic" while the same flag before "help"
// worked — and the order is the user's arbitrary choice. At the root a lone
// token is kept exactly as written, because "-v" and "--version" are themselves
// help topics.
func helpTopicWords(arity map[string]bool, args []string, atRoot bool) []string {
	if atRoot && len(args) == 1 {
		return args
	}
	var words []string
	for i := 0; i < len(args); {
		switch {
		case args[i] == "--":
			i++
		case len(args[i]) > 1 && strings.HasPrefix(args[i], "-"):
			if count, known := scanLeadingFlag(arity, args[i:]); known {
				i += count
				continue
			}
			i++
		default:
			words = append(words, args[i])
			i++
		}
	}
	return words
}

func (a *App) resolveCommand(args []string) (resolution, error) {
	currentCommands := a.Commands
	return a.resolveCommandPath(args, currentCommands)
}

func suggestCommand(typed string, available []Command) string {
	if typed == "" {
		return "" // every short name is within edit distance of nothing
	}
	bestDist := 3
	bestName := ""
	typedRunes := utf8.RuneCountInString(typed)

	// levenshtein(a, b) is at least the difference in their lengths, and only a
	// distance below bestDist can win — so a candidate whose length differs by
	// bestDist or more is decided before the matrix is built. The check is exact
	// and cannot change an answer; without it a mistyped argument was walked in
	// full once per command and once per alias, and that argument comes from
	// argv. A 128 KB word — the most a single argument can be — cost a quarter
	// of a second and 75 MB, on every press of Tab through __complete.
	//
	// Runes, not bytes: 日本語 and 日本 are three bytes apart and one edit apart.
	near := func(name string) bool {
		d := utf8.RuneCountInString(name) - typedRunes
		if d < 0 {
			d = -d
		}
		return d < bestDist
	}

	for _, cmd := range available {
		if cmd.Hidden {
			continue // a hidden command is not one to suggest
		}
		if near(cmd.Name) {
			if d := levenshtein(typed, cmd.Name); d < bestDist {
				bestDist, bestName = d, cmd.Name
			}
		}
		for _, alias := range cmd.Aliases {
			if !near(alias) {
				continue
			}
			if da := levenshtein(typed, alias); da < bestDist {
				bestDist, bestName = da, cmd.Name
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
