# Revised Design Plan: Safe Wrapper Generation & Author Controls

**Target Repository:** [`github.com/sarielhp/clihelp`](file:///home/sariel/prog/26/go/clihelp)  
**Date:** 2026-09-23  
**Status:** Approved for Implementation (post-critical review)

---

## 1. Executive Summary & Review Verdict

A critical systems review evaluated the proposals from [`plans/clihelp_install.md`](file:///home/sariel/prog/26/go/clihelp/plans/clihelp_install.md) and identified three fatal issues with automated in-place wrapper installation:
1. **Ownership Violation**: In-place mutation of caller scripts breaks `clihelp`'s core ownership invariant (`ownership_test.go`: *"Does not carry our marker means someone else wrote it"*), risking user-defined environment variables, secrets, and cleanup traps.
2. **Process Model Hostility**: `$PPID` inspection is non-portable (fails on Darwin/BSD) and completely collapses when wrappers use idiomatic `exec`.
3. **Upgrade Erasure**: Shell completions for wrappers cannot be safely merged into `~/.config/<app>/shell/<shell>` without being erased on the next binary upgrade by `AutoRefreshIntegration`.

### Features That Survived Critical Review
1. **`App.DisableSetup` (Author Sovereignty & Compliance)**: An explicit opt-out field allowing application authors to disable the `__clihelp` setup verbs in restricted, enterprise, or embedded environments.
2. **Heuristic Wrapper Inspection (`__clihelp wrapper --from <path_or_cmd>`)**: A safe, read-only analyzer that reads an existing wrapper script, extracts preset arguments, and outputs a robust, self-answering `clihelp` wrapper script to `stdout` without touching the filesystem.
3. **Optional Visible `completion wrap` Subcommand**: Exposes wrapper generation under `CompletionCommand()` for applications whose authors opt in.
4. **Refreshed & Themed `__clihelp` Interface**: Upgrades the raw, unstyled `__clihelp` screen with theme colors, clean usage sections, non-repetitive verb tables, shell detection indicators, and examples.

---

## 2. Feature Specification

### Feature 1: `App.DisableSetup`

#### Motivation
Enterprise tools, container sidecars, and embedded/kiosk CLIs often run in restricted environments where commands writing to `$HOME` or modifying dotfiles violate security policies. Authors should have the sovereignty to disable the `__clihelp` setup protocol.

#### Design
- Add `DisableSetup bool` to `clihelp.App`.
- In `execute.go:ExecuteContext`, if `DisableSetup` is true, `protoClihelp` (`__clihelp`) is ignored at the top-level switch and falls through to normal command resolution (yielding an unknown command error or normal dispatch).
- Frozen protocols (`__complete` and `__explain`) remain active so existing shell completions continue functioning.

```go
type App struct {
    // ...
    // DisableSetup suppresses the built-in __clihelp verbs (version, install,
    // uninstall, keys, wrapper, manpage). Tab-completion (__complete) and
    // Alt-H (__explain) remain active.
    DisableSetup bool
}
```

---

### Feature 2: Safe Wrapper Extraction (`__clihelp wrapper --from <path>`)

#### Motivation
Users frequently write simple shims like:
```bash
#!/bin/bash
mail_cli -2 tui "$@"
```
Re-typing `-2 tui` into `mail_cli __clihelp wrapper mt -2 tui` is redundant. The tool should be able to read `~/bin/mt`, extract `-2 tui`, and emit the canonical wrapper script.

#### CLI Syntax
```console
$ mail_cli __clihelp wrapper --from ~/bin/mt [wrapper_name]
$ mail_cli __clihelp wrapper --from mt
```

#### Behavior & Invariants
1. **Read-Only**: Never mutates or overwrites the input script. Emits the generated wrapper script to `stdout` and shell registration instructions to `stderr`.
2. **Resolution**: If the `--from` target is not a direct file path, resolve via `exec.LookPath`.
3. **Parser Rules**:
   - Strip comments and blank lines.
   - Scan for an execution line invoking `appName` (e.g., `mail_cli`, `/path/to/mail_cli`, or `exec mail_cli`).
   - Validate that the line forwards arguments via `"$@"`, `"${@}"`, or `"$*"`.
   - Extract intermediate tokens as preset arguments.
   - **Fail-Fast**: If the script contains complex logic (unexpanded shell variables other than `$@`, multiple branches, or loops), abort cleanly with an actionable error prompting manual argument specification.

---

### Feature 3: `completion wrap` in `CompletionCommand()`

#### Motivation
Provides discoverability for authors who attach `clihelp.CompletionCommand()` to their command tree.

#### CLI Syntax
```console
$ mail_cli completion wrap mt [-- -2 tui]
$ mail_cli completion wrap --from ~/bin/mt [mt]
```

---

### Feature 4: Refreshed & Themed `__clihelp` Interface

#### Motivation
The current output of `<app> __clihelp` is plain unstyled text that repeats the verb token on every line (`__clihelp install`, `__clihelp uninstall`), uses a rigid 52-column gap that breaks on normal terminals, lacks section headers, and provides no examples. It feels disconnected from the rest of `clihelp`'s polished, colorized interface.

#### Design & Visual Layout
1. **Theme Integration**:
   - Read `th := Options{}.theme(a)` to inherit the application's color palette (`th.Hdr`, `th.Accent`, `th.Subcommand`, `th.Flag`, `th.Body`, `th.ExampleComment`).
   - Respect `App.NoColor` and global color settings automatically.
2. **Structured Sections**:
   - **Header**: Styled banner showing application name, protocol badge, and version.
   - **Usage**:
     ```
     Usage:
       podctl __clihelp <verb> [options...]
     ```
   - **Verbs Table**:
     - Remove the repeated `__clihelp` prefix on each line.
     - Verb name in `th.Subcommand` (Bold Green).
     - Verb options/parameters in `th.Flag` (Cyan).
     - Align descriptions cleanly using dynamic column indentation and word-wrapping.
     ```
     Verbs:
       version                                    Report the clihelp version this program was built with
       install [--no-keys] [--no-man] [<shell>]   Set up completion, Alt-H binding, and manual page
       uninstall [<shell>]                        Remove integration files written by install
       keys [<shell>]                             Print Alt-H key bindings for manual inspection
       wrapper [--from <path>] <name> [<args>...] Generate a self-answering wrapper script with preset args
       manpage [--install|--uninstall] [--force]  Generate or install roff manual page for man(1)
     ```
   - **Shells Section**:
     - Dynamically inspect active shell:
       ```
       Shells:
         Detected:  fish (active)
         Supported: bash, zsh, fish
       ```
   - **Curated Examples**:
     - Practical, color-coded examples:
       ```
       Examples:
         podctl __clihelp install                        # Set up tab-completion and Alt-H
         podctl __clihelp wrapper pd deploy > ~/bin/pd   # Wrap 'podctl deploy' as 'pd'
         podctl __clihelp manpage --install              # Install man(1) manual page
       ```
   - **Extended Detail (`-H`)**:
     - Styled sections for `Reserved Argument Names:` and `Install Locations (for <shell>):`.
   - **Individual Verb Help (`__clihelp <verb> -h`)**:
     - Formatted sub-help showing Usage, Description, and per-verb Examples.

---

## 3. Implementation Steps

1. **`clihelp.go` & `execute.go`**:
   - Add `DisableSetup bool` field to `App`.
   - Update `ExecuteContext` to check `a.DisableSetup` before handling `protoClihelp`.
2. **`wrapper_parse.go` (New Helper, ~60 lines)**:
   - Implement `extractWrapperArgs(r io.Reader, appName string) ([]string, error)`.
   - Implement `resolveWrapperPath(target string) (string, error)`.
3. **`protocol.go`**:
   - Re-implement `printClihelpVerbs`, `printClihelpDetail`, and `printVerbHelp` using `th := Options{}.theme(a)`.
   - Add structured sections (`Usage:`, `Verbs:`, `Shells:`, `Examples:`).
   - Add `--from <path>` parsing to `clihelpWrapper`.
   - Wire extracted arguments to `GenWrapperScript`.
4. **`completion_command.go`**:
   - Add `completionWrapSubcommand()` to `CompletionCommand()`.
5. **Tests**:
   - `protocol_test.go`: Test `App.DisableSetup`, color output, and verb list formatting.
   - `wrapper_parse_test.go`: Test extraction with simple scripts, `exec` prefixes, quoted args, and fail-fast triggers.
   - `docs_drift_test.go`: Verify documentation sync.
6. **Documentation**:
   - Update `docs/completion.md` and `docs/how-clihelp-decides.md`.

