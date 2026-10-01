package text

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "plain", "plain"},
		{"csi", "\x1b[31mred\x1b[0m", "red"},
		{"osc8 link", "\x1b]8;;https://example.com\x07link\x1b]8;;\x07", "link"},
		{"osc title", "\x1b]0;a window title\x07after", "after"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripANSI(tt.in); got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "hello", 5},
		{"colour costs nothing", "\x1b[31mred\x1b[0m", 3},
		{"hyperlink costs nothing", "\x1b]8;;https://example.com\x07link\x1b]8;;\x07", 4},
		{"wide runes", "日本語", 6},
		{"tab advances to the stop", "a\tb", 9},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VisualWidth(tt.in); got != tt.want {
				t.Errorf("VisualWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestFirstSentence(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"one sentence", "One sentence.", "One sentence."},
		{"cuts at a full stop", "First. Second.", "First."},
		{"cuts at a line break", "Line one\nLine two. Three", "Line one"},
		{"cuts at a paragraph", "Summary\n\nDetail. More.", "Summary"},
		{"no full stop", "No full stop", "No full stop"},
		{"stop inside a code span", "Run `a. b` now. Then", "Run `a. b` now."},
		{"stop inside a link", "See [v1. 2](http://x/v1. 2/y) now. Then", "See [v1. 2](http://x/v1. 2/y) now."},
		{"empty", "  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FirstSentence(tt.in); got != tt.want {
				t.Errorf("FirstSentence(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
