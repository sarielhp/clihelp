// Package clihelptest is the test harness for applications built on clihelp.
//
// It lives in its own package because clihelp itself must not import "testing".
// While these helpers sat in the library, every binary that imported clihelp
// linked the test framework and the standard "flag" package — the one this
// library deliberately avoids in favour of pflag. Measured before the move: 100
// dependency packages against 95, and about 19 KB on a small example binary.
//
// Import it only from your own tests:
//
//	res := clihelptest.Execute(app, []string{"build", "--fast"})
//	res.AssertStdoutContains(t, "Building")
package clihelptest

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

// Result holds the outcome of one Execute call.
type Result struct {
	Stdout string
	Stderr string
	Error  error
}

// AssertNoError asserts that the command executed successfully without error.
func (tr *Result) AssertNoError(t *testing.T) {
	t.Helper()
	if tr.Error != nil {
		t.Fatalf("expected no error, got: %v", tr.Error)
	}
}

// AssertErrorContains asserts that an error occurred and contains the substring.
func (tr *Result) AssertErrorContains(t *testing.T, substring string) {
	t.Helper()
	if tr.Error == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(tr.Error.Error(), substring) {
		t.Fatalf("expected error containing %q, got: %v", substring, tr.Error)
	}
}

// AssertStdoutContains asserts that stdout contains the substring.
func (tr *Result) AssertStdoutContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stdout, substring) {
		t.Fatalf("expected stdout containing %q, got:\n%s", substring, tr.Stdout)
	}
}

// AssertStderrContains asserts that stderr contains the substring.
func (tr *Result) AssertStderrContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stderr, substring) {
		t.Fatalf("expected stderr containing %q, got:\n%s", substring, tr.Stderr)
	}
}

// Execute runs the app with mock buffers and redirected stdout/stderr.
func Execute(app *clihelp.App, args []string) *Result {
	return ExecuteWithStdin(app, args, nil)
}

// ExecuteWithStdin runs the app redirecting stdout, stderr, and stdin.
func ExecuteWithStdin(app *clihelp.App, args []string, stdin io.Reader) *Result {
	var stdout, stderr bytes.Buffer
	origStdin := app.Stdin
	origStdout := app.Stdout
	origStderr := app.Stderr

	app.Stdin = stdin
	app.Stdout = &stdout
	app.Stderr = &stderr
	defer func() {
		app.Stdin = origStdin
		app.Stdout = origStdout
		app.Stderr = origStderr
	}()

	err := app.Execute(args)
	return &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Error:  err,
	}
}
