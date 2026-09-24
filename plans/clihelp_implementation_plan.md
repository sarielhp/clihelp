# Implementation Plan: Enhancing clihelp (and Integrating with mail_cli)

This plan details the implementation of the three approved features in `github.com/sarielhp/clihelp` (`/home/sariel/prog/26/go/clihelp`), followed by their integration and cleanup in `mail_cli` (`/home/sariel/prog/26/mail_cli`).

---

## Feature 1: Standardized `clihelp.ErrUsage` Sentinel Error

### Objective
Provide a standardized sentinel error `clihelp.ErrUsage` so caller applications can distinguish command-line syntax/usage errors (exit code 2) from runtime operational errors (exit code 1) using standard Go `errors.Is(err, clihelp.ErrUsage)`.

### clihelp Changes
1. **Define Sentinel Error (`errors.go`)**:
   ```go
   package clihelp

   import "errors"

   // ErrUsage indicates a command-line syntax, argument, flag, or resolution error.
   var ErrUsage = errors.New("usage error")

   // IsUsageError reports whether err was caused by a command-line usage or syntax error.
   func IsUsageError(err error) bool {
       return errors.Is(err, ErrUsage)
   }
   ```
2. **Wrap All Syntax / Resolution Failures**:
   - **Flag parsing errors (`execute.go`)**:
     Wrap `fs.Parse(args)` errors with `fmt.Errorf("%w: %v", ErrUsage, err)`.
   - **Missing required flags (`execute.go`)**:
     Wrap `required flag(s) %s not set` with `%w: ErrUsage`.
   - **Arity rejections (`validation.go`)**:
     Wrap errors returned by `ExactArgs`, `MinimumNArgs`, `MaximumNArgs`, `RangeArgs`, and `NoArgs` with `%w: ErrUsage`.
   - **Option constraints (`validation.go`)**:
     Wrap errors from `MutuallyExclusive`, `RequiredTogether`, `RequiredWith`, etc., with `%w: ErrUsage`.
   - **Unknown commands / Leftover arguments (`resolve.go`, `execute.go`)**:
     Wrap errors from `checkUnknownCommand` and `checkLeftoverArgument` with `%w: ErrUsage`.
   - **Enum validation (`options.go`)**:
     Wrap enum mismatch errors in flag binder with `%w: ErrUsage`.
3. **Tests in `clihelp`**:
   - Add unit tests verifying `errors.Is(err, clihelp.ErrUsage)` for:
     - Unknown flag (`--bogus`)
     - Unknown subcommand (`mytool foobar`)
     - Arity mismatch (`ExactArgs(1)` given 0 or 2 args)
     - Missing required flag
     - Mutually exclusive flags passed together

### mail_cli Integration
1. In `main.go`:
   Simplify `exitCodeFor(err error)`:
   ```go
   func exitCodeFor(err error) int {
       var unknownAccount *cfg_g.UnknownAccountError
       var ambiguousAccount *cfg_g.AmbiguousAccountError

       switch {
       case errors.Is(err, clihelp.ErrUsage),
           errors.Is(err, app.ErrUsage),
           errors.Is(err, cfg_g.ErrNoSenderAccount),
           errors.As(err, &unknownAccount),
           errors.As(err, &ambiguousAccount):
           return exitUsageError
       default:
           return exitRuntimeError
       }
   }
   ```
   All `clihelp` syntax/flag/arity failures will automatically yield exit code 2 without special-casing.

---

## Feature 2: Positional Filtering & Subcommand Resolution Hook

### Objective
Allow a command with both subcommands and positional arguments (such as `mail_cli` root supporting `%inbox`) to reject mistyped subcommands (e.g. `mail_cli scann`) and trigger `Did you mean?` suggestions rather than silently capturing the typo as an argument. Additionally, export suggestion utilities for custom routing needs.

### clihelp Changes
1. **Add `PositionalFilter` to `Command` and `App` (`command.go`, `app.go`)**:
   ```go
   // PositionalFilter tests whether an argument is a valid positional value for this command.
   // When a command has both subcommands and positionals, an argument at slot 0 that does
   // NOT match any subcommand is evaluated by PositionalFilter. If it returns false,
   // clihelp treats the argument as an unknown/misspelled command and suggests alternatives.
   PositionalFilter func(arg string) bool
   ```
2. **Update `takesPositionals` / Resolution (`resolve.go`)**:
   Update resolution in `resolve.go`:
   When checking whether an unrecognized word at slot 0 should be treated as an unknown command or accepted as a positional:
   - If `currentCmd.PositionalFilter != nil` (or `App.PositionalFilter` at root), evaluate `PositionalFilter(word)`.
   - If it returns `false`, `takesPositionals` returns `false` for that word, letting `checkUnknownCommand` format:
     `unknown command "scann" for "mail_cli". Did you mean "scan"?`
3. **Export Suggestion Utilities (`resolve.go`)**:
   Export existing internal helpers:
   ```go
   // SuggestCommand finds the closest matching command name from candidates for the input word.
   func SuggestCommand(input string, candidates []Command) string

   // FindNearestCommands searches the entire command hierarchy for closest matching command paths.
   func (a *App) FindNearestCommands(input string) []string
   ```
4. **Tests in `clihelp`**:
   - Add test with a command having subcommands `["scan", "search"]` and `PositionalFilter: func(s string) bool { return strings.HasPrefix(s, "%") }`:
     - Argument `%receipts` -> passes filter, treated as positional.
     - Argument `scann` -> fails filter, treated as unknown command, suggests `scan`.

### mail_cli Integration
1. Set `cliApp.PositionalFilter = func(arg string) bool { return strings.HasPrefix(arg, "%") }` in `cli/cli.go`.
2. Remove the custom Levenshtein algorithm and tree walk from `cli/cli.go`:
   - Delete `editDistance(a, b string) int` (~20 lines).
   - Delete `nameDistance(typed string, c clihelp.Command) int` (~15 lines).
   - Delete `nearestCommands(ctx *clihelp.Context, arg string) []string` (~30 lines).
   - Simplify `unknownRootCommand` to delegate directly to `clihelp` or eliminate it entirely as `clihelp`'s native resolution now handles `scann` automatically.

---

## Feature 3: Built-in `-E, --examples [command]` Meta-Flag

### Objective
Make `-E, --examples [command]` a first-class feature of `clihelp` alongside `-h, --help`, eliminating the need for manual `os.Args` preprocessing, short-flag unbundling, and dummy flag bindings in client applications.

### clihelp Changes
1. **Add `EnableExamplesFlag` to `App` (`app.go`)**:
   ```go
   type App struct {
       ...
       // EnableExamplesFlag enables the built-in -E, --examples flag, rendering
       // the examples help topic (optionally narrowed by command name).
       EnableExamplesFlag bool
   }
   ```
2. **Register and Recognize Examples Flags (`execute.go`)**:
   - In `helpFlagNames()` or dedicated `examplesFlagNames()`:
     Include `-E` and `--examples` when `a.EnableExamplesFlag` is true.
   - When `-E` or `--examples` is passed:
     - If followed by arguments (e.g. `mytool -E scan` or `mytool scan -E`), narrow to that command path: `a.renderExamplesTopic(stdout, []string{"scan"})`.
     - If bare (e.g. `mytool -E`), render all examples: `a.renderExamplesTopic(stdout, nil)`.
     - Exit cleanly with `nil` (similar to `--help`).
   - In `RenderGlobal()`:
     Advertise `-E, --examples` in global options overview:
     `-E, --examples       Show every example in one place; add a command to narrow it`.
3. **Tests in `clihelp`**:
   - Test `mytool -E` outputs all examples.
   - Test `mytool -E subcmd` and `mytool subcmd -E` outputs examples scoped to `subcmd`.
   - Test bundled flag behavior (e.g. `-E`).

### mail_cli Integration
1. In `cli/cli.go`:
   - Enable `cliApp.EnableExamplesFlag = true`.
   - Delete dummy `flagExamplesView` binding in `cliPersistentOptions()`.
2. In `main.go`:
   - Remove `cli.HandleExamples(...)` check before `cliApp.Execute()`.
3. In `cli/examples.go`:
   - Delete `cli/examples.go` (and `cli/examples_test.go` or convert tests to verify `cliApp.Execute([]string{"-E"})`).

---

## Execution Order & Verification Gates

1. **Phase 1: Implement in `clihelp` (`/home/sariel/prog/26/go/clihelp`)**:
   - Step 1.1: Add `ErrUsage` and wrap all syntax/flag/validation errors.
   - Step 1.2: Add `PositionalFilter` and export suggestion utilities.
   - Step 1.3: Add `EnableExamplesFlag` and wire to examples topic.
   - Step 1.4: Run full `clihelp` test suite (`go test ./...`) and lint checks.
   - Step 1.5: Commit and tag/release (e.g. `v0.3.45`).

2. **Phase 2: Integrate in `mail_cli` (`/home/sariel/prog/26/mail_cli`)**:
   - Step 2.1: Bump `go.mod` to new `clihelp` version.
   - Step 2.2: Simplify `main.go` exit codes using `errors.Is(err, clihelp.ErrUsage)`.
   - Step 2.3: Configure `PositionalFilter` in `cli/cli.go`, delete duplicate Levenshtein / tree walk code.
   - Step 2.4: Enable `EnableExamplesFlag`, delete `cli/examples.go`, remove arg interception in `main.go`.
   - Step 2.5: Run `mail_cli` pre-commit gate:
     - `./tools/check.rb`
     - `./tools/go-audit`
     - `go test -v ./...`
