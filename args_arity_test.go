package clihelp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The arity is the whole point of the interface: a validator that cannot say how
// many arguments it wants forces the caller to discover it by calling, which is
// what a real application ended up doing four times per invocation.
func TestBuiltinValidatorsReportTheirArity(t *testing.T) {
	for _, tt := range []struct {
		name     string
		v        ArgsValidator
		min, max int
		bare     bool // does it accept no arguments at all?
	}{
		{"NoArgs", NoArgs, 0, 0, true},
		{"ExactArgs(0)", ExactArgs(0), 0, 0, true},
		{"ExactArgs(1)", ExactArgs(1), 1, 1, false},
		{"ExactArgs(3)", ExactArgs(3), 3, 3, false},
		{"MinimumNArgs(0)", MinimumNArgs(0), 0, -1, true},
		{"MinimumNArgs(2)", MinimumNArgs(2), 2, -1, false},
		{"MaximumNArgs(1)", MaximumNArgs(1), 0, 1, true},
		{"RangeArgs(0,1)", RangeArgs(0, 1), 0, 1, true},
		{"RangeArgs(1,2)", RangeArgs(1, 2), 1, 2, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			min, max := tt.v.Arity()
			if min != tt.min || max != tt.max {
				t.Errorf("Arity() = (%d, %d), want (%d, %d)", min, max, tt.min, tt.max)
			}
			if got := acceptsNoArgs(tt.v); got != tt.bare {
				t.Errorf("acceptsNoArgs = %v, want %v", got, tt.bare)
			}
			// The arity and the validation must agree, or the library would be
			// reading one and enforcing the other.
			if err := tt.v.ValidateArgs(nil); (err == nil) != tt.bare {
				t.Errorf("ValidateArgs(nil) = %v, but the arity says bare=%v", err, tt.bare)
			}
			if tt.max >= 0 {
				over := make([]string, tt.max+1)
				if tt.v.ValidateArgs(over) == nil {
					t.Errorf("%d arguments accepted although max is %d", len(over), tt.max)
				}
			}
			if tt.min > 0 {
				under := make([]string, tt.min-1)
				if tt.v.ValidateArgs(under) == nil {
					t.Errorf("%d arguments accepted although min is %d", len(under), tt.min)
				}
			}
		})
	}
}

// A custom rule the built-ins cannot express says so, and is then asked rather
// than assumed.
func TestArgsFuncReportsUnknownArityAndIsStillAsked(t *testing.T) {
	calls := 0
	v := ArgsFunc(func(args []string) error {
		calls++
		if len(args) == 0 {
			return errors.New("needs something")
		}
		return nil
	})
	if min, max := v.Arity(); min >= 0 || max >= 0 {
		t.Errorf("Arity() = (%d, %d), want both negative for a custom function", min, max)
	}
	if acceptsNoArgs(v) {
		t.Error("a function that rejects no arguments was reported as accepting them")
	}
	if calls != 1 {
		t.Errorf("the custom function was called %d times to answer one question", calls)
	}
	if err := v.ValidateArgs([]string{"x"}); err != nil {
		t.Errorf("a valid call was rejected: %v", err)
	}
}

// The built-ins answer from their declared arity and are never used as an oracle.
func TestABuiltinValidatorIsNotCalledToLearnItsArity(t *testing.T) {
	probed := 0
	v := arityProbe{inner: ExactArgs(1), calls: &probed}
	if acceptsNoArgs(v) {
		t.Error("ExactArgs(1) was reported as accepting no arguments")
	}
	if probed != 0 {
		t.Errorf("the validator was called %d times to answer a question its arity answers", probed)
	}
}

type arityProbe struct {
	inner ArgsValidator
	calls *int
}

func (p arityProbe) ValidateArgs(args []string) error { *p.calls++; return p.inner.ValidateArgs(args) }
func (p arityProbe) Arity() (int, int)                { return p.inner.Arity() }

// F1: a command invoked with nothing, which needs something, answers with its
// own help — and still fails, so that a script is not told it ran.
func TestMissingArgumentsShowTheCommandsHelp(t *testing.T) {
	newApp := func(out, errOut *bytes.Buffer) *App {
		return &App{
			Name: "demo", Stdout: out, Stderr: errOut,
			Commands: []Command{
				{
					Name: "scan", Description: "Scan a label.", UsageLine: "demo scan <prefix>",
					Parameters: []Param{{Name: "<prefix>", Description: "Label prefix to scan."}},
					Args:       ExactArgs(1),
					Run:        func(*Context) error { return nil },
				},
				{
					Name: "status", Description: "Show status.", Args: MaximumNArgs(1),
					Run: func(*Context) error { return nil },
				},
			},
		}
	}

	t.Run("nothing at all gets the help", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := newApp(&out, &errOut).Execute([]string{"scan"})
		if err == nil {
			t.Fatal("the command printed help and reported success; a script would think it ran")
		}
		body := StripANSI(errOut.String())
		if !strings.Contains(body, "<prefix>") || !strings.Contains(body, "Label prefix to scan.") {
			t.Errorf("the help did not name the missing parameter:\n%s", body)
		}
		if out.Len() != 0 {
			t.Errorf("help printed because of an error went to stdout: %q", out.String())
		}
	})

	t.Run("a wrong count keeps its precise message", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := newApp(&out, &errOut).Execute([]string{"scan", "a", "b"})
		if err == nil {
			t.Fatal("two arguments were accepted by ExactArgs(1)")
		}
		if strings.Contains(StripANSI(errOut.String()), "Label prefix to scan.") {
			t.Errorf("a page of help was printed for a user who did supply arguments:\n%s", errOut.String())
		}
	})

	t.Run("a command that runs bare is untouched", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if err := newApp(&out, &errOut).Execute([]string{"status"}); err != nil {
			t.Fatalf("a command accepting zero arguments failed: %v", err)
		}
		if errOut.Len() != 0 {
			t.Errorf("help was printed for a command that ran: %q", errOut.String())
		}
	})
}
