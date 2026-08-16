# clihelp

`clihelp` is a standalone Go package for formatting colored, width-aware terminal help text for CLI applications with single or multi-level subcommand hierarchies.

## Features

- **Theme-Driven Styling** — One renderer, any look: colors, separator bars, and header wording are controlled by a `Theme` (defaults to a bold-yellow / white / bold-cyan palette like `mail_cli`).
- **io.Writer Output** — All rendering writes to a caller-supplied `io.Writer`, so output is easy to test, capture, or route to stdout vs stderr.
- **Deterministic Width** — Terminal width auto-detection (`golang.org/x/term`) with a 70-column fallback for non-TTY / piped environments, overridable per-render via `Options.Width`.
- **ANSI-Aware Word Wrapping** — Reflows descriptions cleanly without breaking ANSI color sequences (`github.com/acarl005/stripansi`).
- **Rich Sections** — Description, Usage, Subcommands, Parameters, Flags, Examples, and Notes, all first-class.
- **Multi-Level Subcommands** — Arbitrary nesting (e.g. `podctl config set location`).

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

	// Render global usage, or command help when a path is supplied.
	app.Render(clihelp.Options{})
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

// Render help for the nested path: podctl config set location.
app.RenderCommand(clihelp.Options{}, "config", "set", "location")
```

## Migrating from 0.1.x to v0.2.0

`v0.2.0` replaced the old `Print*` renderer API with a unified, theme-driven,
`io.Writer`-first engine. Your **data model (`App`, `Command`, `Option`,
`Example`, and `LookupCommand`) is unchanged** — only the method calls change.

| Before (v0.1.x) | After (v0.2.0) |
|-----------------|-----------------|
| `app.PrintGlobalUsage()` | `app.RenderGlobal(clihelp.Options{})` |
| `app.PrintCommandUsage("config", "set")` | `app.RenderCommand(clihelp.Options{}, "config", "set")` |
| `app.PrintUsage(args...)` | `app.Render(clihelp.Options{}, args...)` |
| `PrintSection("USAGE")` | removed (use `Theme`/sections) |

Capturing output to a buffer (rather than stdout) used to require the old
`Print*To` helpers; now pass a writer directly:

```go
var buf bytes.Buffer
app.RenderCommand(clihelp.Options{Writer: &buf, Width: 80}, "config", "set")
```

Two properties are new but optional:
- `Options.Writer` defaults to `os.Stdout` when nil.
- `Options.Width` defaults to auto-detection (70-column fallback) when `0`.

Stay on the old API with `go get github.com/sarielhp/clihelp@v0.1.1` if you
don't want to migrate yet.

## API Reference

### Structs

| Struct | Purpose | Key Fields |
|--------|---------|------------|
| `App` | Top-level CLI app definition | `Name`, `Description`, `GlobalNote`, `Commands`, `Theme`, `GlobalFlags`, `Shortcuts`, `Version`, `ConfigPath` |
| `Command` | Subcommand or nested command | `Name`, `Title`, `Aliases`, `Description`, `UsageLine`, `Options`, `Parameters`, `SubcommandEntries`, `Notes`, `Examples`, `Subcommands` |
| `Option` | Command-line option/flag | `Flags` (e.g. `-o, --output`), `Description` |
| `Example` | Demonstrative usage line | `Line`, `Description` |
| `Param` | Positional arg / list entry | `Name`, `Description` |
| `Note` | Prose section | `Heading`, `Text` |
| `Theme` | Renderer styling | `Hdr`, `Body`, `Accent`, `Separator`, `TitlePrefix` |
| `Options` | Per-render control | `Writer`, `Width`, `Theme` |
| `MarkdownOptions` | Markdown generation | `Dir` (default: `"docs/clihelp"`) |

### Key Methods

#### `func (a *App) Render(o Options, path ...string) bool`
Dispatches to the global overview when `path` is empty, otherwise renders help for the command path. Returns `true` when a command path renders.

#### `func (a *App) RenderGlobal(o Options)`
Renders the top-level overview: `Usage of <name>:`, command list (with aliases), shortcut commands, global flags, config path, and version.

#### `func (a *App) RenderCommand(o Options, path ...string) bool`
Renders a detailed help page for a command or nested subcommand path (e.g. `"config"`, `"set"`, `"location"`), with Description, Usage, Subcommands, Parameters, Flags, Examples, and Notes sections. Returns `true` if found.

#### `func (a *App) LookupCommand(path ...string) *Command`
Traverses the command hierarchy and returns the matching `*Command` pointer, or `nil` if not found.

#### `func RenderMarkdown(a *App, o MarkdownOptions) (changed bool, err error)`
Generates GitHub-friendly markdown help pages under `o.Dir` (defaults to `docs/clihelp/`). Creates one `.md` file per command plus `index.md`, with embedded relative links between commands. Uses SHA-256 content hashing to skip regeneration when the usage tree is unchanged. Gated by the `CLIHELP_GEN` environment variable so deployed binaries stay silent.

```go
// Developer bootstrap — run: CLIHELP_GEN=1 go run .
if os.Getenv("CLIHELP_GEN") != "" {
    clihelp.RenderMarkdown(app, clihelp.MarkdownOptions{})
    return
}
```

## Markdown Documentation Generation

`clihelp` can generate a **GitHub-friendly markdown help site** from your `App` definition. Each command gets its own `.md` page with embedded links to subcommands, so the rendered output on GitHub is a fully navigable documentation tree.

### Workflow

```bash
# Generate the docs
CLIHELP_GEN=1 go run .

# Commit and push to GitHub
git add -A docs/clihelp
git commit -m "docs: update CLI help pages"
git push
```

The output goes to `docs/clihelp/` by default. A `.clihelp-hash` sidecar file tracks content staleness and is automatically gitignored. Subsequent runs without `CLIHELP_GEN` will regenerate only when the command tree changes.

See `example/main.go` for a complete integration example.

## Development & Automation

This project includes a full development workflow:

- `make check` — Run format, tidy, vet, staticcheck, unit tests, and build checks.
- `make run` — Run the demonstration application in `example/main.go`.
- `make test` — Execute unit tests.
- `make map` — Show exported package architecture.

## License

MIT License. See [LICENSE](LICENSE) for details.
