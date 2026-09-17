package clihelp

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func explainApp() *App {
	long := strings.Repeat("This command does a great many things, each of which needs describing at length. ", 12)
	return &App{
		Name:           "prog",
		AbbrevCommands: true,
		Commands: []Command{
			{
				Name:            "xylophone",
				Description:     "Play it",
				LongDescription: long,
				Subcommands: []Command{
					{
						Name:            "yesterday",
						Description:     "Yesterday's run",
						LongDescription: long,
						Parameters:      []Param{{Name: "<file>", Description: "The file to use"}},
						Examples: []Example{
							{Line: "prog xylophone yesterday out.txt", Description: "Write to out.txt"},
						},
						Run: func(*Context) error { return nil },
					},
					{Name: "yonder", Description: "Over there", Run: func(*Context) error { return nil }},
				},
			},
			{Name: "other", Description: "Something else", Run: func(*Context) error { return nil }},
		},
	}
}

func TestExpandCommandLine(t *testing.T) {
	app := explainApp()
	for _, tt := range []struct {
		name string
		line string
		want string
	}{
		{"one abbreviation", "prog x", "prog xylophone"},
		{"two abbreviations", "prog x ye", "prog xylophone yesterday"},
		{"already full", "prog xylophone yesterday", "prog xylophone yesterday"},
		{"trailing argument is untouched", "prog x ye out.txt", "prog xylophone yesterday out.txt"},
		{"quoting and spacing are preserved", `prog   x   ye   'a b'`, `prog   xylophone   yesterday   'a b'`},
		{"the rest of the line is untouched", "prog x ye | grep x", "prog xylophone yesterday | grep x"},
		{"redirection is untouched", "prog x ye > out.txt", "prog xylophone yesterday > out.txt"},
		{"ambiguous prefix stops expansion", "prog x y", "prog xylophone y"},
		{"unknown word stops expansion", "prog nope ye", "prog nope ye"},
		{"path-qualified program name", "./prog x ye", "./prog xylophone yesterday"},
		{"bare program name", "prog", "prog"},
		{"empty line", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := app.expandCommandLine(tt.line)
			if got != tt.want {
				t.Errorf("expandCommandLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestExpandCommandLineReportsThePath(t *testing.T) {
	app := explainApp()
	_, path := app.expandCommandLine("prog x ye out.txt")
	if strings.Join(path, " ") != "xylophone yesterday" {
		t.Errorf("path = %v, want [xylophone yesterday]", path)
	}
}

// The explanation has to leave the command it explains on screen, so it may use
// at most two thirds of the terminal's height — the requirement this test exists
// to hold.
func TestExplainFitsTwoThirdsOfTheScreen(t *testing.T) {
	app := explainApp()
	for _, rows := range []int{9, 12, 24, 40, 60, 100} {
		t.Run(strings.Repeat("x", 0)+itoa(rows), func(t *testing.T) {
			var buf bytes.Buffer
			budget := explainBudget(rows)
			app.Explain(&buf, "prog x ye", 80, budget)

			lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
			if len(lines) > budget {
				t.Errorf("explanation is %d lines, over its budget of %d", len(lines), budget)
			}
			if twoThirds := rows * 2 / 3; budget > 3 && len(lines) > twoThirds {
				t.Errorf("explanation is %d lines, over two thirds of a %d-line screen (%d)", len(lines), rows, twoThirds)
			}
			if lines[0] != "prog xylophone yesterday" {
				t.Errorf("first line = %q, want the expanded command line", lines[0])
			}
		})
	}
}

func TestExplainSaysWhereTheRestIs(t *testing.T) {
	app := explainApp()
	var buf bytes.Buffer
	app.Explain(&buf, "prog x ye", 80, explainBudget(12))
	out := buf.String()
	if !strings.Contains(out, "more lines") {
		t.Fatalf("a truncated explanation should say how much was cut:\n%s", out)
	}
	if !strings.Contains(out, "prog help xylophone yesterday") {
		t.Errorf("the hint should name the command that prints the rest:\n%s", out)
	}
}

func TestExplainShortHelpIsNotTruncated(t *testing.T) {
	app := &App{
		Name:     "prog",
		Commands: []Command{{Name: "run", Description: "Run it", Run: func(*Context) error { return nil }}},
	}
	var buf bytes.Buffer
	app.Explain(&buf, "prog run", 80, explainBudget(40))
	out := buf.String()
	if strings.Contains(out, "more lines") {
		t.Errorf("short help should not be truncated:\n%s", out)
	}
	if !strings.HasPrefix(out, "prog run\n") {
		t.Errorf("first line = %q, want the expanded command line", out)
	}
}

func TestExplainProtocolUsesTheTerminalSizeItIsGiven(t *testing.T) {
	t.Setenv(envTermLines, "12")
	t.Setenv(envTermColumns, "60")

	var out bytes.Buffer
	app := explainApp()
	app.Stdout = &out
	if err := app.ExecuteContext(context.Background(), []string{"__explain", "prog x ye"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) > 8 { // two thirds of 12
		t.Errorf("__explain produced %d lines on a 12-line screen:\n%s", len(lines), out.String())
	}
	for _, l := range lines {
		if visualLen(l) > 60 {
			t.Errorf("line wider than the 60 columns given: %q", l)
		}
	}
}

func TestKeyBindingSnippets(t *testing.T) {
	app := &App{Name: "pod-ctl"}
	for _, tt := range []struct {
		shell string
		want  []string
	}{
		{"bash", []string{`bind -x '"\eh": _pod_ctl_clihelp_explain'`, "$READLINE_LINE", "__explain", "CLIHELP_TERM_LINES"}},
		{"zsh", []string{"bindkey '^[h'", "zle run-help", "$BUFFER", "__explain"}},
		{"fish", []string{`bind \eh`, "commandline -r", "__explain", "repaint"}},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			var b strings.Builder
			if err := GenKeyBindings(app, tt.shell, &b); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(b.String(), want) {
					t.Errorf("%s snippet is missing %q:\n%s", tt.shell, want, b.String())
				}
			}
			if !strings.Contains(b.String(), "pod-ctl __explain") {
				t.Errorf("%s snippet does not call the app:\n%s", tt.shell, b.String())
			}
		})
	}

	if err := GenKeyBindings(app, "ksh", &strings.Builder{}); err == nil {
		t.Errorf("GenKeyBindings accepted a shell it has no snippet for")
	}
	if err := GenKeyBindings(nil, "bash", &strings.Builder{}); err == nil {
		t.Errorf("GenKeyBindings accepted a nil app")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
