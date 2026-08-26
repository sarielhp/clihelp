# Recipes & Common Patterns

Practical design patterns, testing strategies, and advanced recipes for building command-line applications with `clihelp`.

---

## Table of Contents

- [Graceful Shutdown & Signal Cancellation](#graceful-shutdown--signal-cancellation)
- [Unit Testing Commands & Output](#unit-testing-commands--output)
- [Dynamic Shell Autocompletion Callbacks](#dynamic-shell-autocompletion-callbacks)
- [Persistent Flags & Subcommand Inheritance](#persistent-flags--subcommand-inheritance)
- [Rendering the Command Tree](#rendering-the-command-tree)
- [Customizing Terminal Themes](#customizing-terminal-themes)
- [Prefix Command Abbreviations](#prefix-command-abbreviations)

---

## Graceful Shutdown & Signal Cancellation

Use `app.ExecuteContext` in combination with standard library `signal.NotifyContext` to handle `SIGINT` (Ctrl+C) and `SIGTERM` gracefully:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sarielhp/clihelp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := &clihelp.App{
		Name:  "worker",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "start",
				Run: func(c *clihelp.Context) error {
					fmt.Fprintln(c.Stdout, "Starting long-running process...")
					select {
					case <-time.After(30 * time.Second):
						fmt.Fprintln(c.Stdout, "Task completed.")
					case <-c.Context.Done():
						fmt.Fprintln(c.Stderr, "Received shutdown signal. Cleaning up...")
						return c.Context.Err()
					}
					return nil
				},
			},
		},
	}

	if err := app.ExecuteContext(ctx, os.Args[1:]); err != nil {
		clihelp.PrintError(err)
		os.Exit(1)
	}
}
```

Inside any `Run`, `PreRun`, or `PostRun` handler, access `c.Context` to listen for cancellation or pass it downward to I/O operations and database clients.

---

## Unit Testing Commands & Output

Because `clihelp.App` accepts customizable I/O streams and pure argument slices, testing CLI commands in Go tests is straightforward without spawning sub-processes:

```go
package main_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

func TestBuildCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var output string
	var verbose bool

	app := &clihelp.App{
		Name:   "podctl",
		Pager:  true,
		Stdout: &stdout,
		Stderr: &stderr,
		Commands: []clihelp.Command{
			{
				Name: "build",
				Args: clihelp.ExactArgs(1),
				Options: []clihelp.Option{
					clihelp.String(&output, "-o, --output PATH", "dist", "Output file"),
					clihelp.Bool(&verbose, "-v, --verbose", false, "Verbose logging"),
				},
				Run: func(ctx *clihelp.Context) error {
					ctx.Stdout.Write([]byte("building " + ctx.Args[0] + " to " + output + "\n"))
					return nil
				},
			},
		},
	}

	// Execute with mock command line arguments
	err := app.Execute([]string{"build", "-o", "bin/out", "input.wav"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := stdout.String()
	want := "building input.wav to bin/out\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

---

## Dynamic Shell Autocompletion Callbacks

To provide dynamic, context-aware shell autocompletion (such as completing available cluster environments, remote accounts, or local files), attach a `Complete` callback to your `Option`:

```go
var region string

clihelp.Option{
	Flags:       "-r, --region REGION",
	Description: "AWS region name",
	Complete: func(toComplete string) []string {
		regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
		var matches []string
		for _, r := range regions {
			if strings.HasPrefix(r, toComplete) {
				matches = append(matches, r)
			}
		}
		return matches
	},
}
```

`clihelp.Enum` automatically configures dynamic completion for all allowed values out-of-the-box:

```go
var env string
clihelp.Enum(&env, "-e, --env ENV", []string{"dev", "staging", "prod"}, "dev", "Target deployment environment")
```

---

## Persistent Flags & Subcommand Inheritance

Flags defined in `PersistentOptions` on the root `App` or on parent `Command`s are automatically inherited by all descendant subcommands:

```go
type GlobalConfig struct {
	ConfigFile string
	Verbose    bool
}

func buildApp() *clihelp.App {
	var cfg GlobalConfig

	return &clihelp.App{
		Name:  "cloudctl",
		Pager: true,
		PersistentOptions: []clihelp.Option{
			clihelp.String(&cfg.ConfigFile, "-c, --config PATH", "~/.cloudctl.yaml", "Path to configuration file"),
			clihelp.Bool(&cfg.Verbose, "-v, --verbose", false, "Enable verbose output"),
		},
		BeforeRun: func(ctx *clihelp.Context) error {
			// Global initialization runs before any command Run
			if cfg.Verbose {
				fmt.Fprintf(ctx.Stdout, "[DEBUG] Using config file: %s\n", cfg.ConfigFile)
			}
			return nil
		},
		Commands: []clihelp.Command{
			{
				Name: "cluster",
				Subcommands: []clihelp.Command{
					{
						Name: "list",
						Run: func(ctx *clihelp.Context) error {
							// Both --config and --verbose have been bound and parsed here
							fmt.Fprintln(ctx.Stdout, "Listing clusters...")
							return nil
						},
					},
				},
			},
		},
	}
}
```

Running `cloudctl cluster list -v -c ./dev.yaml` correctly populates `cfg.Verbose` and `cfg.ConfigFile`.

---

## Rendering the Command Tree

To output a complete visual hierarchy of all available commands and subcommands with box-drawing characters, use `app.RenderTree`:

```go
// Render full command tree to stdout
app.RenderTree(clihelp.Options{Writer: os.Stdout})
```

Example rendered tree:

```
podctl
├── build
├── serve
├── config
│   ├── get
│   └── set
│       └── space
└── deploy
```

---

## Customizing Terminal Themes

`clihelp` provides full control over ANSI colors, section title formatting, and separators via `clihelp.Theme`:

```go
import (
	"github.com/fatih/color"
	"github.com/sarielhp/clihelp"
)

customTheme := clihelp.Theme{
	Header:      color.New(color.FgCyan, color.Bold),
	Command:     color.New(color.FgHiBlue, color.Bold),
	Subcommand:  color.New(color.FgHiGreen),
	Option:      color.New(color.FgYellow),
	Separator:   true,       // Draws horizontal dividers between sections
	TitlePrefix: "▶ ",      // Prefix before section titles
}

app := &clihelp.App{
	Name:  "mytool",
	Pager: true,
	Theme: customTheme,
	// ...
}
```

---

## Prefix Command Abbreviations

When `AbbrevCommands: true` is enabled on `App`, users can type unique prefixes instead of full command names:

```go
app := &clihelp.App{
	Name:           "podctl",
	AbbrevCommands: true,
	Pager:          true,
	Commands: []clihelp.Command{
		{Name: "build", Description: "Build audio"},
		{Name: "serve", Description: "Start server"},
		{Name: "status", Description: "Show status"},
	},
}
```

- `podctl b` resolves to `podctl build`
- `podctl se` resolves to `podctl serve`
- `podctl s` returns a clear error: `ambiguous command "s": matching "serve", "status"`
