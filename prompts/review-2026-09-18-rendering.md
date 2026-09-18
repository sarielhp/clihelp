# Deep review: the terminal rendering path

## Why this scope

`clihelp` exists to render beautiful terminal help. Two deep reviews have been done on it
and **neither opened this code**. Citation counts across both review documents:

| File | Lines | findings-2026-09-17 | findings-2026-09-17-shell-integration |
|---|---:|---:|---:|
| `render.go` | 598 | 0 | 0 |
| `resolve.go` | 506 | 0 | 0 |
| `format.go` | 436 | 2 | 0 |
| `topics.go` | 369 | 1 | 0 |
| `inline.go` | 151 | 2 | 0 |
| `pager.go` | 116 | 2 | 0 |

The first review went deep on flag binding, resolution semantics and example validation;
the second was scoped to shell integration and explicitly excluded everything else. So
roughly 2,000 lines of the library's primary purpose have never been examined.

There is also positive evidence this neighbourhood is fertile. Every defect found in the
last two days was a *text measurement* bug here or next door:

- `tree/`'s width measurement used a stripper that does not understand OSC sequences, so an
  OSC 8 hyperlink measured 22 columns instead of 4.
- `tree/`'s `firstSentence` returned strings containing newlines into a reflow that assumes
  one line.
- A `Command` → `clihelp.Command` rename mangled two Markdown table headers, and the golden
  test agreed with the mangled output.

## Scope

**In scope:** `render.go`, `format.go`, `topics.go`, `inline.go`, `pager.go`, and their
tests. `resolve.go` only where it feeds rendering (help-topic routing, `DisplayName`,
`commandArgs`).

**Out of scope:** the shell-integration surface (`completion*.go`, `install.go`,
`autoinstall.go`, `explain.go`, `man.go`, `protocol.go`, `shell.go`, `names.go`,
`atomicwrite.go`, `lock_*.go`, `versions.go`) — audited in
`review/findings-2026-09-17-shell-integration.md`, every finding closed. Flag binding and
resolution — audited in `review/findings-2026-09-17.md`, every finding closed. Do not
re-review either except where rendering reaches into them.

## Guardrails — pass these on, and obey them

1. **This is a read-only review. Do not modify, create or delete any file in the
   repository.** Write findings in your reply, not to disk.
2. **Never run the example binary** (`go run ./example`, `./example/mail_cli_fake`, or any
   built copy) unless you set **both** `CLIHELP_NO_AUTO_COMPLETION=1` **and** a sandboxed
   `HOME`, `XDG_CONFIG_HOME` and `XDG_DATA_HOME` pointing into a temp directory. It sets
   `AutoInstallCompletion: true`. Prefer not running it at all — `go test` covers it.
3. **Never write to the real home directory**, and never touch `~/.bashrc`, `~/.zshrc`,
   `~/.config/fish/` or `~/.local/share/`.
4. **No network access.** Nothing in this review may contact a live service.
5. **Never run** `make bump`, `tools/bump-version.sh`, `git push`, `git tag`, `git commit`,
   or anything else that publishes or rewrites history.
6. Do not read or repeat credentials, and do not put real user data in a finding.
7. `go test ./...`, `go vet`, `staticcheck` and writing throwaway programs under
   `$TMPDIR` are all fine and encouraged.

## House rules that outrank your taste

Read `AGENTS.md`. In particular: 80-line function limit; file comfort 300–700, warn 800,
hard 1100; no raw ANSI escapes in source (use the theme); table-driven tests; backward
compatibility matters — this is a published module with a `retract` directive already in
`go.mod`, so a breaking API change needs a very good reason. `make check` and `make audit`
are the gate and are currently green.

## Seeded pressure points

Each of these is a specific, falsifiable claim about *this* code. Settle it by reading the
code and, where you can, by a runnable proof. Report the ones that are real and say plainly
which ones you disproved — a disproved hypothesis is a result.

1. **`Options.theme` clears `Separator` unconditionally.** `render.go`'s `theme()` copies
   every field behind an `if src.X != nil` / `!= ""` guard, then ends with a bare
   `th.Separator = src.Separator`. Does a partial theme — say `&Theme{Hdr: color.New(...)}`
   — therefore silently lose the default separator? What does a zero `Separator` render as?
2. **`width()` says 70 and `maxContent()` says 80.** For a non-`*os.File` writer (a
   `bytes.Buffer`, which is what every test and `doc/` uses) `width()` returns 70 while
   `maxContent()` returns 80, and `wrapWidth` takes `min(termWidth, indent+maxContent)`. Is
   the effective width for buffered output 70, and is that what the tests and the markdown
   generator assume? Is 70 ever wider than the real terminal?
3. **`pageOutput` counts logical lines, not screen rows.** It pages on
   `strings.Count(buf.String(), "\n") > h`. A page of 20 logical lines that each wrap to
   three rows occupies 60 rows and counts as 20. Does help that overflows the screen fail to
   page? Conversely, do ANSI escapes or a trailing newline miscount it the other way?
4. **Width measured before or after inline rendering.** `inline()` turns
   `[text](url)` into an OSC 8 sequence, and `VisualWidth` discounts that correctly. But is
   the wrap decision ever made on the *markdown source* rather than the rendered string? If
   so, a description containing a link wraps as though the URL were visible.
5. **`colIndent` drops params that are too wide.** A name wider than
   `DefaultMaxColIndent - 4` is excluded from the maximum; if *every* name is that wide,
   `maxW` stays 0 and the function returns `DefaultMaxColIndent`. Do those names then
   overflow their column and collide with the description text?
6. **`reflowWords` and a word wider than the line.** An over-long word gets its own line
   and overflows. Is there any input for which it loops, drops the word, or emits a line of
   only indentation? Consider a zero-width word, a word of only ANSI escapes, and
   `indent >= width`.
7. **Wide runes and combining marks.** `runewidth` is used consistently as far as a grep
   shows. Do CJK command names, emoji (including ZWJ sequences), and combining accents
   actually align in a grouped command list, a flags table, and a wrapped note?
8. **`pageOutput` measures one fd and writes to another.** `isTerminal` and `height()` are
   resolved from `os.Stdout` when `o.Writer` is nil or `os.Stdout`, and from `o.Writer`'s fd
   otherwise. Is there a combination — `Writer` set to stderr, or to a `*os.File` that is a
   pipe — where it pages into something that is not a terminal, or refuses to page when it
   should?
9. **The pager subprocess.** `runPager` pipes bytes to `$PAGER`. What happens when the
   pager exits non-zero, is killed, does not exist, or the user presses `q` early
   (`EPIPE`)? Does `countingWriter` correctly distinguish "never ran" from "ran and quit"?
   Is `$PAGER` split safely — can a `$PAGER` containing shell metacharacters execute
   something unintended?
10. **`renderInline`'s markdown parsing against hostile input.** Descriptions, usage lines
    and notes are author-supplied, but authors make mistakes: unterminated `**`, a nested
    `[a](b](c)`, a URL containing an OSC terminator (`\x07` or `\x1b\`), a link whose text
    is empty, ten thousand asterisks. Can any of those emit an unterminated escape sequence
    that corrupts the user's terminal, or loop, or panic?
11. **`splitLines` and `\r\n`.** Notes and descriptions may be written with Windows line
    endings, or contain a lone `\r`. Does a stray `\r` survive into output and overwrite the
    line the user is reading?
12. **The tiered help paths agree.** `-h` is meant to stay within about 24 lines while
    `--help`/`-H` renders everything. Is the concise path's budget actually enforced, or is
    it a convention that a long `Description` or many groups can blow past silently?

## Required output format

For every finding:

- **ID and one-line title.**
- **Severity** (critical / high / medium / low) **and confidence** (high / medium / low),
  rated *separately*.
- **`file:line` citations**, re-read at the source. A citation you did not open is worth
  nothing.
- **The failure path, end to end**, with concrete inputs that produce the bad outcome. If
  you cannot state the inputs, it is not a finding yet — say so and downgrade it.
- **A runnable proof** where the claim is about behaviour: a Go test, a tiny program, a
  command line. Keep it in your reply; it becomes the regression test.
- **A concrete fix.** A finding without one is an opinion; drop it.

Also report, explicitly: **what you checked and found sound.** The seeded points you
disproved are as valuable as the ones you confirmed, and they stop the next reviewer
repeating the work.
