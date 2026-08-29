# AI Coding Agent & Pair Programming Guidelines

When implementing or modifying applications using `clihelp`, AI assistants, coding agents, and pair programmers should adhere to the following best practices:

---

## LLM Documentation Resources

- [`llms.txt`](../llms.txt) — Clean, token-efficient single-file specification designed for direct LLM ingestion.
- [Execution Lifecycle Guide](lifecycle-and-routing.md) — Comprehensive execution pipeline details.
- [Flags & Options Reference](flags-and-options.md) — Complete flag constructor signatures and type behaviors.

---

## Core Rules for AI Assistants

1. **Persistent Target Variables**:
   Always pass pointers to fields within an options struct or command-level variable scope (e.g. `&globals.Verbose`, `&opts.OutputDir`). Never pass pointers to temporary local variables that go out of scope.

2. **Never Manually Register `-h` / `--help`**:
   `clihelp` automatically binds help flags to all command flagsets during `app.Execute()`. Explicitly registering a help option will return a validation error.

3. **Use Subcommands for Nested Workflows**:
   Model nested subcommands (e.g. `config set space`) using `Command.Subcommands` slices rather than doing manual token parsing inside `Run` handlers.

4. **Read Arguments from `ctx.Args`**:
   Inside `Run`, `PreRun`, or `PostRun` handlers, always read positional arguments from `ctx.Args`. Flags and options are already stripped, and arguments are already validated.

5. **Declare Positional Validators**:
   Use `clihelp.ExactArgs(n)`, `clihelp.RangeArgs(min, max)`, or `clihelp.NoArgs` on commands that accept positional inputs to prevent out-of-bounds index errors.

6. **Standard Error Output**:
   Use `clihelp.PrintError(err)` at application entry points to render consistent bold-red error messages to `os.Stderr`.

7. **Shell Completion Integration**:
   When implementing completion, prefer mounting `clihelp.CompletionCommand()` for zero-boilerplate setup across `bash`, `zsh`, `fish`, and user-level self-installation (`InstallCompletion`). For manual pipelines, use `clihelp.GenBashCompletion`, `clihelp.GenZshCompletion`, `clihelp.GenFishCompletion`, and `clihelp.InstallCompletion`.

---

## Canonical Application Template

```go
package main

import (
    "fmt"
    "os"

    "github.com/sarielhp/clihelp"
)

type Config struct {
    Verbose bool
    Output  string
}

func main() {
    var cfg Config

    app := &clihelp.App{
        Name:        "myapp",
        Description: "Sample CLI tool",
        Version:     "1.0.0",
        Pager:       true,
        PersistentOptions: []clihelp.Option{
            clihelp.Bool(&cfg.Verbose, "-v, --verbose", false, "Enable verbose output"),
        },
        Commands: []clihelp.Command{
            {
                Name:        "process",
                Description: "Process input file",
                UsageLine:   "myapp process [options] <input>",
                Args:        clihelp.ExactArgs(1),
                Options: []clihelp.Option{
                    clihelp.String(&cfg.Output, "-o, --output PATH", "out.bin", "Output file path"),
                },
                Run: func(ctx *clihelp.Context) error {
                    input := ctx.Args[0]
                    fmt.Fprintf(ctx.Stdout, "Processing %s -> %s (verbose=%v)\n", input, cfg.Output, cfg.Verbose)
                    return nil
                },
            },
        },
    }

    if err := app.Execute(os.Args[1:]); err != nil {
        clihelp.PrintError(err)
        os.Exit(1)
    }
}
```
