package clihelp

import (
	"fmt"
	"io"
	"strings"
)

const (
	osc8         = "\x1b]8;;"
	oscEnd       = "\x1b\\"
	sgrBoldOn    = "\x1b[1m"
	sgrBoldOff   = "\x1b[22m"
	sgrItalicOn  = "\x1b[3m"
	sgrItalicOff = "\x1b[23m"
	sgrStrikeOn  = "\x1b[9m"
	sgrStrikeOff = "\x1b[29m"
	sgrCodeOn    = "\x1b[32m"
	sgrCodeOff   = "\x1b[39m"
)

// Inline renders inline markdown in s to a string with ANSI/OSC8 sequences.
// It is the exported form of the internal inline helper used by the renderer.
func Inline(s string) string {
	return inline(s)
}

// emphasisEnd returns the index in s at which the emphasis span opened by
// marker and beginning at start closes, or -1 when it does not close.
//
// The span must not begin or end with whitespace, which is what tells emphasis
// from arithmetic: without the rule, "2 * 3 * 4" rendered as "2  3  4", the
// asterisks eaten by an italic span nobody wrote.
func emphasisEnd(s string, start int, marker string) int {
	if start >= len(s) || isSpaceByte(s[start]) {
		return -1
	}
	rel := strings.Index(s[start:], marker)
	if rel <= 0 {
		return -1
	}
	end := start + rel
	if isSpaceByte(s[end-1]) {
		return -1
	}
	return end
}

// renderInline writes s to w, translating inline markdown patterns into
// ANSI escape sequences and OSC 8 hyperlinks for terminal display.
//
// Recognized patterns (in priority order):
//   - `code`          inline code (green foreground)
//   - [text](url)     OSC 8 clickable hyperlink (or visible text when showURLs is true)
//   - **bold**        bold text
//   - *italic*        italic text
//   - ~~strikethrough~~  strikethrough text
//   - \X              backslash escapes the next character
func renderInline(w io.Writer, s string, showURLs ...bool) {
	show := len(showURLs) > 0 && showURLs[0]
	for i := 0; i < len(s); {
		// backslash escape
		if s[i] == '\\' && i+1 < len(s) {
			w.Write([]byte{s[i+1]})
			i += 2
			continue
		}
		// inline code `...`
		if s[i] == '`' {
			j := strings.IndexByte(s[i+1:], '`')
			if j >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrCodeOn, s[i+1:i+1+j], sgrCodeOff)
				i += j + 2
				continue
			}
		}
		// link [text](url)
		if text, url, advance, ok := parseMarkdownLink(s, i); ok {
			if show {
				fmt.Fprintf(w, "%s (%s)", text, url)
			} else {
				fmt.Fprintf(w, "%s%s%s%s%s", osc8, url, oscEnd, text, osc8+oscEnd)
			}
			i += advance
			continue
		}
		// **bold**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			if end := emphasisEnd(s, i+2, "**"); end >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrBoldOn, s[i+2:end], sgrBoldOff)
				i = end + 2
				continue
			}
			// Not emphasis: two literal asterisks. Falling through here let the
			// italic branch take the second one as its own closer, swallowing
			// both and emitting a pair of escapes around nothing.
			w.Write([]byte{'*'})
			i++
			continue
		}
		// *italic*
		if s[i] == '*' {
			if end := emphasisEnd(s, i+1, "*"); end >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrItalicOn, s[i+1:end], sgrItalicOff)
				i = end + 1
				continue
			}
		}
		// ~~strikethrough~~
		if i+1 < len(s) && s[i] == '~' && s[i+1] == '~' {
			end := strings.Index(s[i+2:], "~~")
			if end >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrStrikeOn, s[i+2:i+2+end], sgrStrikeOff)
				i += end + 4
				continue
			}
		}
		w.Write([]byte{s[i]})
		i++
	}
}

func parseMarkdownLink(s string, i int) (text, url string, advance int, ok bool) {
	if s[i] != '[' {
		return "", "", 0, false
	}
	cb := strings.IndexByte(s[i+1:], ']')
	if cb < 0 || i+1+cb+1 >= len(s) || s[i+1+cb+1] != '(' {
		return "", "", 0, false
	}
	depth := 1
	ue := -1
	for j := i + 1 + cb + 2; j < len(s); j++ {
		if s[j] == '(' {
			depth++
		} else if s[j] == ')' {
			depth--
			if depth == 0 {
				ue = j - (i + 1 + cb + 2)
				break
			}
		}
	}
	if ue < 0 {
		return "", "", 0, false
	}
	text = s[i+1 : i+1+cb]
	url = s[i+1+cb+2 : i+1+cb+2+ue]
	advance = cb + 2 + ue + 2
	return text, url, advance, true
}
