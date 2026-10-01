package clihelp

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/sarielhp/clihelp/internal/text"
)

// defaultMaxColIndent defines the standard column threshold for description
// text alignment in two-column command and option listings (GNU standard: 24).
const defaultMaxColIndent = 24

// stripANSI, visualLen and firstSentence are the text helpers shared with the
// doc and tree subpackages; internal/text owns the one copy.
func stripANSI(s string) string { return text.StripANSI(s) }

func visualLen(s string) int { return text.VisualWidth(s) }

func firstSentence(s string) string { return text.FirstSentence(s) }

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
// padPrefix lays the prefix out in its own column: margin columns of leading
// space, the prefix, then padding to indent.
//
// The margin used to be hardcoded at two. Only the description column could be
// moved, so RenderMan's nested Parameters and Flags lists — which add 6 to the
// indent to sit inside their command — put their names at column 2, outdented
// from their own headings, with the extra six columns appearing as a wider gap
// instead.
func padPrefix(prefix string, margin, indent int) string {
	pad := indent - margin - visualLen(prefix)
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", margin) + prefix + strings.Repeat(" ", pad)
}

func formatPrefix(w io.Writer, c *color.Color, prefixColor *color.Color, margin, indent int, prefix string, noWords bool) (string, int, bool) {
	prefixDisplay := strings.Repeat(" ", margin) + prefix
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
	prefixStr := padPrefix(prefix, margin, indent)
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
func emitLine(w io.Writer, c *color.Color, lead, text string) string {
	// The lead — a coloured prefix, or plain indent — is written as it is, and
	// only the text is wrapped in the ambient colour.
	//
	// Colouring the whole line at once was wrong: fatih/color closes a nested
	// colour with a full SGR reset, not with the outer colour, so the prefix's
	// closer killed the body colour for the rest of that row while the
	// continuation rows — which have no nested colour — kept it. Every flags
	// table had its first description line uncoloured and its wrapped remainder
	// coloured.
	text = strings.TrimRight(text, " ")
	reopen := ""
	if url := openHyperlink(lead + text); url != "" {
		text += osc8 + oscEnd
		reopen = osc8 + url + oscEnd
	}
	if text == "" {
		fmt.Fprintln(w, strings.TrimRight(lead, " "))
		return reopen
	}
	fmt.Fprintln(w, lead+c.Sprint(text))
	return reopen
}

func reflowWords(w io.Writer, c *color.Color, width, indent int, initialStr string, initialLen int, words []string) {
	indentStr := strings.Repeat(" ", indent)
	lead := initialStr
	var cur strings.Builder
	curLen := initialLen
	wrote := false
	lineHasWords := initialLen > indent
	for _, word := range words {
		wlen := visualLen(word)
		space := 0
		if lineHasWords {
			space = 1
		}
		// "does this line already hold something" is a fact about the line, not
		// about its width. Testing the width instead meant an over-long first
		// word flushed a line holding only the indent — or only the padded
		// prefix — and a zero-width word left the next one glued to it.
		if lineHasWords && curLen+space+wlen > width {
			reopen := emitLine(w, c, lead, cur.String())
			lead = indentStr
			cur.Reset()
			cur.WriteString(reopen) // zero width, so curLen is unchanged by it
			cur.WriteString(word)
			curLen = indent + wlen
			lineHasWords = true
		} else {
			if space > 0 {
				cur.WriteString(" ")
				curLen++
			}
			cur.WriteString(word)
			curLen += wlen
			lineHasWords = true
		}
		wrote = true
	}
	// A word of zero display width — a "[](url)" link, a stray "**", a
	// zero-width space — still belongs to a row that has a prefix to show, so
	// the test is whether anything was written, not how wide it came out.
	if wrote || curLen > indent {
		emitLine(w, c, lead, cur.String())
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
func reflowSegment(w io.Writer, c *color.Color, prefixColor *color.Color, width, margin, indent int, prefix, text string) {
	listPrefix, isList := detectListPrefix(text)
	if isList {
		remText := text[len(listPrefix):]
		words := strings.Fields(remText)
		hangingIndent := indent + visualLen(listPrefix)
		if prefix != "" {
			initialStr, initialLen, done := formatPrefix(w, c, prefixColor, margin, indent, prefix, false)
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
		initialStr, initialLen, done := formatPrefix(w, c, prefixColor, margin, indent, prefix, len(words) == 0)
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
// reflow lays text out at the default two-column margin. reflowMargin takes one
// explicitly, for lists nested inside another section.
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string, prefixColors ...*color.Color) {
	reflowMargin(w, c, width, 2, indent, prefix, text, prefixColors...)
}

func reflowMargin(w io.Writer, c *color.Color, width, margin, indent int, prefix, text string, prefixColors ...*color.Color) {
	var prefixColor *color.Color
	if len(prefixColors) > 0 {
		prefixColor = prefixColors[0]
	}
	// One line, always. The prefix is a name — a command, a flag, a parameter —
	// and the column arithmetic below measures it as a single token; a newline in
	// one split it across two rows while the width was computed on the whole
	// thing. Same class as the tree/firstSentence defect already fixed.
	prefix = strings.Join(strings.Fields(prefix), " ")
	if prefix != "" && indent < margin {
		indent = margin
	}
	if strings.TrimSpace(text) == "" {
		if prefix != "" {
			// The prefix is the name of something — a command, a flag — and it
			// has to be listed whether or not it came with a description.
			formatPrefix(w, c, prefixColor, margin, indent, prefix, true)
		}
		return
	}
	segments := splitLines(strings.Trim(text, "\r\n"))
	for i, seg := range segments {
		// A line of spaces is a blank line. Testing for exact emptiness meant a
		// paragraph break pasted from an editor that leaves indentation behind
		// fell through to the word reflow, which found no words and returned
		// silently — so the break vanished.
		if strings.TrimSpace(seg) == "" {
			seg = ""
		}
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := padPrefix(prefix, margin, indent)
				if prefixColor != nil {
					prefixStr = prefixColor.Sprint(prefixStr)
				}
				c.Fprintln(w, prefixStr)
				prefix = ""
			} else {
				c.Fprintln(w) // not indent columns of trailing spaces
			}
			continue
		}
		if seg == "" {
			continue
		}
		reflowSegment(w, c, prefixColor, width, margin, indent, prefix, seg)
		prefix = ""
	}
}

// separator writes a horizontal rule in the accent color.
func separator(w io.Writer, th Theme, width int) {
	th.Accent.Fprintln(w, strings.Repeat("=", width))
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

// displayNameWithAliases renders a command name followed by its aliases in
// parentheses. The name says which of the two it is: doc/ has its own
// displayName that deliberately returns the bare name, because a Markdown
// heading carrying aliases is not a heading anyone can link to.
func displayNameWithAliases(c Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + " (" + strings.Join(c.Aliases, ", ") + ")"
}

// displayNameWithArgs renders a command name with aliases and positional argument signature.
func displayNameWithArgs(c Command) string {
	name := displayNameWithAliases(c)
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
				Name:        displayNameWithAliases(c.Subcommands[i]),
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

// colIndent returns the indent (max visible width + 4, capped at defaultMaxColIndent)
// so that entries line up cleanly without excessive horizontal spacing.
// colIndentFor is colIndent, reduced so that at least minText columns are left
// for the description.
//
// colIndent knows nothing about the terminal, so a 13-column flag name at width
// 20 left about three columns for the text and every word took its own line.
// Explain feeds the shell's real $COLUMNS, so a narrow pane reaches this. A name
// that no longer fits the reduced column takes formatPrefix's own-line branch,
// which is already the tested behaviour for a name wider than its column.
func colIndentFor(params []Param, termWidth, minText int) int {
	return clampIndent(colIndent(params), termWidth, minText)
}

// clampIndent reduces a description column so that at least minText columns are
// left for the text, splitting the difference on a terminal too narrow to give
// minText away outright.
func clampIndent(indent, termWidth, minText int) int {
	if termWidth <= 0 {
		return indent
	}
	if half := termWidth / 2; minText > half {
		minText = half
	}
	if limit := termWidth - minText; limit >= 6 && indent > limit {
		return limit
	}
	return indent
}

// minTextColumns is the narrowest description column worth wrapping into.
const minTextColumns = 20

func colIndent(params []Param) int {
	maxW := 0
	for _, p := range params {
		l := visualLen(p.Name)
		if l+4 <= defaultMaxColIndent && l > maxW {
			maxW = l
		}
	}
	if maxW == 0 {
		return defaultMaxColIndent
	}
	return maxW + 4
}

// inline renders inline markdown in s, honouring the global colour setting.
//
// fatih/color sets NoColor when stdout is not a terminal, and every theme colour
// then disappears — but this renderer had its own hardcoded escapes and kept
// emitting them, so `myapp --help > help.txt` produced a file that was not plain
// text and `grep '^  --flag'` on it did not match. With colour off a link is
// spelled as its label alone: a visible URL in a help page is what OSC 8 exists
// to avoid, and TestExampleAppNoBareMarkdownAndNoVisibleURLs says so. Only
// man.go asks for the "text (url)" form, and it asks explicitly.
func inline(s string) string {
	return renderInlineTo(s, color.NoColor)
}

func renderInlineTo(s string, plain bool) string {
	var buf strings.Builder
	renderInline(&buf, s, plain)
	return buf.String()
}
