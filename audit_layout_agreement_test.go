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

// descriptionLines counts the lines the entry starting at the line whose
// trimmed text begins with label occupies, not counting a line that holds only
// the label (a name too wide for its column sits on a line of its own).
func descriptionLines(page, label string) int {
	lines := strings.Split(stripANSI(page), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, label) {
			continue
		}
		n := 0
		for _, l := range lines[i:] {
			if strings.TrimSpace(l) == "" {
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
	}
	for _, tc := range cases {
		for lead := 1; lead <= 4; lead++ {
			for words := 0; words <= 22; words++ {
				desc := rowDescription(lead, words)
				t.Run(fmt.Sprintf("%s/%d", tc.name, len(desc)), func(t *testing.T) {
					app := tc.build(desc)
					var buf bytes.Buffer
					o := Options{Writer: &buf, Width: 80}
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
					err := Audit(app, AuditOptions{SkipExampleValidation: true})
					if oneRow != (err == nil) {
						t.Errorf("description of %d columns: rendered on %d line(s), Audit = %v\n%s",
							len(desc), lines, err, buf.String())
					}
				})
			}
		}
	}
}
