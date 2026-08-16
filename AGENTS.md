# AGENTS.md — Guidelines for AI-assisted development (Go)

## Build & Quality

- **Go 1.26+** — package `clihelp` with `example/` demo application
- Build example: `go build -o /dev/null ./example`
- Test: `go test -timeout 30s ./...`
- Lint: `go vet ./...` then `staticcheck ./...`
- Format: `gofmt -s -w .` before committing

## Automation Scripts (`scripts/`)

| Script | Purpose |
|--------|---------|
| `scripts/check.sh` | Full quality gate: format → tidy → vet → staticcheck → test → build example |
| `scripts/format.sh` | Run `gofmt -s -w .` only |
| `scripts/lint.sh` | Static analysis: `go vet` + `staticcheck` |
| `scripts/map.sh` | Print package structure, key types, and exported functions |
| `scripts/version.sh` | Print current version from `VERSION` file |
| `scripts/bump-version.sh` | Bump patch version in `VERSION`, git add/commit/push |
| `scripts/commit.sh <msg>` | Quality gate + stage + commit (silent, outputs "Success <msg>") |
| `scripts/checkpoint.sh` | Auto micro-commit of all changes (saves work state) |
| `scripts/run_example.sh` | Run the demonstration CLI application (`example/main.go`) |

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
- Current version: `0.2.0`
- Run `make bump` to bump patch version, commit, tag, and push version in one step
- `VERSION` file is the single source of truth for release versioning

## API Stability & Backward Compatibility

- **Stable Interface**: Preserve backward compatibility for all exported types and methods (`App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, `App.Render`, `App.RenderGlobal`, `App.RenderCommand`, `App.LookupCommand`).
- **Additive Changes**: Adding new fields, structs, or methods is encouraged. Avoid breaking existing function signatures or struct field semantics in future development.

## File Sizing

- **Target**: 150–300 lines per file (~1.5k–3k tokens)
- **Trigger**: Split into logical units when exceeding 600 lines
- **Rationale**: Keeping files under 300 lines maintains focused AI agent context and improves AST editing accuracy.

## Code Style

- Go 1.26+ with minimal dependencies (`github.com/fatih/color`, `github.com/acarl005/stripansi`, `golang.org/x/term`)
- Self-documenting, clean, formatted Go code (`gofmt -s -w .`)
- **No direct ANSI escape codes** in code or tests (`\033`, `\x1b`) — always use external packages (`github.com/fatih/color`, `github.com/acarl005/stripansi`)
- ANSI color formatting for terminal headers and labels
- Terminal width auto-detection with fallback to 80 characters for non-TTY environments
- All functions return clean outputs; no `os.Exit` inside library code

## File Organization

| File | Purpose |
|------|---------|
| `clihelp.go` | Core CLI help printing logic, section layout, string wrapping, and terminal width helpers |
| `md.go` | GitHub-friendly markdown help-page generator (RenderMarkdown, MarkdownOptions) |
| `clihelp_test.go` | Unit tests for formatting, wrapping, ANSI stripping, usage output, and markdown generation |
| `example/main.go` | Demonstration CLI app using `clihelp` |
| `Makefile` | Make targets for standard workflows |
| `VERSION` | Version source of truth |
| `CHANGES.md` | Version changelog |
| `scripts/` | Automation shell scripts |

## Agent Development Rules

1. **Verification**: After modifying any Go file, run `make check` to verify formatting, vet, lint, tests, and build.
2. **Error Resolution**: If `make check` fails, focus on fixing the first reported error before making additional changes.
3. **Exploration**: Run `make map` before introducing new types or functions to inspect existing API signatures.
4. **Checkpointing**: Run `make checkpoint` after passing checks to preserve working states during long sessions.
5. **No Direct ANSI Codes**: Do not hardcode ANSI escape sequences (`\033`, `\x1b`) in source or test files — use `fatih/color` or `stripansi`.
6. **Backward Compatibility**: Maintain strict backward compatibility for exported APIs. Introduce non-breaking additive fields or methods rather than modifying existing public signatures.
7. **Commit Messages**: Use conventional commits format (`feat:`, `fix:`, `chore:`, `docs:`, `test:`).
