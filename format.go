package clihelp

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

// DefaultMaxColIndent defines the standard column threshold for description
// text alignment in two-column command and option listings (GNU standard: 24).
const DefaultMaxColIndent = 24

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(?:\x1b\\|\x07)`)

// stripANSI removes both CSI escape sequences (e.g. \x1b[31m) and OSC
// sequences (e.g. \x1b]8;;url\x1b\ for hyperlinks, \x1b]0;title\x07 for
// window titles) from s, returning only the visible text.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// visualLen returns the display column width of s, ignoring ANSI escape
// codes. Wide East-Asian characters count as two columns.
func visualLen(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// splitLines splits text on '\n', preserving empty segments so consecutive
// newlines produce blank lines. Trailing '\r' (CRLF line endings) is trimmed.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	var out []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			out = append(out, strings.TrimSuffix(text[start:i], "\r"))
			start = i + 1
		}
	}
	out = append(out, strings.TrimSuffix(text[start:], "\r"))
	return out
}

// reflowSegment word-wraps a single paragraph (no newlines) so that no visual
// line exceeds width columns. An optional prefix is placed in its own first-line
// column and following lines are indented to align it.
func reflowSegment(w io.Writer, c *color.Color, prefixColor *color.Color, width, indent int, prefix, text string) {
	words := strings.Fields(text)

	if prefix != "" {
		prefixDisplay := "  " + prefix
		if prefixColor != nil {
			prefixDisplay = prefixColor.Sprint(prefixDisplay)
		}
		var prefixStr string
		var curLen int
		if visualLen(prefixDisplay)+2 > indent {
			c.Fprintln(w, prefixDisplay)
			prefixStr = strings.Repeat(" ", indent)
			curLen = indent
		} else {
			prefixStr = fmt.Sprintf("  %-*s", indent-2, prefix)
			if prefixColor != nil {
				prefixStr = prefixColor.Sprint(prefixStr)
			}
			curLen = visualLen(prefixStr)
		}
		if len(words) == 0 {
			if visualLen(prefixDisplay)+2 <= indent {
				c.Fprintln(w, prefixDisplay)
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

// reflow word-wraps text so that no visual line exceeds width columns. It
// preserves intentional newlines in text by processing each line segment
// independently. An optional prefix is placed in its own first-line column
// and following lines are indented to align it. Width is measured in visible
// characters, so ANSI escape codes and multi-byte runes are ignored when
// deciding where to wrap.
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string, prefixColors ...*color.Color) {
	var prefixColor *color.Color
	if len(prefixColors) > 0 {
		prefixColor = prefixColors[0]
	}
	if prefix != "" && indent < 2 {
		indent = 2
	}
	segments := splitLines(strings.TrimSpace(text))
	for i, seg := range segments {
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
				if prefixColor != nil {
					prefixStr = prefixColor.Sprint(prefixStr)
				}
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
		reflowSegment(w, c, prefixColor, width, indent, prefix, seg)
		prefix = ""
	}
}

// separator writes a horizontal rule in the accent color.
func separator(w io.Writer, th Theme, width int) {
	th.Accent.Fprintln(w, strings.Repeat("=", width))
}

// FirstSentence returns the first sentence of s, or the first line/paragraph if shorter.
func FirstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "\n\n"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, "\n"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, ". "); idx != -1 {
		return s[:idx+1]
	}
	return s
}

// firstSentence is an internal alias for FirstSentence.
func firstSentence(s string) string {
	return FirstSentence(s)
}

// commandArgs extracts the positional argument signature for cmd, if any.
func commandArgs(cmd Command) string {
	if len(cmd.Parameters) > 0 {
		var parts []string
		for _, p := range cmd.Parameters {
			if p.Name != "" {
				parts = append(parts, p.Name)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	if cmd.UsageLine == "" {
		return ""
	}

	synopsis := strings.Split(cmd.UsageLine, " — ")[0]
	synopsis = strings.Split(synopsis, " - ")[0]
	words := strings.Fields(synopsis)
	if len(words) <= 1 {
		return ""
	}

	var args []string
	for _, w := range words {
		lower := strings.ToLower(w)
		switch lower {
		case "[options]", "[flags]", "[options...]", "[flags...]",
			"<subcommand>", "[subcommand]", "<command>", "[command]",
			cmd.Name, strings.ToLower(cmd.Name):
			continue
		}
		if strings.HasPrefix(w, "<") || strings.HasPrefix(w, "[") {
			args = append(args, w)
		}
	}
	return strings.Join(args, " ")
}

// DisplayName renders a command name followed by its aliases in parentheses.
func DisplayName(c Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + " (" + strings.Join(c.Aliases, ", ") + ")"
}

// DisplayNameWithArgs renders a command name with aliases and positional argument signature.
func DisplayNameWithArgs(c Command) string {
	name := DisplayName(c)
	args := commandArgs(c)
	if args != "" {
		name += " " + args
	}
	return name
}

// displayNameWithArgs is an internal alias for DisplayNameWithArgs.
func displayNameWithArgs(c Command) string {
	return DisplayNameWithArgs(c)
}

// title returns the explicit help title of cmd, falling back to its name.
func title(c *Command) string {
	if c.Title != "" {
		return c.Title
	}
	return c.Name
}

// subcommandEntries returns the display list for the Subcommands section,
// preferring explicit entries over the structural Subcommands tree.
func subcommandEntries(c *Command) []Param {
	if len(c.SubcommandEntries) > 0 {
		return c.SubcommandEntries
	}
	var out []Param
	for i := range c.Subcommands {
		if !c.Subcommands[i].Hidden {
			out = append(out, Param{
				Name:        displayNameWithArgs(c.Subcommands[i]),
				Description: c.Subcommands[i].Description,
			})
		}
	}
	return out
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wrapWidth calculates the effective wrapping width for a given terminal
// width, indent, and content cap (maxContent).
func wrapWidth(termWidth, indent, maxContent int) int {
	return min(termWidth, indent+maxContent)
}

// appName returns the display name for the app, falling back to "app".
func appName(a *App) string {
	if a.Name != "" {
		return a.Name
	}
	return "app"
}

// colIndent returns the indent (max visible width + 4, capped at DefaultMaxColIndent)
// so that entries line up cleanly without excessive horizontal spacing.
func colIndent(params []Param) int {
	maxW := 0
	for _, p := range params {
		l := visualLen(p.Name)
		if l+4 <= DefaultMaxColIndent && l > maxW {
			maxW = l
		}
	}
	if maxW == 0 {
		return DefaultMaxColIndent
	}
	return maxW + 4
}

// inline renders inline markdown in s to a string with ANSI/OSC8 sequences.
func inline(s string) string {
	var buf strings.Builder
	renderInline(&buf, s)
	return buf.String()
}
