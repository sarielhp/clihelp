package clihelp

import (
	"context"
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
		// A command that only groups subcommands prints its help instead of
		// running, but PostRun and AfterRun still owe their BeforeRun and PreRun
		// whatever those acquired, so this path falls through rather than
		// returning here.
		o := Options{Writer: a.stdout(), Theme: a.Theme, Pager: a.Pager}
		if len(path) == 0 {
			a.RenderGlobal(o)
		} else {
			a.RenderCommand(o, path...)
		}
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
	if len(args) > 0 && args[0] == "__explain" {
		return a.handleExplain(args[1:])
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
