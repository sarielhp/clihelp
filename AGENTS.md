# AGENTS.md — Guidelines for AI-assisted development (Go)

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

- **Stable Interface**: Preserve backward compatibility for all exported types and methods (`App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, `App.Render`, `App.RenderGlobal`, `App.RenderCommand`, `App.LookupCommand`, `App.Walk`).
- **Additive Changes**: Adding new fields, structs, or methods is encouraged. Avoid breaking existing function signatures or struct field semantics in future development.

## Sizing

Two limits, and they are not equally important:

### Functions — hard limit 80 lines (exceptions apply)

- **Production functions** (`*.go`, excluding `*_test.go`): Hard limit **80 lines**. This limit protects correctness, so when it conflicts with anything else, it wins. Extracting a function is a *semantic* edit: the extracted piece needs a name, parameters and return values, and the compiler checks every call site.
  - **Declarative builders exception**: Functions named `build*` with cyclomatic branches <= 2 have a relaxed limit of **150 lines** (e.g. static CLI command tree builders).
- **Test functions** (`*_test.go`): Relaxed limit of **200 lines** to permit thorough table-driven test cases without unnatural fragmentation.

### Files — comfort metrics, warnings, and hard limits

File length is a *comfort* metric, not a correctness one. Keep functions under their limits and files land in the comfort range on their own:
- **Production files**: Comfort **300–700 lines**, warn > **800 lines**, hard limit **1100 lines**.
- **Test files**: Comfort **300–1000 lines**, warn > **1200 lines**, hard limit **1600 lines**.

### Never split a file through a function body

When a file grows past the warning threshold, **decompose its long functions in place** into named helpers rather than cutting the file underneath an oversized function. Never split a file across a function body.
Enforce sizing via `tools/audit_lines.rb` (`make audit`).

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

- **UV-Style Command Listings**: Command and subcommand index tables render strictly bare command names and aliases without argument or flag signatures, guaranteeing clean single-line scannability.
- **Command Tree Traversal (`App.Walk`)**: Programmatic depth-first traversal of all commands and nested subcommands with path slice isolation and early error-exit for testing and interface coverage.
- **Global Flag De-Cluttering & Topic Routing**: Added `Option.Group` and `Group()` helper to organize options by category, `App.OmitGlobalFlagsInCommands` to suppress verbose global flags in subcommands, and dedicated help topic routing (`help flags`, `help man`, `help topics`).
- **Comprehensive Paged Manual (`help man`)**: Built-in `RenderMan()` renders an exhaustive Unix man page with all commands, subcommands, arguments, flags, and notes paged through `$PAGER`.
- **Pager Support**: When `App.Pager` or `Options.Pager` is true, help output is automatically paged through `$PAGER` when it exceeds terminal height.
- **GNU-Standard Column Formatting**: Two-column command/option listings cap the description column at `DefaultMaxColIndent = 24`. Long command or flag signatures automatically place description text on the next line indented at column 24.
- **Modular Subpackages**: `github.com/sarielhp/clihelp/doc` for GitHub Markdown documentation site generation and `github.com/sarielhp/clihelp/tree` for command hierarchy visualization.
- **Prefix Command Matching**: Added `App.AbbrevCommands` field to enable abbreviated command names (e.g. `podctl b` instead of `podctl build`).
- **Self-Installing Shell Autocompletion**: Added `CompletionCommand()` and `InstallCompletion()` supporting Bash, Zsh, and Fish with one-command user XDG self-installation.

## File Organization

| File / Package | Purpose |
|------|---------|
| `clihelp.go` | Core data types (`App`, `Command`, `Option`, `Param`, `Example`, `Note`, `Context`) and `App.Walk` |
| `topics.go` | Specialized help topic renderers (`RenderFlags`, `RenderMan`, `RenderHelpTopics`, grouped option reflow) |
| `render.go` | Terminal help rendering for global app, individual commands, and grouped commands |
| `format.go` | Text layout, word-wrapping, string reflow, ANSI stripping, and column indentation utilities |
| `format_test.go` | Unit tests for word-wrapping, line reflow, visual string measurement, and column indent |
| `execute.go` | Command lookup, flag parsing, command execution dispatch, alias handling, and error formatting |
| `options.go` | Option builder functions (`Bool`, `String`, `Int`, `Duration`, `Enum`, `StringSlice`) and flag binding |
| `inline.go` | Inline markdown parsing and ANSI/OSC8 terminal formatting (bold, italic, code, hyperlinks) |
| `pager.go` | Pager detection/execution (`$PAGER`, `less`), terminal height check, and paged output |
| `completion.go` | Shell autocompletion script generation (Bash, Zsh, Fish), dynamic completion, and XDG auto-installation |
| `completion_test.go` | Unit tests for shell completion protocol, installation, and shared completion helpers |
| `completion_bash_test.go` | Live Bash tab-completion integration and dynamic callback tests |
| `completion_zsh_test.go` | Live Zsh tab-completion integration and dynamic callback tests |
| `completion_fish_test.go` | Live Fish tab-completion integration and dynamic callback tests |
| `doc/` | Subpackage for GitHub-friendly markdown documentation site generation (`doc.RenderMarkdown`) |
| `tree/` | Subpackage for command hierarchy tree visualization (`tree.Render`) |
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

## Agent Development Rules

1. **Verification**: After modifying any Go file, run `make check` to verify formatting, vet, lint, tests, and build.
2. **Error Resolution**: If `make check` fails, focus on fixing the first reported error before making additional changes.
3. **Exploration**: Run `make map` before introducing new types or functions to inspect existing API signatures.
4. **Checkpointing**: Run `make checkpoint` after passing checks to preserve working states during long sessions.
5. **No Direct ANSI Codes**: Do not hardcode ANSI escape sequences (`\033`, `\x1b`) in source or test files — use `fatih/color` or `stripansi`.
6. **Backward Compatibility**: Maintain strict backward compatibility for exported APIs. Introduce non-breaking additive fields or methods rather than modifying existing public signatures.
7. **Commit Messages**: Use conventional commits format (`feat:`, `fix:`, `chore:`, `docs:`, `test:`).

## AI Agent Keywords

These keywords can be used to trigger specific workflows when working with AI agents:

- `audit`: Run file line counts, staticcheck, and error propagation checks.
- `harden`: Add edge-case table tests (nil pointers, context cancellation, empty slices, malformed inputs).
- `simplify`: Remove dead code, redundant abstractions, and simplify complex loops/conditionals without changing behavior.
