package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

// This package used to hold a 499-line second renderer that reproduced clihelp's
// output so the two could be compared byte for byte. It was retired.
//
// The reason is measured rather than argued. A mutation survey of the rendering
// path — render.go, format.go, inline.go — killed 27 of 38 mutations with the
// oracle in place: a 29% escape rate, indistinguishable from install.go and
// man.go at 29% and doc/ and tree/ at 32%, neither of which has an oracle. A
// second implementation of the same logic makes the same assumptions; what it
// catches is a typo, and what it costs is maintaining a renderer nobody ships.
// It also had to be pinned to one width, because making it agree at every width
// would have meant maintaining it as a real renderer.
//
// What replaces it is this: the same 43-page corpus — a realistic command tree
// with grouped commands, aliases, notes, examples, hyperlinks and hidden
// entries — checked against properties that must hold of any correct rendering,
// at several widths rather than one. A property does not have to be kept in step
// with the code it checks.

func paths(t *testing.T) (*clihelp.App, [][]string) {
	t.Helper()
	app := buildApp()
	p := detailedPaths(app)
	if len(p) < 40 {
		t.Fatalf("the corpus has shrunk to %d pages; these properties are only worth "+
			"as much as the tree they run over", len(p))
	}
	return app, p
}

func render(t *testing.T, app *clihelp.App, width int, path []string) string {
	t.Helper()
	var buf bytes.Buffer
	if len(path) == 0 {
		app.RenderGlobal(clihelp.Options{Writer: &buf, Width: width})
	} else if !app.RenderCommand(clihelp.Options{Writer: &buf, Width: width}, path...) {
		t.Fatalf("no page for %q", strings.Join(path, " "))
	}
	if strings.TrimSpace(clihelp.StripANSI(buf.String())) == "" {
		t.Fatalf("%q rendered nothing, so every property below holds vacuously",
			strings.Join(path, " "))
	}
	return buf.String()
}

// A line goes over its width only when a single unbreakable token makes it
// unavoidable — a path, a URL, a long flag spelling. If every word on an
// over-wide line would have fitted, the line could have been broken and was not.
//
// Example lines are exempt outright: they are emitted verbatim so that one can
// be copied and pasted, which examples.go documents and a test in the library
// pins. The same reasoning covers App.ConfigPath, which is printed as written so
// the path survives a copy — at a narrow width it overflows, and no arrangement
// of it would not.
func TestEveryPageFitsItsWidth(t *testing.T) {
	app, pages := paths(t)
	for _, width := range []int{40, 70, 100} {
		for _, path := range append([][]string{nil}, pages...) {
			inExamples := false
			for _, line := range strings.Split(render(t, app, width, path), "\n") {
				plain := clihelp.StripANSI(line)
				if trimmed := strings.TrimSpace(plain); strings.HasSuffix(trimmed, ":") {
					inExamples = trimmed == "Examples:"
				}
				if inExamples || clihelp.VisualWidth(line) <= width {
					continue
				}
				// A continuation line starts at its column, and the column is
				// decided by the layout rather than by wrapping — so the room a
				// wrapper actually had is the width less the indent.
				indent := clihelp.VisualWidth(plain) - clihelp.VisualWidth(strings.TrimLeft(plain, " "))
				widest := 0
				for _, word := range strings.Fields(plain) {
					if w := clihelp.VisualWidth(word); w > widest {
						widest = w
					}
				}
				if indent+widest <= width {
					t.Errorf("width %d, page %q: a line of %d columns whose widest word is "+
						"%d could have been broken and was not:\n%q",
						width, strings.Join(path, " "), clihelp.VisualWidth(line), widest, plain)
				}
			}
		}
	}
}

// Nothing the author wrote as markdown reaches the terminal as markdown, and no
// URL is ever spelled out: that is what OSC 8 exists for.
var (
	bareMarkdown = regexp.MustCompile(`\*\*|\[[^\]]*\]\([^)]*\)|` + "`")
	bareURL      = regexp.MustCompile(`https?://`)
)

func TestNoPageLeaksMarkupOrURLs(t *testing.T) {
	app, pages := paths(t)
	for _, path := range append([][]string{nil}, pages...) {
		plain := clihelp.StripANSI(render(t, app, 70, path))
		if m := bareMarkdown.FindString(plain); m != "" {
			t.Errorf("page %q shows raw markdown %q", strings.Join(path, " "), m)
		}
		if m := bareURL.FindString(plain); m != "" {
			t.Errorf("page %q shows a bare URL (%q); the label is what a reader sees",
				strings.Join(path, " "), m)
		}
	}
}

// Every escape this library opens on a line is closed on that line. A hyperlink
// left open at a line break bleeds into everything after it, including the
// shell prompt.
func TestEveryPageClosesWhatItOpens(t *testing.T) {
	app, pages := paths(t)
	for _, path := range append([][]string{nil}, pages...) {
		for i, line := range strings.Split(render(t, app, 70, path), "\n") {
			opens := strings.Count(line, "\x1b]8;;")
			if opens%2 != 0 {
				t.Errorf("page %q line %d leaves a hyperlink open: %q",
					strings.Join(path, " "), i, line)
			}
			if strings.Contains(line, "\x1b[") && !strings.Contains(line, "\x1b[0m") &&
				strings.Count(line, "\x1b[") == 1 {
				t.Errorf("page %q line %d opens a colour it never closes: %q",
					strings.Join(path, " "), i, line)
			}
		}
	}
}

// No line ends in whitespace. It is invisible until someone copies the output
// into a file that a linter reads.
func TestNoPageHasTrailingWhitespace(t *testing.T) {
	app, pages := paths(t)
	for _, path := range append([][]string{nil}, pages...) {
		for i, line := range strings.Split(strings.TrimRight(render(t, app, 70, path), "\n"), "\n") {
			plain := clihelp.StripANSI(line)
			if plain != strings.TrimRight(plain, " \t") {
				t.Errorf("page %q line %d ends in whitespace: %q",
					strings.Join(path, " "), i, plain)
			}
		}
	}
}

// Rendering is a pure function of the declaration: the same page twice is the
// same bytes. Map iteration over groups or options would show up here.
func TestRenderingIsDeterministic(t *testing.T) {
	app, pages := paths(t)
	for _, path := range append([][]string{nil}, pages...) {
		first := render(t, app, 70, path)
		for i := 0; i < 3; i++ {
			if again := render(t, app, 70, path); again != first {
				t.Fatalf("page %q differs between renders:\n--- first ---\n%s\n--- again ---\n%s",
					strings.Join(path, " "), first, again)
			}
		}
	}
}

// Each page is about its own command: it names it, and no two pages are the
// same. A renderer that ignored the path would pass everything above.
func TestEachPageIsAboutItsOwnCommand(t *testing.T) {
	app, pages := paths(t)
	seen := map[string]string{}
	for _, path := range pages {
		body := clihelp.StripANSI(render(t, app, 70, path))
		leaf := path[len(path)-1]
		if !strings.Contains(body, leaf) {
			t.Errorf("page %q never names %q", strings.Join(path, " "), leaf)
		}
		if prev, dup := seen[body]; dup {
			t.Errorf("pages %q and %q render identically", prev, strings.Join(path, " "))
		}
		seen[body] = strings.Join(path, " ")
	}
}

func TestMailCLIPagerEnabled(t *testing.T) {
	if app := buildApp(); !app.Pager {
		t.Error("expected buildApp().Pager to be true, got false")
	}
}
