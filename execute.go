package clihelp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

// PrintError prints a formatted error message to os.Stderr with colored prefix.
func PrintError(err error) {
	if err == nil {
		return
	}
	errColor := color.New(color.FgRed, color.Bold)
	errColor.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintln(os.Stderr, err.Error())
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
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v" || args[0] == "version") {
		if !a.hasCommandNamed("version") {
			if a.Version != "" {
				if a.Name != "" {
					fmt.Fprintf(a.stdout(), "%s %s\n", a.Name, a.Version)
				} else {
					fmt.Fprintln(a.stdout(), a.Version)
				}
			}
			return nil
		}
	}

	// 3. Resolve command hierarchy and separate command path from remaining args
	targetCmd, ancestors, path, remaining, err := a.resolveCommand(args)
	if err != nil {
		return err
	}

	// If resolveCommand handled the "help" subcommand, return early to avoid double render
	if len(args) > 0 && args[0] == "help" && !a.hasCommandNamed("help") {
		return nil
	}

	// If resolved to help subcommand
	if targetCmd == nil && len(path) == 0 && len(remaining) == 0 && a.Run == nil {
		a.RenderGlobal(Options{Writer: a.stdout()})
		return nil
	}

	// 4. Gather persistent + local options and bind to pflag.FlagSet
	cmdName := a.Name
	if targetCmd != nil {
		cmdName = targetCmd.Name
	}

	fs := pflag.NewFlagSet(cmdName, pflag.ContinueOnError)
	fs.SetOutput(a.stderr())

	// Add help flag if not explicitly defined
	var helpRequested bool
	if fs.Lookup("help") == nil {
		fs.BoolVarP(&helpRequested, "help", "h", false, "Help for "+cmdName)
		_ = fs.MarkHidden("help")
	}

	// Bind App PersistentOptions
	for _, opt := range a.PersistentOptions {
		if opt.Binder != nil {
			if err := opt.Binder(fs); err != nil {
				return err
			}
		}
	}

	// Bind Ancestors' PersistentOptions
	for _, anc := range ancestors {
		for _, opt := range anc.PersistentOptions {
			if opt.Binder != nil {
				if err := opt.Binder(fs); err != nil {
					return err
				}
			}
		}
	}

	// Bind Target Command PersistentOptions & Options
	if targetCmd != nil {
		for _, opt := range targetCmd.PersistentOptions {
			if opt.Binder != nil {
				if err := opt.Binder(fs); err != nil {
					return err
				}
			}
		}
		for _, opt := range targetCmd.Options {
			if opt.Binder != nil {
				if err := opt.Binder(fs); err != nil {
					return err
				}
			}
		}
	}

	// Parse flags
	if err := fs.Parse(remaining); err != nil {
		return err
	}

	// Render help if -h / --help was passed
	if helpRequested {
		if len(path) == 0 {
			a.RenderGlobal(Options{Writer: a.stdout()})
		} else {
			a.RenderCommand(Options{Writer: a.stdout()}, path...)
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
		if len(path) == 0 {
			a.RenderGlobal(Options{Writer: a.stdout()})
		} else {
			a.RenderCommand(Options{Writer: a.stdout()}, path...)
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
	for _, c := range a.Commands {
		if c.Name == name {
			return true
		}
	}
	return false
}

func hasSubcommandNamed(cmds []Command, name string) bool {
	for _, c := range cmds {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (a *App) resolveCommand(args []string) (*Command, []*Command, []string, []string, error) {
	currentCommands := a.Commands
	var currentCmd *Command
	var ancestors []*Command
	var path []string
	idx := 0

	for idx < len(args) {
		arg := args[idx]

		// If help subcommand is passed: e.g. "app help scan"
		if arg == "help" && !a.hasCommandNamed("help") && !hasSubcommandNamed(currentCommands, "help") {
			helpPath := append(path, args[idx+1:]...)
			if len(helpPath) == 0 {
				a.RenderGlobal(Options{Writer: a.stdout()})
			} else {
				a.RenderCommand(Options{Writer: a.stdout()}, helpPath...)
			}
			return nil, nil, nil, nil, nil
		}

		// Flags denote end of command tree traversal
		if strings.HasPrefix(arg, "-") {
			break
		}

		// Look for matching command or alias
		var matched *Command
		for i := range currentCommands {
			if currentCommands[i].Name == arg {
				matched = &currentCommands[i]
				break
			}
			for _, alias := range currentCommands[i].Aliases {
				if alias == arg {
					matched = &currentCommands[i]
					break
				}
			}
			if matched != nil {
				break
			}
		}

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

		// Unknown command token: if current node has subcommands and no Run handler, treat as typo/error
		if len(currentCommands) > 0 && (currentCmd == nil || currentCmd.Run == nil) {
			parentName := a.Name
			if currentCmd != nil {
				parentName = currentCmd.Name
			}
			suggestion := suggestCommand(arg, currentCommands)
			if suggestion != "" {
				return nil, nil, nil, nil, fmt.Errorf("unknown command %q for %q. Did you mean %q?", arg, parentName, suggestion)
			}
			return nil, nil, nil, nil, fmt.Errorf("unknown command %q for %q", arg, parentName)
		}

		// Otherwise positional argument for current command
		break
	}

	return currentCmd, ancestors, path, args[idx:], nil
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
