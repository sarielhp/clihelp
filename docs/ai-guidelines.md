# AI Coding Agent & Pair Programming Guidelines

When implementing or modifying applications using `clihelp`, AI assistants, coding agents, and pair programmers should adhere to the following best practices:

---

> [!IMPORTANT]
> Read [How clihelp Decides](how-clihelp-decides.md) first. It states the
> decisions this library makes on your behalf and what each one reads, and the
> rules below follow from it. Rules memorised without the mechanism get
> misapplied the moment the situation shifts — a real application declared `Args`
> on all 71 of its commands, exactly as rule 5 says, and still wrote 81 lines to
> get a behaviour that declaration already entitled it to, because nothing said
> `Args` was an input the library reads.

## LLM Documentation Resources

- [How clihelp Decides](how-clihelp-decides.md) — the decision model. Start here.
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
   Use `clihelp.ExactArgs(n)`, `clihelp.RangeArgs(min, max)`, or `clihelp.NoArgs` on every command, including ones taking no arguments. This is not only a guard against out-of-bounds indexing: the declared arity is what the library reads to decide whether a bare invocation gets the command's help, whether an unrecognised word is a typo or an argument, and what the generated usage line says. A command with no validator declares nothing, and the library has to assume.

   On the application itself, set `App.Args: clihelp.NoArgs` whenever you define `App.Run` for a bare invocation, unless the root genuinely takes positional arguments. Without it the root is assumed to take none — which is usually right — but saying so makes it explicit and survives the app later growing arguments.

6. **Standard Error Output**:
   Use `app.PrintError(err)` at application entry points to render consistent bold-red error messages to `os.Stderr`.

7. **Shell Completion Integration**:
   When implementing completion, mount `clihelp.CompletionCommand()`: the user runs one command and gets tab completion, the Alt-H key binding and the manual page together. For a build-time pipeline — a packager generating scripts into `/usr/share` — use `clihelp.GenBashCompletion`, `clihelp.GenZshCompletion`, `clihelp.GenFishCompletion` and `clihelp.GenManPage`, which write to an `io.Writer` and touch nobody's home directory.

8. **Structure Concise and Extended Help**:
   Keep `Command.Description` to a concise single-sentence summary used in index tables and `-h`. Place architectural details and multi-paragraph guides in `Command.LongDescription`. Use `Note.Raw: true` or markdown code fences to output ASCII tables, diagrams, or configurations verbatim.

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
        app.PrintError(err)
        os.Exit(1)
    }
}
```
