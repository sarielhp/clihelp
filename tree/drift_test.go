package tree

import (
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

// This package once carried its own copies of clihelp's text helpers, and both
// drifted. The tests below do not check what the helpers return — clihelp's own
// tests do that — they check that this package still asks clihelp rather than
// answering for itself, which is the property that broke.

// visualLen used a third-party stripper that does not understand OSC sequences.
// clihelp emits OSC 8 hyperlinks, so a linked word measured 22 columns wide
// instead of 4 and every column beside it was placed wrong.
func TestVisualLenAgreesWithClihelp(t *testing.T) {
	for _, s := range []string{
		"plain",
		"\x1b[31mred\x1b[0m",
		"\x1b]8;;https://example.com\x07link\x1b]8;;\x07",
		"\x1b]0;a window title\x07after",
		"日本語",
	} {
		if got, want := visualLen(s), clihelp.VisualWidth(s); got != want {
			t.Errorf("visualLen(%q) = %d, clihelp says %d", s, got, want)
		}
	}
}

// The copy checked for ". " before truncating at a line break, so a description
// whose first full stop fell on a later line came back with newlines still in
// it — and reflowTree assumes one line.
func TestFirstSentenceAgreesWithClihelp(t *testing.T) {
	for _, s := range []string{
		"One sentence.",
		"Line one\nLine two. Line three",
		"Summary\n\nDetail. More detail.",
		"No full stop at all",
		"",
	} {
		got := firstSentence(s)
		if want := clihelp.FirstSentence(s); got != want {
			t.Errorf("firstSentence(%q) = %q, clihelp says %q", s, got, want)
		}
		if strings.ContainsAny(got, "\n") {
			t.Errorf("firstSentence(%q) returned a newline: %q — reflowTree assumes one line", s, got)
		}
	}
}
