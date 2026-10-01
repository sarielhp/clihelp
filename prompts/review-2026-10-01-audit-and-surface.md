# Deep review: the v0.3.45 → v0.3.48 changes and their blast radius

## Why this scope

Four deep reviews already cover resolution, flag binding, shell integration and the rendering
path as they stood on 2026-09-18 (`review/findings-*.md`, every finding closed). Since then
(`git diff 64e0ddf..HEAD`) roughly twenty commits landed in one sitting, none of them reviewed
by anyone but their author:

- **Automatic paging removed** (`pager.go`, `signals_*.go`, `App.Pager`, `Options.Pager`).
- **`Audit` grew a layout standard** (`audit_layout.go`): one-row short descriptions at a
  configurable width, trailing-period consistency per listing, warnings for unwrappable
  long-tier text, errors collected with `errors.Join` and de-duplicated. It restates the
  renderer's own column decision, so it can drift from it; `audit_layout_agreement_test.go`
  guards that, but only for the shapes it enumerates.
- **Redirected-help width fallback 70 → 80** (`render.go` `fallbackWidth`).
- **Concise help drops an orphan section heading** (`explain.go` `dropOrphanHeading`).
- **Usage lines keep bracketed groups whole** (`render.go` `glueGroups`, `writeUsage`, using a
  private-use sentinel rune).
- **API pruning:** `App.Render`, `SuggestCommand`, `ColorizeExampleLineWithApp` unexported or
  removed; `IsCompletionInstalled` renamed `CompletionScriptExists`; `StripANSI`,
  `VisualWidth`, `FirstSentence` moved to `internal/text`.
- **Tooling:** `Makefile` quoting of `ARGS`, `tools/commit.sh` rewritten (shows failing
  output, optional `COMMIT_TRAILER`).
- **Demo app** `example/` brought up to the new standard; generated docs regenerated.

## Scope

**In scope:** everything touched by `git diff 64e0ddf..HEAD`, plus whatever it reaches:
the code that *consumes* the renamed or removed names, the renderers whose layout `Audit`
claims to mirror, and the documentation that describes any of the above.

**Out of scope:** re-reviewing the closed findings in `review/`. Shell-integration internals
(`install.go`, `completion*.go`, `atomicwrite.go`, `lock_*.go`) only where this change set
touched them (`completion_command.go`, `man.go` for `markLibraryOwned`).

## Guardrails — pass these on, and obey them

1. **Read-only.** Do not modify, create or delete any file in the repository. Write findings
   in your reply. Scratch programs and tests belong under `$TMPDIR` or the scratchpad
   directory, never in the repo.
2. **Never run the example binaries** (`go run ./example`, `./example/mail_cli_fake`, or any
   built copy) unless you set `CLIHELP_NO_AUTO_COMPLETION=1` **and** sandbox `HOME`,
   `XDG_CONFIG_HOME` and `XDG_DATA_HOME` into a temp directory. Prefer `go test`.
3. **Never touch the real home directory**: `~/.bashrc`, `~/.zshrc`, `~/.config/fish/`,
   `~/.local/share/`. Never run any `install`/`uninstall`/`__clihelp` verb outside a sandbox.
4. **No network access.** **No ssh, no sudo.**
5. **Never run** `make bump`, `make commit`, `make push`, `make checkpoint`,
   `tools/bump-version.sh`, `tools/commit.sh`, `tools/checkpoint.sh`, `git push`, `git tag`,
   `git commit`, `git add`, `git reset`, `git checkout -- <path>`, or anything that
   publishes, stages, or rewrites history. `git diff`, `git log`, `git show` are fine.
6. Do not read or repeat credentials or real user data.
7. `go test ./...` (with `-race` where useful), `go vet`, `staticcheck`, and throwaway
   programs under `$TMPDIR` are all fine and encouraged. Do **not** run `make check` — it
   formats and tidies in place.

## House rules that outrank your taste (`AGENTS.md`)

- Go 1.26+. Function limits: standard logic soft 80 / hard 110 lines; declarative builders
  (`build*`, `init*`, `render*`, `generate*`) soft 120 / hard 160; dispatchers soft 150 /
  hard 200; table-driven `Test*` soft 180 / hard 250. Files: soft 800 / hard 1100 (tests
  1200 / 1600). Nesting depth ≤ 4, branches ≤ 15 (`tools/audit_lines.rb`).
- **Before 1.0 the API is not frozen.** A wrong name is removed, not aliased. Do not
  recommend deprecated aliases.
- No `os.Exit`/`log.Fatal` in library code. No hardcoded ANSI escapes outside `inline.go`
  (the one other `\x1b` is the regex in `internal/text`; flag it if you think that breaks
  the rule).
- Tests are table-driven and must pass under `-race`.
- stdout carries only machine-readable answers; stderr is for humans.
- Commits are conventional-commit style; `VERSION` bumps are patch-only.

## Seeded pressure points

Starting hypotheses, each answerable from the code. Settle the ones in your dimension and say
plainly when one is a non-issue.

1. **`Audit` vs the renderer.** `auditRows` assumes one `colIndentFor` over a whole listing
   and `wrapWidth(width, indent, 80) - indent` as the text width. But `topics.go:269/299/323`
   lays out topics, parameters and option lists with `clampIndent(colIndent(..)+2|+4)`, and
   `renderManPage` / `help topics` / `help flags` pages use different margins than the `-h`
   pages. Does a description that passes `Audit` ever wrap on one of those pages? The
   agreement test only drives `RenderGlobal` and `RenderCommand`.
2. **What the audit measures vs what the renderer draws.** The audit measures
   `firstSentence(Description)` for commands but the *full* `Description` appears in the
   command's own `-h` header and in `renderCommandSubcommands` via `SubcommandEntries`
   (`Command.SubcommandEntries` rows are never audited). `Group`ed command listings
   (`renderCommandGrouped`) compute one indent across all groups — confirm the audit does too.
   Shortcuts, aliases in `displayNameWithAliases`, `Hidden` handling, `OmitGlobalFlagsInCommands`,
   `EnableExamplesFlag`, `ExtendedHelpFlag` built-in rows: any that shift the column but are
   absent from the audit's lists?
3. **Error aggregation.** `audit()` returns `errors.Join(structural, auditLayout(..))` but
   returns early on example-validation and option-scope errors, so layout is skipped then.
   `dedupe` keys on the message — can two distinct problems collapse into one, or can the
   "global flags on command pages" scope hide a per-command difference? Is `Warn` ever
   invoked twice for the same line?
4. **`libraryOwned` propagation.** `markLibraryOwned` copies `Subcommands`; is the flag lost
   when an app wraps, re-groups, copies or appends a library command (e.g. in
   `Shortcuts`, or after `App.Walk`)? Is it observable to `doc/`/`tree/` (they are other
   packages)? Does exempting these commands from the period rule hide a real inconsistency
   inside the library's own listings?
5. **`dropOrphanHeading`.** `isHeadingLine` treats any unindented line ending in `:` as a
   heading. Can a legitimate content line (a `Usage:` continuation, a note ending in a
   colon, a wrapped description line at column 0) be dropped? It is also used by the
   `__explain` path (`explain.go`); does Alt-H output change? Is the "N more lines" count
   still correct when blank lines are trimmed?
6. **Private-use sentinel in usage lines.** `groupSpace` is `U+E000`. `runewidth` may treat
   private-use / ambiguous-width runes as width 2 in East-Asian locales (`RUNEWIDTH_EASTASIAN`),
   making `reflowMargin` mis-measure every glued group. Does `o.inline()` (markdown/links),
   `stripANSI`, or `VisualWidth` treat it differently from a space? Can the sentinel survive
   into output on any path — an error return, a panic recovery, `reflowMargin` writing
   partial output, `NoColor` vs colour — or appear in a test's captured buffer?
   Are there other `Usage:` renderers (`protocol.go`, `man.go`, `doc/`) that bypass
   `writeUsage` and therefore still split groups?
7. **Width fallback 70 → 80.** Everything that called `Options.width()` for a non-TTY
   changed. Which generated artifacts depend on it (man pages, `doc.RenderMarkdown`, golden
   files, `docs/clihelp/*`, tree rendering, Alt-H `explainGeometry`)? Any place that still
   hardcodes 70, or documents 70? Does `COLUMNS` handling still precede the fallback?
8. **Pager removal residue.** `termFd`, `height`, `screenRows`-style helpers, `LESS`,
   `signals_*`: is anything now dead or misleading (`Options.height()` is still used by the
   concise budget)? Does any doc, comment, example, `llms.txt`, `.agents/` rule, generated
   Markdown or CHANGES entry still promise paging or `$PAGER`? Does a consumer that set
   `Pager: true` get a *compile error with a useful message*, or silent surprise?
9. **`internal/text` move.** `StripANSI`/`VisualWidth`/`FirstSentence` now come from
   `internal/text`; root keeps `stripANSI`/`visualLen`/`firstSentence` wrappers and `tree/`
   keeps its own. `crosspackage_test.go` has exception text that may now be false. Is the
   layering rule in `AGENTS.md` ("leaves depend on nothing above them") still true? Is the
   `internal/text` regex identical to the one it replaced? Does `doc/md.go` still carry its
   own copies of anything?
10. **Public-surface leftovers and doc drift.** Re-run the audit of `go doc -all .`: what
    remains exported that only the library itself calls (`SubcommandList`, `CollectOptions`,
    `GenWrapperScript`, `PrintError`, `CompletionPath`…)? `docs_drift_test.go` guards prose
    naming real symbols — does it still run over `README.md`, `docs/*.md`, `AGENTS.md`,
    `llms.txt`, `doc.go`? Are there references to removed names (`App.Render`,
    `IsCompletionInstalled`, `SuggestCommand`, `Pager`) in any non-code file?
11. **`commit.sh` / `Makefile` / `bump-version.sh`.** `ARGS` now travels via the environment
    (`"$$ARGS"`). Does `make commit ARGS=` (empty) or a message with a leading `-` behave?
    `git add -A` stages everything including untracked scratch files — is that guarded?
    `bump-version.sh` pushes and tags: what happens on a dirty tree, a failed push, or a
    pre-existing tag? `CHANGES.md` still has an `[Unreleased]` heading after three bumps —
    is the changelog ever promoted, and does a script check it?
12. **Tests that assert the current behaviour.** `audit_layout_agreement_test.go` sweeps
    description *lengths* but with one filler shape; does it have teeth against a plausible
    regression (indent off by one, `firstSentence` vs full text, decoration dropped)? The
    suite was mutated by hand at the time of writing — are there other mutations it misses?
    Any test that now asserts a width, heading or text that is incidental?

## Part A — Architecture

Characterize the real dependency structure after the move to `internal/text` and the removal
of the pager. Is the audit coupled to renderer internals in a way that will rot
(`audit_layout.go` calls `colIndentFor`, `wrapWidth`, `decorateOptionDescription`,
`collectLocalOptions`, `collectGlobalOptions`, `rootFlags`)? Is there a better seam — e.g. the
renderer exposing a "layout plan" the audit consumes — and what is the mechanical path there?

## Part B — Bugs & correctness

Prioritize: wrong output a user sees (help text corrupted, text dropped, sentinel leaked) >
audit false negatives/positives that ship bad help > crashes (nil `Warn`, nil app, empty
lists, `Hidden` everything) > cosmetic. Give concrete inputs. Pay particular attention to the
twelve points above.

## Part C — Refactoring

Only changes with a defensible payoff. `render.go` is 883 lines and `options.go` 823 (soft
limit 800): is there a natural, non-function-splitting cut? Is `audit_layout.go` the right
shape?

## Part D — Interface design

How does an author *experience* the new standard? Read the error messages `Audit` produces:
are they actionable (do they name the command, the width, the fix)? Is the 80-column default
and `AuditOptions.Width`/`Warn` surface the smallest that works? Is a failing audit on a
description that was fine yesterday a break that needs a migration note? Run (in a sandbox)
`go test ./example` and read `podctl -h` pages at widths 40/60/80/120 via the library's
test harness, not the binary, and report what looks wrong.

## Part E — Tests, tooling, documentation

Which documented claims are now false? Which tests encode incidental behaviour? What single
test-infrastructure change would catch the next audit/renderer drift?

## Output format

### 1. Executive summary
Ten lines at most.

### 2. Findings
Full detail for critical/high; a compact table for medium/low. Per finding:
`[ID] title` — **Severity** (critical|high|medium|low) and **Confidence** (high|medium|low),
rated independently — **Category** — **Location** (`file:line`) — **What's wrong** —
**Evidence** (quoted code and the concrete input → bad output path; a runnable repro under
`$TMPDIR` is best) — **Why it matters** — **Fix** (specific enough to implement) —
**Effort** — **Risk & compatibility** — **Verification** (the test that would have caught it).

A finding without a concrete fix is an opinion — turn it into a fix or drop it.

### 3. Ordered roadmap
`Now` / `Next` / `Later`, ordered so that stopping after any step leaves the tree no worse.

### 4. What I would look at with more time
