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

// lookupFlagName resolves a name as a constraint writes it. pflag.Lookup indexes
// long names only, so a constraint written with shorthands used to resolve to
// nothing and enforce nothing.
func lookupFlagName(fs *pflag.FlagSet, name string) *pflag.Flag {
	if f := fs.Lookup(name); f != nil {
		return f
	}
	if len(name) == 1 {
		return fs.ShorthandLookup(name)
	}
	return nil
}

// isFlagSet reports whether the option named by flagName was given on the command
// line, under any of its spellings. A "name=value" condition additionally
// requires the value to match. A name no flag answers to is an error rather than
// a quiet false: a constraint over a flag that does not exist can never fire, and
// nothing else would ever say so.
func isFlagSet(fs *pflag.FlagSet, flagName string) (bool, error) {
	clean := cleanFlagName(flagName)
	want := ""
	hasValue := false
	if i := strings.Index(clean, "="); i >= 0 {
		clean, want, hasValue = clean[:i], clean[i+1:], true
	}
	flg := lookupFlagName(fs, clean)
	if flg == nil {
		return false, fmt.Errorf("option constraint names %q, which is not a flag of this command", flagName)
	}
	if !optionChanged(fs, optionGroup(flg)) {
		return false, nil
	}
	if hasValue {
		return flg.Value.String() == want, nil
	}
	return true, nil
}

// partitionFlags splits the given names into those that were set and those that
// were not.
func partitionFlags(fs *pflag.FlagSet, flags []string) (set, unset []string, err error) {
	for _, f := range flags {
		on, err := isFlagSet(fs, f)
		if err != nil {
			return nil, nil, err
		}
		if on {
			set = append(set, f)
		} else {
			unset = append(unset, f)
		}
	}
	return set, unset, nil
}

// MutuallyExclusive ensures at most one of the specified flags is set.
func MutuallyExclusive(flags ...string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		setFlags, _, err := partitionFlags(fs, flags)
		if err != nil {
			return err
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
		setFlags, missingFlags, err := partitionFlags(fs, flags)
		if err != nil {
			return err
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
		on, err := isFlagSet(fs, target)
		if err != nil || !on {
			return err
		}
		_, missing, err := partitionFlags(fs, required)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return fmt.Errorf("flag %s is required when using %s", missing[0], target)
		}
		return nil
	}
}

// RequiredIf ensures flag is required if condition is met.
// The condition can be a bare flag name (meaning the condition flag is set),
// or key=value format (meaning the condition flag is set to value).
func RequiredIf(flag string, condition string) OptionsValidator {
	return func(fs *pflag.FlagSet) error {
		on, err := isFlagSet(fs, condition)
		if err != nil || !on {
			return err
		}
		set, _, err := partitionFlags(fs, []string{flag})
		if err != nil || len(set) > 0 {
			return err
		}
		cleanCond := cleanFlagName(condition)
		if i := strings.Index(cleanCond, "="); i >= 0 {
			return fmt.Errorf("flag %s is required when %s is set to %q", flag, cleanCond[:i], cleanCond[i+1:])
		}
		return fmt.Errorf("flag %s is required when %s is set", flag, condition)
	}
}
