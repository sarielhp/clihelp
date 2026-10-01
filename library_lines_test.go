package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

// CompletionCommand and ManPageCommand are written before anyone knows the
// application's name or where the author will mount them, so their usage and
// example lines were relative to their own root: "completion zsh",
// "manpage > myapp.1". Every other command's lines start with the program name,
// and a user who copies one of these gets "unknown command". Mounted a level or
// two down, the page's own heading disagreed with the line beneath it.
func TestLibraryCommandsNameTheProgramAndTheirMountPoint(t *testing.T) {
	mounts := []struct {
		name    string
		parents []string
	}{
		{"at the root", nil},
		{"one level down", []string{"cfg"}},
		{"two levels down", []string{"cfg", "sub"}},
	}
	for _, m := range mounts {
		t.Run(m.name, func(t *testing.T) {
			lib := []Command{CompletionCommand(), ManPageCommand()}
			cmds := lib
			for i := len(m.parents) - 1; i >= 0; i-- {
				cmds = []Command{{Name: m.parents[i], Description: "Group", Subcommands: cmds}}
			}
			app := &App{Name: "tool", Commands: cmds}

			prefix := strings.TrimSpace("tool " + strings.Join(m.parents, " "))
			for _, root := range []string{"completion", "manpage"} {
				want := prefix + " " + root
				var paths [][]string
				base := append(append([]string{}, m.parents...), root)
				paths = append(paths, base)
				if root == "completion" {
					for _, sub := range []string{"zsh", "install", "keys", "wrap"} {
						paths = append(paths, append(append([]string{}, base...), sub))
					}
				}
				for _, path := range paths {
					var buf bytes.Buffer
					if !app.RenderCommand(Options{Writer: &buf, Width: 100, NoColor: true, Extended: true}, path...) {
						t.Fatalf("no page for %v", path)
					}
					page := buf.String()
					checkLibraryPage(t, page, path, want)
				}
			}

			var ex bytes.Buffer
			app.renderExamplesPage(Options{Writer: &ex, Width: 100, NoColor: true})
			for _, line := range strings.Split(ex.String(), "\n") {
				if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") && !strings.HasPrefix(strings.TrimSpace(line), "tool") {
					t.Errorf("-E lists an example line that does not start with the program name: %q\n%s", line, ex.String())
				}
			}
		})
	}
}

func checkLibraryPage(t *testing.T, page string, path []string, want string) {
	t.Helper()
	inExamples := false
	sawUsage := false
	for _, line := range strings.Split(page, "\n") {
		switch {
		case strings.HasPrefix(line, "Usage:"):
			sawUsage = true
			if got := strings.TrimSpace(strings.TrimPrefix(line, "Usage:")); !strings.HasPrefix(got, want) {
				t.Errorf("%v: usage %q does not start with %q", path, got, want)
			}
		case line == "Examples:":
			inExamples = true
		case line != "" && line[0] != ' ':
			inExamples = false
		case inExamples && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    "):
			if got := strings.TrimSpace(line); !strings.HasPrefix(got, want) {
				t.Errorf("%v: example line %q does not start with %q", path, got, want)
			}
		}
		if strings.Contains(line, "<app>") || strings.Contains(line, "myapp") {
			t.Errorf("%v: an unfilled placeholder survived on %q", path, line)
		}
	}
	if !sawUsage {
		t.Errorf("%v: page has no usage line", path)
	}
}
