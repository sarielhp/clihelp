package tree

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/sarielhp/clihelp"
)

func renderPlain(t *testing.T, app *clihelp.App, width int) []string {
	t.Helper()
	had := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = had }()

	var buf bytes.Buffer
	Render(&buf, app, Options{Writer: &buf, Width: width})
	return strings.Split(strings.TrimRight(clihelp.StripANSI(buf.String()), "\n"), "\n")
}

// lineFor returns the rendered line whose name column is display, and the
// box-drawing run that precedes it. A child is drawn under its full path, so
// display is "alpha alpha-one" rather than "alpha-one".
func lineFor(lines []string, display string) (line, prefix string, ok bool) {
	for _, l := range lines {
		plain := strings.TrimRight(l, " ")
		head := plain
		if i := strings.Index(plain, display); i >= 0 {
			head = plain[:i]
		} else {
			continue
		}
		if strings.Trim(head, "│├└─ ") != "" {
			continue // the text appeared inside a description
		}
		return plain, head, true
	}
	return "", "", false
}

// The continuation prefixes are what make a tree a tree: a middle child carries
// a vertical bar down its sibling's column, a last child carries blanks, and a
// node with subcommands adds another level. All three decisions live in
// computeContBase and none of them was asserted — every mutation of it passed.
func TestTreeStructureIsDrawn(t *testing.T) {
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{
		{Name: "alpha", Description: "First.", Subcommands: []clihelp.Command{
			{Name: "alpha-one", Description: "A1."},
			{Name: "alpha-two", Description: "A2."},
		}},
		{Name: "beta", Description: "Second."},
		{Name: "omega", Description: "Last.", Subcommands: []clihelp.Command{
			{Name: "omega-one", Description: "O1."},
		}},
	}}
	lines := renderPlain(t, app, 100)

	for _, tt := range []struct {
		name       string
		wantPrefix string
		why        string
	}{
		{"alpha", "├── ", "a middle top-level command"},
		{"beta", "├── ", "another middle top-level command"},
		{"omega", "└── ", "the last top-level command"},
		{"alpha alpha-one", "│   ├── ", "a middle child of a middle parent carries its parent's bar"},
		{"alpha alpha-two", "│   └── ", "the last child of a middle parent still carries the bar"},
		{"omega omega-one", "    └── ", "the last child of the last parent carries blanks, not a bar"},
	} {
		line, prefix, ok := lineFor(lines, tt.name)
		if !ok {
			t.Errorf("%q is not in the tree:\n%s", tt.name, strings.Join(lines, "\n"))
			continue
		}
		if prefix != tt.wantPrefix {
			t.Errorf("%s: %q is drawn with prefix %q, want %q\n  line: %q",
				tt.why, tt.name, prefix, tt.wantPrefix, line)
		}
	}
}

// A hidden command is not in the tree, exactly as it is not in the help.
func TestTreeSkipsHiddenCommands(t *testing.T) {
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{
		{Name: "visible", Description: "V."},
		{Name: "secret", Description: "S.", Hidden: true},
		{Name: "parent", Description: "P.", Subcommands: []clihelp.Command{
			{Name: "shown", Description: "S."},
			{Name: "concealed", Description: "C.", Hidden: true},
		}},
	}}
	out := strings.Join(renderPlain(t, app, 100), "\n")
	for _, name := range []string{"secret", "concealed"} {
		if strings.Contains(out, name) {
			t.Errorf("the tree shows the hidden command %q:\n%s", name, out)
		}
	}
	for _, name := range []string{"visible", "parent", "shown"} {
		if !strings.Contains(out, name) {
			t.Errorf("the tree lost the visible command %q:\n%s", name, out)
		}
	}
}

// A tree is a summary, so each node shows one sentence — the same rule the
// command list in the help follows.
func TestTreeShowsOneSentence(t *testing.T) {
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{
		{Name: "build", Description: "Build it. This second sentence belongs on the manual page."},
	}}
	out := strings.Join(renderPlain(t, app, 120), "\n")
	if !strings.Contains(out, "Build it.") {
		t.Errorf("the first sentence is missing:\n%s", out)
	}
	if strings.Contains(out, "second sentence") {
		t.Errorf("the tree shows more than the first sentence:\n%s", out)
	}
}

// computeContBase draws the column a wrapped description sits in, and it is the
// only place the tree decides whether a parent's vertical bar continues past it.
// Every mutation of its three branches passed the suite: the child prefixes are
// built elsewhere, so asserting those does not reach it.
func TestWrappedDescriptionsSitUnderTheRightColumn(t *testing.T) {
	long := "A description long enough that it must wrap onto a second line so the continuation column can be seen."
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{
		{Name: "alpha", Description: long, Subcommands: []clihelp.Command{
			{Name: "alpha-one", Description: long},
			{Name: "alpha-two", Description: long},
		}},
		{Name: "omega", Description: long, Subcommands: []clihelp.Command{
			{Name: "omega-one", Description: long},
		}},
	}}
	lines := renderPlain(t, app, 60)

	for _, tt := range []struct {
		display    string
		wantIndent string
		why        string
	}{
		{"alpha", "│   │   ", "a middle command with subcommands: its own bar, then its children's"},
		{"omega", "    │   ", "the last command with subcommands: blanks, then its children's bar"},
		{"alpha alpha-one", "│   │   ", "a middle child keeps its own bar for the sibling below it"},
		{"alpha alpha-two", "│       ", "the last child under a middle parent: the parent's bar, then blanks"},
		{"omega omega-one", "        ", "the last child of the last parent carries blanks all the way"},
	} {
		cont, ok := continuationOf(lines, tt.display)
		if !ok {
			t.Errorf("%q has no wrapped continuation line:\n%s", tt.display, strings.Join(lines, "\n"))
			continue
		}
		if !strings.HasPrefix(cont, tt.wantIndent) {
			t.Errorf("%s: the continuation under %q begins %q, want it to begin %q",
				tt.why, tt.display, firstRunes(cont, 12), tt.wantIndent)
		}
	}
}

// continuationOf returns the line after the one naming display — the wrapped
// remainder of its description.
func continuationOf(lines []string, display string) (string, bool) {
	for i, l := range lines {
		head := l
		if j := strings.Index(l, display); j >= 0 {
			head = l[:j]
		} else {
			continue
		}
		if strings.Trim(head, "│├└─ ") != "" {
			continue
		}
		if i+1 < len(lines) {
			return lines[i+1], true
		}
	}
	return "", false
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
