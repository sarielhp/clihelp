package clihelp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type flagSpec struct {
	raw         string
	longNames   []string
	shortNames  []string
	placeholder string
	isToggle    bool
	baseToggle  string
}

func (fs flagSpec) hasHelpFlag() bool {
	for _, l := range fs.longNames {
		if l == "help" {
			return true
		}
	}
	for _, s := range fs.shortNames {
		if s == "h" {
			return true
		}
	}
	return false
}

func parseFlagSpec(spec string) flagSpec {
	fs := flagSpec{raw: spec}
	// Clean commas and split by whitespace
	parts := strings.Fields(spec)
	for _, raw := range parts {
		token := strings.Trim(raw, ",")
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, "--[no-]") || strings.HasPrefix(token, "-[no-]") {
			fs.isToggle = true
			base := strings.TrimPrefix(token, "--[no-]")
			base = strings.TrimPrefix(base, "-[no-]")
			fs.baseToggle = base
			fs.longNames = append(fs.longNames, base, "no-"+base)
			continue
		}
		if strings.HasPrefix(token, "--") {
			name := strings.TrimPrefix(token, "--")
			if name != "" {
				fs.longNames = append(fs.longNames, name)
			}
			continue
		}
		if strings.HasPrefix(token, "-") {
			name := strings.TrimPrefix(token, "-")
			if name != "" {
				fs.shortNames = append(fs.shortNames, name)
			}
			continue
		}
		if strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[") || strings.ToUpper(token) == token {
			fs.placeholder = token
		}
	}
	return fs
}

func bindHelper(fs *pflag.FlagSet, spec flagSpec, fn func(long, short string)) error {
	if spec.hasHelpFlag() {
		return fmt.Errorf("flag spec %q: -h/--help flags are automatically managed by clihelp and must not be declared in Options", spec.raw)
	}

	primaryLong := ""
	if len(spec.longNames) > 0 {
		primaryLong = spec.longNames[0]
	}
	primaryShort := ""
	if len(spec.shortNames) > 0 {
		primaryShort = spec.shortNames[0]
	}

	if primaryLong != "" || primaryShort != "" {
		fn(primaryLong, primaryShort)
	}

	// Register additional long names
	for i := 1; i < len(spec.longNames); i++ {
		fn(spec.longNames[i], "")
	}
	// Register additional short names with alias names
	for i := 1; i < len(spec.shortNames); i++ {
		short := spec.shortNames[i]
		aliasLong := fmt.Sprintf("%s-alias-%s", primaryLong, short)
		if primaryLong == "" {
			aliasLong = "alias-" + short
		}
		fn(aliasLong, short)
		_ = fs.MarkHidden(aliasLong)
	}
	return nil
}

// String binds a string flag to target.
func String(target *string, flags string, defaultVal string, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.StringVarP(target, long, short, defaultVal, usage)
				} else if long != "" {
					fs.StringVar(target, long, defaultVal, usage)
				} else if short != "" {
					fs.StringVarP(target, "flag-"+short, short, defaultVal, usage)
				}
			})
		},
	}
}

// Int binds an integer flag to target.
func Int(target *int, flags string, defaultVal int, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.IntVarP(target, long, short, defaultVal, usage)
				} else if long != "" {
					fs.IntVar(target, long, defaultVal, usage)
				} else if short != "" {
					fs.IntVarP(target, "flag-"+short, short, defaultVal, usage)
				}
			})
		},
	}
}

// Bool binds a boolean flag to target.
func Bool(target *bool, flags string, defaultVal bool, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.BoolVarP(target, long, short, defaultVal, usage)
				} else if long != "" {
					fs.BoolVar(target, long, defaultVal, usage)
				} else if short != "" {
					fs.BoolVarP(target, "flag-"+short, short, defaultVal, usage)
				}
			})
		},
	}
}

type toggleVal struct {
	target   *bool
	positive bool
}

func (t *toggleVal) String() string {
	if t.target == nil {
		return "false"
	}
	if t.positive {
		return strconv.FormatBool(*t.target)
	}
	return strconv.FormatBool(!*t.target)
}

func (t *toggleVal) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if t.positive {
		*t.target = b
	} else {
		*t.target = !b
	}
	return nil
}

func (t *toggleVal) Type() string {
	return "bool"
}

func (t *toggleVal) IsBoolFlag() bool {
	return true
}

// BoolToggle binds a boolean toggle pair (e.g. --[no-]check-new).
func BoolToggle(target *bool, flags string, defaultVal bool, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	base := spec.baseToggle
	if base == "" && len(spec.longNames) > 0 {
		base = spec.longNames[0]
	}

	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			if spec.hasHelpFlag() {
				return fmt.Errorf("flag spec %q: -h/--help flags are automatically managed by clihelp and must not be declared in Options", flags)
			}
			pos := &toggleVal{target: target, positive: true}
			neg := &toggleVal{target: target, positive: false}

			primaryShort := ""
			if len(spec.shortNames) > 0 {
				primaryShort = spec.shortNames[0]
			}

			if primaryShort != "" {
				fs.VarP(pos, base, primaryShort, usage)
			} else {
				fs.Var(pos, base, usage)
			}
			if f := fs.Lookup(base); f != nil {
				f.NoOptDefVal = "true"
			}

			noFlag := "no-" + base
			fs.Var(neg, noFlag, "Disable "+usage)
			if f := fs.Lookup(noFlag); f != nil {
				f.NoOptDefVal = "true"
			}
			_ = fs.MarkHidden(noFlag)
			return nil
		},
	}
}

// Duration binds a time.Duration flag to target.
func Duration(target *time.Duration, flags string, defaultVal time.Duration, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.DurationVarP(target, long, short, defaultVal, usage)
				} else if long != "" {
					fs.DurationVar(target, long, defaultVal, usage)
				} else if short != "" {
					fs.DurationVarP(target, "flag-"+short, short, defaultVal, usage)
				}
			})
		},
	}
}

// StringSlice binds a repeatable or comma-separated string slice flag to target.
func StringSlice(target *[]string, flags string, defaultVal []string, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.StringSliceVarP(target, long, short, defaultVal, usage)
				} else if long != "" {
					fs.StringSliceVar(target, long, defaultVal, usage)
				} else if short != "" {
					fs.StringSliceVarP(target, "flag-"+short, short, defaultVal, usage)
				}
			})
		},
	}
}

type enumVal struct {
	target  *string
	allowed []string
}

func (e *enumVal) String() string {
	if e.target == nil {
		return ""
	}
	return *e.target
}

func (e *enumVal) Set(val string) error {
	for _, a := range e.allowed {
		if a == val {
			*e.target = val
			return nil
		}
	}
	return fmt.Errorf("invalid value %q: must be one of [%s]", val, strings.Join(e.allowed, ", "))
}

func (e *enumVal) Type() string {
	return "string"
}

// Enum restricts input to an enumerated list of valid strings.
func Enum(target *string, flags string, allowed []string, defaultVal string, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Complete: func(toComplete string) []string {
			var matches []string
			for _, a := range allowed {
				if strings.HasPrefix(a, toComplete) {
					matches = append(matches, a)
				}
			}
			return matches
		},
		Binder: func(fs *pflag.FlagSet) error {
			val := &enumVal{target: target, allowed: allowed}
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.VarP(val, long, short, usage)
				} else if long != "" {
					fs.Var(val, long, usage)
				} else if short != "" {
					fs.VarP(val, "flag-"+short, short, usage)
				}
			})
		},
	}
}

// Var binds a custom user-defined pflag.Value interface.
func Var(target pflag.Value, flags string, usage string) Option {
	spec := parseFlagSpec(flags)
	return Option{
		Flags:       flags,
		Description: usage,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				if long != "" && short != "" {
					fs.VarP(target, long, short, usage)
				} else if long != "" {
					fs.Var(target, long, usage)
				} else if short != "" {
					fs.VarP(target, "flag-"+short, short, usage)
				}
			})
		},
	}
}
