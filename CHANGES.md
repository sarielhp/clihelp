# Changelog

All notable changes to `clihelp` will be documented in this file.

## [0.2.10] - 2026-08-18

### Added
- Demonstrated clickable OSC 8 hyperlinks and inline formatting in `example/main.go` and `example_test.go` (`ExampleApp_Render`).

## [0.2.9] - 2026-08-18

### Changed
- Clarified and refined package description at the top of `README.md` and `llms.txt`.

## [0.2.7] - 2026-08-18

### Added
- Prominently integrated `llms.txt` and AI optimization documentation across `README.md`, `docs/index.md`, and `docs/ai-guidelines.md`.

## [0.2.6] - 2026-08-18

### Added
- In-depth architectural comparison guide with `spf13/cobra` in `docs/comparison-with-cobra.md` covering global state tradeoffs, declarative vs imperative patterns, and terminal aesthetics.

## [0.2.5] - 2026-08-18

### Added
- Standardized `llms.txt` AI specification at repository root for single-fetch LLM consumption.
- Testable Go examples in `example_test.go` (`ExampleApp_Execute`, `ExampleBoolToggle`, `ExampleExactArgs`) for `pkg.go.dev`.
- Automatic help flag collision protection: Intercepts accidental `-h`/`--help` declarations in `Option` constructors and returns actionable error messages.

## [0.2.0] - 2026-08-15

### Changed (breaking rendering API)
- Unified the two renderers (the original classic layout and the mail_cli-style detailed layout) into a single theme-driven engine. The old printing methods (`PrintUsage`, `PrintGlobalUsage`, `PrintCommandUsage`, `PrintSection`, `Print*To`) and helpers (`wrapText`, `indentLines`, `describeLabel`, `describeFlags`) were removed in favor of:
  - `(*App).Render(o Options, path ...string) bool` — dispatch global vs command help.
  - `(*App).RenderGlobal(o Options)` — application overview.
  - `(*App).RenderCommand(o Options, path ...string) bool` — detailed command page.
- Rendering is now `io.Writer`-first: `Options.Writer` (defaults to stdout) replaces hardcoded stdout output.
- Render width is deterministic and injectable via `Options.Width` (0 = auto-detect with a 70-column fallback), replacing the two previously inconsistent width functions and the separate 80/70 defaults.
- One ANSI-aware word reflow (`reflow`) replaces the diverging `wrapText` and reflow implementations, so all sections wrap identically.
- `Theme` (colors, `Separator`, `TitlePrefix`) now drives all styling via `App.Theme` or `Options.Theme`; nil color fields fall back to the default mail_cli palette.

### Remaining API
- `App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, and `(*App).LookupCommand` are unchanged.

### Fixed
- `reflow` and `colIndent` now measure **visible width** (ANSI-stripped runes) instead of raw byte length, so multi-byte runes and colored labels wrap and align correctly.
- `App.Description` and `App.GlobalNote` are now rendered on the global overview (when set).
- `Example.Description` is now rendered beneath its example line (when set).
- Corrected the `Theme` and `reflow` doc comments (`Theme`'s zero value is safe; nil colors fall back to defaults).

### Notes
- The `example/mailcli` reconstruction still reproduces mail_cli's usage pages **byte-for-byte** (verified by the oracle test), now via the unified `Render` API.

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
