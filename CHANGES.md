# Changelog

All notable changes to `clihelp` will be documented in this file.

## [Unreleased]

## [0.1.1] - 2026-08-13

### Added
- Added receiver methods on `*App` (`a.PrintGlobalUsage()`, `a.PrintCommandUsage(path...)`, `a.PrintUsage(path...)`, `a.LookupCommand(path...)`) for a clean, object-oriented Go API surface.
- Added comprehensive Go doc comments across all exported structs, fields, and functions in `clihelp.go`.
- Extensively documented `example/main.go` step-by-step to demonstrate how external applications should integrate `clihelp`.
- Rewrote `README.md` into a complete package guide featuring API reference tables, multi-level subcommand examples, and quick start instructions.
- Added recursive subcommand support (`Subcommands []Command`) to `clihelp` with variadic command path resolution in `PrintCommandUsage(a *App, path ...string)`.
- Added nested `config set` subcommands in `example/main.go` supporting `time`, `space`, and `location` attributes with recursive help dispatching and flag options.
- Added unit tests in `clihelp_test.go` verifying multi-level nested subcommand help generation.
- Expanded demonstration CLI application in `example/main.go` (`podctl`) with multiple subcommands (`build`, `serve`, `config`, `deploy`, `status`), extensive options, usage examples, and command-line help dispatching.
- Improved `describeLabel` multi-line description alignment and tuned command column width.
- Replaced custom ANSI stripping loop and direct escape code literals (`\033`, `\x1b`) with external package `github.com/acarl005/stripansi`.
- Added rule in `AGENTS.md` prohibiting direct ANSI escape codes in favor of external packages (`fatih/color`, `stripansi`).
- Added green bold ANSI color styling (`color.FgGreen, color.Bold`) for subcommand names under `COMMANDS`.
- Added ANSI-aware padding calculation (`formatPadded`) to ensure aligned description columns when labels include ANSI escape codes.
- Added `AGENTS.md` guidelines for AI-assisted development adapted for `clihelp`.
- Added automation scripts in `scripts/` (`check.sh`, `format.sh`, `lint.sh`, `map.sh`, `version.sh`, `bump-version.sh`, `commit.sh`, `checkpoint.sh`, `run_example.sh`).
- Added `Makefile` for standard development tasks (`make check`, `make lint`, `make test`, `make map`, `make run`, etc.).
- Added `VERSION` file (`0.1.0`) as single source of truth for versioning.
- Added unit tests in `clihelp_test.go`.
- Added `CHANGES.md` project changelog.


## [0.1.0] - 2026-08-10

### Added
- Initial release of `clihelp` Go package.
- Colored section headers (`USAGE`, `COMMANDS`, `OPTIONS`, `EXAMPLES`) powered by `fatih/color`.
- Terminal width auto-detection and ANSI-aware text wrapping via `golang.org/x/term`.
- Core data structures: `App`, `Command`, `Option`, and `Example`.
- `PrintGlobalUsage` for displaying global application overview with registered commands.
- `PrintCommandUsage` for detailed command help text, flags, and usage examples.
- Sample command-line application in `example/main.go`.
