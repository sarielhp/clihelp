# clihelp

`clihelp` is a standalone Go package for formatting colored, width-aware terminal help text for CLI applications with single or multi-level subcommand hierarchies.

## Features

- **ANSI Color Styling** — Bold yellow section headers (`USAGE`, `COMMANDS`, `OPTIONS`, `EXAMPLES`), bold green command labels, and cyan flag options via `github.com/fatih/color`.
- **Terminal Width Auto-Detection** — Automatically queries stdout terminal width (`golang.org/x/term`) with a clean 80-character fallback for non-TTY / piped environments.
- **ANSI-Aware Text Wrapping** — Wraps descriptions cleanly without breaking ANSI color control sequences (`github.com/acarl005/stripansi`).
- **Multi-Level Subcommands** — Supports arbitrary subcommand nesting (e.g. `podctl config set location`).
- **Clean API & Structs** — Intuitive data modeling (`App`, `Command`, `Option`, `Example`) with convenience methods on `*App`.

## Installation

```bash
go get github.com/sarielhp/clihelp
```

## Quick Start

```go
package main

import (
	"os"

	"github.com/sarielhp/clihelp"
)

func main() {
	app := &clihelp.App{
		Name:        "myapp",
		Description: "A sample CLI application",
		GlobalNote:  "Run 'myapp <command> --help' for command-specific options.",
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile the project binary",
				UsageLine:   "myapp build [options] <target>",
				Options: []clihelp.Option{
					{Flags: "-o, --output PATH", Description: "Output binary path"},
					{Flags: "-v, --verbose", Description: "Enable verbose logging"},
				},
				Examples: []clihelp.Example{
					{Line: "myapp build -o out.bin main.go"},
				},
			},
		},
	}

	// Print global usage or command help automatically based on os.Args
	app.PrintUsage(os.Args[1:]...)
}
```

## Multi-Level Subcommands

`clihelp` supports nested subcommands by populating the `Subcommands` slice inside any `Command`:

```go
app := &clihelp.App{
	Name: "podctl",
	Commands: []clihelp.Command{
		{
			Name:        "config",
			Description: "Configuration management",
			UsageLine:   "podctl config <subcommand>",
			Subcommands: []clihelp.Command{
				{
					Name:        "set",
					Description: "Set configuration attribute values",
					UsageLine:   "podctl config set <attribute> <value>",
					Subcommands: []clihelp.Command{
						{
							Name:        "location",
							Description: "Set storage location zone ID",
							UsageLine:   "podctl config set location <id> [options]",
							Options: []clihelp.Option{
								{Flags: "--zone NAME", Description: "Availability zone"},
							},
							Examples: []clihelp.Example{
								{Line: "podctl config set location 5"},
							},
						},
					},
				},
			},
		},
	},
}

// Print help for nested path: podctl config set location
app.PrintCommandUsage("config", "set", "location")
```

## API Reference

### Structs

| Struct | Purpose | Key Fields |
|--------|---------|------------|
| `App` | Top-level CLI app definition | `Name`, `Description`, `GlobalNote`, `Commands` |
| `Command` | Subcommand or nested command | `Name`, `Description`, `UsageLine`, `Options`, `Examples`, `Subcommands` |
| `Option` | Command-line option/flag | `Flags` (e.g. `-o, --output`), `Description` |
| `Example` | Demonstrative usage line | `Line`, `Description` |

### Key Methods

#### `func (a *App) PrintGlobalUsage()`
Prints top-level usage overview listing all registered root commands.

#### `func (a *App) PrintCommandUsage(path ...string) bool`
Renders formatted help text for a specific command or nested subcommand path (e.g., `"config"`, `"set"`, `"location"`). Returns `true` if found, `false` otherwise.

#### `func (a *App) PrintUsage(path ...string) bool`
Convenience method. If `path` is empty, calls `PrintGlobalUsage()`. If `path` is provided, calls `PrintCommandUsage(path...)`.

#### `func (a *App) LookupCommand(path ...string) *Command`
Traverses the command hierarchy and returns the matching `*Command` pointer, or `nil` if not found.

#### `func PrintSection(name string)`
Prints a stand-alone yellow bold section header (e.g., `PrintSection("ENVIRONMENT")`).

## Development & Automation

This project includes a full development workflow:

- `make check` — Run format, tidy, vet, staticcheck, unit tests, and build checks.
- `make run` — Run the demonstration application in `example/main.go`.
- `make test` — Execute unit tests.
- `make map` — Show exported package architecture.

## License

MIT License. See [LICENSE](LICENSE) for details.
