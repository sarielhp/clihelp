# clihelp 2.0: Architectural Blueprint & Implementation Plan
**A Modern, Declarative, Lightweight CLI Application Framework for Go**

---

## 1. Executive Summary & Vision

`clihelp` was originally conceived as a specialized terminal help formatter to replace the dense, unstyled output of standard CLI generators with clean, readable, modern typography.

**`clihelp 2.0`** evolves this into a **complete, declarative, lightweight CLI application framework** for Go. It merges:
1. **Battle-Tested POSIX/GNU Flag Parsing** via internal integration with [`github.com/spf13/pflag`](https://github.com/spf13/pflag).
2. **Pure Declarative Subcommand Routing** with zero `init()` side-effects, zero package-level globals, and first-class Dependency Injection.
3. **Industry-Leading Terminal Presentation**, structured examples (`Example{Line, Description}`), and native modern flag conventions (such as `--[no-]flag` boolean pairs).
4. **Live Dynamic Shell Autocompletions** for Bash, Zsh, and Fish.

---

## 2. Core Design Principles

### 2.1 Single Source of Truth
Metadata, flags, documentation, examples, and execution handlers are declared in a single, cohesive struct. Adding a flag automatically configures both the parser and the formatted help display.

### 2.2 Zero `init()` Functions & Zero Global State
Unlike Cobra, which relies heavily on package-level `init()` functions mutating a shared `rootCmd`, `clihelp` uses pure functional composition and factory functions (`NewScanCommand(client)`). This makes commands trivial to unit-test and instantiate in isolation.

### 2.3 Strict Compile-Time & Runtime Type Safety
Typed flag constructor functions (`clihelp.String`, `clihelp.Int`, `clihelp.Duration`, `clihelp.Bool`, `clihelp.Enum`) bind directly to typed Go struct pointers. Type validation occurs before command handlers are invoked—`Run` never has to manually parse strings or handle type conversion errors.

### 2.4 Lightweight Footprint
By building on modern Go (Go 1.20+) and avoiding legacy 2013-era backwards compatibility shims, the entire framework remains under **1,000 lines of code** with a single external dependency (`spf13/pflag`).

---

## 3. Public API & Type System Specification

### 3.1 Core Structs

```go
package clihelp

import (
	"context"
	"time"
)

// App represents the root CLI application.
type App struct {
	Name              string
	Description       string
	Version           string
	GlobalNote        string
	PersistentOptions []Option
	Commands          []Command
	BeforeRun         func(ctx *Context) error
	AfterRun          func(ctx *Context) error
	Run               func(ctx *Context) error
}

// Command represents an executable command or category node.
type Command struct {
	Name              string
	Aliases           []string
	Description       string
	UsageLine         string
	Group             string
	Hidden            bool
	PersistentOptions []Option
	Options           []Option
	Subcommands       []Command
	Examples          []Example
	Args              ArgsValidator
	PreRun            func(ctx *Context) error
	Run               func(ctx *Context) error
	PostRun           func(ctx *Context) error
}

// Example represents a structured usage example.
type Example struct {
	Line        string
	Description string
}

// Context encapsulates execution state passed to command handlers.
type Context struct {
	Context     context.Context
	App         *App
	Command     *Command
	Args        []string
	RawArgs     []string
}
```

---

### 3.2 Typed Option / Flag Constructors

`Option` defines both presentation metadata and runtime flag bindings:

```go
type Option struct {
	Flags       string                            // e.g. "-p, --podcast <name>"
	Description string                            // e.g. "Podcast title, index, or ID"
	DefaultText string                            // Custom display override for default
	Hidden      bool                              // Hidden from help output
	Deprecated  string                            // Deprecation notice
	Complete    func(toComplete string) []string  // Dynamic shell tab-completion
	Binder      func(fs *pflag.FlagSet) error     // Internal pflag binder
}

// String binds a string flag.
func String(target *string, flags string, defaultVal string, usage string) Option

// Int binds an integer flag.
func Int(target *int, flags string, defaultVal int, usage string) Option

// Bool binds a boolean flag (supports -s, --silent).
func Bool(target *bool, flags string, defaultVal bool, usage string) Option

// BoolToggle binds a boolean toggle pair (e.g. --[no-]check-new).
func BoolToggle(target *bool, flags string, defaultVal bool, usage string) Option

// Duration binds a time.Duration flag (e.g. --timeout 30s).
func Duration(target *time.Duration, flags string, defaultVal time.Duration, usage string) Option

// StringSlice binds a repeatable or comma-separated string list.
func StringSlice(target *[]string, flags string, defaultVal []string, usage string) Option

// Enum restricts input to an enumerated list of valid strings.
func Enum(target *string, flags string, allowed []string, defaultVal string, usage string) Option

// Var binds a custom user-defined pflag.Value interface.
func Var(target pflag.Value, flags string, usage string) Option
```

---

### 3.3 Positional Argument Validators

```go
type ArgsValidator func(args []string) error

func NoArgs(args []string) error
func ExactArgs(n int) ArgsValidator
func MinimumNArgs(n int) ArgsValidator
func MaximumNArgs(n int) ArgsValidator
func RangeArgs(min, max int) ArgsValidator
```

---

## 4. Execution Engine Architecture

```
                                  os.Args[1:]
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │     1. System Interception        │
                     │  • Check for --version / version  │
                     │  • Check for __complete (shell)   │
                     └─────────────────┬─────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │     2. Subcommand Resolution      │
                     │  • Match path & command aliases   │
                     │  • Check for -h / --help / help   │
                     └─────────────────┬─────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │     3. Flag Binding & Parsing     │
                     │  • Instantiate pflag.FlagSet      │
                     │  • Bind Persistent + Local Flags  │
                     │  • Parse & enforce types/defaults │
                     └─────────────────┬─────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │     4. Validation & Lifecycle     │
                     │  • Validate Positional Args       │
                     │  • App.BeforeRun                  │
                     │  • Command.PreRun                 │
                     │  • Command.Run                    │
                     │  • Command.PostRun                │
                     │  • App.AfterRun                   │
                     └───────────────────────────────────┘
```

---

## 5. Live Shell Tab-Completion Protocol

### 5.1 The Hidden `__complete` Protocol

When `app.Execute(os.Args[1:])` is invoked with `__complete` as the first parameter:
1. The routing engine walks the partial command line to locate the active command and flag.
2. If completing a flag value with a registered `Complete(toComplete)` callback, the callback is executed.
3. If completing a subcommand or flag name, available matching child commands and flags are collected.
4. Candidates are printed to `stdout` in the standard tab-separated format:
   ```text
   Dan Snow's History Hit	Podcast ID 92f0ecd1
   Hard Fork	Podcast ID f64abdc6
   The Ezra Klein Show	Podcast ID e17569d9
   ```
5. Process terminates cleanly with exit code `0`.

### 5.2 Shell Script Generation

Provide built-in generators:
* `clihelp.GenBashCompletion(app, writer)`
* `clihelp.GenZshCompletion(app, writer)`
* `clihelp.GenFishCompletion(app, writer)`

---

## 6. Target Usage in `podcast_manager`

Here is how `podcast_manager` is structured with `clihelp 2.0`:

```go
package main

import (
	"context"
	"os"
	"time"

	"github.com/sarielhp/clihelp"
)

type AppOptions struct {
	Host        string
	Token       string
	PodcastsDir string
	Silent      bool
	Verbose     bool
}

func main() {
	var globalOpts AppOptions
	cfg := loadConfig()

	app := &clihelp.App{
		Name:        "podcast_manager",
		Description: "Manage, Download, Scan, and Purge Audiobookshelf Podcast Libraries",
		Version:     "0.2.0",
		PersistentOptions: []clihelp.Option{
			clihelp.String(&globalOpts.Host, "--host <url>", cfg.Host, "Audiobookshelf server URL"),
			clihelp.String(&globalOpts.Token, "--token <token>", cfg.Token, "API Bearer Token"),
			clihelp.Bool(&globalOpts.Silent, "-s, --silent", false, "Silent mode (suppress non-error output)"),
			clihelp.Bool(&globalOpts.Verbose, "-v, --verbose", false, "Detailed output"),
		},
		Commands: []clihelp.Command{
			newScanCommand(&globalOpts),
			newDownloadCommand(&globalOpts),
			newNewCommand(&globalOpts),
			newKeepCommand(&globalOpts),
			newListCommand(&globalOpts),
			newConfigCommand(cfg),
		},
	}

	if err := app.ExecuteContext(context.Background(), os.Args[1:]); err != nil {
		clihelp.PrintError(err)
		os.Exit(1)
	}
}

func newScanCommand(global *AppOptions) clihelp.Command {
	var opts struct {
		Podcast  string
		Fill     bool
		CheckNew bool
		DryRun   bool
	}

	return clihelp.Command{
		Name:        "scan",
		Aliases:     []string{"rescan"},
		Description: "Check MP3 file lengths on disk and check for new episodes",
		UsageLine:   "podcast_manager scan [options]",
		Options: []clihelp.Option{
			clihelp.String(&opts.Podcast, "-p, -P, --podcast <name>", "", "Podcast title, index, or ID"),
			clihelp.Bool(&opts.Fill, "-f, --fill", false, "Check for gaps in downloaded episodes"),
			clihelp.BoolToggle(&opts.CheckNew, "--[no-]check-new", true, "Check and download new episodes"),
			clihelp.Bool(&opts.DryRun, "--dry-run", false, "Preview actions without executing"),
		},
		Examples: []clihelp.Example{
			{Line: "podcast_manager scan", Description: "Scan all podcasts"},
			{Line: "podcast_manager scan -p 1 --dry-run", Description: "Simulate scan for podcast #1"},
		},
		Run: func(ctx *clihelp.Context) error {
			client := NewABSClient(global.Host, global.Token)
			client.Silent = global.Silent
			return runScan(client, opts, global)
		},
	}
}
```

---

## 7. Implementation Roadmap & Phased Steps

### Phase 1: Repository Setup & `pflag` Flag Engine
- [ ] Add `github.com/spf13/pflag` as sole external dependency in `clihelp`.
- [ ] Implement `Option` struct and flag parsing helpers (`String`, `Int`, `Bool`, `BoolToggle`, `Duration`, `StringSlice`, `Enum`).
- [ ] Implement flag parsing engine that extracts short/long names and binds to `pflag.FlagSet`.

### Phase 2: Command Routing & Execution Engine
- [ ] Implement recursive `Command` tree resolver with alias normalization.
- [ ] Implement positional arguments validator functions (`ExactArgs`, `NoArgs`, etc.).
- [ ] Implement `App.ExecuteContext(ctx, args)` lifecycle runner (`BeforeRun` -> `PreRun` -> `Run` -> `PostRun` -> `AfterRun`).

### Phase 3: Formatted Output & Help Interception
- [ ] Integrate existing `clihelp` layout formatting engine into the `Command` tree.
- [ ] Automatically wire `-h`, `--help`, and `help <cmd>` interception into `app.Execute()`.
- [ ] Implement Levenshtein typo suggestion for misspelled commands (`did you mean 'scan'?`).

### Phase 4: Shell Autocompletions & Tooling
- [ ] Implement hidden `__complete` protocol in `app.Execute()`.
- [ ] Add `Complete` callback support to `clihelp.Option`.
- [ ] Implement `GenBashCompletion`, `GenZshCompletion`, and `GenFishCompletion`.

### Phase 5: Test Suite & Documentation
- [ ] Comprehensive test suite verifying flag types, invalid inputs, subcommand resolution, lifecycle hooks, and error handling.
- [ ] Benchmark execution time (target < 0.1ms dispatch overhead).
- [ ] Update documentation with quickstart examples.

---

## 8. Migration Guide for Dependent Projects

When migrating `podcast_manager` to `clihelp 2.0`:
1. `cli.go` drops custom argument parsing loops and becomes a pure factory assembling `clihelp.App`.
2. All `strconv.Atoi` and type conversions are eliminated.
3. Commands (`scan`, `download`, `keep`, `list`, `config`) receive clean options structs via closures.
4. Total line count in `podcast_manager` will drop by ~250–350 lines while gaining full shell autocompletion.
