package clihelp

// Re-entrancy and silence: that resolving does not render, and that the paths
// which merely resolve — completion, example validation, example colourisation —
// emit nothing of their own.
//
// This file used to be called resolution_purity_test.go and contained no purity
// test at all; the real one is TestResolutionMutatesNothing in
// leading_flags_test.go.

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// helpExampleApp is an app whose own example line is a help invocation. Rendering
// its help colorizes that line, which resolves it — the shape that made resolution
// re-enter the renderer.
func helpExampleApp(out io.Writer) *App {
	return &App{
		Name:     "app",
		Stdout:   out,
		Stderr:   out,
		Examples: []Example{{Line: "app help"}},
		Commands: []Command{{Name: "build", Description: "Build it", Run: func(*Context) error { return nil }}},
	}
}

// Rendering help for an app whose example is itself a help invocation must
// terminate. It is checked in a subprocess because the failure mode is a stack
// overflow, which kills the process rather than failing an assertion.
func TestHelpExampleDoesNotRecurse(t *testing.T) {
	if os.Getenv("CLIHELP_HELP_RECURSION_CHILD") == "1" {
		helpExampleApp(io.Discard).RenderGlobal(Options{Writer: io.Discard, Width: 80})
		return
	}

	cmd := sandboxedCommand(t, os.Args[0], "-test.run=TestHelpExampleDoesNotRecurse", "-test.timeout=60s")
	cmd.Env = append(cmd.Env, "CLIHELP_HELP_RECURSION_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		summary := string(output)
		if len(summary) > 400 {
			summary = summary[:400] + "..."
		}
		t.Fatalf("rendering help for an app whose example is a help invocation failed: %v\n%s", err, summary)
	}
}

// Resolving a help token must not write anything: completion consumers parse the
// stream as candidates.
func TestCompleteHelpEmitsOnlyCandidates(t *testing.T) {
	res := testExecute(helpExampleApp(nil), []string{"__complete", "help", ""})
	res.AssertNoError(t)
	if strings.Contains(res.Stdout, "Usage:") {
		t.Errorf("help page rendered into the completion stream:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "build") {
		t.Errorf("expected the command candidates, got:\n%s", res.Stdout)
	}
}

// Example validation must stay silent; it is called from Audit, and from tests.
func TestValidateExamplesEmitsNoHelp(t *testing.T) {
	var out bytes.Buffer
	app := helpExampleApp(&out)
	if errs := app.validateExamples(); len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %v", errs)
	}
	if out.Len() != 0 {
		t.Errorf("validation wrote %d bytes to App.Stdout:\n%s", out.Len(), out.String())
	}
}
