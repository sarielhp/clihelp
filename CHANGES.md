# Changelog

All notable changes to `clihelp` will be documented in this file.

## [0.3.1] - 2026-08-29

### Added
- **Example Syntax Colorization & Theming** - Examples in command help, global overview, and manual pages are now rendered with ANSI syntax highlighting for commands, flags, arguments, values, comments, and prompts (`Theme.ExampleCmd`, `Theme.ExampleFlag`, `Theme.ExampleArg`, `Theme.ExampleComment`, `Theme.ExampleDesc`).
- **Static Example Validation & CLI Parsing** - Added `clihelp.ValidateExample`, `(*App).ValidateExamples()`, `(*App).ValidateAllExamples()`, and POSIX shell tokenizer `SplitExampleCommandLine` to statically parse and verify example strings against real flag specs, option validators, and positional argument constraints.
- **Example Tree Auditing** - `clihelp.Audit(app)` and `clihelp.AuditWithOptions` now automatically validate all examples declared on the application and across the entire command hierarchy.
- **Top-Level Application Examples** - Added `App.Examples` field rendered under an `Examples:` section in `RenderGlobal`, `RenderMan`, and `RenderMarkdown`.
- **Global Flag De-Cluttering & Topic Routing** - Added `Option.Group` and `clihelp.Group()` to categorize options under section headings, and `App.OmitGlobalFlagsInCommands` to suppress verbose global flags in subcommand help.
- **Dedicated Flags Directory (`help flags`)** - Added `App.RenderFlags()` (accessible via `help flags` or `help options`) to display an exhaustive, categorized reference for all global and persistent options.
- **Comprehensive Paged Manual (`help man`)** - Added `App.RenderMan()` (accessible via `help man` or `help all`) to render a full Unix manual with all commands, subcommands, arguments, flags, and examples through `$PAGER`.
- **Help Topic Index (`help topics`)** - Added `App.RenderHelpTopics()` (accessible via `help topics` or `help help`) to list available help topics.
- **Fish Shell Autocompletion** - Added `clihelp.GenFishCompletion(app, writer)` generator with native tab-separated description formatting.
- **Zero-Touch Auto-Installation** - Added `App.AutoInstallCompletion` field, `clihelp.CompletionPath()`, and `clihelp.IsCompletionInstalled()` to silently drop shell autocompletion scripts into standard user XDG directories on the first interactive execution.
- **Auto-Installation Support** - Added `clihelp.InstallCompletion(app, shell)` for installing Bash, Zsh, and Fish completions directly to standard XDG user directories without root permissions.
- **Pre-Built `CompletionCommand`** - Added `clihelp.CompletionCommand()` factory function returning ready-to-mount subcommands for `bash`, `zsh`, `fish`, and `install`.
- **Zsh Autocompletion Robustness** - Handled dynamic `compdef` registration, colon escaping in descriptions for `_describe`, and cursor-aware word slicing.

## [0.3.0] - 2026-08-28

### Added
- **Declarative Options Validation** - Added support for attaching an `OptionsValidator` callback to `Command` using built-in rule helpers (`MutuallyExclusive`, `RequiredTogether`, `RequiredWith`, and `RequiredIf`).
- **Required Option Constraints** - Added support for marking individual flags as required using `clihelp.Required()` which automatically appends `(required)` to help descriptions and returns a validation error if omitted.
- **Interactive Fallback** - When `App.InteractiveFallback` is true and execution runs in a terminal (TTY), `clihelp` automatically prompts the user interactively to collect missing required options.
- **CLI Constructor Tip** - After collecting missing required flags interactively, `clihelp` prints an educational shortcut tip showing how to bypass prompts in future executions.
- **Option Deprecations** - Added option deprecation message formatting in help pages, and warning prints during CLI execution if a deprecated flag is supplied.
- **Unit Testing Harness** - Added `clihelp.TestExecute` and `clihelp.TestExecuteWithStdin` to run mock executions and assert output/error content cleanly.
- **Command Tree Audit** - Added `clihelp.Audit` and `clihelp.AuditWithOptions` to statically verify command trees (valid descriptions, no collisions, and consistent subcommand path ordering).
- **Custom Flag Coloring** - Added a configurable `Flag` color field to `Theme` (defaults to Cyan).

## [0.2.19] - 2026-08-25

### Added
- **Pager support** - When `App.Pager` or `Options.Pager` is true, help output is automatically paged through `$PAGER` when it exceeds terminal height.
- **Command tree view** - Added `App.RenderTree()` method to render the full command hierarchy as a tree with box-drawing characters.
- **Prefix command matching** - Added `App.AbbrevCommands` field to enable abbreviated command names (e.g. `podctl b` instead of `podctl build`).
- `Options.MaxContentWidth` (default 80) makes the content wrap cap configurable (`min(termWidth, indent+MaxContentWidth)`).
- `Command.Group` group headings in global command lists and structural subcommand lists.
- `parseFlagSpec` now accepts `--flag=VALUE` specs.
- `tools/check.sh` verifies `example/main.go`'s `Version:` literal matches the `VERSION` file.
- CJK-aware column measurement via `go-runewidth` (wide East-Asian chars count as two columns).

### Fixed
- `Enum` rejects a default value outside the allowed set at bind time.
- `StringSlice` no longer aliases the caller's slice backing array.
- Root `Run` handlers now receive positional args even when subcommands exist.
- `--version` returns an error when `App.Version` is empty instead of silently exiting 0.
- `Options.width()` probes the render Writer when it is a terminal before falling back to stdout.
- `splitLines` strips trailing CR for CRLF input.
- Markdown pages/nav/index skip `Hidden` commands; slug collisions error instead of silently overwriting; front-matter title/parent are YAML-quoted.
- Completion supports `--flag=` prefixes, de-duplicates root command/shortcut names, and propagates `resolveCommand` errors.
- `Command` tree traversal unified through `findCommand`; option collection unified through `App.collectOptions`.
- Example Run handlers write to `ctx.Stdout` instead of `fmt.Printf`.

## [0.2.17] - 

### Added
- `Options.MaxContentWidth` (default 80) makes the content wrap cap configurable (`min(termWidth, indent+MaxContentWidth)`).
- `Command.Group` group headings in global command lists and structural subcommand lists.
- `parseFlagSpec` now accepts `--flag=VALUE` specs.
- `tools/check.sh` verifies `example/main.go`'s `Version:` literal matches the `VERSION` file.
- CJK-aware column measurement via `go-runewidth` (wide East-Asian chars count as two columns).

### Fixed
- `Enum` rejects a default value outside the allowed set at bind time.
- `StringSlice` no longer aliases the caller's slice backing array.
- Root `Run` handlers now receive positional args even when subcommands exist.
- `--version` returns an error when `App.Version` is empty instead of silently exiting 0.
- `Options.width()` probes the render Writer when it is a terminal before falling back to stdout.
- `splitLines` strips trailing CR for CRLF input.
- Markdown pages/nav/index skip `Hidden` commands; slug collisions error instead of silently overwriting; front-matter title/parent are YAML-quoted.
- Completion supports `--flag=` prefixes, de-duplicates root command/shortcut names, and propagates `resolveCommand` errors.
- `Command` tree traversal unified through `findCommand`; option collection unified through `App.collectOptions`.
- Example Run handlers write to `ctx.Stdout` instead of `fmt.Printf`.

## [0.2.15] - 

### Added
- Green subcommand names in command and global help via `Theme.Subcommand` (default green).
- Per-section wrap width: `wrapWidth(termWidth, indent) = min(termWidth, indent+80)` so indented lists can use more horizontal space.
- Exported `Inline` helper for rendering inline markdown to ANSI/OSC8 strings.
- `RenderCommand` now includes app-level and ancestor persistent options in the Flags section.

### Fixed
- `UsageLine` and `Examples` now pass through inline markdown rendering (no raw `**` or visible URLs).
- Oracle test (`example/mail_cli_fake`) updated to match green subcommands, per-section wrap width, and inline rendering.

## [0.2.13] - 
n### Fixed
- Fixed nested help double-render for `podctl config help` and similar nested commands
- Fixed `-v` version flag hijacking `-v, --verbose` persistent flags
- Fixed `App.GlobalFlags` not being parsed
- Fixed `Theme.Separator` dead code
- Fixed `BoolToggle` duplicate registration panicking instead of returning friendly error
- Fixed markdown subcommand alias links broken
- Fixed markdown tables unescaped `|` in descriptions
- Fixed silent `help <unknown>` behavior
- Fixed `PrintError` bypassing `App.Stderr`
- Fixed execute-path help ignoring custom theme
- Fixed markdown command pages omitting app/ancestor persistent flags

### Changed
- Updated example to version 0.2.13

## [0.2.11] - 2026-08-18

### Added
- Expanded Cobra comparison with detailed breakdown of terminal styling, plain-text defaults, and ANSI-aware width wrapping.
- Enriched `example/main.go` demonstrating all inline Markdown formatting features (bold, italic, code, strikethrough, links, notes).
- Added pre-checks in `bindHelper` for duplicate flag and shorthand declarations across parent and child flagsets.

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
