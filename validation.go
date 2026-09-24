package clihelp

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// Flags is what an OptionsValidator is given: the options of the command being
// run, as the user left them.
//
// It is an interface clihelp owns rather than *pflag.FlagSet, so that writing a
// custom validator does not require importing pflag. While the signature named
// pflag, this library's compatibility promise depended on pflag's — a pflag v2
// would have broken every consumer who had ever written a validator, and clihelp
// could not have shielded them. The binding library is meant to be an
// implementation detail of the "--tag <v>, -t" spec string.
type Flags interface {
	// Changed reports whether the named option was given on the command line,
	// under any of its spellings. A "name=value" form additionally requires the
	// value to match. Naming an option the command does not have is an error
	// rather than a quiet false, because a constraint over a flag that does not
	// exist can never fire and nothing else would ever say so.
	Changed(name string) (bool, error)
	// Value returns the named option's value as a string.
	Value(name string) (string, error)
}

// flagSetView adapts a pflag.FlagSet to Flags.
type flagSetView struct{ fs *pflag.FlagSet }

func (v flagSetView) Changed(name string) (bool, error) { return isFlagSet(v.fs, name) }

func (v flagSetView) Value(name string) (string, error) {
	flg := lookupFlagName(v.fs, cleanFlagName(name))
	if flg == nil {
		return "", fmt.Errorf("option constraint names %q, which is not a flag of this command", name)
	}
	return flg.Value.String(), nil
}

// ValidateOptions chains multiple OptionsValidators into a single validator.
func ValidateOptions(validators ...OptionsValidator) OptionsValidator {
	return func(f Flags) error {
		for _, v := range validators {
			if v != nil {
				if err := v(f); err != nil {
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
func partitionFlags(opts Flags, flags []string) (set, unset []string, err error) {
	for _, name := range flags {
		on, err := opts.Changed(name)
		if err != nil {
			return nil, nil, err
		}
		if on {
			set = append(set, name)
		} else {
			unset = append(unset, name)
		}
	}
	return set, unset, nil
}

// MutuallyExclusive ensures at most one of the specified flags is set.
func MutuallyExclusive(flags ...string) OptionsValidator {
	return func(f Flags) error {
		setFlags, _, err := partitionFlags(f, flags)
		if err != nil {
			return err
		}
		if len(setFlags) > 1 {
			return fmt.Errorf("%w: flags %s are mutually exclusive", ErrUsage, strings.Join(setFlags, " and "))
		}
		return nil
	}
}

// RequiredTogether ensures if any of the flags are set, all of them must be set.
func RequiredTogether(flags ...string) OptionsValidator {
	return func(f Flags) error {
		setFlags, missingFlags, err := partitionFlags(f, flags)
		if err != nil {
			return err
		}
		if len(setFlags) > 0 && len(missingFlags) > 0 {
			return fmt.Errorf("%w: flags %s must be used together", ErrUsage, strings.Join(flags, " and "))
		}
		return nil
	}
}

// RequiredWith ensures if target is set, all required flags must be set.
func RequiredWith(target string, required ...string) OptionsValidator {
	return func(f Flags) error {
		on, err := f.Changed(target)
		if err != nil || !on {
			return err
		}
		_, missing, err := partitionFlags(f, required)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return fmt.Errorf("%w: flag %s is required when using %s", ErrUsage, missing[0], target)
		}
		return nil
	}
}

// RequiredIf ensures flag is required if condition is met.
// The condition can be a bare flag name (meaning the condition flag is set),
// or key=value format (meaning the condition flag is set to value).
func RequiredIf(flag string, condition string) OptionsValidator {
	return func(f Flags) error {
		on, err := f.Changed(condition)
		if err != nil || !on {
			return err
		}
		set, _, err := partitionFlags(f, []string{flag})
		if err != nil || len(set) > 0 {
			return err
		}
		cleanCond := cleanFlagName(condition)
		if i := strings.Index(cleanCond, "="); i >= 0 {
			return fmt.Errorf("%w: flag %s is required when %s is set to %q", ErrUsage, flag, cleanCond[:i], cleanCond[i+1:])
		}
		return fmt.Errorf("%w: flag %s is required when %s is set", ErrUsage, flag, condition)
	}
}
