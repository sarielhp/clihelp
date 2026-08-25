# clihelp Review Decisions Log

**Last updated:** 2026-08-25 · version reviewed against: `0.2.17`

This document replaces the earlier `report.md` review (written against `0.2.12`).
It records the disposition of every finding rather than the original prose,
so a future session can tell at a glance what was fixed, what was deliberately
not done, and what still remains.

---

## Fixed

### 0.2.12 review — confirmed bugs (all fixed by 0.2.16)
- Nested `help` double-render (`... <cmd> help`) — fixed via `handled` flag in `resolveCommand`.
- `-v` version shorthand hijacking `-v, --verbose` — version check now only accepts `--version`/`version`.
- `App.GlobalFlags` displayed but never parsed — now bound in `ExecuteContext`.
- `Theme.Separator` dead code — now wired into render paths.
- `BoolToggle` bypassing duplicate detection (pflag panic) — now checks duplicates.
- Markdown subcommand alias links broken — canonical names used for links.
- Markdown tables broken on `|` — `|` and newlines escaped for cells.
- `help <unknown>` silently succeeding — returns `unknown help topic %q` error.

### 0.2.12 review — design/consistency items (fixed by 0.2.17)
- Version drift across `VERSION`/`example`/`AGENTS.md` — `check.sh` now guards the example literal; AGENTS.md no longer hardcodes a version.
- `AGENTS.md` 80-vs-70 width fallback — corrected to 70; explicit exception documented for the SGR/OSC constants in `inline.go`.
- `Options.Width` capped at 80 with no escape hatch — added `Options.MaxContentWidth` (default 80).
- `PrintError` bypassing `App.Stderr` — routed through `a.stderr()`.
- Execute-path help ignoring `App.Theme` — now passes `Theme: a.Theme`.
- Markdown command pages omitting app/ancestor persistent flags — now uses `App.collectOptions`.
- `Command.Group` unused — group headings implemented in global and structural subcommand lists.
- Root positional args rejected when `a.Run` is set — root `Run` now receives positionals.
- `--version` with empty `App.Version` silent success — now returns an error.
- Example `Run` handlers writing via `fmt.Printf` — now `fmt.Fprintf(ctx.Stdout, …)`.
- `Option.Binder` misleading "Internal pflag flag binder" doc — clarified.
- Misleading help-flag comment in `ExecuteContext` — corrected.
- `RenderCommand` subcommands listing bare names vs global `displayName` — now consistent (structural lists use `displayName`).
- Empty/zero app rendering `Usage of :` and empty sections — falls back to `app` and suppresses empty sections.
- Hidden commands generating markdown pages — skipped in collect/index/nav.
- `markdownSlug` collisions silently overwriting pages — now return an error.
- `pageHeader` raw YAML with unsafe title/parent — fields are now YAML-quoted.
- `splitLines` keeping trailing `\r` — trimmed for CRLF input.
- `visualLen` rune-count misaligning wide CJK — now `go-runewidth` (display columns).
- `Enum` not validating `defaultVal` membership — validated at bind time.
- `StringSlice` aliasing caller's backing array — defensive copy.
- `parseFlagSpec` not supporting `--flag=value` — value suffix stripped, bare name registered.
- Completion lacking `--flag=` prefix, duplicate shortcut/command names, swallowed `resolveCommand` errors — all fixed.
- Tree traversal duplicated across `hasCommandNamed`/`LookupCommand`/`ancestorsForPath`/`resolveCommand` — unified via `findCommand`; options unified via `App.collectOptions`.
- `Options.width()` always probing `os.Stdout` — probes the Writer first when it is a terminal file.

---

## Intentional non-fixes

- **`Options.ShowURLs` (render URLs as visible text):** dropped deliberately. The feature was lost in the v0.2.14 merge blunder; the decision is to keep OSC8-hidden hyperlinks only (modern-terminal-first). The `inline()` OSC8 rendering remains intact.
- **OSC8/SGR hardcoded escapes in `inline.go`:** intentional and documented — neither `fatih/color` nor `stripansi` can emit OSC8 links, so this is the sole sanctioned exception to the no-raw-ANSI rule.
- **`BooleanToggle` extra alias support:** only the first short name and base long name are registered; extra aliases are not. Left as-is (documented behavior).

---

## Historical note — v0.2.14 merge blunder

A `git merge` of `origin/master` into local `master` (`b1146aa`) resolved `render.go`
in favor of the *older remote* version, silently dropping local work:
`prefixColor`, `Options.ShowURLs`, `wrapWidth`, `ancestorsForPath` in `RenderCommand`,
and `Theme.Subcommand`. Most were re-implemented in this session (0.2.15–0.2.17);
`ShowURLs` was intentionally not restored. The local tag list was also missing
`v0.2.14` — fetch tags (`git fetch --tags`) to sync. Lesson: verify merge diffs
(or use `--no-commit`) when two branches both modify the same files.
