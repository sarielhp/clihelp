# clihelp Bug: `\n` in `UsageLine` Ignored by `reflow`

## Summary

`reflow` in `render.go` uses `strings.Fields` to tokenize text, which
collapses **all** whitespace (including intentional `\n` newlines) into
single spaces. Any `Command` with a multi-line `UsageLine` has its lines
joined into one paragraph.

## Root Cause

**`render.go` (v0.2.11), the buggy `reflow` in full:**

```go
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string) {
    if indent < 2 {
        indent = 2
    }
    words := strings.Fields(strings.TrimSpace(text))  // ← BUG

    if prefix != "" {
        prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
        curLen := visualLen(prefixStr)
        if curLen > indent {
            c.Fprintln(w, prefixStr)
            prefixStr = strings.Repeat(" ", indent)
            curLen = indent
        }
        if len(words) == 0 {
            if curLen > 0 {
                c.Fprintln(w, prefixStr)
            }
            return
        }
        indentStr := strings.Repeat(" ", indent)
        var cur strings.Builder
        cur.WriteString(prefixStr)
        for _, word := range words {
            wlen := visualLen(word)
            space := 0
            if curLen > indent {
                space = 1
            }
            if curLen+space+wlen > width {
                c.Fprintln(w, cur.String())
                cur.Reset()
                cur.WriteString(indentStr)
                cur.WriteString(word)
                curLen = indent + wlen
            } else {
                if space > 0 {
                    cur.WriteString(" ")
                    curLen++
                }
                cur.WriteString(word)
                curLen += wlen
            }
        }
        if curLen > indent {
            c.Fprintln(w, cur.String())
        }
        return
    }

    if len(words) == 0 {
        return
    }
    indentStr := strings.Repeat(" ", indent)
    var cur strings.Builder
    cur.WriteString(indentStr)
    curLen := indent
    for _, word := range words {
        wlen := visualLen(word)
        space := 0
        if curLen > indent {
            space = 1
        }
        if curLen+space+wlen > width {
            c.Fprintln(w, cur.String())
            cur.Reset()
            cur.WriteString(indentStr)
            cur.WriteString(word)
            curLen = indent + wlen
        } else {
            if space > 0 {
                cur.WriteString(" ")
                curLen++
            }
            cur.WriteString(word)
            curLen += wlen
        }
    }
    if curLen > indent {
        c.Fprintln(w, cur.String())
    }
}
```

[`strings.Fields`](https://pkg.go.dev/strings#Fields) splits on any run of
whitespace (spaces, tabs, newlines) and **discards the separators**. So

```
"mail_cli spam <subcommand>\nmail_cli spam <message_id...>"
```

becomes the single token list

```
["mail_cli", "spam", "<subcommand>", "mail_cli", "spam", "<message_id...>"]
```

and reflow reassembles them into one continuous paragraph.

## Impact

Every `Command` with `\n` in its `UsageLine` renders incorrectly. In mail_cli:

**`cli/spam.go`:16**
```go
UsageLine: "mail_cli spam <subcommand>\nmail_cli spam <message_id...>",
```

**`cli/misc.go`:143**
```go
UsageLine: "mail_cli unspam <message_id...>\n  mail_cli unspam folder <folder_name>",
```

The clihelp **example itself** demonstrates the same pattern in
`example/mail_cli_fake/tree.go`:

```go
// line 58
UsageLine: "mail_cli unspam <message_id...>\n  mail_cli unspam folder <folder_name>",
// line 211
UsageLine: "mail_cli rule <subcommand> [args...]\nmail_cli rule -export <file>\nmail_cli rule -import <file>",
```

… yet the example's golden tests (`mail_cli_fake_test.go`) were written to match
the **broken** output, so the bug was never caught.

## Before / After

### Broken (old) output for `spam --help`

```
Usage:
  mail_cli spam <subcommand> mail_cli spam <message_id...>
```

### Fixed output for `spam --help`

```
Usage:
  mail_cli spam <subcommand>
  mail_cli spam <message_id...>
```

### Broken (old) output for `unspam --help`

```
Usage:
  mail_cli unspam <message_id...> mail_cli unspam folder <folder_name>
```

### Fixed output for `unspam --help`

```
Usage:
  mail_cli unspam <message_id...>
  mail_cli unspam folder <folder_name>
```

## The Fix

Refactor `reflow` into three parts:

1. **`reflow`** — entry point that splits `text` on `\n` via `splitLines` and
   calls `reflowSegment` for each line.
2. **`splitLines`** — splits a string on `\n`, preserving empty segments so
   consecutive newlines produce blank lines.
3. **`reflowSegment`** — the original word-wrapping logic moved into its own
   function, unchanged except it operates on a single paragraph.

### `reflow` (new entry point)

```go
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string) {
    if indent < 2 {
        indent = 2
    }
    segments := splitLines(strings.TrimSpace(text))
    for i, seg := range segments {
        if seg == "" && i+1 < len(segments) {
            // Empty segment followed by more content => blank line
            if prefix != "" {
                prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
                c.Fprintln(w, prefixStr)
                prefix = ""
            } else {
                c.Fprintln(w, strings.Repeat(" ", indent))
            }
            continue
        }
        if seg == "" {
            continue
        }
        reflowSegment(w, c, width, indent, prefix, seg)
        prefix = ""
    }
}
```

### `splitLines` (new helper)

```go
func splitLines(text string) []string {
    if text == "" {
        return nil
    }
    var out []string
    start := 0
    for i, r := range text {
        if r == '\n' {
            out = append(out, text[start:i])
            start = i + 1
        }
    }
    out = append(out, text[start:])
    return out
}
```

### `reflowSegment` (unchanged body, moved out)

```go
func reflowSegment(w io.Writer, c *color.Color, width, indent int, prefix, text string) {
    words := strings.Fields(text)

    if prefix != "" {
        prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
        curLen := visualLen(prefixStr)
        if curLen > indent {
            c.Fprintln(w, prefixStr)
            prefixStr = strings.Repeat(" ", indent)
            curLen = indent
        }
        if len(words) == 0 {
            if curLen > 0 {
                c.Fprintln(w, prefixStr)
            }
            return
        }
        indentStr := strings.Repeat(" ", indent)
        var cur strings.Builder
        cur.WriteString(prefixStr)
        for _, word := range words {
            wlen := visualLen(word)
            space := 0
            if curLen > indent {
                space = 1
            }
            if curLen+space+wlen > width {
                c.Fprintln(w, cur.String())
                cur.Reset()
                cur.WriteString(indentStr)
                cur.WriteString(word)
                curLen = indent + wlen
            } else {
                if space > 0 {
                    cur.WriteString(" ")
                    curLen++
                }
                cur.WriteString(word)
                curLen += wlen
            }
        }
        if curLen > indent {
            c.Fprintln(w, cur.String())
        }
        return
    }

    if len(words) == 0 {
        return
    }
    indentStr := strings.Repeat(" ", indent)
    var cur strings.Builder
    cur.WriteString(indentStr)
    curLen := indent
    for _, word := range words {
        wlen := visualLen(word)
        space := 0
        if curLen > indent {
            space = 1
        }
        if curLen+space+wlen > width {
            c.Fprintln(w, cur.String())
            cur.Reset()
            cur.WriteString(indentStr)
            cur.WriteString(word)
            curLen = indent + wlen
        } else {
            if space > 0 {
                cur.WriteString(" ")
                curLen++
            }
            cur.WriteString(word)
            curLen += wlen
        }
    }
    if curLen > indent {
        c.Fprintln(w, cur.String())
    }
}
```

## Testing

After the fix, existing test snapshots that captured the broken output must be
updated. New test cases should cover:

- `UsageLine` with a single `\n` (two usage lines)
- `UsageLine` with multiple `\n` (three+ usage lines)
- `UsageLine` with a trailing `\n` (no extra blank line emitted)
- A segment wider than `width` that still wraps mid-line
- `Description` text (no newlines) still wraps identically to before
