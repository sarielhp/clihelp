# Execution Lifecycle, Routing & Context

`clihelp` provides a structured execution pipeline that manages argument parsing, subcommand routing, argument validation, and lifecycle hooks with context propagation.

---

## Table of Contents

- [Execution Pipeline](#execution-pipeline)
- [Lifecycle Hooks & Hook Order](#lifecycle-hooks--hook-order)
- [Abort Semantics & Error Handling](#abort-semantics--error-handling)
- [The `clihelp.Context` Object](#the-clihelpcontext-object)
- [Subcommands & Persistent Options](#subcommands--persistent-options)
- [Positional Argument Validation](#positional-argument-validation)
- [Fuzzy Typo Suggestions & Built-in `help`](#fuzzy-typo-suggestions--built-in-help)

---

## Execution Pipeline

When `app.Execute(os.Args[1:])` or `app.ExecuteContext(ctx, args)` is called, the application processes arguments in the following sequence:

```
┌────────────────────────────────────────────────────────┐
│ 1. Shell Completion Check (__complete argument)        │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 2. Version Check (--version, -v, version)              │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 3. Resolve Subcommand Hierarchy & Command Aliases      │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 4. Bind Persistent & Local Options to pflag.FlagSet    │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 5. Parse Flags (populates target variables)            │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 6. Validate Positional Arguments (Command.Args)        │
└──────────────────────────┬─────────────────────────────┘
                           ▼
┌────────────────────────────────────────────────────────┐
│ 7. Execute Lifecycle Hooks:                            │
│    App.BeforeRun ──► Command.PreRun ──► Command.Run    │
│    ──► Command.PostRun ──► App.AfterRun                │
└────────────────────────────────────────────────────────┘
```

---

## Lifecycle Hooks & Hook Order

Lifecycle hooks allow you to structure global pre-flight checks (like loading config or initializing telemetry), command-specific setup, core execution, and teardown:

| Hook | Level | Signature | Typical Use Case |
|---|---|---|---|
| `App.BeforeRun` | Application | `func(ctx *clihelp.Context) error` | Global setup, reading config files, auth checks |
| `Command.PreRun` | Command | `func(ctx *clihelp.Context) error` | Command-specific validation or database connections |
| `Command.Run` | Command | `func(ctx *clihelp.Context) error` | Main command business logic |
| `App.Run` | Application | `func(ctx *clihelp.Context) error` | Default action when no subcommand is given |
| `Command.PostRun` | Command | `func(ctx *clihelp.Context) error` | Command-level cleanup or resource release |
| `App.AfterRun` | Application | `func(ctx *clihelp.Context) error` | Global teardown, telemetry flush, log sync |

### Lifecycle Example

```go
app := &clihelp.App{
    Name:  "mycli",
    Pager: true,
    BeforeRun: func(ctx *clihelp.Context) error {
        // Global pre-flight (e.g. initialize logger)
        return nil
    },
    Commands: []clihelp.Command{
        {
            Name: "deploy",
            PreRun: func(ctx *clihelp.Context) error {
                // Ensure cloud credentials are valid
                return nil
            },
            Run: func(ctx *clihelp.Context) error {
                fmt.Fprintln(ctx.Stdout, "Deploying...")
                return nil
            },
            PostRun: func(ctx *clihelp.Context) error {
                // Clean up local artifacts
                return nil
            },
        },
    },
    AfterRun: func(ctx *clihelp.Context) error {
        // Global teardown
        return nil
    },
}
```

---

## Abort Semantics & Error Handling

- If flag parsing fails, positional argument validation fails, or **any hook returns a non-nil error**, downstream execution is aborted immediately.
- The returned error is propagated up through `app.Execute()`.
- Standardize your error output using `clihelp.PrintError(err)`.

```go
if err := app.Execute(os.Args[1:]); err != nil {
    clihelp.PrintError(err) // Formats in bold red to stderr
    os.Exit(1)
}
```

---

## The `clihelp.Context` Object

All lifecycle hooks receive a pointer to `clihelp.Context`, which encapsulates runtime state:

```go
type Context struct {
    Context context.Context // Go context for cancellation and deadlines
    App     *App            // Pointer to the root App definition
    Command *Command        // Pointer to the matched Command (or nil if root)
    Args    []string        // Validated positional arguments (flags stripped)
    RawArgs []string        // Raw unmodified argument slice passed to Execute
    Stdout  io.Writer       // Destination stdout (defaults to os.Stdout)
    Stderr  io.Writer       // Destination stderr (defaults to os.Stderr)
}
```

---

## Subcommands & Persistent Options

Subcommands can be nested to arbitrary depths via `Command.Subcommands`. 

Options defined on `App.PersistentOptions` or parent `Command.PersistentOptions` are inherited and parsed by all downstream child subcommands.

```go
app := &clihelp.App{
    Name:  "podctl",
    Pager: true,
    PersistentOptions: []clihelp.Option{
        clihelp.Bool(&verbose, "-v, --verbose", false, "Verbose output"),
    },
    Commands: []clihelp.Command{
        {
            Name: "config",
            Description: "Configuration management",
            Subcommands: []clihelp.Command{
                {
                    Name: "set",
                    Description: "Set configuration attribute",
                    Subcommands: []clihelp.Command{
                        {
                            Name: "space",
                            Description: "Set disk space limit",
                            Args: clihelp.ExactArgs(1),
                            Run: func(ctx *clihelp.Context) error {
                                fmt.Fprintf(ctx.Stdout, "Space limit set to: %s\n", ctx.Args[0])
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

---

## Positional Argument Validation

Positional arguments are validated after flag stripping. Configure `Command.Args` using built-in validators:

| Validator | Behavior |
|---|---|
| `clihelp.NoArgs` | Errors if any positional arguments are supplied. |
| `clihelp.ExactArgs(n)` | Errors unless exactly `n` positional arguments are provided. |
| `clihelp.MinimumNArgs(n)` | Errors if fewer than `n` positional arguments are provided. |
| `clihelp.MaximumNArgs(n)` | Errors if more than `n` positional arguments are provided. |
| `clihelp.RangeArgs(min, max)` | Errors if argument count is outside `[min, max]`. |

In `Run` handlers, positional arguments are accessible directly via `ctx.Args`:

```go
Command{
    Name: "convert",
    Args: clihelp.ExactArgs(2),
    Run: func(ctx *clihelp.Context) error {
        input := ctx.Args[0]
        output := ctx.Args[1]
        // ...
        return nil
    },
}
```

---

## Fuzzy Typo Suggestions & Built-in `help`

### Typo Suggestions

When an unknown subcommand is entered, `clihelp` runs Levenshtein distance calculations against registered command names and aliases:

```bash
$ podctl biuld
Error: unknown command "biuld" for "podctl". Did you mean "build"?
```

### Built-in `help` Subcommand

`clihelp` automatically routes `app help <subcommand>` (e.g. `podctl help config set`) to render detailed help pages for that command path, unless you register a custom command named `"help"`.
