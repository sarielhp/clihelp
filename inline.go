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

// renderInline writes s to w, translating inline markdown patterns into
// ANSI escape sequences and OSC 8 hyperlinks for terminal display.
//
// Recognized patterns (in priority order):
//   - `code`          inline code (green foreground)
//   - [text](url)     OSC 8 clickable hyperlink
//   - **bold**        bold text
//   - *italic*        italic text
//   - ~~strikethrough~~  strikethrough text
//   - \X              backslash escapes the next character
func renderInline(w io.Writer, s string) {
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
		if s[i] == '[' {
			cb := strings.IndexByte(s[i+1:], ']')
			if cb >= 0 && i+1+cb+1 < len(s) && s[i+1+cb+1] == '(' {
				ue := strings.IndexByte(s[i+1+cb+2:], ')')
				if ue >= 0 {
					text := s[i+1 : i+1+cb]
					url := s[i+1+cb+2 : i+1+cb+2+ue]
					fmt.Fprintf(w, "%s%s%s%s%s", osc8, url, oscEnd, text, osc8+oscEnd)
					i += cb + 2 + ue + 2
					continue
				}
			}
		}
		// **bold**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "**")
			if end >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrBoldOn, s[i+2:i+2+end], sgrBoldOff)
				i += end + 4
				continue
			}
		}
		// *italic*
		if s[i] == '*' {
			end := strings.IndexByte(s[i+1:], '*')
			if end >= 0 {
				fmt.Fprintf(w, "%s%s%s", sgrItalicOn, s[i+1:i+1+end], sgrItalicOff)
				i += end + 2
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
