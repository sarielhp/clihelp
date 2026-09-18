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

// sanitizeControl replaces the control bytes a terminal would act on.
//
// Everything this package renders is a string the application's author wrote,
// and an author string is not always a compile-time literal: descriptions come
// from config files, embedded JSON and translation catalogues, and __clihelp
// exists so a packager can set a program up whose author mounted nothing. A
// stray ESC in one of those used to be copied to the terminal verbatim, where
// "\x1b[2J" clears the screen, "\x1b[?1049h" switches to the alternate buffer
// and "\x1b[?25l" hides the cursor for good.
//
// Newline and tab survive, because reflow splits prose on one and
// detectListPrefix reads the other. Everything else becomes U+FFFD, which is
// what a terminal shows for a byte it cannot render anyway.
func sanitizeControl(s string) string {
	if strings.IndexFunc(s, isActionableControl) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isActionableControl(r) {
			return '\uFFFD'
		}
		return r
	}, s)
}

func isActionableControl(r rune) bool {
	return r != '\n' && r != '\t' && (r < 0x20 || r == 0x7f)
}

// oscSafeURL percent-encodes every byte that must not appear inside an OSC 8
// string.
//
// A terminal reads the payload up to ST or BEL, and reflow splits the rendered
// string on whitespace — so a space, a tab or a newline in the author's URL ends
// the sequence in the middle of itself, and the user sees the escape's own bytes
// printed. Percent-encoding is what a browser and xdg-open expect, so no working
// hyperlink stops working.
func oscSafeURL(url string) string {
	var b strings.Builder
	for i := 0; i < len(url); i++ {
		if c := url[i]; c <= 0x20 || c == 0x7f {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(url[i])
	}
	return b.String()
}

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
	s = sanitizeControl(s)
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
				fmt.Fprintf(w, "%s%s%s%s%s", osc8, oscSafeURL(url), oscEnd, text, osc8+oscEnd)
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
