# clihelp Library Fixes

This document describes issues found in the `clihelp` library (v0.2.13) during the mail_cli usage message audit. Each issue includes the root cause, affected code, and a proposed fix.

---

## Fix 1: `visualLen()` does not strip OSC 8 sequences

**File:** `render.go:99-101`

```go
func visualLen(s string) int {
    return len([]rune(stripansi.Strip(s)))
}
```

**Problem:** The `acarl005/stripansi` library strips CSI escape sequences (`\x1b[31m`) but **not** OSC (Operating System Command) sequences. Two types of OSC sequences are relevant:

1. **OSC 8 hyperlinks** — `\x1b]8;;https://url\x1b\` (clickable terminal hyperlinks)
2. **OSC 2 window title** — `\x1b]0;title\x07`

When a description or note contains an OSC 8 hyperlink (e.g. the `GlobalNote` in mail_cli: `[GitHub](https://github.com/sarielhp/gmail_cli)`), `visualLen()` counts the escape bytes as visible characters. This causes:

- **Incorrect word-wrap decisions** — `reflowSegment` thinks the line is much longer than it actually is, so it may fail to wrap or wrap at the wrong position.
- **Corrupted output** — When the escape sequence straddles a wrap boundary, the visible text gets mangled (e.g. the "h" in "https://" disappears from the output).
- **Incorrect `colIndent`** — The command/flag name column width is computed from `visualLen()`, so hyperlinked names produce wrong indentation.

**Fix:** Replace or augment the ANSI stripping with a function that also removes OSC sequences. Add a helper:

```go
func stripANSI(s string) string {
    // Remove CSI sequences: \x1b[ ... m
    // Remove OSC sequences: \x1b] ... (\x1b\|\x07)
    re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(?:\x1b\\|\x07)`)
    return re.ReplaceAllString(s, "")
}
```

Then use this in `visualLen()` and anywhere else that needs visible character length.

**Impact:** Without this fix, any usage of OSC 8 hyperlinks in descriptions, notes, or global notes will produce corrupted output at all terminal widths.

---

## Fix 2: Command title line is never reflowed

**File:** `render.go:379`

```go
th.Accent.Fprintln(w, th.TitlePrefix+title(cmd))
```

**Problem:** The title header line is written verbatim with no wrapping. If the title exceeds the terminal width, it overflows silently. The title is composed of:

- `TitlePrefix` = `"Detailed Usage: "` (16 chars)
- `title(cmd)` = `cmd.Title` if set, else `cmd.Name`

For example, `account associate` has `Title = "account associate <account_name> <program_name>"` (53 chars), producing a total of 69 visible characters — overflowing a 60-char terminal by 9 chars.

**Fix:** Wrap the title line using `reflow()` with a 2-space indent, or at minimum truncate with an ellipsis when it exceeds the available width. The separator lines above/below should also respect the same width.

```go
// Option A: reflow the title
reflow(w, th.Accent, wrapW, 2, "", th.TitlePrefix+title(cmd))

// Option B: truncate with ellipsis
titleStr := th.TitlePrefix + title(cmd)
if visualLen(titleStr) > wrapW {
    // truncate logic
}
th.Accent.Fprintln(w, titleStr)
```

**Impact:** Any command with a long `Title` field will overflow at narrow terminal widths. The current workaround is to keep all titles under ~44 chars, which is fragile.

---

## Fix 3: Separator and wrap width caps are inconsistent

**File:** `render.go:372-373`

```go
sepW := min(o.width(), 70)    // separator ==== width
wrapW := min(o.width(), 80)   // text wrapping width
```

**Problem:** The separator width and content wrap width use different caps (70 vs 80). This creates a visual mismatch at wider terminals:

| Terminal width | Separator width | Wrap width | Visual |
|---------------|----------------|------------|--------|
| 60 | 60 | 60 | Consistent ✓ |
| 80 | 70 | 80 | Separator 10 chars narrower than content |
| 100 | 70 | 80 | Separator 30 chars narrower than content |
| 120 | 70 | 80 | Separator 50 chars narrower than content |

The separator is meant to be a horizontal rule framing the command help page. Having it significantly narrower than the content looks like a rendering bug.

**Fix:** Use the same cap for both, or make the separator width proportional to the wrap width:

```go
// Option A: same cap for both
sepW := min(o.width(), 80)
wrapW := min(o.width(), 80)

// Option B: separator matches wrap width
wrapW := min(o.width(), 80)
sepW := wrapW
```

The 70-char cap on the separator appears to be a legacy value from the old mail_cli formatter. There is no technical reason for the separator to be narrower than the content.

**Impact:** Cosmetic issue affecting all command help pages at widths > 70. Most noticeable at width 100+ where the separator is dramatically shorter than the text block.

---

## Fix 4: `RenderGlobal` also uses inconsistent width caps

**File:** `render.go:276`

```go
wrapW := min(o.width(), 80)
```

**Problem:** The global overview caps wrap width at 80 but has no separate separator width. The separator is not used in `RenderGlobal`, so this is less visible. However, the same 80-char cap means that on very wide terminals (120+), the global overview is still only 80 chars wide, leaving unused horizontal space.

**Fix:** Consider whether the 80-char cap is intentional for readability. If so, document it. If not, remove the cap or increase it.

**Impact:** Minor — the 80-char cap is reasonable for readability on wide screens, but it should be a conscious design choice rather than an arbitrary limit.

---

## Summary of required changes

| # | File | Change | Severity |
|---|------|--------|----------|
| 1 | `render.go` | Fix `visualLen()` to strip OSC 8 sequences | **High** — causes corrupted output |
| 2 | `render.go` | Wrap or truncate title line in `RenderCommand` | **Medium** — overflow at narrow widths |
| 3 | `render.go` | Unify separator and wrap width caps | **Low** — cosmetic mismatch |
| 4 | `render.go` | Document or adjust 80-char wrap cap in `RenderGlobal` | **Low** — cosmetic |