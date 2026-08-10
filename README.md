# clihelp

Colored, width-respecting CLI help text for Go programs with subcommands.

## Features

- **Colored section headers** — yellow, bold headers for USAGE, COMMANDS, OPTIONS, EXAMPLES
- **Width-respecting wrap** — text wraps at terminal width (80 fallback), ANSI-aware
- **Command-based** — register subcommands with per-command options and examples
- **Global usage** — overview listing all commands (no options)
- **Per-command usage** — detailed view with options and examples

## Install

```
go get github.com/sarielhp/clihelp
```

## Quick start

```go
package main

import "github.com/sarielhp/clihelp"

func main() {
    app := &clihelp.App{
        Name:        "myapp",
        Description: "Does something useful",
        Commands: []clihelp.Command{
            {
                Name:        "build",
                Description: "Build the project",
                UsageLine:   "myapp build [options]",
                Options: []clihelp.Option{
                    {Flags: "-o, --output PATH", Description: "Output path"},
                    {Flags: "--verbose", Description: "Verbose output"},
                },
                Examples: []clihelp.Example{
                    {Line: "myapp build -o out.bin"},
                },
            },
        },
    }

    clihelp.PrintGlobalUsage(app)
}
```
