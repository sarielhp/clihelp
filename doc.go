// Package clihelp provides a declarative, lightweight CLI application framework
// and width-aware, colorized help text formatter for Go applications.
//
// # Core Concepts
//
// clihelp combines declarative command and flag definitions with robust execution
// lifecycles, pflag-backed option parsing, positional argument validation,
// shell completion, and GitHub-friendly Markdown generation.
//
// An application is defined using [App], which contains a hierarchy of [Command]
// nodes, persistent/local [Option] flags, and lifecycle hooks.
//
// # Quick Start
//
// A minimal CLI application:
//
//	package main
//
//	import (
//		"fmt"
//		"os"
//
//		"github.com/sarielhp/clihelp"
//	)
//
//	func main() {
//		var verbose bool
//		var output string
//
//		app := &clihelp.App{
//			Name:        "demo",
//			Description: "Demonstration command-line tool",
//			Version:     "1.0.0",
//			Commands: []clihelp.Command{
//				{
//					Name:        "build",
//					Description: "Compile the target package",
//					UsageLine:   "demo build [options] <target>",
//					Args:        clihelp.ExactArgs(1),
//					Options: []clihelp.Option{
//						clihelp.String(&output, "-o, --output PATH", "dist", "Output directory"),
//						clihelp.Bool(&verbose, "-v, --verbose", false, "Enable verbose logging"),
//					},
//					Run: func(ctx *clihelp.Context) error {
//						target := ctx.Args[0]
//						fmt.Fprintf(ctx.Stdout, "Building %s to %s (verbose=%v)\n", target, output, verbose)
//						return nil
//					},
//				},
//			},
//		}
//
//		if err := app.Execute(os.Args[1:]); err != nil {
//			clihelp.PrintError(err)
//			os.Exit(1)
//		}
//	}
//
// # Execution Lifecycle
//
// When [App.Execute] or [App.ExecuteContext] is called, the execution follows
// a deterministic multi-stage pipeline:
//
//  1. Completion Check: If the first argument is "__complete", shell completion runs.
//  2. Version Check: If "--version", "-v", or "version" is passed without a custom version command,
//     the version string is printed to stdout.
//  3. Command Resolution: Subcommands and aliases are resolved hierarchically. Unknown command
//     tokens trigger typo suggestions via Levenshtein distance. Built-in "help <subcommand>" is
//     automatically routed.
//  4. Flag Binding & Parsing: [App.PersistentOptions], ancestor persistent options, and target
//     command options are bound to a [pflag.FlagSet] and parsed against remaining arguments.
//     Help flags (-h, --help) are automatically registered.
//  5. Argument Validation: The command's [ArgsValidator] validates remaining positional arguments.
//  6. Lifecycle Hooks:
//     App.BeforeRun -> Command.PreRun -> Command.Run (or App.Run) -> Command.PostRun -> App.AfterRun
//
// If any hook or validator returns a non-nil error, execution halts immediately and the error is returned.
//
// # Flag Specifications
//
// Options are configured using typed helper constructors such as [String], [Int], [Bool],
// [BoolToggle], [Duration], [StringSlice], [Enum], and [Var].
//
// Flag specification strings support rich syntax:
//   - Short and long flags: "-o, --output PATH"
//   - Multiple aliases: "-p, -P, --port, --listen-port <port>"
//   - Boolean toggle pairs: "--[no-]cache" (registers both --cache and --no-cache)
//   - Value hints / placeholders: "<file>", "PATH", "[value]"
//
// Caution: Do not manually register "-h" or "--help" flags in your Options slices.
// [App.Execute] automatically manages help flag registration and help rendering.
//
// # Positional Argument Validators
//
// Positional arguments are validated after flag parsing. Built-in validators include:
//   - [NoArgs]: rejects any positional arguments
//   - [ExactArgs](n): requires exactly n positional arguments
//   - [MinimumNArgs](n): requires at least n positional arguments
//   - [MaximumNArgs](n): requires at most n positional arguments
//   - [RangeArgs](min, max): requires between min and max positional arguments
//
// Validated arguments are available via [Context.Args].
//
// # Shell Autocompletion
//
// clihelp includes generators for Bash and Zsh autocompletion:
//   - [GenBashCompletion]
//   - [GenZshCompletion]
//
// Dynamic completion is supported by setting the Option.Complete callback.
//
// # Markdown Help Generation
//
// Call [RenderMarkdown] to generate a GitHub-friendly markdown help site from the
// command hierarchy. The generator uses SHA-256 content hashing to avoid unnecessary
// disk writes.
package clihelp
