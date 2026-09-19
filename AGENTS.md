# AGENTS.md — Guidelines for AI-assisted development (Go)

## Start Here

- **Read `llms.txt` first**: It is the top-level conceptual index for the repository.
- **Before modifying resolution, flags, completion, or help**: Read [`docs/how-clihelp-decides.md`](docs/how-clihelp-decides.md) before inspecting the source code.
  - Pay special attention to **"Invariants you can rely on"** (§ Invariants) — they are normative behavioral specifications, not advisory guidelines.
  - Be aware of the distinct option collectors and arity scopes (`scopeLeading` vs `scopePositional` vs `CollectOptions`): do not pick internal helpers without verifying their inclusion/exclusion rules regarding command `Options` and `Hidden` flags.

## Build & Quality

- **Go 1.26+** — package `clihelp` with `example/` demo application
- Build example: `go build -o /dev/null ./example`
- Test: `go test -timeout 30s ./...`
- Lint: `go vet ./...` then `staticcheck ./...`
- Format: `gofmt -s -w .` before committing

## Automation Scripts (`tools/`)

| Script | Purpose |
|--------|---------|
| `tools/audit_lines.rb` | Audit Go source files for function length and file sizing |
| `tools/check.sh` | Full quality gate: format → tidy → vet → staticcheck → test → build example |
| `tools/format.sh` | Run `gofmt -s -w .` only |
| `tools/lint.sh` | Static analysis: `go vet` + `staticcheck` |
| `tools/map.sh` | Print package structure, key types, and exported functions |
| `tools/version.sh` | Print current version from `VERSION` file |
| `tools/bump-version.sh` | Bump patch version in `VERSION`, git add/commit/push |
| `tools/commit.sh <msg>` | Quality gate + stage + commit (silent, outputs "Success <msg>") |
| `tools/checkpoint.sh` | Auto micro-commit of all changes (saves work state) |
| `tools/ex_podcl [args]` | Incrementally build and execute `podctl` example with CLI arguments |
| `tools/ex_mail_cli [args]` | Incrementally build and execute `mail_cli` example with CLI arguments |
| `tools/run_example.sh` | Run the demonstration CLI application (`example/main.go`) |

## Makefile

A `Makefile` at the project root delegates to all scripts:

| Target | Action |
|--------|--------|
| `make audit` | Check function and file line limits |
| `make check` | Full quality gate |
| `make lint` | Static analysis (vet + staticcheck) |
| `make test` | Run tests |
| `make build` | Build example binary |
| `make format` | Format code |
| `make map` | Show architecture overview |
| `make version` | Show current version |
| `make bump` | Bump patch version |
| `make commit` | Quality gate + commit |
| `make push` | Alias for `make bump` |
| `make ci` | Alias for `make check` |
| `make run` | Run example application |
| `make checkpoint` | Micro-commit all changes |
| `make clean` | Clean build artifacts |

### Workflow

```bash
# Standard development loop:
make commit ARGS="feat: add new feature"   # quality gate + commits (silent)
make bump                                   # bumps version, commits, pushes (silent)

# Quick checks without commit:
make check

# Explore architecture before making changes:
make map

# Save work state during long sessions:
make checkpoint

# Run the example CLI application:
make run
```

## Version Management

- Version is stored in `VERSION` file (semver: `major.minor.patch`)
- Current version: `<read from VERSION at build time — kept in sync by tools/check.sh>`
- Run `make bump` to bump patch version, commit, tag, and push version in one step
- `VERSION` file is the single source of truth for release versioning; `example/main.go`'s `Version:` literal must match it and is verified by `make check`

## API Stability & Backward Compatibility

- **Before 1.0, the surface is not frozen.** The library is pre-1.0 and backward compatibility is not yet a promise: a name, field, or function that is wrong should be changed or removed outright. Prefer that to adding a correct alternative beside it — a deprecated alias is a permanent cost paid to avoid a break that is currently free. `review/api-surface-2026-09-18.md` is the audit of what should go.
- **After 1.0**: preserve backward compatibility for all exported types and methods (`App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, `App.Render`, `App.RenderGlobal`, `App.RenderCommand`, `App.LookupCommand`, `App.Walk`), and prefer additive changes — new fields, structs, or methods — over breaking existing signatures or struct field semantics.

## Sizing

Two limits, and they are not equally important:

### Functions — cognitive tiering (GUIDELINES.md)

Follow `/home/sariel/prog/standards/go/GUIDELINES.md` (§3):
- **Standard Logic**: Comfort 20–60 lines, soft warn **80 lines**, hard limit **110 lines**.
- **Declarative Builders** (`build*`, `init*`, `render*`, `generate*`, `View`): Soft warn **120 lines**, hard limit **160 lines** (branches <= 2).
- **Event / Key Dispatchers** (`handle*`, `dispatch*`, `Execute*`, `*Key`, `*Route`): Soft warn **150 lines**, hard limit **200 lines**.
- **Table-Driven Tests** (`Test*`): Soft warn **180 lines**, hard limit **250 lines**.

### Files — comfort metrics, warnings, and hard limits

File length is a *comfort* metric, not a correctness one. Keep functions under their limits and files land in the comfort range on their own:
- **Production files**: Comfort **300–700 lines**, warn > **800 lines**, hard limit **1100 lines**.
- **Test files**: Comfort **300–1000 lines**, warn > **1200 lines**, hard limit **1600 lines**.

### Never split a file through a function body

When a file grows past the warning threshold, **decompose its long functions in place** into named helpers rather than cutting the file underneath an oversized function. Never split a file across a function body.
Enforce sizing via `tools/audit_lines.rb` (`make audit`).

## Output Streams

One rule, for every command and for every `__clihelp` verb:

- **stdout is what a script captures** — a path, a generated script, a version — one item
  per line and nothing else. A command that produces no machine-readable answer writes
  nothing to stdout.
- **stderr is everything addressed to a human** — reports, tips, "restart your shell".
- **A visible command obeys the same rule as its `__clihelp` twin**, so the two are
  interchangeable in a script.

For an interactive user this is invisible, since both streams reach the same terminal. Only
a redirecting caller sees it, and that caller wants it.

## Hard Constraints & Code Hygiene

- **Error Handling & Exit Hygiene**:
  - **Never use `log.Fatalf` or `os.Exit` in libraries or handlers**.
  - All domain logic, storage operations, and command handlers must return errors upward using `fmt.Errorf("context: %w", err)`.
  - Use a centralized `handleError(err)` helper at the presentation/CLI boundary.
  - Reserve `log.Fatalf` strictly for fatal, unrecoverable startup configuration failures in `main.go`.
- **Context Propagation**:
  - Always accept `ctx context.Context` as the first parameter for I/O, database, network calls, and background workers.
  - Never store `Context` in a struct.
- **Goroutine & Resource Management**:
  - Always ensure deterministic cleanup of goroutines, channels, timers, and file descriptors.
  - Use `context.Context` for cancellation, `sync.WaitGroup` for synchronization, and `defer` for resource cleanup.

## Code Style

- Go 1.26+ with minimal dependencies (`github.com/fatih/color`, `github.com/acarl005/stripansi`, `golang.org/x/term`)
- Self-documenting, clean, formatted Go code (`gofmt -s -w .`)
- **No direct ANSI escape codes** in code or tests (`\033`, `\x1b`) — always use external packages (`github.com/fatih/color`, `github.com/acarl005/stripansi`). **Sole exception:** the SGR/OSC8 constants in `inline.go` (neither dependency can emit OSC8 link sequences). Do not add ANSI escapes anywhere else.
- ANSI color formatting for terminal headers and labels
- Terminal width auto-detection with fallback to 70 characters for non-TTY environments
- All functions return clean outputs; no `os.Exit` inside library code

## Testing

- **Table-Driven Tests**: Use table-driven tests (`[]struct{ name string, ... }` with `t.Run(tt.name, func(t *testing.T) { ... })`) as the default testing idiom for Go.
- **Race Detection**: Always run tests with `-race` enabled during verification:
  ```bash
  go test -v -race ./...
  ```

## New Features

- **Tiered Progressive Help (`-h` vs `--help` / `-H`)**: Differentiates concise help (`-h`, suppressing `Notes`, using `Description`, and displaying a footer hint) from extended help (`--help`, `help <cmd>`, or opt-in `-H` via `App.ExtendedHelpFlag` displaying `LongDescription`, parameters, flags, examples, all notes, and paging through `$PAGER`).
- **Extended Command Documentation (`Command.LongDescription`)**: In-depth command documentation rendered in extended help and documentation sites, keeping `Description` concise for listings.
- **Verbatim Text & Fenced Code Blocks (`Note.Raw`)**: Preserves preformatted text, indentation, and markdown code fences in notes and manual pages without word-wrapping or collapsing spaces.
- **Hanging Indentation for Lists**: `reflowSegment` automatically detects bullet lists (`- `, `* `, `• `) and numbered lists (`1. `, `2. `, etc.), aligning continuation lines with hanging indents to the start of the list item text.
- **UV-Style Command Listings**: Command and subcommand index tables render strictly bare command names and aliases without argument or flag signatures, guaranteeing clean single-line scannability.
- **Command Tree Traversal (`App.Walk`)**: Programmatic depth-first traversal of all commands and nested subcommands with path slice isolation and early error-exit for testing and interface coverage.
- **Global Flag De-Cluttering & Topic Routing**: Added `Option.Group` and `Group()` helper to organize options by category, `App.OmitGlobalFlagsInCommands` to suppress verbose global flags in subcommands, and dedicated help topic routing (`help flags`, `help man`, `help topics`).
- **Comprehensive Paged Manual (`help man`)**: Built-in `RenderMan()` renders an exhaustive Unix man page with all commands, subcommands, arguments, flags, and notes paged through `$PAGER`.
- **Pager Support**: When `App.Pager` or `Options.Pager` is true, help output is automatically paged through `$PAGER` when it exceeds terminal height.
- **GNU-Standard Column Formatting**: Two-column command/option listings cap the description column at `DefaultMaxColIndent = 24`, and reduce it further when the terminal is too narrow to leave a usable text column. Long command or flag signatures automatically place description text on the next line, indented to the shared description column — the widest name that fits within `DefaultMaxColIndent`, plus four; `DefaultMaxColIndent` itself when no name fits.
- **Modular Subpackages**: `github.com/sarielhp/clihelp/doc` for GitHub Markdown documentation site generation and `github.com/sarielhp/clihelp/tree` for command hierarchy visualization.
- **Prefix Command Matching**: Added `App.AbbrevCommands` field to enable abbreviated command names (e.g. `podctl b` instead of `podctl build`).
- **Self-Installing Shell Autocompletion**: Added `CompletionCommand()` supporting Bash, Zsh, and Fish with one-command user XDG self-installation. The installer functions themselves are unexported: setup goes through the command, so that the "refresh only, never create" rule has one place to live.

## File Organization

| File / Package | Purpose |
|------|---------|
| `clihelp.go` | Core data types (`App`, `Command`, `Option`, `Param`, `Example`, `Note`, `Context`) and `App.Walk` |
| `topics.go` | Specialized help topic renderers (`RenderFlags`, `RenderMan`, `RenderHelpTopics`, grouped option reflow) |
| `topics_test.go` | Unit tests for topic routing, manual pages, and help flags |
| `render.go` | Terminal help rendering for global app, individual commands, and grouped commands |
| `format.go` | Text layout, word-wrapping, hanging list indentation, ANSI stripping, and column indentation utilities |
| `format_test.go` | Unit tests for word-wrapping, list hanging indents, visual string measurement, and column indent |
| `execute.go` | Flag-set construction, flag validation, execution dispatch, and the run lifecycle |
| `resolve.go` | Command matching and resolution, leading-flag arity scanning, help-token and help-topic routing, and command suggestions |
| `execute_test.go` | Unit tests for command execution, tiered help (`-h` vs `--help` / `-H`), and lifecycle hooks |
| `options.go` | Option builder functions (`Bool`, `String`, `Int`, `Duration`, `Enum`, `StringSlice`) and flag binding |
| `args.go` | Positional argument validators (`ExactArgs`, `RangeArgs`, `MinimumNArgs`, `NoArgs`) |
| `interactive.go` | Interactive prompt fallback for missing required options in TTY environments |
| `validation.go` | Declarative option constraint validation (`MutuallyExclusive`, `RequiredTogether`, etc.) |
| `testing.go` | Testing harnesses (`TestExecute`, `Audit`) for simulating execution and verifying command trees |
| `inline.go` | Inline markdown parsing and ANSI/OSC8 terminal formatting (bold, italic, code, hyperlinks) |
| `pager.go` | Pager detection/execution (`$PAGER`, `less`), terminal height check, and paged output |
| `completion.go` | The `__complete` protocol, dynamic completion, and the three script generators |
| `completion_templates.go` | The generated bash, zsh and fish completion scripts, as shell source |
| `completion_command.go` | The optional `completion` command and the install/uninstall report |
| `autorefresh.go` | `AutoRefreshIntegration`: the only unattended writer, allowed to refresh and never to create |
| `atomicwrite.go` | `writeFileAtomically` — symlink-resolving, fsynced, owner-preserving replace |
| `shell.go` | `resolveShell` — the one answer to "which shell, and can we write for it?" |
| `versions.go` | Every version number stamped into a generated artifact, in one place |
| `explain.go` | Command-line expansion and the height-capped explanation behind Alt-H (`__explain`, `App.Explain`, `GenKeyBindings`) |
| `protocol.go` | The reserved `__clihelp` setup verbs (version, install, uninstall, keys, wrapper, manpage), their shared argument parser, and wrapper-script generation |
| `names.go` | `safeAppName` — the one gate for any name that becomes a file path, a shell symbol or an rc-file marker — plus the shell quoters |
| `lock_unix.go`, `lock_other.go` | Advisory locking for the startup-file read-modify-write |
| `install.go` | One-command setup: the generated per-shell file, the marked startup-file block, install/uninstall/refresh, and the manual page that goes with them |
| `man.go` | roff manual page generation (`GenManPage`), installation under `$XDG_DATA_HOME/man`, and `ManPageCommand` |
| `completion_test.go` | Unit tests for shell completion protocol, installation, and shared completion helpers |
| `completion_bash_test.go` | Live Bash tab-completion integration and dynamic callback tests |
| `completion_zsh_test.go` | Live Zsh tab-completion integration and dynamic callback tests |
| `completion_fish_test.go` | Live Fish tab-completion integration and dynamic callback tests |
| `completion_install_test.go` | Shell detection, installation, script version markers, and completion descriptions |
| `install_test.go` | Shell-integration install/uninstall, startup-file editing, refresh, and concurrency |
| `ownership_test.go` | That nothing overwrites or deletes a file clihelp did not write |
| `atomicwrite_test.go` | Symlinked dotfiles and the atomic-replace contract |
| `autopath_test.go` | What an ordinary program run is and is not allowed to do |
| `sandbox_test.go` | `TestMain`'s guard that the suite never writes to the real home directory |
| `shell_harness_test.go` | Drives the generated zsh and fish code headlessly by stubbing `zle`, `bindkey`, `bind` and `commandline` |
| `install_live_test.go` | Starts a real zsh and fish against a sandboxed home and checks what the install left them |
| `install_manpage_test.go` | That one `install` sets everything up, and one `uninstall` removes it |
| `shell_test.go` | That every entry point resolves a shell the same way |
| `man_test.go` | roff generation, escaping, installation, and live `man` rendering |
| `protocol_test.go` | The `__clihelp` verbs, their argument grammar, and wrapper generation |
| `explain_test.go`, `explain_shell_test.go` | Command-line expansion, the height budget, and the live shell key bindings |
| `shellquote_test.go` | Shell quoting of generated source and `App.Name` validation |
| `doc/` | Subpackage for GitHub-friendly markdown documentation site generation (`doc.RenderMarkdown`) |
| `docs/` | User and developer documentation guides, site index, and generated markdown reference sites |
| `docs_drift_test.go` | Guard verifying that documentation prose only names real exported symbols and members |
| `tree/` | Subpackage for command hierarchy tree visualization (`tree.Render`) |
| `audit.go` | Static analysis audit (`Audit`) verifying command uniqueness, flag collision, and parameter invariants |
| `examples.go` | Example command syntax colorizer, shell tokenizer, and static example validator (`ValidateExample`, `ValidateAllExamples`) |
| `examples_test.go` | Unit tests for example shell splitting, ANSI syntax colorization, and CLI constraint validation |
| `clihelp_test.go` | Unit tests for help formatting, command dispatch, ANSI stripping, and usage output |
| `walk_test.go` | Unit tests for `App.Walk` depth-first traversal, path isolation, and error propagation |
| `example/main.go` | Demonstration CLI app (`podctl`) using `clihelp` |
| `example/main_test.go` | Testing demonstration verifying command coverage, leaf usage lines, examples, and smoke rendering |
| `Makefile` | Make targets for standard workflows |
| `VERSION` | Version source of truth |
| `CHANGES.md` | Version changelog |
| `tools/` | Automation shell and ruby scripts |

### Documentation Guides (`docs/`)

The `docs/` directory contains GitHub Pages/Jekyll documentation and guides for the library:

| File / Subdirectory | Purpose |
|---------------------|---------|
| `docs/index.md` | Documentation hub and guide index |
| `docs/how-clihelp-decides.md` | Decision matrix: resolution rules, help-tier selection, line budgets, and error reporting |
| `docs/flags-and-options.md` | Flags and options: option constructors, flag specifications, groups, toggles, and arity |
| `docs/lifecycle-and-routing.md` | Command lifecycle: resolution, leading flags, argument validators, and error routing |
| `docs/completion.md` | Shell completion: installation protocols, Bash/Zsh/Fish script generation, and dynamic callbacks |
| `docs/recipes-and-patterns.md` | Idiomatic recipes: command patterns, configuration binding, interactive prompts, and migration |
| `docs/comparison-with-cobra.md` | Architectural differences, cognitive complexity limits, and trade-offs compared to `spf13/cobra` |
| `docs/markdown-generation.md` | Documentation site generation guide via the `doc` subpackage (`doc.RenderMarkdown`) |
| `docs/clihelp/` | Generated Markdown documentation site for the `clihelp` tool itself |
| `docs/mail_cli_fake/` | Generated Markdown documentation site for the demonstration `mail_cli` command tree |
| `docs/mailcli/` | Generated Markdown documentation site for `mailcli` |


**The shell-integration surface is layered, and the layering is load-bearing.** It was a
dependency cycle across five files until it was untangled; keep it acyclic:

1. **Leaves** — `shell.go`, `names.go`, `versions.go`, `atomicwrite.go`,
   `completion_templates.go`. They depend on nothing above them.
2. **Generators** — `completion.go`, `explain.go`, `man.go`. They produce text and touch no
   files.
3. **Installers** — `install.go`, `autoinstall.go`. They write files, using layer 2.
4. **Commands** — `protocol.go`, `completion_command.go`. Presentation over layer 3.

A reference from a lower layer to a higher one is the cycle coming back. If a constant is
what you need from above, move the constant down — that is why the `__complete` / `__explain`
/ `__clihelp` names live in `names.go` and not in `protocol.go`.

## Agent Development Rules

1. **Verification**: After modifying any Go file, run `make check` to verify formatting, vet, lint, tests, and build.
2. **Error Resolution**: If `make check` fails, focus on fixing the first reported error before making additional changes.
3. **Exploration**: Run `make map` before introducing new types or functions to inspect existing API signatures.
4. **Checkpointing**: Run `make checkpoint` after passing checks to preserve working states during long sessions.
5. **No Direct ANSI Codes**: Do not hardcode ANSI escape sequences (`\033`, `\x1b`) in source or test files — use `fatih/color` or `stripansi`.
6. **Backward Compatibility**: After 1.0, maintain strict backward compatibility for exported APIs and prefer additive fields or methods to modified public signatures. Before 1.0 — where the library is now — a wrong name or a redundant entry point should be removed rather than aliased.
7. **Commit Messages**: Use conventional commits format (`feat:`, `fix:`, `chore:`, `docs:`, `test:`).

## AI Agent Keywords

These keywords can be used to trigger specific workflows when working with AI agents:

- `audit`: Run file line counts, staticcheck, and error propagation checks.
- `harden`: Add edge-case table tests (nil pointers, context cancellation, empty slices, malformed inputs).
- `simplify`: Remove dead code, redundant abstractions, and simplify complex loops/conditionals without changing behavior.
