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

// ExecuteContext runs the application using the given context and argument slice.
func (a *App) ExecuteContext(ctx context.Context, args []string) error {
	// 1. Check for shell autocompletion protocol
	if len(args) > 0 && args[0] == "__complete" {
		return a.handleComplete(ctx, args[1:])
	}

	// 2. Check for top-level version flag
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		if !a.hasCommandNamed("version") {
			if a.Version == "" {
				return fmt.Errorf("%s: no version is set for this application", appName(a))
			}
			if a.Name != "" {
				fmt.Fprintf(a.stdout(), "%s %s\n", a.Name, a.Version)
			} else {
				fmt.Fprintln(a.stdout(), a.Version)
			}
			return nil
		}
	}

	// 3. Resolve command hierarchy and separate command path from remaining args
	targetCmd, ancestors, path, remaining, handled, err := a.resolveCommand(args)
	if err != nil {
		return err
	}

	// If resolveCommand handled the "help" subcommand, return early to avoid double render
	if handled {
		return nil
	}

	// If resolved to help subcommand
	if targetCmd == nil && len(path) == 0 && len(remaining) == 0 && a.Run == nil {
		a.RenderGlobal(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
		return nil
	}

	// 4. Gather persistent + local options and bind to pflag.FlagSet
	cmdName := a.Name
	if targetCmd != nil {
		cmdName = targetCmd.Name
	}

	fs := pflag.NewFlagSet(cmdName, pflag.ContinueOnError)
	fs.SetOutput(a.stderr())

	// Register the built-in help flag for -h/--help handling.
	var helpRequested bool
	fs.BoolVarP(&helpRequested, "help", "h", false, "Help for "+cmdName)
	_ = fs.MarkHidden("help")

	// Bind App PersistentOptions
	if err := bindAndMark(fs, a.PersistentOptions); err != nil {
		return err
	}

	// Bind App GlobalFlags
	if err := bindAndMark(fs, a.GlobalFlags); err != nil {
		return err
	}

	// Bind Ancestors' PersistentOptions
	for _, anc := range ancestors {
		if err := bindAndMark(fs, anc.PersistentOptions); err != nil {
			return err
		}
	}

	// Bind Target Command PersistentOptions & Options
	if targetCmd != nil {
		if err := bindAndMark(fs, targetCmd.PersistentOptions); err != nil {
			return err
		}
		if err := bindAndMark(fs, targetCmd.Options); err != nil {
			return err
		}
	}

	// Parse flags
	parseErr := fs.Parse(remaining)

	// Gather all options for validation, warnings, and fallbacks
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

	if parseErr != nil {
		return parseErr
	}

	// Check if any required options are missing
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

	// Render help if -h / --help was passed
	if helpRequested {
		o := Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager}
		if len(path) == 0 {
			a.RenderGlobal(o)
		} else {
			a.RenderCommand(o, path...)
		}
		return nil
	}

	// 5. Validate positional args
	cmdArgs := fs.Args()
	if targetCmd != nil && targetCmd.Args != nil {
		if err := targetCmd.Args(cmdArgs); err != nil {
			return err
		}
	}

	// 6. Lifecycle execution
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
		// Default to rendering help when command has no Run handler
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

// resolveCommandPath resolves a path of command names, using prefix matching when
// AbbrevCommands is enabled and exact match fails.
func (a *App) resolveCommandPath(args []string, currentCommands []Command) (*Command, []*Command, []string, []string, bool, error) {
	var currentCmd *Command
	var ancestors []*Command
	var path []string
	idx := 0

	for idx < len(args) {
		arg := args[idx]

		// If help subcommand is passed: e.g. "app help scan", "app h", "app h build", "app help tree"
		if isHelpToken(arg, currentCommands, a.AbbrevCommands) {
			helpPath := append(path, args[idx+1:]...)
			if len(helpPath) == 0 {
				a.RenderGlobal(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
				return nil, nil, nil, nil, true, nil
			}

			// If command exists at this path, render it
			if helpCmd, resolvedPath := a.lookupCommandPath(helpPath); helpCmd != nil {
				a.RenderCommand(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager}, resolvedPath...)
				return nil, nil, nil, nil, true, nil
			}

			// Built-in root help topics
			if len(helpPath) == 1 {
				topic := helpPath[0]
				switch {
				case topic == "t" || topic == "-t" || topic == "tree" || topic == "--tree" || (a.AbbrevCommands && strings.HasPrefix("tree", topic)):
					a.RenderTree(Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager})
					return nil, nil, nil, nil, true, nil
				case topic == "v" || topic == "-v" || topic == "version" || topic == "--version" || (a.AbbrevCommands && strings.HasPrefix("version", topic)):
					if a.Version == "" {
						return nil, nil, nil, nil, false, fmt.Errorf("%s: no version is set for this application", appName(a))
					}
					if a.Name != "" {
						fmt.Fprintf(a.stdout(), "%s %s\n", a.Name, a.Version)
					} else {
						fmt.Fprintln(a.stdout(), a.Version)
					}
					return nil, nil, nil, nil, true, nil
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
					return nil, nil, nil, nil, true, nil
				}
			}

			return nil, nil, nil, nil, false, fmt.Errorf("unknown help topic %q", helpPath[0])
		}

		// Flags denote end of command tree traversal
		if strings.HasPrefix(arg, "-") {
			break
		}

		// Look for matching command or alias
		matched, _ := findCommand(currentCommands, arg)

		if matched != nil {
			if currentCmd != nil {
				ancestors = append(ancestors, currentCmd)
			}
			currentCmd = matched
			path = append(path, matched.Name)
			currentCommands = matched.Subcommands
			idx++
			continue
		}

		// If exact match failed and abbreviation is enabled, try prefix matching
		if a.AbbrevCommands {
			matches := filterCommandsByPrefix(currentCommands, arg)
			if len(matches) == 1 {
				// Unique prefix match
				if currentCmd != nil {
					ancestors = append(ancestors, currentCmd)
				}
				currentCmd = matches[0]
				path = append(path, matches[0].Name)
				currentCommands = matches[0].Subcommands
				idx++
				continue
			} else if len(matches) > 1 {
				// Ambiguous prefix - show all candidates
				names := make([]string, len(matches))
				for i, cmd := range matches {
					names[i] = cmd.Name
				}
				var buf strings.Builder
				buf.WriteString(fmt.Sprintf("command %q is ambiguous. Did you mean one of these?\n", arg))
				for _, name := range names {
					buf.WriteString(fmt.Sprintf("  %s\n", name))
				}
				return nil, nil, nil, nil, false, errors.New(buf.String())
			}
		}

		// Unknown command token: if the receiving node has subcommands and no
		// Run handler (root uses a.Run), treat as typo/error. Otherwise it is
		// a positional argument.
		if len(currentCommands) > 0 && ((currentCmd == nil && a.Run == nil) || (currentCmd != nil && currentCmd.Run == nil)) {
			parentName := a.Name
			if currentCmd != nil {
				parentName = currentCmd.Name
			}
			suggestion := suggestCommand(arg, currentCommands)
			if suggestion != "" {
				return nil, nil, nil, nil, false, fmt.Errorf("unknown command %q for %q. Did you mean %q?", arg, parentName, suggestion)
			}
			return nil, nil, nil, nil, false, fmt.Errorf("unknown command %q for %q", arg, parentName)
		}

		// Otherwise positional argument for current command
		break
	}

	return currentCmd, ancestors, path, args[idx:], false, nil
}

func (a *App) resolveCommand(args []string) (*Command, []*Command, []string, []string, bool, error) {
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

func checkDeprecatedFlags(fs *pflag.FlagSet, opts []Option, stderr io.Writer) {
	fs.Visit(func(f *pflag.Flag) {
		for _, opt := range opts {
			if opt.Deprecated == "" {
				continue
			}
			spec := parseFlagSpec(opt.Flags)
			matched := false
			for _, l := range spec.longNames {
				if l == f.Name {
					matched = true
					break
				}
			}
			if !matched && len(spec.longNames) == 0 && len(spec.shortNames) > 0 {
				if f.Name == "flag-"+spec.shortNames[0] {
					matched = true
				}
			}
			for i := 1; i < len(spec.shortNames); i++ {
				aliasLong := fmt.Sprintf("%s-alias-%s", spec.longNames[0], spec.shortNames[i])
				if f.Name == aliasLong {
					matched = true
					break
				}
			}
			if matched {
				fmt.Fprintf(stderr, "Warning: flag --%s is deprecated: %s\n", f.Name, opt.Deprecated)
			}
		}
	})
}
