package clihelp

import (
	"bytes"
	"io"
	"strings"
)

// writeUsage writes the "Usage:" line, wrapped under itself. A bracketed group
// such as "[--tags TAGS]" is one unit: breaking it between the flag and its
// value leaves "[--tags" at the end of one row and "TAGS]" at the start of the
// next, which reads as two different things.
func writeUsage(w io.Writer, th Theme, o Options, termWidth int, usage string) {
	room := wrapWidth(termWidth, 8, o.maxContent()) - 8
	wrapGlued(w, room, usage, func(b io.Writer, text string) {
		reflowMargin(b, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 0, 8,
			"Usage:", o.inline(text), th.Hdr)
	})
}

// wrapGlued runs render over text with the spaces inside its bracketed groups
// made unbreakable, then turns them back into spaces in the output. room is the
// columns a row of that text has; a group wider than a row is left breakable,
// because holding it together would overflow the terminal.
func wrapGlued(w io.Writer, room int, text string, render func(io.Writer, string)) {
	var buf bytes.Buffer
	render(&buf, glueGroups(text, min(maxGluedGroup, room-1)))
	_, _ = io.WriteString(w, strings.ReplaceAll(buf.String(), groupSpace, " "))
}

// groupSpace stands in for a space that must not be a line break. It is the
// Unicode noncharacter U+FDD0, set aside for exactly this kind of internal use:
// it is not white space to strings.Fields, so the reflow keeps it inside its
// word; it cannot occur in real text; and go-runewidth measures it as one column
// under both the narrow and the East Asian width rules, like the space it
// replaces. A private-use rune looked equivalent and was not — U+E000 is two
// columns under East Asian rules, so a glued group cost an extra column per
// space and the same page wrapped differently depending on the user's locale.
const groupSpace = "\ufdd0"

// maxGluedGroup bounds the groups kept whole; a longer one is prose that
// happens to sit in brackets, and holding it together would overflow the line.
const maxGluedGroup = 40

// glueGroups replaces the white space inside each balanced [..] or <..> group of
// s with groupSpace, for groups whose brackets are at most limit runes apart. An
// unbalanced opener is left alone. An angle bracket only counts when it hugs its
// text — "<file>" opens a group, "< in.txt" and "a > b" do not — so a shell
// redirect or a comparison in a usage line is not mistaken for one.
func glueGroups(s string, limit int) string {
	runes := []rune(s)
	glued := make([]bool, len(runes))
	isSpace := func(i int) bool { return i < 0 || i >= len(runes) || runes[i] == ' ' || runes[i] == '\t' }
	for _, pair := range [][2]rune{{'[', ']'}, {'<', '>'}} {
		var open []int
		for i, r := range runes {
			switch {
			case r == pair[0] && (pair[0] == '[' || !isSpace(i+1)):
				open = append(open, i)
			case r == pair[1] && len(open) > 0 && (pair[1] == ']' || !isSpace(i-1)):
				start := open[len(open)-1]
				open = open[:len(open)-1]
				if i-start <= limit {
					for j := start; j <= i; j++ {
						glued[j] = true
					}
				}
			}
		}
	}
	for i, r := range runes {
		if (r == ' ' || r == '\t') && glued[i] {
			runes[i] = []rune(groupSpace)[0]
		}
	}
	return string(runes)
}
