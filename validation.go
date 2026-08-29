package clihelp

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// ValidateOptions chains multiple OptionsValidators into a single validator.
func ValidateOptions(validators ...OptionsValidator) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		for _, v := range validators {
			if v != nil {
				if err := v(fs); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// cleanFlagName removes leading dashes from a flag name.
func cleanFlagName(f string) string {
	return strings.TrimLeft(f, "-")
}

// isFlagSet checks if a flag was explicitly set on the command line.
func isFlagSet(fs *pflag.FlagSet, flagName string) bool {
	clean := cleanFlagName(flagName)
	// Check if condition contains value check, e.g. "--auth-method=token"
	if strings.Contains(clean, "=") {
		parts := strings.SplitN(clean, "=", 2)
		name := parts[0]
		val := parts[1]
		flg := fs.Lookup(name)
		if flg == nil {
			return false
		}
		// If the condition flag was set and its value matches, return true
		return fs.Changed(name) && flg.Value.String() == val
	}
	return fs.Changed(clean)
}

// MutuallyExclusive ensures at most one of the specified flags is set.
func MutuallyExclusive(flags ...string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		var setFlags []string
		for _, f := range flags {
			if isFlagSet(fs, f) {
				setFlags = append(setFlags, f)
			}
		}
		if len(setFlags) > 1 {
			return fmt.Errorf("flags %s are mutually exclusive", strings.Join(setFlags, " and "))
		}
		return nil
	}
}

// RequiredTogether ensures if any of the flags are set, all of them must be set.
func RequiredTogether(flags ...string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		var setFlags []string
		var missingFlags []string
		for _, f := range flags {
			if isFlagSet(fs, f) {
				setFlags = append(setFlags, f)
			} else {
				missingFlags = append(missingFlags, f)
			}
		}
		if len(setFlags) > 0 && len(missingFlags) > 0 {
			return fmt.Errorf("flags %s must be used together", strings.Join(flags, " and "))
		}
		return nil
	}
}

// RequiredWith ensures if target is set, all required flags must be set.
func RequiredWith(target string, required ...string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		if isFlagSet(fs, target) {
			for _, req := range required {
				if !isFlagSet(fs, req) {
					return fmt.Errorf("flag %s is required when using %s", req, target)
				}
			}
		}
		return nil
	}
}

// RequiredIf ensures flag is required if condition is met.
// The condition can be a bare flag name (meaning the condition flag is set),
// or key=value format (meaning the condition flag is set to value).
func RequiredIf(flag string, condition string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		if isFlagSet(fs, condition) {
			if !isFlagSet(fs, flag) {
				cleanCond := cleanFlagName(condition)
				if strings.Contains(cleanCond, "=") {
					parts := strings.SplitN(cleanCond, "=", 2)
					return fmt.Errorf("flag %s is required when %s is set to %q", flag, parts[0], parts[1])
				}
				return fmt.Errorf("flag %s is required when %s is set", flag, condition)
			}
		}
		return nil
	}
}
