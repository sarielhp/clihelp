# Deep review: the terminal rendering path (2026-09-18)

> **Reading this later.** Items below struck through have since been settled; the note
> after each says how. The current state of the public API is
> `review/api-surface-2026-09-18.md`, and `docs/how-clihelp-decides.md` is the
> mechanism this library actually implements today.

**Scope:** `render.go`, `format.go`, `topics.go`, `inline.go`, `pager.go` and their tests —
about 2,000 lines that neither previous review opened. Prompt:
`prompts/review-2026-09-18-rendering.md`. Five reviewers, one per dimension. Every
critical and high finding below was re-verified by the author of this document with a
runnable proof, not taken on a reviewer's word.

---

## One sentence

The library's own escape sequences are emitted correctly and measured correctly, and then
every layer above them — the wrapper, the truncator, the stripper, the colour switch — is
free to cut them in half, and the pager that was supposed to display all of it has never
run.

## Executive summary

Two headline features do not work. **The pager has never paged** (P1): `cmd.Stdout` is
wrapped in a `countingWriter`, so `os/exec` gives the child a pipe, and `less` on a pipe
degrades to `cat`. **The `-h` tier does not bound its output** (H4): measured at 92 lines
against a "≤ 24 lines" guarantee repeated in six documents, and byte-identical to
`--help`.

Both were introduced by correct work. The `countingWriter` was added to fix a real bug —
telling "the pager never ran" from "it ran and quit" — and it silently disabled the
feature it was protecting. That is the shape of this whole review: the low-level code is
careful, and the composition is not.

The most serious defect is a chain rather than a single fault (see **Chains**): an author's
description string reaches the terminal unfiltered, can contain or produce a malformed
escape, the stripper cannot measure what it cannot match, the wrapper then cuts the
sequence in half, and `Alt-H` truncates the result — leaving an unterminated OSC 8
hyperlink that keeps hyperlinking the user's shell prompt after the program has exited.

One finding is mine: **`tools/check.sh` was red** when this review started, because of a
guard test I committed an hour earlier. Three of the five reviewers reported it
independently. Fixed in `1998c13`, along with a second defect in the same file — a test
that used `t.Logf` where it meant `t.Errorf` and so could not fail.

---

## High

### H1 — The pager has never paged  **[verified]**

`pager.go:79-83`. `runPager` wraps the destination in `countingWriter` and assigns that to
`cmd.Stdout`. `os/exec` passes a file descriptor straight to the child only when `Stdout`
is an `*os.File`; for any other `io.Writer` it creates a pipe and copies in a goroutine.
`countingWriter` is never an `*os.File`, so the pager's stdout is **always** a pipe.

Measured, same program, only that line differing:

```
Stdout = &counting{os.Stdout}   →  PAGER SEES: PIPE     ← what clihelp does
Stdout = os.Stdout              →  PAGER SEES: TTY
```

`less`, `more`, `most` and `bat` all detect a non-tty stdout and behave like `cat`:
`seq 1 200 | less -R -F -X` emits all 200 lines and exits 0. So `App.Pager`,
`Options.Pager`, the `-R` plumbing in `buildPagerArgs`, and the help text promising "the
complete reference manual (paged)" amount to a fork, an extra copy, and the same
scrolling dump the user would get with paging off.

**Root cause worth recording.** The `countingWriter` was added deliberately, with its own
comment (`pager.go:56-58`) and a regression test (`pager_test.go:159-190`), to fix a real
bug. It disabled the feature it was protecting, and nothing noticed because no test starts
a real terminal.

**Fix.** Hand the pager the terminal when the destination is one: `if f, ok :=
out.(*os.File); ok { cmd.Stdout = f }`, and classify the result by exit code —
`*exec.Error` and exit 126/127 mean "never displayed anything", everything else means a
pager ran. Keep the counting path for non-file writers. This is what `git`'s `pager.c`
does. **Ordering hazard: see the roadmap.**

### H2 — A wrapped OSC 8 hyperlink is left open, and `Alt-H` makes it permanent  **[verified]**

`format.go:215` word-splits the *rendered* string, so an OSC 8 opener and its terminator
land on different lines:

```
line 2: "Read \x1b]8;;https://example.com/m\x1b\\the complete operations manual for"
line 3: "this command\x1b]8;;\x1b\\ before you begin."
```

`fatih/color` closes every line with `\x1b[0m`, an SGR reset, which does not close a
hyperlink. Any consumer that truncates by line then drops the terminator — and
`App.Explain` (`explain.go:174`), the Alt-H key binding, does exactly that, as does
`myapp --help | head -20`. VTE, kitty, iTerm2 and WezTerm all then hyperlink every cell
they print afterwards, including the shell prompt. **It survives the process exiting.**

Two triggers, neither needing an adversary: a space in a URL, and a markdown link
hard-wrapped across two source lines in a `Note.Text` (`render.go:488` rejoins note lines
with `\n`, and `inline()` swallows it into the OSC payload).

**Fix.** Two independent halves, both wanted. Percent-encode every byte `<= 0x20` and
`0x7f` in the URL before it enters the OSC payload, so the payload cannot contain a byte
the wrapper or the terminal will act on. And have `reflowWords` close and reopen the link
at a line break, which is what the OSC 8 specification requires.

### H3 — Author strings reach the terminal byte for byte  **[verified]**

`inline.go:118` copies every byte of an author string including ESC; `inline.go:82`
interpolates the author's URL straight into clihelp's own OSC 8 payload. Nothing in the
rendering path filters control characters. Verified:

| `Description` contains | effect |
|---|---|
| `\x1b[2J\x1b[1;1H` | clears the user's screen mid-help |
| `\x1b[?25l` | hides the cursor, never restored; **measures 14 columns, should be 5** |
| `\x1b[?1049h` | switches to the alternate screen buffer; **measures 11, should be 5** |
| `[x](\x1b\\…)` | the ST in the URL closes clihelp's own OSC; the rest executes |
| `[x](http://a\x07…)` | BEL does the same |
| `\x1b]0;pwn` | rewrites the window title, and survives `StripANSI` |

The `?`-marker forms are worse than passthrough: `ansiRegex` (`format.go:16`) cannot match
them, so they reach the terminal *and* corrupt the column arithmetic.

Severity depends on provenance. In most programs these are compile-time literals and this
is a careless-author hazard. But `__clihelp` exists precisely so a program can be set up
by a packager, descriptions in real CLIs come from config files, embedded JSON and
translation catalogues, and `completion.go:50` and `man.go:152` push the same strings
through the same path — the completion one into a generated shell script.

**Fix.** One sanitiser applied at the render boundary: replace every C0 byte except `\n`
and `\t` (which `reflow` and `detectListPrefix` need) with U+FFFD. Call it at the top of
`renderInline`, and at the four verbatim sites that bypass it — `render.go:471`,
`render.go:511`, `render.go:523`, `topics.go:346`.

### H4 — The documented `-h ≤ 24 lines` guarantee is false, and nothing enforces it  **[verified]**

`render.go:559-581` is the whole of `Concise`: prefer `Description` over
`LongDescription`, skip `Notes`, add a footer. Subcommands, Parameters, Flags, Global
Flags and **every Example** render in full. `RenderGlobal` ignores `o.Concise` entirely, so
`myapp -h` and `myapp --help` are byte-identical at the root.

Measured on a realistic command (2 parameters, 30 flags, 6 examples) at width 70:
**concise 92 lines, extended 92 lines.** At width 40 a single command's `-h` is 50 lines;
at width 20 it is 100.

Promised as a guarantee in six places: `README.md:17`, `README.md:216`, `llms.txt:121`,
`docs/comparison-with-cobra.md:24`, `:132`, and `docs/flags-and-options.md:215`. The
comparison-document instance is a head-to-head table cell against cobra — it is the
library's stated differentiator.

**Fix.** The codebase already knows how: `explain.go:163`'s `writeWithinBudget`, with the
property test `explain_test.go:79-100` run at six terminal heights. Buffer the concise
render and pass it through the same budget, with the concise footer as the hint. Or drop
the numeric promise from all six documents. Either is small; leaving it is not an option,
because the number is currently a false claim about a competitor comparison.

### H5 — A pager that exits 0 having printed nothing swallows the help entirely  **[verified]**

`pager.go:90-93` consults `counted.n` only on the error path and returns `true`
unconditionally on success. Measured: `cmd.Run` err=nil, 0 bytes written, `runPager`
returns `true`, `pageOutput` skips its fallback, **the user sees no help at all** — not
truncated, none. `PAGER=true` and `PAGER=/bin/true` are both real ways people disable
paging, and a `$PAGER` wrapper ending in `|| exit 0` is another.

**Fix.** `return counted.n > 0 || len(data) == 0` on the counting path. On the terminal
path introduced by H1, exit-code classification plus treating `PAGER` values of `""`,
`cat` and `true` as "no pager".

### H6 — Five rendering behaviours are guarded only by a width-locked oracle in `example/`  **[verified by mutation]**

`example/mail_cli_fake/mail_cli_fake_test.go` compares renderer output byte-for-byte
against a hand-written reimplementation of the renderer living in the test file, at a
hardcoded width of 70. Mutations that `go test .` does not notice, and only that oracle
catches:

- `colIndent`: `return maxW + 4` → `return DefaultMaxColIndent`
- the grouped list: `firstSentence(c.Description)` → `c.Description`
- `renderGlobalShortcuts`: the whole "Shortcut Commands:" section deleted
- `RenderGlobal`: the `Config:` footer deleted
- `RenderGlobal`: the `Global Flags:` heading deleted

And the oracle is not an independent specification: changing its `Width: 70` to 90
produces a 110-line diff, and its `anyMultiLine` rule differs from production's
(`render.go:212-218` lacks the oracle's `visualLen(p.Name)+4 > indent ||` term — they
agree only at width 70). This is the "golden updated to match the defect" hazard with the
golden being an entire second renderer.

**Fix.** Parameterise the oracle on width and run it at 40/70/100, and move the five
behaviours into root-package assertions so the library is not guarded from `example/`.

### H7 — `tools/check.sh` was red  **[verified; fixed in `1998c13`]**

`go/parser.ParseDir` is deprecated as of Go 1.25, so staticcheck reports SA1019 and
`tools/check.sh` — which runs `staticcheck ./...` under `set -euo pipefail` — exits 1
before reaching vet, the tests or the build. `make bump` would have aborted.

The offending file is `crosspackage_test.go`, the drift guard committed in `df321f9` an
hour before this review started. I reported the gate as green having run `go vet`, the
line audit and the review's own `gate.sh` — which classes staticcheck as *advisory* — but
not the project's `make check`. Three of the five reviewers found it independently.

The same file carried a second defect: `TestSharedHelperExceptionsAreSorted` used
`t.Logf` where it meant `t.Errorf`, so it passed for any input, and it ranged a map, whose
order Go randomises deliberately, so the source order it meant to check was not observable
from inside the test. It now reads the map literal out of the source, and it failed on the
first run and caught that my own exception list was unsorted.

---

## Chains

The severe path here is composed, not singular.

**C1 — from a description string to a permanently corrupted terminal.** H3 lets an author
string carry arbitrary bytes. M5 lets an ordinary typo — a space in a URL — split
clihelp's own escape. L14/M11 means the stripper cannot match either result, so
`VisualWidth` reports 27 columns for a fragment that displays zero, and every wrap
decision on that line is wrong. H2 then cuts the sequence at a line break, and
`writeWithinBudget` in the Alt-H path truncates away the terminator. The user pressed a
key to *inspect* a command line they had not decided to run, and their shell prompt is
hyperlinked from then on. Each link is medium on its own; the path is critical.

**C2 — the pager's fixes are load-bearing on each other, and the order matters.** H1 is
caused by the fix for H5's sibling problem. Fixing H1 alone removes the pipe and therefore
also removes M8's unbounded hang (no copy goroutine, nothing to wait for) — good. But it
also means the pager *starts running for real*, at which point M9's wrong paging decision
(newlines counted instead of screen rows) becomes visible for the first time, and H5
becomes undetectable on the terminal path because there is no longer a writer to count.
**H1 must land with M9 and H5, not before them.**

---

## Medium and low

| ID | Finding | State |
|---|---|---|
| M1 | An ungrouped command between two grouped ones is printed under the previous group's heading, and the return to that group prints no heading — the output states something false about two commands. Same logic for options in `RenderMan`, which unlike `RenderFlags` skips `normalizeFlagGroups`. | **verified** |
| M2 | `buildDefaultUsage` counts `cmd.Subcommands` and ignores `SubcommandEntries`, so `Usage:` promises `[args]` above a populated `Subcommands:` list. Exactly the copy-without-the-preference that `SubcommandList`'s doc comment says already bit `doc/`. | **verified** |
| M3 | `App.UsageLine` is inline-rendered in `RenderCommand` and `RenderMan` and **not** in `RenderGlobal` (`render.go:345`) or `RenderFlags` (`topics.go:168`), so the same app shows raw `**` and a URL the author meant to hide. | **verified** |
| M4 | `Options.Theme` **replaces** `App.Theme` rather than layering onto it, so passing any per-render theme silently drops an app-level `Separator: true` and with it the whole title block. | **verified** |
| M5 | A space in a link URL splits the OSC 8 escape: the user literally sees `]8;;https://example.com/the`, and a bare newline lands inside an OSC string. | **verified** |
| M6 | The grouped-list blank-line decision is made on the markdown source, not the rendered string, so one link in one description double-spaces the entire command list. | **verified** |
| M7 | There is no per-render way to disable colour. `color.NoColor` is global and decided from `os.Stdout` at package init, and `pageOutput`'s mutation of it was removed to fix a data race. The example app's own `--no-color` flag is bound and never read: `myapp --no-color --help` is byte-identical to `myapp --help`. | **verified** |
| M8 | `cmd.WaitDelay` is unset, so a `$PAGER` that backgrounds anything holds the stdout pipe open and `cmd.Run` blocks forever. Measured 3s on a grandchild; unbounded in general. | reported, mechanism confirmed by code read |
| M9 | The paging decision counts `strings.Count(buf, "\n")` against terminal height — logical lines, not screen rows. Help needing 26 rows on a 24-row terminal prints unpaged. Also off by one against the shell prompt, and blind to a final line with no trailing newline. | reported, arithmetic confirmed |
| M10 | No signal handling anywhere: Ctrl-C kills the parent and orphans a pager that traps SIGINT, which keeps the terminal and writes over the shell prompt. | reported |
| M11 | `ansiRegex` cannot match an OSC containing a newline, an unterminated OSC, or a CSI with `?` private markers or `:` sub-parameters — and the first two are things clihelp's own output can now contain. | **verified** |
| M12 | Nested colours close with a full SGR reset, so the first line of every flag and command description loses the body colour while its wrapped continuations keep it. Visible with any custom theme. | reported, mechanism confirmed |
| M13 | `FirstSentence` cuts inside markdown: `"Try [it](http://x/v1. 2/y) now."` → `"Try [it](http://x/v1."`, and the raw markup reaches the help. | **verified** |
| M14 | `parseMarkdownLink`'s paren scan is quadratic — 4× the input costs 16× the time, measured 943ms on 128KB of `[a](` repeated. A truncated translation catalogue entry looks exactly like this. | reported with measurements |
| M15 | `colIndent` never looks at the available width, so at 20 columns a 13-column flag name leaves about three columns for the text and every word gets its own line. `Explain` feeds the shell's real `$COLUMNS`, so a narrow pane reaches it. | **verified** |
| M16 | `RenderMan`'s nested Parameters/Flags lists put their names at column 2 — outdented from their own headings — because the prefix column's left margin is hardcoded at two spaces while only the description column is indented. | reported |
| M17 | `Options.MaxContentWidth` is never tested through a renderer; deleting it from every `-h`/`--help` flag table leaves the suite green. Its own test re-implements the renderer's arithmetic instead of calling the renderer. | **verified by mutation** |
| M18 | `TestRenderGlobalGroupedCommands` constructs the exact M1 input and asserts only presence and heading order, both of which hold while the defect is present. The fourth instance of a test asserting around a defect in this repository. | **verified** |
| M19 | Every width test in the root package counts runes or bytes, not display columns, so the property "a rendered line fits the requested width" is asserted nowhere. `TestReflowMultibyteIsRuneAware`'s comment states that a CJK character is one visible column; the library counts two. Measured tolerance: a 40% overshoot passes. | **verified** |
| M20 | `App.Theme` is set by no root-package test. Mutating `src = a.Theme` to `src = nil` leaves the suite green, and `theme()` shows 100% statement coverage because the assignment always runs with `a.Theme == nil`. | **verified by mutation** |
| M21 | `optionsToParams` and `renderOptionsGrouped` carry the same twelve lines of `(default: …)` / `(required)` / `(deprecated: …)` logic; only the `topics.go` copy is tested. `DefaultText` can vanish from every `-h` flag table while `help flags` still shows it. An intra-package copy, which `crosspackage_test.go` by construction cannot see. | **verified by mutation** |
| M22 | `go test ./...` fails when `CLIHELP_NO_AUTO_COMPLETION` or `NO_AUTO_COMPLETION` is set, because `completion_test.go` got an incomplete copy of the four-variable clearing loop that five other test files have. | **verified** |
| L1 | A first word that does not fit flushes a line holding only padding, and a zero-width word glues the following word to it. `tree/tree.go` solves this with a `lineHasWords` flag; `format.go` never got it. Trailing whitespace confirmed: `"  -v, --verbose  "`. | **verified** |
| L2 | A whitespace-only line in a description loses the paragraph break, and the blank lines that do survive carry `indent` columns of trailing spaces. | reported |
| L3 | `"Shortcut Commands:"` is printed with zero rows when every shortcut is hidden; `renderGlobalFlagsSection` two functions away shows the correct filter-then-heading order. | reported |
| L4 | The command title can be two columns wider than the separator rule it sits inside. | reported |
| L5 | `VisualWidth` is not additive over a `strings.Fields` join for invalid UTF-8, so a stray byte pushes a line one column past the width. | reported |
| L6 | A tab measures zero columns while a terminal advances eight, desynchronising the hanging indent; a lone `\r` survives into raw notes, fenced blocks, `Note.Heading` and example lines, where it overwrites the row the user is reading. Example lines exist to be copy-pasted. | **verified** |
| L7 | `reflow`'s prefix column assumes a single-line prefix — the same class as the `tree/firstSentence` bug already fixed. | reported |
| L8 | A backslash escapes anything, so `Inline("C:\temp\x")` → `"C:tempx"` and `use \d+` → `use d+`. `examples.go:50`'s comment records this as why example lines were taken out of the inline renderer; descriptions never were. | **verified** |
| L9 | `[](url)` renders zero visible characters — a silently blank description. | **verified** |
| L10 | `Note.Heading` is not run through `inline()`, so `**Warning**` prints literal asterisks in the terminal and renders bold in the generated markdown. | reported |
| L11 | `PAGER=""` — the conventional way to disable paging — still launches `less`. `strings.Fields` also cannot parse a quoted `$PAGER`, so `PAGER="sh -c 'fmt \| less'"` fails noisily. | reported |
| L12 | Paging is refused, and width collapses to the 70-column fallback, whenever `Options.Writer` wraps stdout rather than being an `*os.File` — a `bufio.Writer` or a `colorable` writer is a common setup. The same three-branch fd resolution is duplicated in `pageOutput`, `width()` and `height()`. | reported |
| L13 | The pager's stderr always goes to the process stderr, ignoring `App.Stderr`. | reported |
| L14 | `MaxContentWidth` can only ever *narrow* buffered output, because the non-terminal width fallback is 70 and the content cap defaults to 80 — contradicting its own doc comment, which says to set it larger for more space. | reported |
| L15 | `DisplayNameWithArgs` and `commandArgs` (39 lines of usage-line parsing) are exported, have no caller anywhere in the module, and no test. `RenderGlobalFlags` and `CheckExample` are exported with no test. | **verified by coverage** |
| L16 | `doc/drift_test.go`'s subcommand test compares a one-line delegation against the thing it delegates to, so it cannot fail. Contrast `tree/drift_test.go`, whose second assertion is a real property. | reported |
| L17 | The root test package's `strip()` helper uses the third-party stripper that `StripANSI`'s doc comment exists to warn about — it does not merely miss an OSC 8 sequence, it corrupts the string. Latent today (zero hits across the suite), live the moment a test renders a description containing a link. | reported |
| L18 | Documentation: `README.md:295` promises wrapping "without exceeding the terminal width" (false at 20 columns); `AGENTS.md:159` states the wrong continuation column; `README.md:220` and `doc.go:120` present paging as unconditional when it needs `App.Pager`; `README.md:298` documents `Command.Group` without the empty-group consequence. | reported |

---

## Ordered roadmap

Ordered so that stopping after any step leaves the tree no worse. Two hazards govern it:
the pager fixes depend on each other (**C2**), and the escape-integrity fixes compose
(**C1**) — sanitising input first makes every later fix easier to verify, because the
malformed escapes stop arriving.

**Now — stop the terminal corruption and the false promise**

1. **H7 is already done** (`1998c13`). The gate must be green before anything else is
   measured.
2. **H3 — one control-character sanitiser at the render boundary**, plus the four verbatim
   sites. This is the head of C1: it removes the inputs that make H2, M5, M11 and L6 hard
   to reason about, and it is ~20 lines.
3. **M5 — percent-encode the OSC 8 URL.** With 2 in place this closes the second way a
   malformed escape is produced, so what remains in C1 is only clihelp's own wrapping.
4. **H2 — close and reopen the hyperlink at a line break** in `reflowWords`, and have
   `writeWithinBudget` emit a terminator when the text it kept is unbalanced. This is the
   one that stops Alt-H corrupting a terminal permanently.
5. **M11 — widen `ansiRegex`** (`(?s)` on the OSC branch, a bounded body, an optional
   terminator, `[@-~]` for the CSI final byte). Defence in depth behind 2–4, and it fixes
   the mis-measurement that made the wrapping worse.
6. **H4 — decide the `-h` budget and make the six documents true.** Enforce with
   `writeWithinBudget`, or retract the number. Do not leave a false claim in a competitor
   comparison.

**Next — the pager, as one unit**

7. **H1 + M9 + H5 together** (see C2). Hand the terminal to the pager, count screen rows
   rather than newlines, and classify a silent success as failure. Splitting these makes an
   intermediate state where the pager runs for the first time *and* decides wrongly.
8. **M8 — `cmd.WaitDelay`** for the remaining non-file path, and **M10 — hold SIGINT/SIGQUIT**
   for the duration of the pager run, which is what `git` does.
9. **L11, L12, L13 — `PAGER=""`, capability-not-type fd detection, `App.Stderr`.** L12 also
   removes the fd-resolution block triplicated with `render.go`.

**Then — the layout defects that state something false**

10. **M1 — normalise the empty group** (`normalizeFlagGroups` already exists and already
    does this for `RenderFlags`; two of three call sites never got it), with **M18**'s test
    fix landing first so it fails before and passes after.
11. **M2 — `hasSubs := len(SubcommandList(*cmd)) > 0`.** One line.
12. **M3 — `inline()` the usage line** at `render.go:345` and `topics.go:168`, and widen
    `TestExampleAppNoBareMarkdownAndNoVisibleURLs` to cover `help flags`, `help man` and
    `help topics`, which it never rendered.
13. **M4 — layer `Options.Theme` onto `App.Theme`** instead of replacing it.
14. **M6, M13, L8, L9, L10 — the remaining inline and measurement defects.** Each is one
    to five lines.
15. **M15 + M16 + L1 + L2 + L3 + L4 — the width and margin arithmetic.** These want doing
    together: they all touch `format.go`'s prefix/indent model, and M16's fix (an explicit
    margin parameter) is what M15's clamp wants to build on.

**Later — the test surface, which is why none of this was caught**

16. **M19 — one `assertFitsWidth` helper measuring display columns**, used by every width
    test, plus cases at 20 and 40 columns, which no test has ever rendered at.
17. **M17, M20, M21, M22, H6, M18 — the assertion gaps**, and extract the duplicated
    option-suffix logic rather than testing it twice.
18. **L15, L16, L17 — dead exported code, the tautological test, the wrong stripper in the
    test helper.** Dropping `acarl005/stripansi` from `go.mod` is the end of L17: it exists
    in this module only to be the wrong answer.
19. **L18 — the five documentation corrections**, after the code they describe has settled.

---

## Tests and tooling

The recurring finding is not missing execution — all 24 functions in `render.go` run — but
missing *assertions*. Proven by mutation: 28 mutations, 8 survivors, each named above.

Three property tests exist in the whole module and only one is in the root package:
`format_width_test.go:13-47` (column alignment in display columns),
`tree/tree_test.go:55-88` (line width in display columns), and
`examples_test.go:447-479` (example lines are verbatim). The model to copy is
`explain_test.go:79-100`, which asserts a budget at six terminal heights — the `-h` tier
simply never got the equivalent.

Untested classes that currently work by luck: emoji, ZWJ sequences and combining marks
appear in no test anywhere in the module. All four align correctly today, but that is a
property of `runewidth v0.0.28`'s grapheme clustering and nothing here would notice a
dependency bump changing it.

**A note on the gate.** `gate.sh` classes staticcheck as advisory; `tools/check.sh` treats
it as fatal. H7 lived in the gap between them. Either the review gate should adopt a
staticcheck baseline, or `make check` should be the only gate anyone reports on.

---

## Disproved — recorded because a disproved hypothesis is a result

- **Seed 1, `Options.theme` clearing `Separator` — my own hypothesis, and wrong.**
  `render.go:145`'s unguarded `th.Separator = src.Separator` loses nothing, because
  `defaultTheme()` already sets `Separator: false` (`render.go:52`), so the assignment
  writes the value a guard would have preserved. A latent trap if that default ever
  changes, not a bug. The real defect is one line up and is **M4**: `Options.Theme`
  replaces `App.Theme` wholesale. I pointed at the right ten lines for the wrong reason;
  one reviewer disproved the mechanism and another found the actual fault beside it.
- **Seed 5, `colIndent` dropping over-wide params.** Sound. `colIndent`'s
  `l+4 <= DefaultMaxColIndent` filter is the exact complement of `formatPrefix`'s
  `visualLen(prefixDisplay)+2 > indent` guard, so any name excluded from the maximum
  necessarily takes the own-line branch. No collision at any width tried.
- **Seed 7, wide runes and combining marks.** Sound, and measured: `検索`=4,
  `👩‍💻dev`=5, `e+U+0301`+`xport2`=7, `🚀ship`=6, and a hyperlink label of CJK measures its
  visible width. `visualLen`/`VisualWidth` is used consistently; `grep` finds no
  `RuneCountInString` or `%-*s` in the rendering path.
- **Seed 6, `reflowWords` looping or dropping words.** No loop, and content integrity
  holds: a fuzz target asserting `Fields(in) == Fields(StripANSI(out))` ran over 6 million
  executions without a dropped, duplicated or reordered word. `indent >= width` and
  `width == 0` both degrade to one word per line rather than faulting. The padding-only
  line (L1) is the only real defect here.
- **Seed 8, paging into a non-terminal.** Disproved as stated. The three branches are
  self-consistent, including `Writer = os.Stderr` with stdout redirected. The failure is
  only the conservative direction — refusing to page when it should (L12).
- **Seed 9's security half, `$PAGER` metacharacters.** Disproved. There is no shell:
  `exec.Command` execs `parts[0]` directly, so `PAGER="cat ; touch /tmp/pwned"` passes
  `";"` as a literal argument and creates nothing. Go's `ErrDot` additionally blocks a
  relative `./pager` resolved out of `PATH`.
- **Seed 10, hostile markdown panicking or looping.** Disproved on its own terms, with the
  strongest evidence in the review: 13.5M fuzz executions over `Inline` asserting every
  OSC 8 opener is terminated, and 447K over the whole render path with the fuzzed string in
  thirteen author fields — zero panics, zero unterminated escapes *of clihelp's own
  making*, and no input slower than 20ms beyond M14. Every unterminated marker degrades to
  literal text. What is broken is what happens to those correct escapes downstream (H2) and
  what arrives in them from outside (H3).
- **No reachable nil `*color.Color`.** Every theme reaching `format.go` comes from
  `Options.theme`, which nil-guards all ten colour fields. `reflow` and `separator` *would*
  fault on a literal nil, which matters only if a Theme-taking reflow is ever exported.
- **No zombies or descriptor leaks in the pager.** 600 failing runs moved `/proc/self/fd`
  from 15 entries to 15 and left no children. Early quit (`EPIPE`) is handled correctly by
  `os/exec` and does not cause a double print.
- **`doc/` is unaffected by any of the width arithmetic.** It never calls `reflow`,
  `Render*` or `VisualWidth`; it emits unwrapped markdown.
- **`ansiRegex` covers everything `fatih/color` v1.19 emits**, including truecolour
  `\x1b[38;2;R;G;Bm` and its `\x1b[0;22;…m` resets. The gaps (M11) are in sequences
  clihelp does not emit — except the two that H2 and M5 now cause it to emit.

---

## What I would look at with more time

- **Whether `Options` should carry the terminal instead of re-deriving it.** The same
  `*os.File` type assertion appeared three times with three different fallbacks
  (`pageOutput`, `width()`, `height()`), and L12 was the consequence.
  *Half done:* the `termFd()` helper exists and all three now go through it — one type
  assertion, in `render.go`. What remains is the design question the helper does not
  answer: the terminal is still re-derived per render rather than resolved once and
  carried, and honouring `$COLUMNS` still has nowhere to live.
- ~~**A golden-free rendering test strategy.** `example/mail_cli_fake`'s oracle is a second
  renderer, and H6 is what that costs. The property tests that exist are worth more than
  all the byte comparisons put together.~~ **Done 2026-09-18:** the oracle is retired. 499 lines became ~190 of properties over the same corpus, at three widths rather than one; the three behaviours it alone guarded now have direct tests, and the survey is back to 38 of 38.
- ~~**Whether `Inline` should be exported at all**, given that the plain-text path for
  `NoColor` (M7) and the sanitiser (H3) both want to live inside it, and `man.go` already
  reaches past it for the `showURLs` form.~~ **Settled 2026-09-18:** unexported, along with `DisplayName`, `ColorizeExampleLineWithApp` and `DefaultMaxColIndent`. The oracle was the only thing outside the library using them.
- ~~**`resolve.go`, 506 lines, zero citations in three reviews now.** It was out of scope
  here except where it feeds rendering. It is the last large file nobody has read
  adversarially.~~ **Done:** reviewed in `findings-2026-09-18-resolve.md` — every finding closed — and mutation-surveyed to 25 of 26.

---

## Status

**Every finding in this document is fixed**, on branch
`fix/rendering-review-2026-09-18`, one commit per block of the roadmap, each with
regression tests whose teeth were checked against the unfixed behaviour. `make check` and
`make audit` are green.

Two deviations from the roadmap as written, both recorded here rather than quietly:

- **H6 was answered by its second half, not its first.** The roadmap suggested
  parameterising `example/mail_cli_fake`'s oracle on width and running it at 40/70/100. I
  tried that and reverted it: the oracle is a second renderer, and making it agree at
  every width means maintaining it as one, which is the opposite of the finding. The five
  behaviours it uniquely guarded are asserted in the library's own tests now
  (`TestGlobalHelpListsShortcutsAndConfig`), `oracleWidth` is a variable rather than a
  literal, and the comparison still runs at one width with a comment saying why.
- ~~**L15's dead exported code was kept and tested, not deleted.** `DisplayNameWithArgs`,
  `commandArgs`, `RenderGlobalFlags` and `CheckExample` have no caller in this module, but
  this is a published module with a `retract` directive already in `go.mod`, and removing
  exported symbols to tidy an internal audit is not a trade worth making.~~ **Settled 2026-09-18:** `displayNameWithArgs` is unexported; the API audit removed the rest of the dead surface.

Things the fix pass found that the review did not:

- **`writeExampleLine` had the same nested-colour defect as `reflowWords`**, and only
  surfaced once the latter was fixed and the oracle disagreed.
- **The oracle flattened a multi-line `UsageLine` into one flow**, which the real renderer
  has never done. It had no line segmentation at all in the path the usage line takes.
- **`emitLine` in the first draft sanitised inside `splitLines`**, which also runs on
  already-rendered text — so the sanitiser destroyed this package's own escapes and
  exposed the URLs. `TestExampleAppNoBareMarkdownAndNoVisibleURLs` caught it in the first
  run after the change, which is the test doing exactly its job.
- **Under `NoColor`, spelling a link out as `text (url)` was the wrong plain form.** It
  changes the width of every line containing a link and puts a bare URL in help output,
  which this repository already has a test forbidding. The label alone is the right answer;
  `man.go` asks for the spelled-out form explicitly because a manual page cannot
  hyperlink.

Every high finding was re-verified by the author with an independent runnable proof in a
throwaway module outside the repository before any fix was written; the proofs are in the session scratchpad and use
only the exported API, so they transfer into the repository almost verbatim. Findings
marked **verified** in the medium/low table were reproduced the same way. The rest are
recorded as the reviewer reported them, with the mechanism confirmed by reading the cited
code.

`make check` and `make audit` are green as of `1998c13`.
