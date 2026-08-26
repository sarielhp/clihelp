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

- **Stable Interface**: Preserve backward compatibility for all exported types and methods (`App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, `App.Render`, `App.RenderGlobal`, `App.RenderCommand`, `App.LookupCommand`).
- **Additive Changes**: Adding new fields, structs, or methods is encouraged. Avoid breaking existing function signatures or struct field semantics in future development.

## File Sizing

- **Target**: 150–300 lines per file (~1.5k–3k tokens)
- **Trigger**: Split into logical units when exceeding 600 lines
- **Rationale**: Keeping files under 300 lines maintains focused AI agent context and improves AST editing accuracy.

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

- **Pager Support**: When `App.Pager` or `Options.Pager` is true, help output is automatically paged through `$PAGER` when it exceeds terminal height. Automatically injects `-no-linenumbers` when `moar` is detected as the pager.
- **GNU-Standard Column Formatting**: Two-column command/option listings cap the description column at `DefaultMaxColIndent = 24`. Long command or flag signatures automatically place description text on the next line indented at column 24.
- **Command Tree View**: Added `App.RenderTree()` method to render the full command hierarchy as a tree with box-drawing characters.
- **Prefix Command Matching**: Added `App.AbbrevCommands` field to enable abbreviated command names (e.g. `podctl b` instead of `podctl build`).

## File Organization

| File | Purpose |
|------|---------|
| `clihelp.go` | Core data types (`App`, `Command`, `Option`, `Param`, `Example`, `Note`, `Context`) |
| `render.go` | Terminal help rendering for global app, individual commands, command tree, and grouped commands |
| `format.go` | Text layout, word-wrapping, string reflow, ANSI stripping, and column indentation utilities |
| `execute.go` | Command lookup, flag parsing, command execution dispatch, alias handling, and error formatting |
| `options.go` | Option builder functions (`Bool`, `String`, `Int`, `Duration`, `Enum`, `StringSlice`) and flag binding |
| `inline.go` | Inline markdown parsing and ANSI/OSC8 terminal formatting (bold, italic, code, hyperlinks) |
| `pager.go` | Pager detection/execution (`$PAGER`, `less`, `moar`), terminal height check, and paged output |
| `completion.go` | Shell autocompletion script generation (Bash and Zsh) |
| `md.go` | GitHub-friendly markdown documentation generator (`RenderMarkdown`, `MarkdownOptions`) |
| `clihelp_test.go` | Unit tests for help formatting, wrapping, ANSI stripping, usage output, and tree rendering |
| `example/main.go` | Demonstration CLI app using `clihelp` |
| `Makefile` | Make targets for standard workflows |
| `VERSION` | Version source of truth |
| `CHANGES.md` | Version changelog |
| `tools/` | Automation shell scripts |

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
