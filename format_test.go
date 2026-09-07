package clihelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestVisualLen(t *testing.T) {
	s := "hello"
	if visualLen(s) != 5 {
		t.Errorf("visualLen(%q) = %d, want 5", s, visualLen(s))
	}
	s = "  hello"
	if visualLen(s) != 7 {
		t.Errorf("visualLen(%q) = %d, want 7", s, visualLen(s))
	}
	green := color.New(color.FgGreen).Sprint("podctl")
	if visualLen(green) != 6 {
		t.Fatalf("visualLen of colored %q = %d, want 6", green, visualLen(green))
	}
}

func TestReflowHonorsWidth(t *testing.T) {
	longText := "The quick brown fox jumps over the lazy dog near the riverside cottage daily."
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 20, 2, "", longText)
	max := 0
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if l := len(strip(line)); l > max {
			max = l
		}
	}
	if max > 20 {
		t.Fatalf("reflow exceeded width: max line %d (>20)\n%s", max, buf.String())
	}
}

func TestReflowMultibyteIsRuneAware(t *testing.T) {
	// Each CJK character is 3 bytes but 1 visible char. At width 10, only
	// ~10 visible chars fit per line regardless of byte lengths.
	longText := strings.Repeat("界 ", 30)
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 10, 0, "", longText)
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if n := len([]rune(strip(line))); n > 10 {
			t.Fatalf("line exceeded 10 visible chars (got %d): %q", n, strip(line))
		}
	}
}

func TestReflowNoSeparatorTheme(t *testing.T) {
	o, buf := captureOptions(80)
	o.Theme = &Theme{Separator: false, TitlePrefix: "USAGE: "}
	testApp().RenderCommand(o, "build")
	out := strip(buf.String())
	if strings.Contains(out, "===") {
		t.Errorf("separator rendered despite Theme.Separator=false:\n%q", out)
	}
	if !strings.Contains(out, "USAGE: build") {
		t.Errorf("custom TitlePrefix not applied: got %q, want %q", out, "USAGE: build")
	}
}

func TestRenderCommandReflowAtWidth60(t *testing.T) {
	th := defaultTheme()
	th.Separator = false
	o, buf := captureOptions(60)
	o.Theme = &th

	longDesc := "Compile, encode, and package raw audio source files into fully tagged MP3 podcast episodes with configurable bitrate, loudness normalization, and embedded ID3 metadata tags for distribution across multiple platforms and aggregators like Apple Podcasts, Spotify, and Google Podcasts."

	app := &App{
		Name: "podctl",
		Commands: []Command{
			{
				Name:        "build",
				Description: longDesc,
				UsageLine:   "podctl build [options] <source-file>",
				Options: []Option{
					{Flags: "-o, --output PATH", Description: "Write compiled MP3 output to specified PATH for later distribution and archival storage on cloud platforms"},
					{Flags: "--tags TAGS", Description: "Embed ID3 metadata tags including title, artist, album, episode number, season, and publication date for rich podcast episode descriptions"},
				},
				Notes: []Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use --bitrate 320 for highest quality music episodes or --bitrate 128 for voice-only spoken word content to optimize file size while maintaining acceptable audio fidelity for listeners on mobile connections.",
					},
				},
			},
		},
	}

	app.RenderCommand(o, "build")
	output := strip(buf.String())

	maxLine := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
		if len([]rune(line)) > 60 {
			t.Errorf("line exceeds 60 chars at width 60:\n  len=%d  %q", len([]rune(line)), line)
		}
	}
	t.Logf("width=60: max line = %d chars", maxLine)
}

func TestRenderCommandReflowAtWidth100(t *testing.T) {
	th := defaultTheme()
	th.Separator = false
	o, buf := captureOptions(100)
	o.Theme = &th

	longDesc := "Compile, encode, and package raw audio source files into fully tagged MP3 podcast episodes with configurable bitrate, loudness normalization, and embedded ID3 metadata tags for distribution across multiple platforms and aggregators like Apple Podcasts, Spotify, and Google Podcasts."

	app := &App{
		Name: "podctl",
		Commands: []Command{
			{
				Name:        "build",
				Description: longDesc,
				UsageLine:   "podctl build [options] <source-file>",
				Options: []Option{
					{Flags: "-o, --output PATH", Description: "Write compiled MP3 output to specified PATH for later distribution and archival storage on cloud platforms"},
					{Flags: "--tags TAGS", Description: "Embed ID3 metadata tags including title, artist, album, episode number, season, and publication date for rich podcast episode descriptions"},
				},
				Notes: []Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use --bitrate 320 for highest quality music episodes or --bitrate 128 for voice-only spoken word content to optimize file size while maintaining acceptable audio fidelity for listeners on mobile connections.",
					},
				},
			},
		},
	}

	app.RenderCommand(o, "build")
	output := strip(buf.String())

	expectedMax := 100
	maxLine := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
		if len([]rune(line)) > expectedMax {
			t.Errorf("line exceeds %d chars at width 100:\n  len=%d  %q", expectedMax, len([]rune(line)), line)
		}
	}
	t.Logf("width=100: max line = %d chars (expected max %d)", maxLine, expectedMax)
}

func testWideIndentWrap(t *testing.T) {
	o, buf := captureOptions(200)
	app := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "A short description.",
				Options: []Option{
					{Flags: "--very-long-flag-name-with-many-characters", Description: "This is a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines and test that the wrapping width respects the indent plus 80 rule."},
				},
			},
		},
	}
	app.RenderCommand(o, "cmd")
	raw := stripANSI(buf.String())

	maxLine := 0
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if l := len([]rune(line)); l > maxLine {
			maxLine = l
		}
	}
	if maxLine > 127 {
		t.Errorf("at width 200 with indent 47, max line = %d, expected <= 127", maxLine)
	}
	if maxLine < 100 {
		t.Errorf("at width 200 with indent 47, max line = %d, expected >= 100 (should use wide terminal)", maxLine)
	}
}

func testNarrowWrap(t *testing.T) {
	o2, buf2 := captureOptions(60)
	o2.Theme = &Theme{Separator: false}
	app2 := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "A short description that should not wrap at all at this width because it is short.",
			},
		},
	}
	app2.RenderCommand(o2, "cmd")
	raw2 := stripANSI(buf2.String())

	maxLine2 := 0
	for _, line := range strings.Split(strings.TrimSpace(raw2), "\n") {
		if l := len([]rune(line)); l > maxLine2 {
			maxLine2 = l
		}
	}
	if maxLine2 > 60 {
		t.Errorf("at width 60, max line = %d, expected <= 60", maxLine2)
	}
}

func testWideDescriptionCap(t *testing.T) {
	o3, buf3 := captureOptions(200)
	app3 := &App{
		Name: "test",
		Commands: []Command{
			{
				Name:        "cmd",
				Description: "This is a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines and test that the wrapping width respects the indent plus 80 rule.",
			},
		},
	}
	app3.RenderCommand(o3, "cmd")
	raw3 := stripANSI(buf3.String())

	maxLine3 := 0
	lines3 := strings.Split(strings.TrimSpace(raw3), "\n")
	for _, line := range lines3 {
		if l := len([]rune(line)); l > maxLine3 {
			maxLine3 = l
		}
	}
	if maxLine3 > 82 {
		t.Errorf("at width 200 with indent 2, max line = %d, expected <= 82", maxLine3)
	}
	if len(lines3) > 1 && maxLine3 < 60 {
		t.Errorf("at width 200 with indent 2, max line = %d, expected >= 60 (should wrap to ~80)", maxLine3)
	}
}

func TestReflowWrapWidthRespectsIndentPlus80(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	testWideIndentWrap(t)
	testNarrowWrap(t)
	testWideDescriptionCap(t)
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello world", "hello world"},
		{"sgr bold", "\x1b[1mhello\x1b[22m", "hello"},
		{"sgr color", "\x1b[35;1mhello\x1b[0m", "hello"},
		{"sgr short reset", "\x1b[mhello", "hello"},
		{"osc8 link", "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\", "text"},
		{"mixed", "\x1b[1m**bold**\x1b[22m \x1b]8;;https://x.com\x1b\\link\x1b]8;;\x1b\\", "**bold** link"},
		{"link in sentence", "see \x1b]8;;https://docs.example.com\x1b\\the docs\x1b]8;;\x1b\\ for more", "see the docs for more"},
	}
	for _, tc := range tests {
		got := stripANSI(tc.in)
		if got != tc.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVisualLenWithOSC8Links(t *testing.T) {
	link := "\x1b]8;;https://example.com\x1b\\docs\x1b]8;;\x1b\\"
	if n := visualLen(link); n != 4 {
		t.Errorf("visualLen of OSC8 link = %d, want 4 (visible text 'docs', raw=%q)", n, link)
	}

	link2 := "\x1b]8;;https://example.com/very/long/url\x1b\\alpha-two command\x1b]8;;\x1b\\"
	if n := visualLen(link2); n != 17 {
		t.Errorf("visualLen of long OSC8 link = %d, want 17 (visible text 'alpha-two command')", n)
	}
}

func TestUniformWrappingWithLinks(t *testing.T) {
	text := "This is a **bold command** with a [link to docs](https://docs.example.com/very/long/path) and some more text that should wrap nicely across multiple lines without any single line being too short because of the embedded link escape codes that would previously corrupt the visual length calculation."
	var buf bytes.Buffer
	reflow(&buf, color.New(color.FgWhite), 40, 2, "", inline(text))
	plain := stripANSI(buf.String())

	lines := strings.Split(strings.TrimSpace(plain), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 wrapped lines, got %d:\n%s", len(lines), plain)
	}

	maxGap := 0
	for i, line := range lines {
		l := len([]rune(line))
		gap := 40 - l
		if gap > maxGap {
			maxGap = gap
		}
		if i == 0 && l > 38 {
			continue
		}
		if i == len(lines)-1 {
			continue
		}
		if gap > 16 {
			t.Errorf("line %d: len=%d (gap=%d), too short at width 40:\n  %q", i, l, gap, line)
		}
	}
	t.Logf("uniform wrapping: max gap from width 40 = %d chars (mid-lines)", maxGap)
}
