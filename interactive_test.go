package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestInteractiveFallbackRejectsNonterminalInput(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "answers")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString("json\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		in   io.Reader
	}{
		{"buffer", strings.NewReader("json\n")},
		{"file", file},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var format string
			var out, report bytes.Buffer
			ran := false
			app := &App{
				Name: "test", InteractiveFallback: true,
				Stdin: tt.in, Stdout: &out, Stderr: &report,
				Options: []Option{Required(String(&format, "--format FORMAT", "", "Output format"))},
				Run:     func(*Context) error { ran = true; return nil },
			}
			if err := app.Execute(nil); !IsUsageError(err) {
				t.Fatalf("expected usage error, got %v", err)
			}
			if ran || format != "" || report.Len() != 0 {
				t.Fatalf("nonterminal input prompted or ran: ran=%v format=%q stderr=%q", ran, format, report.String())
			}
		})
	}
}

// The child runs with real terminal input and redirected output, proving that
// interactivity follows stdin rather than the output writer.
func TestInteractiveFallbackTerminal(t *testing.T) {
	if os.Getenv("CLIHELP_TEST_TERMINAL_INPUT") == "1" {
		var format string
		var report bytes.Buffer
		app := &App{
			Name: "test", InteractiveFallback: true, Stderr: &report,
			Options: []Option{Required(String(&format, "--format FORMAT", "", "Output format"))},
			Run:     func(ctx *Context) error { _, err := fmt.Fprintln(ctx.Stdout, "exported", format); return err },
		}
		if err := app.Execute(nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(report.String(), "Next time") {
			t.Fatalf("missing equivalent-command tip: %q", report.String())
		}
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("uses util-linux script to supply a terminal")
	}
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script is unavailable")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script, "-qec", escapeShellArg(executable)+" -test.run '^TestInteractiveFallbackTerminal$'", "/dev/null")
	cmd.Env = append(os.Environ(), "CLIHELP_TEST_TERMINAL_INPUT=1")
	cmd.Stdin = strings.NewReader("json\n")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "exported json") {
		t.Fatalf("terminal prompt: %v\n%s", err, out)
	}
}
