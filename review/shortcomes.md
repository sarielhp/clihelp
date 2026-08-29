# Architectural Review: Proposed Features & Core Roadmaps

To keep `clihelp` lightweight, coherent, and idiomatic to Go, we must strictly filter out feature bloat. A good library does one thing well: command routing, POSIX flag binding, and beautiful help rendering. 

Below is a highly critical division of all discussed features into **Should Be Done** (improving core functionality without adding API complexity) and **Should Not Be Done** (preventing API bloat, redundant paradigms, or out-of-scope features).

---

## 1. 🟢 Should Be Done (High Value, Coherent & Low Complexity)

These features fix existing gaps, improve testability, or enhance validation clarity without complicating the programmer's coding interface.

### A. Implement `Option.Deprecated` warning logic
* **Critique:** The `Deprecated` field already exists in the `Option` struct but is non-functional. Implementing it is a bug fix rather than a new feature.
* **Why it's clean:** Fulfills the existing API contract. It automatically prints warnings to `Stderr` at execution time and applies visual striking/coloring in the help rendering, requiring zero code changes from the programmer.

### B. Declarative Flag Inter-dependencies (`OptionsValidator`)
* **Critique:** Although conditionals inside handlers work, declarative validation keeps the interface self-documenting and generates standardized error messages.
* **Why it's clean:** To maintain package coherence, this mirrors the existing positional argument validation pattern (`Args: clihelp.ExactArgs(1)`). By defining an `OptionsValidator` type and a few standard helpers (`MutuallyExclusive`, `RequiredTogether`), the API remains small and consistent while allowing custom validation functions where needed.
  ```go
  OptionsValidator: clihelp.ValidateOptions(
      clihelp.MutuallyExclusive("--json", "--yaml"),
      clihelp.RequiredTogether("--cert", "--key"),
  ),
  ```

#### Relational Validation Commands Provided:
1. `clihelp.MutuallyExclusive(flags ...string)`: At most one of the specified flags can be set.
   * *Error:* `Error: flags --json and --yaml are mutually exclusive`
2. `clihelp.RequiredTogether(flags ...string)`: If any of the flags are set, all of them must be set.
   * *Error:* `Error: flags --cert and --key must be used together`
3. `clihelp.RequiredWith(target string, required ...string)`: If `target` is set, all of the accompanying `required` flags must be set.
   * *Error:* `Error: flag --bucket is required when using --upload`
4. `clihelp.RequiredIf(flag string, condition string)`: The `flag` is required only if another flag is set to a specific value.
   * *Example:* `clihelp.RequiredIf("--token", "--auth-method=token")`
   * *Error:* `Error: flag --token is required when --auth-method is set to "token"`

### C. Built-in Testing Harness & Audit Check (`clihelp.TestExecute` & `clihelp.Audit`)
* **Critique:** Writing unit tests for Go CLI commands usually requires complex boilerplate (mocking stdin/stdout, capturing exit codes, and managing global state). Developers also frequently forget to add help descriptions or duplicate flag characters.
* **Why it's clean:** 
  1. `clihelp.TestExecute(app, args)` runs the application against captured internal buffers, simplifying unit testing.
  2. `clihelp.Audit(app)` statically scans the entire command tree during unit tests, ensuring no commands have empty descriptions, duplicate flags, subcommand name collisions, or malformed flag specifications.

### D. Interactive Fallback & Command Constructor Mode
* **Critique:** Instead of importing a heavy prompt library, this can be integrated using a simple config toggle: `App.InteractiveFallback bool`.
* **Why it's clean:** If a required parameter is missing, the command prompts the user, runs, and prints the exact direct CLI equivalent to `Stderr` (e.g. `💡 Shortcut next time: mytool build -o out.mp3 in.wav`). It requires **zero extra code** from the programmer, teaches the end-user how to use the CLI, and serves as an interactive script builder.

---

## 2. 🔴 Should Not Be Done (Rejected due to API Bloat / Complexity)

To maintain simplicity and package coherence, the following features are rejected because they introduce redundant paradigms, duplicate existing patterns, or violate the single-responsibility principle.

### A. Hybrid Struct-Tag Parser (`clihelp.FromStruct`) — REJECTED
* **Critique:** While struct-tag parsing (like Rust's `clap`) looks clean, Go lacks compile-time macros. Doing this in Go requires runtime reflection.
* **Why it fails simplicity:** It introduces a second, completely separate way of defining flags (Struct Tags vs. Functional Bindings), hurting package coherence. Typos inside struct tags are not caught by the compiler, leading to silent runtime bugs. The current functional bindings (`clihelp.String(...)`) are compile-time checked, safe, and require no reflection.

### B. Declarative Positional Argument Binding & Validation — REJECTED
* **Critique:** Abstracting positional arguments into custom structs/validators (e.g. `clihelp.IntArg(...)`) creates a heavy type system.
* **Why it fails simplicity:** In Go, accessing slice elements directly (`ctx.Args[0]`) is standard, readable, and idiomatic. Adding a complex positional argument binding engine introduces massive API surface area to solve a problem that is easily handled with Go's built-in control structures.

### C. Cascading Configuration & Environment Variable Binding — REJECTED
* **Critique:** Merging flags, environment variables, and config files is the domain of config managers (such as `Viper` or `Figment`).
* **Why it fails simplicity:** Trying to cram configuration management into `clihelp` violates the single-responsibility principle. Keeping `clihelp` focused strictly on CLI routing and formatting prevents code bloat.

### D. Positional Argument Autocompletion — REJECTED
* **Critique:** Autocompleting flags is straightforward, but autocompleting positional arguments (e.g. looking up database rows or dynamic branch lists) requires custom dynamic completion hooks per shell.
* **Why it fails simplicity:** Extremely high implementation and maintenance complexity for a niche feature that is rarely needed by most CLI applications.

### E. Modular Command Registration (`Register` method) — REJECTED
* **Critique:** Adding a method like `app.Register(Cmd)` to register subcommands.
* **Why it fails simplicity:** It is completely redundant. The `App.Commands` field is a slice of commands. Go programmers can already define commands in separate files and append them to the slice in `main.go`. Adding `Register` introduces a redundant way to mutate the struct.

### F. Context Logger Integration (`ctx.Logger`) — REJECTED
* **Critique:** Adding logging utilities directly into the execution context.
* **Why it fails simplicity:** Every project has its own logging preferences (`slog`, `zap`, etc.). Forcing a custom logger onto `clihelp.Context` introduces unnecessary dependencies and design opinions. Let the developer pass their logger through the standard `context.Context` payload.

### G. Decorated Handlers (Python style) — REJECTED
* **Critique:** Binding command flags and positional arguments directly to function signature parameters.
* **Why it fails simplicity:** Highly non-idiomatic, slow, requires reflection, and loses Go compile-time type checking.
