package clihelp

import "fmt"

// NoArgs returns an error if any positional arguments are provided.
func NoArgs(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown arguments: %v", args)
	}
	return nil
}

// ExactArgs returns an ArgsValidator that ensures exactly n arguments are provided.
func ExactArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// MinimumNArgs returns an ArgsValidator that ensures at least n arguments are provided.
func MinimumNArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) < n {
			return fmt.Errorf("requires at least %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// MaximumNArgs returns an ArgsValidator that ensures at most n arguments are provided.
func MaximumNArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) > n {
			return fmt.Errorf("accepts at most %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// RangeArgs returns an ArgsValidator that ensures between min and max arguments are provided.
func RangeArgs(min, max int) ArgsValidator {
	return func(args []string) error {
		if len(args) < min || len(args) > max {
			return fmt.Errorf("accepts between %d and %d arg(s), received %d", min, max, len(args))
		}
		return nil
	}
}
