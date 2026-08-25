# clihelp

[![Go Reference](https://pkg.go.dev/badge/github.com/sarielhp/clihelp.svg)](https://pkg.go.dev/github.com/sarielhp/clihelp)
[![Go Report Card](https://goreportcard.com/badge/github.com/sarielhp/clihelp)](https://goreportcard.com/report/github.com/sarielhp/clihelp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`clihelp` is a Go library for parsing command-line arguments and generating clean, detailed usage and help messages. Similar in functionality to [Cobra](https://github.com/spf13/cobra), it was written by an AI agent as a reusable tool to be used across other projects.

It provides clean, structured usage messages with support for ANSI colors and clickable OSC 8 hyperlinks in supported terminals, alongside `pflag`-backed flag parsing, hierarchical subcommand routing, argument validation, lifecycle hooks, and automatic GitHub Markdown documentation generation.

---

## Features

- **Declarative CLI Definition** — Define applications, subcommands, persistent options, and flags in clean struct definitions.
- **Pflag-Backed Option Parsing** — Robust flag parsing supporting aliases, boolean toggle pairs (`--[no-]flag`), typed values, and custom value parsers.
- **Execution Lifecycle Hooks** — Coordinated `BeforeRun`, `PreRun`, `Run`, `PostRun`, and `AfterRun` lifecycle execution with context propagation.
- **Positional Argument Validation** — Built-in validators (`ExactArgs`, `RangeArgs`, `MinimumNArgs`, `NoArgs`) executed after flag extraction.
- **Fuzzy Typo Suggestions** — Levenshtein-distance suggestions for mistyped commands (e.g. *Did you mean "build"?*).
- **Shell Autocompletion** — Built-in `__complete` protocol with generators for Bash, Zsh, and Fish, plus dynamic completion callbacks.
- **Rich Terminal Styling** — Theme-driven ANSI colors, auto-detected terminal width with 70-column fallback, and ANSI-aware word wrapping.
- **Inline Markdown & OSC 8 Hyperlinks** — Rich text formatting in descriptions: bold, italic, code, strikethrough, and clickable terminal hyperlinks.
- **Markdown Documentation Generator** — Automatically generates navigable, GitHub-friendly Markdown doc trees with SHA-256 change-detection caching.
- **AI & LLM-Optimized** — Token-efficient single-file [`llms.txt`](llms.txt) specification and declarative syntax eliminating common LLM hallucinations.

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

## Documentation & Topic Guides

Detailed technical guides and reference documentation are available in the [`docs/`](docs/) directory:

| Guide | Description |
|---|---|
| 🔄 [**Execution Lifecycle & Routing**](docs/lifecycle-and-routing.md) | Execution pipeline, lifecycle hooks (`BeforeRun`, `PreRun`, `Run`, etc.), abort semantics, `clihelp.Context`, nested subcommands, typo suggestions, and argument validation. |
| 🏷️ [**Flags & Options Reference**](docs/flags-and-options.md) | Flag spec syntax, constructor reference table (`String`, `Int`, `BoolToggle`, `Enum`, etc.), aliases, custom binders, and help collision safety. |
| 💻 [**Shell Autocompletion**](docs/completion.md) | Setting up Bash, Zsh, and Fish completion, wiring the `completion` command, dynamic callbacks, and live testing. |
| 📄 [**Markdown Doc Generation**](docs/markdown-generation.md) | Generating navigable GitHub Markdown docs with `RenderMarkdown` and SHA-256 change-detection caching. |
| 🤖 [**AI Coding Agent Guidelines**](docs/ai-guidelines.md) | Best practices and prompt rules for LLM coding agents and pair programmers building CLIs with `clihelp`. |
| 🧠 [**AI Context Specification (`llms.txt`)**](llms.txt) | Compact single-file specification formatted for direct ingestion by LLMs and AI developer tools. |
| ⚖️ [**Comparison with Cobra**](docs/comparison-with-cobra.md) | In-depth comparison with `spf13/cobra`, architectural differences, code patterns, and tradeoffs. |

---

## Terminal Formatting & Styling

Descriptions and notes support markdown-like inline formatting:

| Syntax | Terminal Output |
|---|---|
| `` `code` `` | Green highlighted text |
| `**bold**` | Bold text |
| `*italic*` | Italic text |
| `~~strikethrough~~` | Strikethrough text |
| `[Label](https://example.com)` | Clickable OSC 8 terminal hyperlink |
| `\X` | Escapes special character `X` |

### Width & wrapping

- Terminal width is auto-detected with a **70-column fallback** for non-TTY output.
- Content wraps at `indent + MaxContentWidth` columns (default `MaxContentWidth` is 80), so indented lists gain extra horizontal room without exceeding the terminal width. Set `Options.MaxContentWidth` to change the content cap.
- Command lists can be grouped with `Command.Group`; a group heading is rendered when the group value changes.

---

## Migrating from v0.1.x

In `v0.2.0+`, `clihelp` transitioned from a standalone help formatter to a full execution framework:

| Before (v0.1.x) | After (v0.2.x) |
|---|---|
| `app.PrintGlobalUsage()` | `app.RenderGlobal(clihelp.Options{})` |
| `app.PrintCommandUsage("config", "set")` | `app.RenderCommand(clihelp.Options{}, "config", "set")` |
| `app.PrintUsage(args...)` | `app.Render(clihelp.Options{}, args...)` |
| Manual flag parsing via `flag` | `app.Execute(os.Args[1:])` with declarative `clihelp.Option` |

---

## API Reference

Comprehensive Go package API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/sarielhp/clihelp).

---

## License

MIT License. See [LICENSE](LICENSE) for details.
