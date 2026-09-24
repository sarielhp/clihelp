# Candidate Features for clihelp: Learnings from mail_cli

This document captures architectural patterns, custom workarounds, and duplicated logic in `mail_cli` that could be absorbed by upstream `clihelp` (`github.com/sarielhp/clihelp`) to simplify client applications.

---

### 1. Root Positional Validation & Fuzzy Suggestion Hook

- **Current Pain Point**:
  `mail_cli` supports running `mail_cli %` or `mail_cli %label` directly from the root app, so `App` defines a root `Run` handler with `Args: clihelp.MaximumNArgs(1)`.
  Because the root accepts positional arguments, `clihelp` disables unknown command detection at the root level and passes any mistyped command (`mail_cli scann`, `mail_cli prunne`) directly to `App.Run`.
- **Workaround in `mail_cli`**:
  `cli/cli.go` duplicates over 100 lines of `clihelp`'s internal fuzzy suggestion engine: Levenshtein distance (`editDistance`), recursive tree walking (`nearestCommands`), and "Did you mean?" formatting (`unknownRootCommand`).
- **Proposed `clihelp` Feature**:
  - **Option A (Declarative)**: Add a positional validator or matcher to `App` (e.g. `App.PositionalMatcher = func(arg string) bool` or `Param.Validate`). If the argument doesn't match (e.g. doesn't start with `%`), `clihelp` treats it as an unknown command and runs its built-in `suggestCommand` automatically.
  - **Option B (Exposed Utility)**: Export `clihelp.SuggestCommands(app *App, input string) []string` and `clihelp.UnknownCommandError(app *App, input string, hint string) error` so applications don't need to re-implement distance algorithms and tree traversal.

---

### 2. Built-in `-E, --examples [command]` Meta-Flag

- **Current Pain Point**:
  `mail_cli` supports `-E` and `--examples` to view examples for any command (backed by `clihelp`'s `help examples` topic).
- **Workaround in `mail_cli`**:
  `cli/examples.go` and `main.go` intercept `os.Args` manually before `cliApp.Execute()`, unbundle short flags (e.g. `-vE`), filter out arguments, and redirect to `app.Execute([]string{"help", "examples", ...})`. Furthermore, a dummy `flagExamplesView` must be bound in `cli/cli.go` just so `--help` advertises it.
- **Proposed `clihelp` Feature**:
  Add first-class support for an examples meta-flag (e.g. `App.EnableExamplesFlag = true` or `App.ExamplesFlag = "-E, --examples"`), functioning identically to built-in `-h, --help`.

---

### 3. Dynamic / Patterned Flag Matching (e.g. `-1`, `-2`, `-3`)

- **Current Pain Point**:
  `mail_cli` supports numeric flags (`-1`, `-2`, etc.) to select an account by its index in the config file.
- **Workaround in `mail_cli`**:
  `orchestration.go` runs a pre-processing pass (`preprocessArgs`) over `os.Args` to peel off numeric flags before `clihelp` runs, because neither `clihelp` nor `pflag` supports flags without a fixed name or bounded set.
- **Proposed `clihelp` Feature**:
  Add an `UnknownFlagHandler(func(flag string) (handled bool, err error))` or regex-based flag pattern hook (e.g. `clihelp.PatternFlag("^-[0-9]+$", handler)`). This would eliminate `orchestration.go` completely.

---

### 4. Early Flag Extraction / Bootstrap Hooks

- **Current Pain Point**:
  Applications frequently need to read certain global flags before the full application lifecycle starts (e.g., logging destination, config directory).
- **Workaround in `mail_cli`**:
  In `main.go`, `initAppLogger` manually scans `os.Args` for `-logappend` or `--logappend` to initialize the logger *before* executing `cliApp`.
- **Proposed `clihelp` Feature**:
  Provide `App.EarlyParse(args []string, flags ...*Option) error` or an `App.BeforeResolve(ctx)` hook where global/persistent flags have already been parsed.

---

### 5. Standardized `clihelp.ErrUsage` Sentinel Error

- **Current Pain Point**:
  POSIX / standard CLI convention uses exit code `2` for command-line syntax/usage errors and exit code `1` for runtime errors.
  In `main.go`, `exitCodeFor(err)` has to check specific error types. However, when `clihelp` fails due to an unknown flag, mutually exclusive flag violation, missing required flag, arity mismatch, or unknown command, it returns standard untyped errors (`fmt.Errorf(...)`).
- **Proposed `clihelp` Feature**:
  Export a sentinel `clihelp.ErrUsage` (or `clihelp.IsUsageError(err) bool`). All internal CLI parsing errors, arity rejections (`ExactArgs`, `MinimumNArgs`), and option constraint failures (`ValidateOptions`) should wrap `clihelp.ErrUsage`. This allows applications to cleanly classify syntax vs runtime failures using `errors.Is(err, clihelp.ErrUsage)`.

---

### 6. Interactive Confirmation Guard (`-y, --yes`)

- **Current Pain Point**:
  Irreversible actions (`spam empty`, `rule rm-all`, `account rm`, `cache reset`, `label rm`) need confirmation prompts in interactive mode and a mandatory `-y, --yes` flag in non-interactive/scripted mode.
- **Workaround in `mail_cli`**:
  `cli/confirm.go` implements terminal detection, prompt formatting, and refusal logic. Every command re-declares a separate boolean flag variable (`cli/flags.go`) and calls `session.Confirm(...)`.
- **Proposed `clihelp` Feature**:
  `clihelp` already has `InteractiveFallback` for missing required flags. It could offer declarative confirmation on commands:
  ```go
  cmd.Confirm = "delete every routing rule for this account"
  ```
  `clihelp` would automatically register `-y, --yes`, check TTY status, prompt `[y/N]` when interactive, and return `ErrUsage` if non-interactive without `--yes`.

---

### 7. Native Markdown Documentation Subcommand (`clihelp.DocCommand()`)

- **Current Pain Point**:
  While `clihelp` provides `clihelp.CompletionCommand()` and `clihelp.ManPageCommand()`, markdown documentation generation requires custom wiring (`usage/usage.go`) triggered by an environment variable `CLIHELP_GEN=1` in `main.go`.
- **Proposed `clihelp` Feature**:
  Add `clihelp.DocCommand(doc.MarkdownOptions)` (or `clihelp.MarkdownCommand()`), making markdown documentation generation an out-of-the-box subcommand alongside `completion` and `manpage`.
