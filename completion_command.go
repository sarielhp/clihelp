package clihelp

import (
	"fmt"
	"io"
)

// The optional "completion" command, and the report both it and the __clihelp
// verbs print. This is presentation over the installers, which is why it is not
// in completion.go beside the generators: keeping them together made the
// generator file depend on the installer and the installer depend back.

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
			completionWrapSubcommand(),
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

// completionWrapSubcommand generates a wrapper script for this application.
func completionWrapSubcommand() Command {
	var fromPath string
	return Command{
		Name:        "wrap",
		Description: "Generate a wrapper script with preset arguments",
		UsageLine:   "completion wrap [--from <path>] <name> [<args>...]",
		Examples: []Example{
			{Line: "completion wrap pd deploy", Description: "Generate wrapper 'pd' for '<app> deploy'"},
			{Line: "completion wrap --from ~/bin/mt", Description: "Inspect existing script and generate wrapper"},
		},
		Parameters: []Param{
			{Name: "<name>", Description: "Name of the wrapper script"},
			{Name: "[<args>...]", Description: "Preset arguments prepended to wrapped command"},
		},
		Options: []Option{
			String(&fromPath, "--from <path>", "", "Inspect an existing wrapper script to extract preset arguments"),
		},
		Args: MinimumNArgs(0),
		Run: func(ctx *Context) error {
			return executeWrapperGen(ctx.App, fromPath, ctx.Args, ctx.Stdout, ctx.Stderr)
		},
	}
}

// reportInstall names every file the installation touched.
func reportInstall(w io.Writer, app *App, res installResult) {
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

func reportUninstall(w io.Writer, res installResult) {
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
