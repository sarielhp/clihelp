package clihelptest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/clihelptest"
)

func app(run func(*clihelp.Context) error) *clihelp.App {
	return &clihelp.App{
		Name: "harness",
		Commands: []clihelp.Command{
			{Name: "build", Description: "Build it.", Run: run},
		},
	}
}

// What this package is for is capturing the three things a command produces, so
// that is what is asserted — including that they are captured rather than
// reaching the real streams, which is the whole reason a consumer reaches for it.
func TestExecuteCapturesOutputAndError(t *testing.T) {
	a := app(func(ctx *clihelp.Context) error {
		_, _ = ctx.Stdout.Write([]byte("built\n"))
		_, _ = ctx.Stderr.Write([]byte("a warning\n"))
		return errors.New("it failed")
	})

	res := clihelptest.Execute(a, []string{"build"})
	if !strings.Contains(res.Stdout, "built") {
		t.Errorf("stdout was not captured: %q", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "a warning") {
		t.Errorf("stderr was not captured: %q", res.Stderr)
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "it failed") {
		t.Errorf("the handler's error was not returned: %v", res.Error)
	}
}

func TestExecuteWithStdinSuppliesInput(t *testing.T) {
	var got string
	a := app(func(ctx *clihelp.Context) error {
		buf := make([]byte, 16)
		n, _ := ctx.App.Stdin.Read(buf)
		got = string(buf[:n])
		return nil
	})

	res := clihelptest.ExecuteWithStdin(a, []string{"build"}, strings.NewReader("typed\n"))
	res.AssertNoError(t)
	if !strings.HasPrefix(got, "typed") {
		t.Errorf("the handler read %q, want the supplied input", got)
	}
}

// The assertions pass when they should. That they fail when they should cannot
// be exercised here — the helpers take a *testing.T, whose failure methods are
// concrete and cannot be observed from a test — so each is called on a result it
// must accept, and a wrong result would fail this test through them.
func TestAssertionsAcceptAMatchingResult(t *testing.T) {
	a := app(func(ctx *clihelp.Context) error {
		_, _ = ctx.Stdout.Write([]byte("out marker\n"))
		_, _ = ctx.Stderr.Write([]byte("err marker\n"))
		return nil
	})
	res := clihelptest.Execute(a, []string{"build"})
	res.AssertNoError(t)
	res.AssertStdoutContains(t, "out marker")
	res.AssertStderrContains(t, "err marker")

	failing := clihelptest.Execute(app(func(*clihelp.Context) error {
		return errors.New("boom")
	}), []string{"build"})
	failing.AssertErrorContains(t, "boom")
}
