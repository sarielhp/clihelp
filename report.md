# clihelp Library Review Report

**Date:** 2026-08-25
**Version reviewed:** 0.2.12 (`VERSION` file) — working tree at commit `b79d9a7`
**Scope:** All library source files, example application, tests, docs, AGENTS.md, CHANGES.md

---

## Summary

The library is in good overall shape: it builds cleanly, `go vet` passes, `staticcheck` reports only 2 dead-code findings, and all 71 tests pass. The renderer, pflag binding layer, markdown generator, and shell completion are coherent and well documented.

However, this review found **8 confirmed runtime bugs** (several verified live against the example binary), a small set of **design flaws** (silently non-functional API surface), and a number of **consistency/doc-drift issues**. The most damaging category is *silent misbehavior*: features that are rendered or documented but do nothing (`GlobalFlags`, `Theme.Separator`), and cases that produce wrong output without an error (`help` double-render, `-v` version hijack, broken markdown links).

---

## 1. Confirmed Bugs

### BUG-1 [High] Nested `help` subcommand double-renders help

`podctl config help` prints the `config` command help **and then the entire global help page**. Same for `podctl config help set` and `podctl deep alpha help`.

- **Where:** `execute.go:50-58` + `execute.go:61-64`, `execute.go:228-236`
- **Root cause:** `resolveCommand` renders help itself as a side effect and returns `nil, nil, nil, nil, nil`. The guard in `ExecuteContext` only catches the case where `args[0] == "help"` (top-level). For nested `... <cmd> help`, the guard misses and the fallback branch `targetCmd == nil && len(path) == 0 && len(remaining) == 0 && a.Run == nil` runs `RenderGlobal`, producing a second page.
- **Evidence:** `./example config help` output ends with a second `Usage of podctl:` / `Version:` block (verified live). Only the top-level `podctl help` case is covered by `execute_test.go`.
- **Fix suggestion:** Make `resolveCommand` return a `handled bool` (or a sentinel) instead of rendering; `ExecuteContext` returns `nil` immediately when handled. This also removes the rendering side effect from the resolver (see IMP-1).

### BUG-2 [High] `-v` version shorthand hijacks `-v, --verbose`

`podctl -v` prints the version and exits 0 — even when the app declares `-v` as the shorthand for a `--verbose` flag (the example does exactly this).

- **Where:** `execute.go:36-47`
- **Root cause:** the version shortcut is checked before flag binding/parsing, so a lone `-v` never reaches the flag parser. Behavior is inconsistent: `podctl -v` → version, but `podctl -v build` → parsed as the verbose flag.
- **Evidence:** `./example -v` → `podctl 0.2.9`, rc=0 (verified live).
- **Fix suggestion:** Drop `-v` from the version check (keep `--version` and `version`), or only accept `-v` when no option in `App.PersistentOptions`/`Commands` registers the `v` shorthand. Cobra resolves this by requiring an explicit `Version` flag registration.

### BUG-3 [High] `App.GlobalFlags` are displayed but never parsed

Options declared in `App.GlobalFlags` appear in `RenderGlobal` ("Global Flags:" section) and in the markdown index, but they are **never bound to the pflag FlagSet** in `ExecuteContext` (only `App.PersistentOptions` are). Declaring a real flag there silently does nothing at runtime.

- **Where:** `clihelp.go:85` (field, grouped under "Presentation overrides"), `execute.go:82-89` (binding skips it), `render.go:388-391`, `md.go:296-299`
- **Fix suggestion:** Either bind `GlobalFlags` in `ExecuteContext` alongside `PersistentOptions` (with duplicate detection), or formally deprecate the field with a doc comment warning it is display-only. Prefer the former; it matches user expectations.

### BUG-4 [Medium] `Theme.Separator` is dead code — the documented feature does nothing

`Theme.Separator` is documented as toggling "the horizontal rule drawn around the header block", and `separator()` is implemented (`render.go:303`), but **no render path ever calls it**. Setting `Separator: true` has no effect.

- **Where:** `render.go:26-27, 33-42, 91, 302-305`
- **Evidence:** `staticcheck` reports `render.go:303:6: func separator is unused (U1000)` and `render.go:316:6: func title is unused (U1000)` (the latter is a second dead helper).
- **Fix suggestion:** Either wire `separator()` into `RenderCommand`/`RenderGlobal` when `th.Separator` is set, or remove the field and function. Add a test either way.

### BUG-5 [Medium] `BoolToggle` bypasses duplicate detection → pflag panics

`bindHelper` (used by `String`, `Int`, `Bool`, `Duration`, `StringSlice`, `Enum`, `Var`) checks for duplicate long/short names and returns a friendly error. `BoolToggle` implements its own binder that skips these checks; pflag's `FlagSet.Var` **panics** ("flag redefined") on a duplicate name. Duplicate `--[no-]x` specs (e.g., same toggle declared in a parent's and child's `PersistentOptions`) crash the program instead of returning an error.

Additionally `BoolToggle` silently ignores any flags beyond the first short name and the base long name (extra aliases in the spec are never registered), which is inconsistent with the other constructors.

- **Where:** `options.go:218-260`
- **Fix suggestion:** Reuse `bindHelper`-style duplicate checks (or refactor `bindHelper` to accept a registration callback like the others), and either support or reject extra aliases explicitly.

### BUG-6 [Medium] Markdown subcommand links still broken for alias entries

The fix from Bug Review 04-01 (bug-review-04-01.md) is incomplete. When `SubcommandEntries` contains a name that matches a subcommand **alias**, `renderCommandPage` detects the match (the alias loop) but then builds the file path from `s.Name` (the alias) instead of the canonical `cmd.Subcommands[i].Name`. The generated link points at `<alias-slug>.md`, which was never written (pages are keyed by canonical names).

- **Where:** `md.go:356-373`
- **Fix suggestion:** In both match branches, build `p` with `cmd.Subcommands[i].Name`. Add a test with `SubcommandEntries` naming an alias.

### BUG-7 [Medium] Markdown tables break on `|` in descriptions

Command/option descriptions are inserted into markdown tables without escaping pipes. A description containing `|` (plausible for docs) corrupts the table layout. `mdInline` escapes `\ * _ \` [ ] <` but not `|`.

- **Where:** `md.go:266-312` (index tables), `md.go:353-416` (subcommand/parameter/flag tables), `md.go:184-195` (`mdInline`)
- **Fix suggestion:** Escape `|` (and newlines) in table cells — add a dedicated `mdCell()` helper or extend `mdInline` with a table-safe variant.

### BUG-8 [Medium] `help <unknown>` silently succeeds

`podctl help nonexistent` prints nothing and exits 0. The path never reaches `RenderCommand`'s `false` return in a way the user can see.

- **Where:** `execute.go:228-236` (`resolveCommand` renders and swallows the result), `render.go:431-434` (`RenderCommand` returns false)
- **Fix suggestion:** When the built-in help path resolves no command, return an error (e.g., `unknown help topic %q`) so the exit code and error reporting behave like any other failure. Cobra does the same.

---

## 2. Design & Consistency Issues

### DES-1 [High] Doc drift across version sources

- `VERSION` file: `0.2.12`
- `example/main.go:50`: `Version: "0.2.9"`
- `AGENTS.md`: "Current version: `0.2.0`"
- `CHANGES.md`: latest entry is `0.2.11` (the 0.2.12 bump commit exists but no changelog entry)

The example's version is also what `podctl -v`/`--version` prints, so users see `0.2.9`.

**Fix suggestion:** Single source of truth — make the example read `VERSION` at build time (`go:embed` or ldflags), or at minimum update all locations when bumping. Add a `make check` guard (a small script) that cross-checks these.

### DES-2 [Medium] AGENTS.md rules conflict with the code

- AGENTS.md says width fallback is "80 characters for non-TTY environments"; the code and README use **70** (`render.go:49, 103`).
- AGENTS.md says "No direct ANSI escape codes in code or tests"; `inline.go:10-20` hardcodes `\x1b[`-prefixed constants (OSC 8 and SGR). This is *justifiable* — neither `fatih/color` nor `stripansi` can *emit* OSC 8 sequences — but the rule needs an explicit exception, or the AGENTS rule should be reworded to "no direct ANSI except the SGR/OSC constants in inline.go".

### DES-3 [Medium] `Options.Width` semantics vs. the 80-column cap

`Options.Width` is documented as "the target terminal width in columns", but `wrapWidth()` caps the content area at `indent + 80` regardless of `Width`. A user setting `Width: 200` still gets 80-column wraps, and even auto-detected wide terminals never use more than 80 columns of text.

- **Where:** `render.go:108-118`
- **Fix suggestion:** Document the cap explicitly in `Options.Width` (and README), or make the cap configurable (e.g., a `MaxContentWidth` field, default 80).

### DES-4 [Medium] `PrintError` bypasses `App.Stderr`

`PrintError` writes to `os.Stderr` directly, ignoring `App.Stderr`. Every other output path respects the override (tests rely on it). Inconsistent.

- **Where:** `execute.go:13-21`
- **Fix suggestion:** Add `func (a *App) PrintError(err error)` that uses `a.stderr()` (keep the package-level function as a convenience for apps without an `App`).

### DES-5 [Medium] Execute-path help ignores the custom theme

When `--help` (or the built-in help subcommand) renders during `ExecuteContext`, it constructs `Options{Writer: a.stdout()}` — the `App.Theme` is dropped, so colors revert to defaults. Static `Render*` calls respect the theme.

- **Where:** `execute.go:61, 127-131, 176-180, 230-234`
- **Fix suggestion:** Pass `Options{Writer: a.stdout(), Theme: a.Theme}` in all execute-path renders.

### DES-6 [Medium] Markdown command pages omit app/ancestor persistent flags

Terminal `RenderCommand` collects app + ancestor + command persistent options (`render.go:473-497`); `renderCommandPage` only collects the command's own (`md.go:395-405`). The two help outputs disagree.

**Fix suggestion:** Extract a shared `collectOptions(a, path, cmd) []Option` helper used by both renderers.

### DES-7 [Low] `Command.Group` is unused

The `Group` field is never read by renderer, executor, completion, or markdown. Either implement command grouping (a common CLI help feature) or remove the field.

- **Where:** `clihelp.go:53`

### DES-8 [Low] Unknown root-level positional args rejected even when `App.Run` is set

`resolveCommand` treats any non-matching token as an error when `currentCmd == nil` (root) regardless of whether `a.Run` exists. A root command with `Run` (e.g., `podctl <target>`) can never receive positional arguments if any subcommands exist. The parallel check for nested commands correctly tests `currentCmd.Run == nil`.

- **Where:** `execute.go:272-283`
- **Fix suggestion:** Include `a.Run == nil` in the condition, mirroring the nested logic.

### DES-9 [Low] `--version`/`-v` with empty `App.Version` prints nothing and exits 0

`podctl --version` on an app with `Version: ""` succeeds silently. Should return an error (or fall back to help).

- **Where:** `execute.go:36-47`

### DES-10 [Low] Example `Run` handlers write to `fmt.Printf` instead of `ctx.Stdout`

The example's `Run` functions use `fmt.Printf`/`fmt.Println`, bypassing the `Context.Stdout` override the framework provides — a bad pattern for the demonstration app (example/main.go:79, 98, etc.).

### DES-11 [Low] Minor consistency items

- `Option.Binder` doc comment says "Internal pflag flag binder" but the field is exported and necessary for hand-constructed options (clihelp.go:21).
- `ExecuteContext`'s comment "Add help flag if not explicitly defined" is misleading — `fs.Lookup("help") == nil` is always true on a freshly created FlagSet (execute.go:75-80).
- `RenderGlobal` shows aliases for top-level commands (`displayName`) but `RenderCommand`'s Subcommands section lists bare names — inconsistent alias presentation (render.go:308-313 vs 325-336).
- `RenderGlobal` prints "Detailed Help:" even when the app has zero commands; "Usage of :" when `App.Name` is empty.
- Markdown pages are generated for `Hidden` commands (terminal help skips them) — undocumented.
- `markdownSlug` collisions: `set-up` and `set up` map to the same file; later page silently overwrites earlier (md.go:154-182).
- `pageHeader` emits raw YAML — a title containing `": "` produces invalid front matter (md.go:207-219).
- `splitLines`/`reflow` keep trailing `\r` on Windows-style line endings.
- `visualLen` counts runes, not display columns — wide CJK characters misalign columns (use `go-runewidth`/`x/text/width` for true column width).
- `Enum` does not validate that `defaultVal` is a member of `allowed`; help can advertise a default that can't be set (options.go:331).
- `StringSlice` sets `*target = defaultVal`, aliasing the caller's slice backing array — mutating one mutates the other (options.go:285).
- `parseFlagSpec` doesn't support `--flag=value` specs (`--output=PATH` becomes a flag literally named `output=PATH`).
- Completion doesn't support `--flag=value` prefixes, and `Shortcuts` duplicates of `Commands` produce duplicate completion entries (completion.go:76-97, 105-129).
- `handleComplete` ignores errors from `resolveCommand` (completion.go:34).
- `hasCommandNamed`/`LookupCommand`/`ancestorsForPath`/`resolveCommand` each re-implement name+alias tree traversal — a single lookup helper would remove ~60 duplicated lines (clihelp.go:121-177, execute.go:199-215, 244-259).
- `Options.width()` always probes `os.Stdout`, even when `Options.Writer` is a file/buffer — width detection should arguably prefer the Writer when it is a terminal, and needs explicit `Width` otherwise.

---

## 3. Improvement Suggestions

1. **Refactor command resolution to return a result, not render** (IMP-1): `resolveCommand` should return a struct with a `handled` flag (help rendered / completion done). This fixes BUG-1 and BUG-8 structurally and removes hidden side effects.
2. **Unify the option pipeline:** add `(*App).collectOptions(path []string, cmd *Command) []Option` shared by `render.go`, `md.go`, `execute.go`, and `completion.go` (fixes DES-6, reduces drift between the four consumers).
3. **Version management:** embed the version into the example from the `VERSION` file (`go:embed`), update AGENTS.md automatically or drop the hardcoded number, and add a 0.2.12 CHANGES.md entry.
4. **Tests to add** (each confirmed bug gets a regression test first):
   - `podctl config help` / `podctl config help set` render exactly one page
   - `-v` conflict: app with `-v, --verbose` persistent flag — `ExecuteContext(ctx, []string{"-v"})` must not print version
   - `GlobalFlags` bound and parseable after fix
   - `Theme.Separator` produces a rule (or field removed)
   - duplicate `BoolToggle` specs return an error, never panic
   - `SubcommandEntries` alias entries produce working markdown links
   - `|` in descriptions produces valid tables
   - `help unknown-topic` returns an error
5. **Completion:** support `--flag=value` completion and dedupe shortcut/command names; consider `GenFishCompletion`.
6. **Theme:** implement or delete `Separator`; consider letting `Options.Width` disable the 80-column cap (DES-3).
7. **Docs:** document `GlobalFlags` (or deprecate), `SubcommandEntries`, `Shortcuts`, `Group`, and the 70-column fallback + 80-column content cap in README/docs.
8. **Cleanup:** delete `title()`; route `PrintError` through the App; fix the example to use `ctx.Stdout`; align "Internal" comment on `Binder`.

---

## 4. What Is Good (no action needed)

- Clean separation between declarative structs and execution; `Options`/`Theme` layering with safe zero values.
- ANSI-aware wrapping and column alignment (`visualLen`, OSC 8 stripping in `stripANSI`).
- Duplicate flag detection with actionable errors in `bindHelper` (missing only in `BoolToggle`).
- The `reflow`/`splitLines` newline-preservation refactor (documented in `bug/clihelp_bug_and_fix.md`) works correctly.
- The bash/zsh completion templates behave correctly end-to-end (the `%%%%` in the Go template is intentional `fmt.Sprintf` escaping that renders as `%%` + TAB in the emitted script — verified live).
- Markdown generation's hash gating, git-ignore bootstrap, and pruning are thoughtful; `markdownFormatVersion` future-proofing is a nice touch.
- Levenshtein typo suggestions are a pleasant usability feature.

---

## 5. Verification Commands Used

- `go vet ./...` — clean
- `staticcheck ./...` — 2 findings (dead `separator`, dead `title`)
- `go test -timeout 30s ./...` — all pass
- Live checks against `go build ./example`:
  - `./example config help` → double render (BUG-1)
  - `./example -v` → version hijack (BUG-2)
  - `./example help nonexistent` → silent success (BUG-8)
  - generated bash completion sourced in bash 5.3 → correct candidate list (no bug)
