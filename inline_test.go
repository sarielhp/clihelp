package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderInlinePlainText(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "hello world")
	if buf.String() != "hello world" {
		t.Errorf("plain text: got %q", buf.String())
	}
}

func TestRenderInlineBold(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "**bold**")
	want := "\x1b[1mbold\x1b[22m"
	if buf.String() != want {
		t.Errorf("bold: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineItalic(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "*italic*")
	want := "\x1b[3mitalic\x1b[23m"
	if buf.String() != want {
		t.Errorf("italic: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineCode(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "use `--flag` here")
	want := "use \x1b[32m--flag\x1b[39m here"
	if buf.String() != want {
		t.Errorf("code: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineLink(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "see [docs](https://example.com)")
	want := "see \x1b]8;;https://example.com\x1b\\docs\x1b]8;;\x1b\\"
	if buf.String() != want {
		t.Errorf("link: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineStrikethrough(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "~~strike~~")
	want := "\x1b[9mstrike\x1b[29m"
	if buf.String() != want {
		t.Errorf("strikethrough: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineBackslashEscape(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "\\*not italic\\*")
	if buf.String() != "*not italic*" {
		t.Errorf("escape: got %q, want %q", buf.String(), "*not italic*")
	}
}

func TestRenderInlineCodeEscapesMarkdown(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "`**not bold**`")
	want := "\x1b[32m**not bold**\x1b[39m"
	if buf.String() != want {
		t.Errorf("code escapes: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineUnmatchedDelimiter(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "*unclosed")
	if buf.String() != "*unclosed" {
		t.Errorf("unmatched: got %q, want %q", buf.String(), "*unclosed")
	}
}

func TestRenderInlineLinkWithNestedParens(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "see [example](https://example.com/a(b)c)")
	want := "see \x1b]8;;https://example.com/a(b)c\x1b\\example\x1b]8;;\x1b\\"
	if buf.String() != want {
		t.Errorf("nested parens link: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineLinkWithMultipleNestedParens(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "see [wiki](https://en.wikipedia.org/wiki/Go_(programming_language))")
	want := "see \x1b]8;;https://en.wikipedia.org/wiki/Go_(programming_language)\x1b\\wiki\x1b]8;;\x1b\\"
	if buf.String() != want {
		t.Errorf("wiki link: got %q, want %q", buf.String(), want)
	}
}

func TestRenderInlineMixed(t *testing.T) {
	var buf bytes.Buffer
	renderInline(&buf, "**bold** and *italic* and `code`")
	got := buf.String()
	if !strings.Contains(got, "\x1b[1mbold\x1b[22m") {
		t.Errorf("mixed missing bold: %q", got)
	}
	if !strings.Contains(got, "\x1b[3mitalic\x1b[23m") {
		t.Errorf("mixed missing italic: %q", got)
	}
	if !strings.Contains(got, "\x1b[32mcode\x1b[39m") {
		t.Errorf("mixed missing code: %q", got)
	}
}
