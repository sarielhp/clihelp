package clihelp

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// The in-package equivalent of clihelptest, for the library's own tests.
//
// It cannot import clihelptest: that package imports clihelp, and a test file in
// package clihelp importing it would be a cycle. It is a test file rather than
// part of the library because this is exactly what used to make every consumer's
// binary link "testing" and the standard "flag" package.
//
// clihelptest is a thin wrapper over the same public API — App.Stdout,
// App.Stderr, App.Stdin and App.Execute — so there is no behaviour here that
// could drift from it.
type testResult struct {
	Stdout string
	Stderr string
	Error  error
}

func testExecute(app *App, args []string) *testResult {
	return testExecuteWithStdin(app, args, nil)
}

func testExecuteWithStdin(app *App, args []string, stdin io.Reader) *testResult {
	var outBuf, errBuf bytes.Buffer
	app.Stdout, app.Stderr = &outBuf, &errBuf
	if stdin != nil {
		app.Stdin = stdin
	}
	err := app.Execute(args)
	return &testResult{Stdout: outBuf.String(), Stderr: errBuf.String(), Error: err}
}

func (tr *testResult) AssertNoError(t *testing.T) {
	t.Helper()
	if tr.Error != nil {
		t.Fatalf("expected no error, got: %v", tr.Error)
	}
}

func (tr *testResult) AssertErrorContains(t *testing.T, substring string) {
	t.Helper()
	if tr.Error == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(tr.Error.Error(), substring) {
		t.Fatalf("expected error containing %q, got: %v", substring, tr.Error)
	}
}

func (tr *testResult) AssertStdoutContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stdout, substring) {
		t.Fatalf("expected stdout containing %q, got:\n%s", substring, tr.Stdout)
	}
}

func (tr *testResult) AssertStderrContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stderr, substring) {
		t.Fatalf("expected stderr containing %q, got:\n%s", substring, tr.Stderr)
	}
}
