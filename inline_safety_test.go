package clihelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// H3 — author strings reach the terminal byte for byte.
//
// Descriptions, usage lines and notes are author-supplied, but authors are not
// the only source: they come from config files, embedded JSON and translation
// catalogues, and `__clihelp` exists so a packager can set up a program whose
// author never mounted a completion command. A control byte in any of them used
// to be copied straight to the terminal, which acts on it.
func TestAuthorStringsCannotSteerTheTerminal(t *testing.T) {
	for _, tt := range []struct{ name, in string }{
		{"clear screen", "hello \x1b[2J\x1b[1;1H world"},
		{"hide the cursor, never restored", "see \x1b[?25l here"},
		{"alternate screen buffer", "x \x1b[?1049h y"},
		{"reverse video, not reset by the body colour", "x \x1b[7m y"},
		{"window title, survives StripANSI", "t \x1b]0;pwn"},
		{"ST inside a URL closes our own OSC", "[x](\x1b\\\x1b[2Jpwned)"},
		{"BEL inside a URL closes our own OSC", "[x](http://a\x07\x1b[31mRED)"},
		{"bare carriage return overwrites the row", "keep this\rGONE"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := Inline(tt.in)
			for _, b := range []byte(out) {
				if b == 0x1b || b == 0x07 || b == '\r' {
					// Only escapes this package emits are allowed, and it emits
					// none for input carrying no markdown.
					if !strings.Contains(tt.in, "[") {
						t.Fatalf("a control byte from the author string survives: %q -> %q", tt.in, out)
					}
				}
			}
			if strings.ContainsAny(StripANSI(out), "\x1b\a\r") {
				t.Errorf("a control byte reaches the terminal: %q -> %q", tt.in, out)
			}
		})
	}
}

// M5 — a space in a URL split the OSC 8 escape, so its bytes became visible
// text and a bare newline landed inside an OSC string.
func TestLinkURLCannotBreakTheEscape(t *testing.T) {
	had := color.NoColor
	color.NoColor = false // this is a claim about the escape, so emit escapes
	defer func() { color.NoColor = had }()

	for _, url := range []string{
		"https://example.com/the big manual.html",
		"https://example.com/a\nb",
		"https://example.com/a\tb",
		"https://example.com/a\x7fb",
	} {
		out := Inline("see [the manual](" + url + ") now")
		payload := out
		if i := strings.Index(payload, osc8); i >= 0 {
			payload = payload[i+len(osc8):]
			if j := strings.Index(payload, oscEnd); j >= 0 {
				payload = payload[:j]
			}
		}
		if strings.ContainsAny(payload, " \t\n\x7f") {
			t.Errorf("URL %q leaves a splittable byte in the OSC payload: %q", url, payload)
		}
		if strings.Contains(StripANSI(out), "]8;;") {
			t.Errorf("URL %q leaked OSC bytes into visible text: %q", url, StripANSI(out))
		}
	}
}

// The verbatim paths bypass the inline renderer entirely and must be sanitised
// too: a raw note, a fenced block and a heading all used to pass a carriage
// return straight through, and an example line is meant to be copy-pasted.
func TestVerbatimPathsAreSanitised(t *testing.T) {
	app := &App{Name: "demo", Commands: []Command{{
		Name: "run", Description: "Runs it.", Run: func(*Context) error { return nil },
		Notes: []Note{
			{Heading: "Gotcha\rOVERWRITTEN", Raw: true, Text: "keep this\rGONE"},
			{Heading: "Fenced", Text: "```\nkeep that\rGONE\n```"},
		},
	}}}
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Extended: true}, "run")
	requireRendered(t, buf.String())
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.ContainsRune(line, '\r') {
			t.Errorf("a carriage return survives into output: %q", line)
		}
	}
}
