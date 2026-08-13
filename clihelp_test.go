package clihelp

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestStripANSI(t *testing.T) {
	input := color.RedString("Colored") + " Text"
	expected := "Colored Text"
	got := stripANSI(input)
	if got != expected {
		t.Errorf("stripANSI(%q) = %q; want %q", input, got, expected)
	}
}

func TestVisualLen(t *testing.T) {
	input := color.GreenString("Green")
	expected := 5
	got := visualLen(input)
	if got != expected {
		t.Errorf("visualLen(%q) = %d; want %d", input, got, expected)
	}
}

func TestWrapText(t *testing.T) {
	text := "This is a long sentence that should be wrapped across multiple lines properly."
	wrapped := wrapText(text, 30)
	lines := strings.Split(wrapped, "\n")
	if len(lines) < 2 {
		t.Errorf("wrapText did not wrap text into multiple lines: %q", wrapped)
	}
	for _, l := range lines {
		if len([]rune(l)) > 30 {
			t.Errorf("wrapped line exceeded avail width 30: %q", l)
		}
	}
}

func TestIndentLines(t *testing.T) {
	input := "line1\nline2"
	expected := "  line1\n  line2"
	got := indentLines(input, 2)
	if got != expected {
		t.Errorf("indentLines(%q) = %q; want %q", input, got, expected)
	}
}

func TestPrintGlobalUsage(t *testing.T) {
	app := &App{
		Name:        "testapp",
		Description: "A test application",
		GlobalNote:  "Run testapp <command> for details.",
		Commands: []Command{
			{
				Name:        "foo",
				Description: "Do foo",
			},
		},
	}

	out := captureOutput(func() {
		PrintGlobalUsage(app)
	})

	if !strings.Contains(out, "USAGE") {
		t.Errorf("PrintGlobalUsage output missing USAGE header: %s", out)
	}
	if !strings.Contains(out, "testapp [command] [options]") {
		t.Errorf("PrintGlobalUsage output missing usage line: %s", out)
	}
	if !strings.Contains(out, "COMMANDS") {
		t.Errorf("PrintGlobalUsage output missing COMMANDS section: %s", out)
	}
	if !strings.Contains(out, "foo") {
		t.Errorf("PrintGlobalUsage output missing command 'foo': %s", out)
	}
	if !strings.Contains(out, "Run testapp <command> for details.") {
		t.Errorf("PrintGlobalUsage output missing global note: %s", out)
	}
}

func TestPrintCommandUsage(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:        "build",
				Description: "Build binary",
				UsageLine:   "testapp build [flags]",
				Options: []Option{
					{Flags: "-o PATH", Description: "Output path"},
				},
				Examples: []Example{
					{Line: "testapp build -o bin"},
				},
			},
		},
	}

	// Test non-existent command
	if PrintCommandUsage(app, "invalid") {
		t.Errorf("PrintCommandUsage should return false for invalid command")
	}

	// Test valid command
	var found bool
	out := captureOutput(func() {
		found = PrintCommandUsage(app, "build")
	})

	if !found {
		t.Errorf("PrintCommandUsage returned false for valid command 'build'")
	}
	if !strings.Contains(out, "build") {
		t.Errorf("PrintCommandUsage missing command name in output: %s", out)
	}
	if !strings.Contains(out, "OPTIONS") {
		t.Errorf("PrintCommandUsage missing OPTIONS section: %s", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("PrintCommandUsage missing EXAMPLES section: %s", out)
	}
}

func TestPrintNestedCommandUsage(t *testing.T) {
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name:        "config",
				Description: "Configuration settings",
				Subcommands: []Command{
					{
						Name:        "set",
						Description: "Set a config option",
						Subcommands: []Command{
							{
								Name:        "location",
								Description: "Set location attribute",
								UsageLine:   "testapp config set location [options] <value>",
								Examples: []Example{
									{Line: "testapp config set location 5"},
								},
							},
						},
					},
				},
			},
		},
	}

	var found bool
	out := captureOutput(func() {
		found = PrintCommandUsage(app, "config", "set", "location")
	})

	if !found {
		t.Errorf("PrintCommandUsage failed for path [config set location]")
	}
	if !strings.Contains(out, "testapp config set location") {
		t.Errorf("PrintCommandUsage output missing full path header: %s", out)
	}
	if !strings.Contains(out, "testapp config set location 5") {
		t.Errorf("PrintCommandUsage output missing example: %s", out)
	}
}
