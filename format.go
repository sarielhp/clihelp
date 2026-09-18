package clihelp

import (
	"io"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

// DefaultMaxColIndent defines the standard column threshold for description
// text alignment in two-column command and option listings (GNU standard: 24).
const DefaultMaxColIndent = 24

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;:?<=>!]*[@-~]|\x1b\][^\x1b\x07]*(?:\x07|\x1b\\)?`)

// stripANSI removes both CSI escape sequences (e.g. \x1b[31m) and OSC
// sequences (e.g. \x1b]8;;url\x1b\ for hyperlinks, \x1b]0;title\x07 for
// window titles) from s, returning only the visible text.
func stripANSI(s string) string {
	return StripANSI(s)
}

// StripANSI removes the escape sequences clihelp itself emits — CSI colour codes
// and OSC sequences, including the OSC 8 hyperlinks this library sets — leaving
// the text a terminal actually displays.
//
// It is exported because the subpackages need it. tree/ had its own width
// measurement built on a third-party stripper that does not handle OSC, so a
// hyperlink measured 22 columns wide instead of 4.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// visualLen returns the display column width of s, ignoring ANSI escape
// codes. Wide East-Asian characters count as two columns.
func visualLen(s string) int {
	return VisualWidth(s)
}

// VisualWidth is the number of terminal columns s occupies: escape sequences
// cost nothing, and a wide rune costs two. Every layout decision in this library
// and its subpackages has to measure the same way, or columns do not line up.
func VisualWidth(s string) int {
	return runewidth.StringWidth(StripANSI(s))
}

// splitLines splits text on '\n', preserving empty segments so consecutive
// newlines produce blank lines. Trailing '\r' (CRLF line endings) is trimmed.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	// No sanitising here: splitLines also runs on text this package has already
	// rendered, where an ESC is one of ours and replacing it would break the very
	// escape it belongs to. Author strings are sanitised at the boundary instead
	// — renderInline for prose, and the verbatim callers below for the rest.
	var out []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			out = append(out, strings.TrimSuffix(text[start:i], "\r"))
			start = i + 1
		}
	}
	return append(out, strings.TrimSuffix(text[start:], "\r"))
}

// padPrefix lays prefix out in its own column, padded to indent display
// columns. fmt's "%-*s" pads by rune count, so a wide (CJK) name — two columns
// per rune — pushed its description out of line with every other row.
func padPrefix(prefix string, indent int) string {
	pad := indent - 2 - visualLen(prefix)
	if pad < 0 {
		pad = 0
	}
	return "  " + prefix + strings.Repeat(" ", pad)
}

func formatPrefix(w io.Writer, c *color.Color, prefixColor *color.Color, indent int, prefix string, noWords bool) (string, int, bool) {
	prefixDisplay := "  " + prefix
	if prefixColor != nil {
		prefixDisplay = prefixColor.Sprint(prefixDisplay)
	}
	if visualLen(prefixDisplay)+2 > indent {
		c.Fprintln(w, prefixDisplay)
		if noWords {
			return "", 0, true
		}
		return strings.Repeat(" ", indent), indent, false
	}
	if noWords {
		c.Fprintln(w, prefixDisplay)
		return "", 0, true
	}
	prefixStr := padPrefix(prefix, indent)
	if prefixColor != nil {
		prefixStr = prefixColor.Sprint(prefixStr)
	}
	return prefixStr, visualLen(prefixStr), false
}

// openHyperlink returns the URL of an OSC 8 hyperlink that s leaves open, or "".
//
// The last introducer in s is either an opener — osc8, the URL, then oscEnd —
// or the closer, which is osc8 immediately followed by oscEnd. Anything else is
// a sequence we did not emit and cannot reopen.
func openHyperlink(s string) string {
	i := strings.LastIndex(s, osc8)
	if i < 0 {
		return ""
	}
	rest := s[i+len(osc8):]
	if strings.HasPrefix(rest, oscEnd) {
		return "" // that was the closer
	}
	if j := strings.Index(rest, oscEnd); j >= 0 {
		return rest[:j]
	}
	return ""
}

// emitLine writes one wrapped line and returns what the next line must start
// with to carry an unfinished hyperlink across the break.
//
// A hyperlink must not span a line: the per-line reset that fatih/color appends
// is an SGR reset, which does not close an OSC 8 sequence, so a link split
// across a wrap left an opener with no terminator. Anything that then truncates
// by line — App.Explain's budget, or `myapp --help | head` — dropped the
// terminator for good, and the terminal went on hyperlinking every cell it drew
// afterwards, the shell prompt included, after the program had exited.
func emitLine(w io.Writer, c *color.Color, line string) string {
	if url := openHyperlink(line); url != "" {
		c.Fprintln(w, line+osc8+oscEnd)
		return osc8 + url + oscEnd
	}
	c.Fprintln(w, line)
	return ""
}

func reflowWords(w io.Writer, c *color.Color, width, indent int, initialStr string, initialLen int, words []string) {
	indentStr := strings.Repeat(" ", indent)
	var cur strings.Builder
	cur.WriteString(initialStr)
	curLen := initialLen
	wrote := false
	reopen := ""
	for _, word := range words {
		wlen := visualLen(word)
		space := 0
		if curLen > indent {
			space = 1
		}
		if curLen+space+wlen > width {
			reopen = emitLine(w, c, cur.String())
			cur.Reset()
			cur.WriteString(indentStr)
			cur.WriteString(reopen) // zero width, so curLen is unchanged by it
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
		wrote = true
	}
	// A word of zero display width — a "[](url)" link, a stray "**", a
	// zero-width space — still belongs to a row that has a prefix to show, so
	// the test is whether anything was written, not how wide it came out.
	if wrote || curLen > indent {
		emitLine(w, c, cur.String())
	}
}

// detectListPrefix checks if s starts with a list item prefix:
// - numbered lists (1. , 2. , etc., including any leading whitespace)
// - bullet lists (- , * , • , including any leading whitespace)
func detectListPrefix(s string) (string, bool) {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) {
		return "", false
	}

	rem := s[i:]
	for _, b := range []string{"-", "*", "•"} {
		if strings.HasPrefix(rem, b) {
			after := rem[len(b):]
			if len(after) > 0 && (after[0] == ' ' || after[0] == '\t') {
				j := 0
				for j < len(after) && (after[j] == ' ' || after[j] == '\t') {
					j++
				}
				return s[:i+len(b)+j], true
			}
		}
	}

	digitLen := 0
	for digitLen < len(rem) && rem[digitLen] >= '0' && rem[digitLen] <= '9' {
		digitLen++
	}
	if digitLen > 0 && digitLen < len(rem) && rem[digitLen] == '.' {
		after := rem[digitLen+1:]
		if len(after) > 0 && (after[0] == ' ' || after[0] == '\t') {
			j := 0
			for j < len(after) && (after[j] == ' ' || after[j] == '\t') {
				j++
			}
			return s[:i+digitLen+1+j], true
		}
	}

	return "", false
}

// reflowSegment word-wraps a single paragraph (no newlines) so that no visual
// line exceeds width columns. An optional prefix is placed in its own first-line
// column and following lines are indented to align it. When text starts with a
// list prefix (bullet or numbered list), continuation lines use hanging indents
// aligned to the list item text (indent + visualLen(listPrefix)).
func reflowSegment(w io.Writer, c *color.Color, prefixColor *color.Color, width, indent int, prefix, text string) {
	listPrefix, isList := detectListPrefix(text)
	if isList {
		remText := text[len(listPrefix):]
		words := strings.Fields(remText)
		hangingIndent := indent + visualLen(listPrefix)
		if prefix != "" {
			initialStr, initialLen, done := formatPrefix(w, c, prefixColor, indent, prefix, false)
			if done {
				return
			}
			initialStr += listPrefix
			initialLen += visualLen(listPrefix)
			if len(words) == 0 {
				c.Fprintln(w, initialStr)
				return
			}
			reflowWords(w, c, width, hangingIndent, initialStr, initialLen, words)
			return
		}
		if len(words) == 0 {
			c.Fprintln(w, strings.Repeat(" ", indent)+strings.TrimRight(listPrefix, " \t"))
			return
		}
		initialStr := strings.Repeat(" ", indent) + listPrefix
		reflowWords(w, c, width, hangingIndent, initialStr, hangingIndent, words)
		return
	}

	words := strings.Fields(text)
	if prefix != "" {
		initialStr, initialLen, done := formatPrefix(w, c, prefixColor, indent, prefix, len(words) == 0)
		if done {
			return
		}
		reflowWords(w, c, width, indent, initialStr, initialLen, words)
		return
	}
	if len(words) == 0 {
		return
	}
	reflowWords(w, c, width, indent, strings.Repeat(" ", indent), indent, words)
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
	if strings.TrimSpace(text) == "" {
		if prefix != "" {
			// The prefix is the name of something — a command, a flag — and it
			// has to be listed whether or not it came with a description.
			formatPrefix(w, c, prefixColor, indent, prefix, true)
		}
		return
	}
	segments := splitLines(strings.Trim(text, "\r\n"))
	for i, seg := range segments {
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := padPrefix(prefix, indent)
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
	return SubcommandList(*c)
}

// SubcommandList is the subcommand list to display for a command: an explicit
// SubcommandEntries when the author supplied one, and otherwise the visible
// Subcommands under their display names.
//
// The preference is the whole point of SubcommandEntries — it documents
// subcommands the tree does not carry. It is exported so that the terminal help
// and the markdown generator cannot answer differently, which they did: doc/ was
// given a copy of this without the preference, and a documented-only entry
// showed in `--help` and was missing from the generated page.
func SubcommandList(c Command) []Param {
	if len(c.SubcommandEntries) > 0 {
		return c.SubcommandEntries
	}
	var out []Param
	for i := range c.Subcommands {
		if !c.Subcommands[i].Hidden {
			out = append(out, Param{
				Name:        DisplayName(c.Subcommands[i]),
				Description: c.Subcommands[i].Description,
			})
		}
	}
	return out
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
