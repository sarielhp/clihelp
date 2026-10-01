// Package text holds the string measurement and truncation helpers that the
// clihelp package and its doc and tree subpackages must agree on. They live in
// one internal package so that no subpackage grows its own copy, which is how a
// width measurement once came to disagree with the renderer.
package text

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;:?<=>!]*[@-~]|\x1b\][^\x1b\x07]*(?:\x07|\x1b\\)?`)

// StripANSI removes CSI escape sequences (e.g. \x1b[31m) and OSC sequences
// (hyperlinks and window titles), leaving the text a terminal displays. That
// includes the OSC 8 hyperlinks this library sets.
//
// The subpackages share this one copy. tree/ had its own width
// measurement built on a third-party stripper that does not handle OSC, so a
// hyperlink measured 22 columns wide instead of 4.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// VisualWidth is the number of terminal columns s occupies: escape sequences
// cost nothing, and a wide rune costs two. Every layout decision in this library
// and its subpackages has to measure the same way, or columns do not line up.
func VisualWidth(s string) int {
	// ToValidUTF8 makes the measurement additive over a whitespace join, which
	// reflowWords depends on: it accumulates per-word widths and compares the sum
	// to the line width, and runewidth's grapheme segmentation made a stray byte
	// measure differently on its own than in context, pushing a line one column
	// past the width it was given. A terminal draws one replacement glyph for
	// such a byte, which is what this now measures.
	return runewidth.StringWidth(expandTabs(strings.ToValidUTF8(StripANSI(s), "\uFFFD")))
}

// tabStop is the column interval a terminal advances a tab to.
const tabStop = 8

// expandTabs replaces tabs with the spaces a terminal would draw.
//
// runewidth measures a tab as zero columns while a terminal advances to the next
// stop, so a tab in a Param.Name or a list marker desynchronised the hanging
// indent from what was actually on screen.
func expandTabs(s string) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			pad := tabStop - col%tabStop
			b.WriteString(strings.Repeat(" ", pad))
			col += pad
			continue
		}
		b.WriteRune(r)
		col += runewidth.RuneWidth(r)
	}
	return b.String()
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
	if idx := sentenceBreak(s); idx != -1 {
		return s[:idx+1]
	}
	return s
}

// sentenceBreak finds the first ". " that is not inside a markdown construct,
// or -1.
//
// Cutting at the first ". " full stop left raw markup in the help: a version
// number in a URL ("…/v1. 2/y"), an abbreviation in a code span ("`a. b`") or
// an emphasised phrase ("**a. b**") all contain one, and the truncated result
// was then rendered as literal "[it](http://x/v1." on screen.
func sentenceBreak(s string) int {
	code, emphasis, link := false, false, 0
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '`':
			code = !code
		case !code && i+1 < len(s) && s[i] == '*' && s[i+1] == '*':
			emphasis = !emphasis
			i++
		case !code && s[i] == '[':
			link++
		case !code && s[i] == ')' && link > 0:
			link--
		case !code && !emphasis && link == 0 && s[i] == '.' && i+1 < len(s) && s[i+1] == ' ':
			return i
		}
	}
	return -1
}
