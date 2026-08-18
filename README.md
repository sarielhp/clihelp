# clihelp

[![Go Reference](https://pkg.go.dev/badge/github.com/sarielhp/clihelp.svg)](https://pkg.go.dev/github.com/sarielhp/clihelp)
[![Go Report Card](https://goreportcard.com/badge/github.com/sarielhp/clihelp)](https://goreportcard.com/report/github.com/sarielhp/clihelp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`clihelp` is a modern, lightweight, declarative CLI framework and width-aware, colorized help-text generator for Go. It features `pflag`-backed flag parsing, hierarchical subcommand routing, argument validation, lifecycle hooks, shell autocompletion (Bash, Zsh, Fish), OSC 8 terminal hyperlinks, and automatic GitHub Markdown documentation generation.

---

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Execution Lifecycle & Context](#execution-lifecycle--context)
- [Flag & Option Specification Reference](#flag--option-specification-reference)
- [Positional Argument Validation](#positional-argument-validation)
- [Subcommands, Aliases & Typo Suggestions](#subcommands-aliases--typo-suggestions)
- [Shell Autocompletion](#shell-autocompletion)
- [Terminal Styling & Inline Markdown](#terminal-styling--inline-markdown)
- [GitHub Markdown Documentation Generator](#github-markdown-documentation-generator)
- [AI Coding Agent Guidelines](#ai-coding-agent-guidelines)
- [Migrating from v0.1.x](#migrating-from-v01x)
- [API Reference](#api-reference)
- [License](#license)

---

## Features

- **Declarative CLI Definition** — Define applications, nested subcommands, persistent options, and local flags in clean Go struct declarations.
- **Pflag-Backed Option Parsing** — Robust flag parsing supporting multiple short/long aliases, boolean toggle pairs (`--[no-]flag`), typed values, and custom value parsers.
- **Execution Lifecycle Hooks** — Coordinated `BeforeRun`, `PreRun`, `Run`, `PostRun`, and `AfterRun` lifecycle execution with context propagation.
- **Positional Argument Validation** — Built-in validators (`ExactArgs`, `RangeArgs`, `MinimumNArgs`, etc.) executed after flag extraction.
- **Fuzzy Typo Suggestions** — Levenshtein-distance suggestions for mistyped commands (e.g. *Did you mean "build"?*).
- **Shell Autocompletion** — Built-in `__complete` protocol with generators for Bash, Zsh, and Fish, plus dynamic completion callbacks.
- **Rich Terminal Styling** — Theme-driven ANSI colors, auto-detected terminal width with 70-column fallback, and ANSI-aware word wrapping.
- **Inline Markdown & OSC 8 Hyperlinks** — Rich text support in descriptions: bold, italic, code, strikethrough, and clickable terminal hyperlinks.
- **Markdown Documentation Generator** — Automatically generates navigable, GitHub-friendly Markdown doc trees with SHA-256 change-detection caching.

---

## Installation

Requires **Go 1.26+**:

```bash
go get github.com/sarielhp/clihelp
```

---

## Quick Start

```go
package main

import (
	"fmt"
	"os"

	"github.com/sarielhp/clihelp"
)

func main() {
	var verbose bool
	var output string
	var bitrate int
	var normalize bool

	app := &clihelp.App{
		Name:        "podctl",
		Description: "Podcast distribution & audio processing tool",
		Version:     "1.0.0",
		GlobalNote:  "Run 'podctl <command> --help' for command-specific options.",
		PersistentOptions: []clihelp.Option{
			clihelp.Bool(&verbose, "-v, --verbose", false, "Enable verbose logging"),
		},
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile & package audio episodes",
				UsageLine:   "podctl build [options] <source-file>",
				Args:        clihelp.ExactArgs(1),
				Options: []clihelp.Option{
					clihelp.String(&output, "-o, --output PATH", "", "Write output to PATH"),
					clihelp.Int(&bitrate, "-b, --bitrate KBPS", 192, "Target audio bitrate in kbps"),
					clihelp.BoolToggle(&normalize, "--[no-]normalize", true, "Apply LUFS loudness normalization"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl build episode01.wav"},
					{Line: "podctl build -o ep01.mp3 --bitrate 320 --no-normalize ep01.wav"},
				},
				Run: func(ctx *clihelp.Context) error {
					source := ctx.Args[0]
					fmt.Fprintf(ctx.Stdout, "Building %s -> %s (bitrate: %d kbps, normalize: %v, verbose: %v)\n",
						source, output, bitrate, normalize, verbose)
					return nil
				},
			},
		},
	}

	// Execute parses os.Args[1:], routes commands, runs hooks, and handles errors
	if err := app.Execute(os.Args[1:]); err != nil {
		clihelp.PrintError(err)
		os.Exit(1)
	}
}
```

---

## Execution Lifecycle & Context

### Execution Pipeline

When `app.Execute(os.Args[1:])` or `app.ExecuteContext(ctx, args)` is invoked, `clihelp` processes the request through the following deterministic pipeline:

```
Resolve Command Hierarchy & Flags
           │
           ▼
Bind Persistent & Local Options to FlagSet
           │
           ▼
Parse Flags (extracts flags, populates target variables)
           │
           ▼
Validate Positional Arguments (Command.Args)
           │
           ▼
Execute App.BeforeRun (Global setup)
           │
           ▼
Execute Command.PreRun (Command setup)
           │
           ▼
Execute Command.Run (or App.Run if root)
           │
           ▼
Execute Command.PostRun (Command teardown)
           │
           ▼
Execute App.AfterRun (Global teardown)
```

### Abort Semantics & Error Propagation

- If flag parsing fails, positional argument validation fails, or any hook (`BeforeRun`, `PreRun`, `Run`, `PostRun`, `AfterRun`) returns a non-nil `error`, execution **halts immediately** and returns that error.
- Downstream hooks are skipped on error.

### `clihelp.Context`

All lifecycle hooks receive a `*clihelp.Context` containing execution state:

```go
type Context struct {
	Context context.Context // Standard context.Context for cancellation / deadlines
	App     *App            // Pointer to the root App
	Command *Command        // Active matched Command (or nil if root)
	Args    []string        // Validated positional arguments (flags stripped)
	RawArgs []string        // Original unmodified arguments passed to Execute
	Stdout  io.Writer       // Destination stdout writer (defaults to os.Stdout)
	Stderr  io.Writer       // Destination stderr writer (defaults to os.Stderr)
}
```

---

## Flag & Option Specification Reference

### Specification String Syntax

Option constructor functions parse a expressive spec string defining flags, aliases, placeholders, and toggles:

| Syntax Pattern | Example | Description |
|---|---|---|
| Single Short Flag | `"-v"` | Binds short flag `-v`. |
| Single Long Flag | `"--verbose"` | Binds long flag `--verbose`. |
| Combined Short & Long | `"-o, --output PATH"` | Binds `-o` and `--output` with placeholder `PATH`. |
| Multiple Aliases | `"-p, -P, --port, --listen-port <num>"` | Binds multiple short (`-p`, `-P`) and long flags. |
| Boolean Toggle Pair | `"--[no-]cache"` | Binds `--cache` (`true`) and `--no-cache` (`false`). |
| Value Placeholders | `<file>`, `PATH`, `[value]`, `NAME` | Uppercase or bracketed tokens set help placeholder text. |

### Option Constructors

| Constructor | Spec Example | Target Type | Behavior & Default Handling |
|---|---|---|---|
| `clihelp.String(target, spec, default, usage)` | `"-o, --output PATH"` | `*string` | Initializes `*target = default`. Parses string. |
| `clihelp.Int(target, spec, default, usage)` | `"-p, --port N"` | `*int` | Initializes `*target = default`. Parses integer. |
| `clihelp.Bool(target, spec, default, usage)` | `"-v, --verbose"` | `*bool` | Initializes `*target = default`. Sets `true` when flag is supplied. |
| `clihelp.BoolToggle(target, spec, default, usage)` | `"--[no-]normalize"` | `*bool` | Registers `--normalize` and `--no-normalize`. Initializes `*target = default`. |
| `clihelp.Duration(target, spec, default, usage)` | `"-t, --timeout SEC"` | `*time.Duration` | Parses durations like `"30s"`, `"5m"`. Initializes `*target = default`. |
| `clihelp.StringSlice(target, spec, default, usage)` | `"-i, --include ITEM"` | `*[]string` | Repeatable or comma-separated strings (e.g. `-i a -i b`). |
| `clihelp.Enum(target, spec, allowed, default, usage)` | `"-s, --stage STAGE"` | `*string` | Restricts values to `allowed` slice; returns error on invalid input. Supports auto-completion. |
| `clihelp.Var(targetValue, spec, usage)` | `"--custom VAL"` | `pflag.Value` | Binds custom type implementing `pflag.Value`. |

### Advanced Option Fields

You can customize option presentation and behavior on the returned `clihelp.Option`:

```go
opt := clihelp.String(&host, "-H, --host <host>", "127.0.0.1", "Bind IP address")
opt.DefaultText = "localhost"                     // Custom default text in help output
opt.Hidden = true                                 // Hide from help output and completion
opt.Deprecated = "Use --listen-addr instead"       // Deprecation notice
opt.Complete = func(toComplete string) []string { // Dynamic tab completion
    return []string{"127.0.0.1", "0.0.0.0"}
}
```

> [!CAUTION]
> **Do not manually register `-h` or `--help` flags**: `clihelp` automatically binds `-h` and `--help` to every command flagset during execution. Explicitly registering a help flag will trigger a runtime panic (`panic: help flag redefined: help`).

---

## Positional Argument Validation

Configure `Command.Args` using built-in validators:

| Validator | Behavior |
|---|---|
| `clihelp.NoArgs` | Errors if any positional arguments are supplied (`len(args) > 0`). |
| `clihelp.ExactArgs(n)` | Errors unless exactly `n` positional arguments are supplied. |
| `clihelp.MinimumNArgs(n)` | Errors if fewer than `n` positional arguments are supplied. |
| `clihelp.MaximumNArgs(n)` | Errors if more than `n` positional arguments are supplied. |
| `clihelp.RangeArgs(min, max)` | Errors if argument count is not between `min` and `max` (inclusive). |

Custom validation function:

```go
Command{
    Name: "deploy",
    Args: func(args []string) error {
        if len(args) == 0 {
            return fmt.Errorf("missing environment target")
        }
        return nil
    },
}
```

---

## Subcommands, Aliases & Typo Suggestions

### Multi-Level Subcommands

Subcommands can be nested to arbitrary depths via `Command.Subcommands`:

```go
app := &clihelp.App{
    Name: "podctl",
    Commands: []clihelp.Command{
        {
            Name:        "config",
            Description: "Configuration management",
            Subcommands: []clihelp.Command{
                {
                    Name:        "set",
                    Description: "Set configuration attributes",
                    Subcommands: []clihelp.Command{
                        {
                            Name:        "space",
                            Description: "Set disk space limit",
                            Args:        clihelp.ExactArgs(1),
                            Run: func(ctx *clihelp.Context) error {
                                fmt.Printf("Setting space to %s\n", ctx.Args[0])
                                return nil
                            },
                        },
                    },
                },
            },
        },
    },
}
```

### Persistent Options Across Hierarchies

Options placed in `App.PersistentOptions` or parent `Command.PersistentOptions` are inherited and parsed by all child subcommands.

### Typo Suggestions

When an unknown subcommand is entered, `clihelp` calculates the Levenshtein distance across available commands and aliases, returning a helpful suggestion:

```bash
$ podctl biuld
Error: unknown command "biuld" for "podctl". Did you mean "build"?
```

### Built-in `help` Subcommand

`clihelp` automatically routes `app help <subcommand>` (e.g. `podctl help config set`) to render command help, unless you explicitly register a custom command named `"help"`.

---

## Shell Autocompletion

`clihelp` has built-in support for Bash, Zsh, and Fish tab autocompletion via the `__complete` protocol.

### Adding a `completion` Command

```go
Commands: []clihelp.Command{
    {
        Name:        "completion",
        Description: "Generate shell autocompletion script",
        UsageLine:   "podctl completion <bash|zsh|fish>",
        Args:        clihelp.ExactArgs(1),
        Run: func(ctx *clihelp.Context) error {
            switch ctx.Args[0] {
            case "bash":
                return clihelp.GenBashCompletion(ctx.App, ctx.Stdout)
            case "zsh":
                return clihelp.GenZshCompletion(ctx.App, ctx.Stdout)
            case "fish":
                return clihelp.GenFishCompletion(ctx.App, ctx.Stdout)
            default:
                return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", ctx.Args[0])
            }
        },
    },
}
```

### Dynamic Completion Callbacks

Provide contextual dynamic tab completions by attaching a `Complete` function to any `Option`:

```go
clihelp.Option{
    Flags:       "-s, --stage STAGE",
    Description: "Deployment stage",
    Complete: func(toComplete string) []string {
        stages := []string{"dev", "staging", "prod"}
        var matches []string
        for _, s := range stages {
            if strings.HasPrefix(s, toComplete) {
                matches = append(matches, s)
            }
        }
        return matches
    },
}
```

---

## Terminal Styling & Inline Markdown

### ANSI Theming & Width Wrapping

`clihelp` renders help text with clean terminal styling:
- **Terminal Width Auto-Detection**: Uses `golang.org/x/term` with a deterministic 70-column fallback for non-TTY / CI environments.
- **ANSI-Aware Word Wrapping**: Wraps prose sections without corrupting terminal escape codes (`github.com/acarl005/stripansi`).
- **Customizable Themes**: Override colors and header styling via `Theme`.

### Inline Formatting & Hyperlinks

Descriptions and notes support markdown-like inline formatting:

| Syntax | Terminal Output |
|---|---|
| `` `code` `` | Green highlighted text |
| `**bold**` | Bold text |
| `*italic*` | Italic text |
| `~~strikethrough~~` | Strikethrough text |
| `[Label](https://example.com)` | Clickable OSC 8 terminal hyperlink |
| `\X` | Escapes special character `X` |

### Standard Error Formatting

Use `clihelp.PrintError(err)` for consistent, bold-red prefixed error output to `os.Stderr`:

```go
if err := app.Execute(os.Args[1:]); err != nil {
    clihelp.PrintError(err)
    os.Exit(1)
}
```

---

## GitHub Markdown Documentation Generator

`clihelp` can generate a fully navigable GitHub Markdown documentation tree from your `App` definition.

### Generator Integration

```go
// Developer bootstrap — run: CLIHELP_GEN=1 go run .
if os.Getenv("CLIHELP_GEN") != "" {
    changed, err := clihelp.RenderMarkdown(app, clihelp.MarkdownOptions{
        Dir: "docs/clihelp",
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to generate docs: %v\n", err)
        os.Exit(1)
    }
    if changed {
        fmt.Println("Documentation updated in docs/clihelp/")
    }
    return
}
```

- Creates individual `.md` files for every command with relative markdown links to subcommands and parent pages.
- Maintains a `.clihelp-hash` sidecar file to skip regeneration when command structures haven't changed.

---

## AI Coding Agent Guidelines

When building or updating CLI tools with `clihelp`, AI agents and pair programmers should adhere to the following rules:

1. **Persistent Target Variables**: Always pass pointers to fields within a persistent options struct or command scope (e.g. `&opts.OutputDir`).
2. **Never Bind Manual `-h`/`--help` Flags**: Allow `clihelp` to manage help flag registration and help rendering automatically.
3. **Use Subcommands for Nested Workflows**: Represent subcommands via `Command.Subcommands` instead of manual argument parsing in `Run`.
4. **Read Arguments from `ctx.Args`**: Always consume validated positional arguments from `ctx.Args` in `Run` handlers rather than reading `os.Args`.
5. **Use `clihelp.PrintError(err)`**: Standardize error output using `clihelp.PrintError(err)`.
6. **Prefer `ExactArgs` / `RangeArgs`**: Always declare argument validators on commands requiring positional parameters to prevent out-of-bounds panics.

---

## Migrating from v0.1.x

In `v0.2.0`, `clihelp` transitioned from a standalone formatter to a complete execution framework:

| Before (v0.1.x) | After (v0.2.0) |
|---|---|
| `app.PrintGlobalUsage()` | `app.RenderGlobal(clihelp.Options{})` |
| `app.PrintCommandUsage("config", "set")` | `app.RenderCommand(clihelp.Options{}, "config", "set")` |
| `app.PrintUsage(args...)` | `app.Render(clihelp.Options{}, args...)` |
| Manual flag parsing via `flag` | `app.Execute(os.Args[1:])` with declarative `clihelp.Option` |

---

## API Reference

### Core Types

- [`App`](https://pkg.go.dev/github.com/sarielhp/clihelp#App) — Root application definition with commands, options, and lifecycle hooks.
- [`Command`](https://pkg.go.dev/github.com/sarielhp/clihelp#Command) — Executable command or category node with subcommands and flag options.
- [`Context`](https://pkg.go.dev/github.com/sarielhp/clihelp#Context) — Execution state passed to lifecycle hooks (`Context`, `Args`, `Stdout`, `Stderr`, `App`, `Command`).
- [`Option`](https://pkg.go.dev/github.com/sarielhp/clihelp#Option) — Flag specification and binder definition.
- [`ArgsValidator`](https://pkg.go.dev/github.com/sarielhp/clihelp#ArgsValidator) — Function signature for positional argument validation.
- [`Theme`](https://pkg.go.dev/github.com/sarielhp/clihelp#Theme) — Terminal color and header styling options.
- [`MarkdownOptions`](https://pkg.go.dev/github.com/sarielhp/clihelp#MarkdownOptions) — Configuration for Markdown doc generation.

### Primary Functions

- [`App.Execute(args []string) error`](https://pkg.go.dev/github.com/sarielhp/clihelp#App.Execute) — Parses flags, routes commands, validates arguments, and executes hooks.
- [`App.ExecuteContext(ctx context.Context, args []string) error`](https://pkg.go.dev/github.com/sarielhp/clihelp#App.ExecuteContext) — Context-aware execution.
- [`App.Render(opts Options, path ...string) bool`](https://pkg.go.dev/github.com/sarielhp/clihelp#App.Render) — Formats terminal help output for global app or command path.
- [`RenderMarkdown(app *App, opts MarkdownOptions) (bool, error)`](https://pkg.go.dev/github.com/sarielhp/clihelp#RenderMarkdown) — Generates GitHub Markdown documentation.
- [`GenBashCompletion`](https://pkg.go.dev/github.com/sarielhp/clihelp#GenBashCompletion), [`GenZshCompletion`](https://pkg.go.dev/github.com/sarielhp/clihelp#GenZshCompletion), [`GenFishCompletion`](https://pkg.go.dev/github.com/sarielhp/clihelp#GenFishCompletion) — Shell autocompletion script generators.
- [`PrintError(err error)`](https://pkg.go.dev/github.com/sarielhp/clihelp#PrintError) — Prints styled error messages to `os.Stderr`.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
