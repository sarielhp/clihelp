package clihelp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// rowDescription builds a description of exactly varied lengths ending in the
// marker END, so the test can tell from the rendered page whether it wrapped.
func rowDescription(lead, words int) string {
	return strings.Repeat("x", lead) + " " + strings.Repeat("xxx ", words) + "END"
}

// descriptionLines counts the lines the entry whose trimmed text begins with
// label occupies, not counting a line that holds only the label (a name too wide
// for its column sits on a line of its own). Entries are not always separated by
// a blank line, so the entry ends at the first following line that is not
// indented deeper than the entry itself.
func descriptionLines(page, label string) int {
	lines := strings.Split(stripANSI(page), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, label) {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		n := 1
		for _, l := range lines[i+1:] {
			if strings.TrimSpace(l) == "" || len(l)-len(strings.TrimLeft(l, " ")) <= indent {
				break
			}
			n++
		}
		if trimmed == label {
			n--
		}
		return n
	}
	return -1
}

// TestAuditAgreesWithTheRenderer is the guard on audit_layout.go: the audit
// restates the renderer's one-row decision, and this fails if the two drift.
// For every description length around the boundary, Audit passes exactly when
// the rendered page draws the entry on one row.
func TestAuditAgreesWithTheRenderer(t *testing.T) {
	cases := []struct {
		name  string
		label string
		build func(desc string) *App
		path  []string
	}{
		{"short command", "build", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "build", Description: d}}}
		}, nil},
		{"aliased command", "build (b, make)", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "build", Aliases: []string{"b", "make"}, Description: d}}}
		}, nil},
		{"name wider than the column", "a-very-long-command-name-indeed", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "a-very-long-command-name-indeed", Description: d}}}
		}, nil},
		{"shortcut", "sc", func(d string) *App {
			return &App{Name: "a", Shortcuts: []Command{{Name: "sc", Description: d}}}
		}, nil},
		{"global flag with default", "-x, --extra", func(d string) *App {
			return &App{Name: "a", GlobalFlags: []Option{{Flags: "-x, --extra <v>", Description: d, DefaultText: "dflt"}}}
		}, nil},
		{"required flag", "-x, --extra", func(d string) *App {
			return &App{Name: "a", GlobalFlags: []Option{{Flags: "-x, --extra <v>", Description: d, Required: true}}}
		}, nil},
		{"subcommand", "run", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Subcommands: []Command{{Name: "run", Description: d}}}}}
		}, []string{"job"}},
		{"command flag", "--out", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Options: []Option{{Flags: "--out <f>", Description: d}}}}}
		}, []string{"job"}},
		{"local flag beside a wide persistent flag", "--out", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs",
				PersistentOptions: []Option{{Flags: "--persistent-flag-x", Description: "short"}},
				Options:           []Option{{Flags: "--out <f>", Description: d}}}}}
		}, []string{"job"}},
		{"inherited flag beside a wide local flag", "-g", func(d string) *App {
			return &App{Name: "a",
				PersistentOptions: []Option{{Flags: "-g", Description: d}},
				Commands: []Command{{Name: "job", Description: "Jobs",
					Options: []Option{{Flags: "--local-flag-nm-xx", Description: "short"}}}}}
		}, []string{"job"}},
		{"grouped flags", "--grouped", func(d string) *App {
			return &App{Name: "a", GlobalFlags: []Option{
				{Flags: "--grouped", Description: d, Group: "One"},
				{Flags: "--other-wide-flag-x", Description: "short", Group: "Two"}}}
		}, nil},
		{"command parameter", "file", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Parameters: []Param{{Name: "file", Description: d}}}}}
		}, []string{"job"}},
		{"parameter beside a wide parameter name", "file", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Parameters: []Param{{Name: "file", Description: d}, {Name: "<a-wide-name-here>", Description: "short"}}}}}
		}, []string{"job"}},
		{"explicit subcommand entry", "sub", func(d string) *App {
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", SubcommandEntries: []Param{{Name: "sub", Description: d}}}}}
		}, []string{"job"}},
		{"entries replace the real subcommands", "sub", func(d string) *App {
			// The tree's own row is long, but the page draws the entries instead.
			return &App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs",
				SubcommandEntries: []Param{{Name: "sub", Description: d}},
				Subcommands:       []Command{{Name: "sub", Description: strings.Repeat("word ", 40)}}}}}
		}, []string{"job"}},
		{"flag beside the built-in examples flag", "--gx", func(d string) *App {
			return &App{Name: "a", EnableExamplesFlag: true, GlobalFlags: []Option{{Flags: "--gx", Description: d}},
				Commands: []Command{{Name: "job", Description: "Jobs", Examples: []Example{{Line: "a job"}}}}}
		}, nil},
		{"app option beside global flags", "--own", func(d string) *App {
			return &App{Name: "a",
				GlobalFlags: []Option{{Flags: "--global-flag-nm-x", Description: "short"}},
				Options:     []Option{{Flags: "--own", Description: d}}}
		}, nil},
	}
	for _, tc := range cases {
		for _, width := range []int{60, 80, 100} {
			for lead := 1; lead <= 4; lead++ {
				for words := 0; words <= 22; words++ {
					desc := rowDescription(lead, words)
					t.Run(fmt.Sprintf("%s/w%d/%d", tc.name, width, len(desc)), func(t *testing.T) {
						app := tc.build(desc)
						var buf bytes.Buffer
						o := Options{Writer: &buf, Width: width}
						if tc.path != nil {
							if !app.RenderCommand(o, tc.path...) {
								t.Fatal("RenderCommand did not find the command")
							}
						} else {
							app.RenderGlobal(o)
						}
						lines := descriptionLines(buf.String(), tc.label)
						if lines < 0 {
							t.Fatalf("label %q not found in:\n%s", tc.label, buf.String())
						}
						oneRow := lines == 1
						err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: width})
						if oneRow != (err == nil) {
							t.Errorf("description of %d columns: rendered on %d line(s), Audit = %v\n%s",
								len(desc), lines, err, buf.String())
						}
					})
				}
			}
		}
	}
}

// The inherited flags are listed on every command page, so a single over-long
// global description must be reported once and not once per command.
func TestAuditReportsAnInheritedFlagOnce(t *testing.T) {
	app := &App{Name: "a",
		GlobalFlags: []Option{{Flags: "--g", Description: strings.Repeat("word ", 30)}},
		Commands: []Command{
			{Name: "one", Description: "One"},
			{Name: "two", Description: "Two"},
			{Name: "three", Description: "Three"},
		}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil, want an error")
	}
	// Once for the root page and once for the command pages, which share one column.
	if n := strings.Count(err.Error(), `description of "--g"`); n != 2 {
		t.Errorf("--g reported %d times, want 2:\n%v", n, err)
	}
}
