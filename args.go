package clihelp

import "fmt"

// ArgsValidator decides whether a command's positional arguments are acceptable,
// and reports how many the command takes.
//
// It is an interface rather than a plain func because the arity was information
// the library threw away. ExactArgs(1) knows perfectly well that it wants one
// argument, but sealed inside a closure the only way to ask "does this command
// work with no arguments at all" was to call it with an empty slice and see —
// which is what a real application built on this library ended up doing, four
// times per invocation, using the author's own validator as an oracle.
//
// The alternative was a second field on Command saying the same thing in other
// words, and two statements of one fact drift: helpFlagNames and bindHelpFlags
// were exactly that, and the drift made Audit reject examples that ran.
type ArgsValidator interface {
	// ValidateArgs reports whether these positional arguments are acceptable.
	ValidateArgs(args []string) error
	// Arity reports how many positional arguments the command takes. A max below
	// zero means unbounded; a min below zero means the validator cannot say,
	// which is the answer for a custom function whose behaviour is discoverable
	// only by calling it.
	Arity() (min, max int)
}

// argRange is every built-in validator: a bound, and the message to give when
// the count falls outside it.
type argRange struct {
	min, max int
	fail     func(args []string) error
}

func (a argRange) ValidateArgs(args []string) error {
	n := len(args)
	if n < a.min || (a.max >= 0 && n > a.max) {
		return a.fail(args)
	}
	return nil
}

func (a argRange) Arity() (int, int) { return a.min, a.max }

// NoArgs rejects any positional argument.
//
// It is a value rather than a function so that it can report its arity like the
// others; "Args: clihelp.NoArgs" is unchanged at the call site.
var NoArgs ArgsValidator = argRange{
	min: 0, max: 0,
	fail: func(args []string) error { return fmt.Errorf("unknown arguments: %v", args) },
}

// ExactArgs requires exactly n positional arguments.
func ExactArgs(n int) ArgsValidator {
	return argRange{min: n, max: n, fail: func(args []string) error {
		return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
	}}
}

// MinimumNArgs requires at least n positional arguments.
func MinimumNArgs(n int) ArgsValidator {
	return argRange{min: n, max: -1, fail: func(args []string) error {
		return fmt.Errorf("requires at least %d arg(s), received %d", n, len(args))
	}}
}

// MaximumNArgs allows at most n positional arguments.
func MaximumNArgs(n int) ArgsValidator {
	return argRange{min: 0, max: n, fail: func(args []string) error {
		return fmt.Errorf("accepts at most %d arg(s), received %d", n, len(args))
	}}
}

// RangeArgs requires between minArgs and maxArgs positional arguments.
func RangeArgs(minArgs, maxArgs int) ArgsValidator {
	return argRange{min: minArgs, max: maxArgs, fail: func(args []string) error {
		return fmt.Errorf("accepts between %d and %d arg(s), received %d", minArgs, maxArgs, len(args))
	}}
}

// ArgsFunc adapts a plain function to ArgsValidator, for a rule the built-ins
// cannot express. Its arity is unknown, so the library falls back to asking it.
func ArgsFunc(fn func(args []string) error) ArgsValidator { return argFunc(fn) }

type argFunc func(args []string) error

func (f argFunc) ValidateArgs(args []string) error {
	if f == nil {
		return nil
	}
	return f(args)
}

func (argFunc) Arity() (int, int) { return -1, -1 }

// acceptsNoArgs reports whether cmd runs with no positional arguments at all.
//
// It reads the declared arity where there is one and only falls back to asking
// the validator where there is not, so an author's own function is called once,
// to validate, rather than repeatedly as a predicate.
func acceptsNoArgs(v ArgsValidator) bool {
	if v == nil {
		return true
	}
	if min, _ := v.Arity(); min >= 0 {
		return min == 0
	}
	return v.ValidateArgs(nil) == nil
}
