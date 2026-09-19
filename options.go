package clihelp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type flagSpec struct {
	raw           string
	longNames     []string
	shortNames    []string
	placeholder   string
	optionalValue bool
	isToggle      bool
	baseToggle    string
}

// aliasGroupAnnotation labels every pflag flag that stands for one Option with
// the name of that option's primary flag. pflag knows nothing about aliases: it
// sees "--title" and "-t" as two flags with two Changed bits, which is how a
// required option set through an alias came out missing and how a relation
// validator lost sight of a constraint. The annotation travels with the flag
// set, so code holding only the flag set can still ask which option a flag is a
// spelling of.
const aliasGroupAnnotation = "clihelp-option"

func (spec flagSpec) hasHelpFlag() bool {
	for _, l := range spec.longNames {
		if l == "help" {
			return true
		}
	}
	for _, s := range spec.shortNames {
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
		// --flag=VALUE and -o=VALUE: drop the placeholder value so the flag is
		// registered under its bare name; the "=VALUE" form is a parse-time
		// convention, not part of the spec.
		if strings.HasPrefix(token, "-") && strings.Contains(token, "=") {
			token = token[:strings.Index(token, "=")]
		}
		if strings.HasPrefix(token, "--[no-]") || strings.HasPrefix(token, "-[no-]") {
			fs.isToggle = true
			base := strings.TrimPrefix(token, "--[no-]")
			base = strings.TrimPrefix(base, "-[no-]")
			if fs.baseToggle == "" {
				fs.baseToggle = base
			}
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
			// "[From]" is the usage-line convention for a value that may be
			// omitted, and it read identically to "<From>" until now.
			fs.optionalValue = strings.HasPrefix(token, "[")
		}
	}
	return fs
}

// validate reports the spec defects pflag cannot survive and the ones it cannot
// see. A shorthand longer than one ASCII character panics inside
// pflag.ShorthandLookup; a name written without its dashes is discarded by
// parseFlagSpec, leaving an option that binds nothing and silently does not
// exist.
func (spec flagSpec) validate() error {
	for _, s := range spec.shortNames {
		if len(s) != 1 {
			return fmt.Errorf("flag spec %q: shorthand %q must be a single ASCII character (write %q to declare a long name)", spec.raw, "-"+s, "--"+s)
		}
	}
	for _, l := range spec.longNames {
		if l == "" {
			return fmt.Errorf("flag spec %q: empty long flag name", spec.raw)
		}
	}
	if len(spec.longNames) == 0 && len(spec.shortNames) == 0 {
		return fmt.Errorf("flag spec %q declares no flag names: every name needs its leading dashes, as in %q", spec.raw, "--out, -o <file>")
	}
	return nil
}

// validateToggle adds the one requirement a toggle has beyond a plain option: a
// long name to derive the negative spelling from.
func (spec flagSpec) validateToggle() error {
	if err := spec.validate(); err != nil {
		return err
	}
	if spec.toggleBase() == "" {
		return fmt.Errorf("flag spec %q: a toggle needs a long name to derive its --no- form from, as in %q", spec.raw, "--[no-]color")
	}
	return nil
}

// toggleBase is the positive long name of a toggle: the name inside "--[no-]",
// or the first long name when the spec is written without the marker.
func (spec flagSpec) toggleBase() string {
	if spec.baseToggle != "" {
		return spec.baseToggle
	}
	if len(spec.longNames) > 0 {
		return spec.longNames[0]
	}
	return ""
}

// primaryNames returns the long and short name the option is bound under. Either
// may be empty; validate guarantees they are not both empty.
func (spec flagSpec) primaryNames() (string, string) {
	long, short := "", ""
	if len(spec.longNames) > 0 {
		long = spec.longNames[0]
	}
	if len(spec.shortNames) > 0 {
		short = spec.shortNames[0]
	}
	return long, short
}

// primaryFlagName is the pflag name of the option's primary flag, which doubles
// as the key its aliases are annotated with. A short-only option has no long
// name of its own, so it is bound under the synthetic name pflag needs.
func (spec flagSpec) primaryFlagName() string {
	long, short := spec.primaryNames()
	if long != "" {
		return long
	}
	if short != "" {
		return "flag-" + short
	}
	return ""
}

// aliasFlagName is the hidden long name an extra shorthand is registered under:
// pflag offers no way to attach a second shorthand to an existing flag.
func aliasFlagName(primaryLong, short string) string {
	if primaryLong == "" {
		return "alias-" + short
	}
	return primaryLong + "-alias-" + short
}

// optionGroup names the option a flag is a spelling of: the primary flag name it
// was annotated with, or its own name for flags registered outside the option
// machinery, such as the built-in help flags.
func optionGroup(f *pflag.Flag) string {
	if g := f.Annotations[aliasGroupAnnotation]; len(g) > 0 {
		return g[0]
	}
	return f.Name
}

// optionChanged reports whether any spelling of the option identified by
// primaryName appeared on the command line.
func optionChanged(fs *pflag.FlagSet, primaryName string) bool {
	changed := false
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed && optionGroup(f) == primaryName {
			changed = true
		}
	})
	return changed
}

func tagOptionFlag(f *pflag.Flag, primaryName string) {
	if f.Annotations == nil {
		f.Annotations = make(map[string][]string)
	}
	f.Annotations[aliasGroupAnnotation] = []string{primaryName}
}

// checkSpecAvailable reports a name that is already taken, which pflag would
// otherwise answer with a panic.
func checkSpecAvailable(fs *pflag.FlagSet, spec flagSpec) error {
	for _, l := range spec.longNames {
		if fs.Lookup(l) != nil {
			return fmt.Errorf("flag %q in spec %q is already registered in %s flagset", "--"+l, spec.raw, fs.Name())
		}
	}
	for _, s := range spec.shortNames {
		if fs.ShorthandLookup(s) != nil {
			return fmt.Errorf("shorthand %q in spec %q is already registered in %s flagset", "-"+s, spec.raw, fs.Name())
		}
	}
	return nil
}

// bindAlias registers one more spelling of an option that is already bound. The
// alias shares the primary flag's pflag.Value rather than binding a second value
// to the same target: two values mean two "already set" states, which is how
// "--tag a --tag b -T c" used to keep only "c".
func bindAlias(fs *pflag.FlagSet, primary *pflag.Flag, name, short string, hidden bool) error {
	if fs.Lookup(name) != nil {
		return fmt.Errorf("flag %q conflicts with an alias of %q", "--"+name, "--"+primary.Name)
	}
	if short != "" && fs.ShorthandLookup(short) != nil {
		return fmt.Errorf("shorthand %q is declared twice in the spec of %q", "-"+short, "--"+primary.Name)
	}
	f := fs.VarPF(primary.Value, name, short, primary.Usage)
	f.NoOptDefVal = primary.NoOptDefVal
	f.DefValue = primary.DefValue
	f.Hidden = hidden
	tagOptionFlag(f, optionGroup(primary))
	return nil
}

func bindHelper(fs *pflag.FlagSet, spec flagSpec, fn func(long, short string)) error {
	if spec.hasHelpFlag() {
		return fmt.Errorf("flag spec %q: -h/--help flags are automatically managed by clihelp and must not be declared in Options", spec.raw)
	}
	if err := spec.validate(); err != nil {
		return err
	}
	if err := checkSpecAvailable(fs, spec); err != nil {
		return err
	}

	primaryLong, primaryShort := spec.primaryNames()
	fn(primaryLong, primaryShort)
	primary := fs.Lookup(spec.primaryFlagName())
	if primary == nil {
		return fmt.Errorf("flag spec %q: no flag was registered for %q", spec.raw, spec.primaryFlagName())
	}
	tagOptionFlag(primary, primary.Name)

	for i := 1; i < len(spec.longNames); i++ {
		if err := bindAlias(fs, primary, spec.longNames[i], "", false); err != nil {
			return err
		}
	}
	for i := 1; i < len(spec.shortNames); i++ {
		if err := bindAlias(fs, primary, aliasFlagName(primaryLong, spec.shortNames[i]), spec.shortNames[i], true); err != nil {
			return err
		}
	}
	return nil
}

// String binds a string flag to target.
func String(target *string, flags string, defaultVal string, usage string) Option {
	opt := stringOption(target, flags, defaultVal, usage)
	opt.scratch = stringOption(new(string), flags, defaultVal, usage).Binder
	return opt
}

func stringOption(target *string, flags string, defaultVal string, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		arity:       arityValue,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultVal,
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
	opt := intOption(target, flags, defaultVal, usage)
	opt.scratch = intOption(new(int), flags, defaultVal, usage).Binder
	return opt
}

func intOption(target *int, flags string, defaultVal int, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	defaultText := ""
	if defaultVal != 0 {
		defaultText = strconv.Itoa(defaultVal)
	}
	return Option{
		arity:       arityValue,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultText,
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
	opt := boolOption(target, flags, defaultVal, usage)
	opt.scratch = boolOption(new(bool), flags, defaultVal, usage).Binder
	return opt
}

func boolOption(target *bool, flags string, defaultVal bool, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	defaultText := ""
	if defaultVal {
		defaultText = "true"
	}
	return Option{
		arity:       arityFlag,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultText,
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
	opt := boolToggleOption(target, flags, defaultVal, usage)
	opt.scratch = boolToggleOption(new(bool), flags, defaultVal, usage).Binder
	return opt
}

func boolToggleOption(target *bool, flags string, defaultVal bool, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		arity:       arityFlag,
		toggle:      true,
		Flags:       flags,
		Description: usage,
		DefaultText: strconv.FormatBool(defaultVal),
		Binder: func(fs *pflag.FlagSet) error {
			return bindToggle(fs, spec, target, usage)
		},
	}
}

// bindToggle registers a toggle under every spelling its spec declares: the base
// name, the negative counterpart of each positive long name, any further long
// name, and any further shorthand. All of them are tagged with the base name, so
// the option counts as set whichever spelling the user wrote.
func bindToggle(fs *pflag.FlagSet, spec flagSpec, target *bool, usage string) error {
	if spec.hasHelpFlag() {
		return fmt.Errorf("flag spec %q: -h/--help flags are automatically managed by clihelp and must not be declared in Options", spec.raw)
	}
	if err := spec.validateToggle(); err != nil {
		return err
	}
	if err := checkSpecAvailable(fs, spec); err != nil {
		return err
	}

	base := spec.toggleBase()
	_, primaryShort := spec.primaryNames()
	primary := fs.VarPF(&toggleVal{target: target, positive: true}, base, primaryShort, usage)
	primary.NoOptDefVal = "true"
	tagOptionFlag(primary, base)

	if err := bindToggleLongNames(fs, spec, target, usage); err != nil {
		return err
	}
	return bindToggleShorthands(fs, spec, base, primary)
}

// bindToggleLongNames binds the negative form of the base name, every further
// long name the spec declares, and the negative form of each of those. A name
// reachable two ways is bound once.
func bindToggleLongNames(fs *pflag.FlagSet, spec flagSpec, target *bool, usage string) error {
	base := spec.toggleBase()
	bound := map[string]bool{base: true}
	names := append([]string{"no-" + base}, spec.longNames...)
	for _, name := range names {
		if bound[name] {
			continue
		}
		bound[name] = true
		positive := !strings.HasPrefix(name, "no-")
		if err := bindToggleName(fs, name, base, target, usage, positive); err != nil {
			return err
		}
		if counter := "no-" + name; positive && !bound[counter] {
			bound[counter] = true
			if err := bindToggleName(fs, counter, base, target, usage, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func bindToggleName(fs *pflag.FlagSet, name, base string, target *bool, usage string, positive bool) error {
	if fs.Lookup(name) != nil {
		return fmt.Errorf("flag %q conflicts with an alias of %q", "--"+name, "--"+base)
	}
	text := usage
	if !positive {
		text = "Disable " + usage
	}
	f := fs.VarPF(&toggleVal{target: target, positive: positive}, name, "", text)
	f.NoOptDefVal = "true"
	f.Hidden = !positive
	tagOptionFlag(f, base)
	return nil
}

func bindToggleShorthands(fs *pflag.FlagSet, spec flagSpec, base string, primary *pflag.Flag) error {
	for i := 1; i < len(spec.shortNames); i++ {
		if err := bindAlias(fs, primary, aliasFlagName(base, spec.shortNames[i]), spec.shortNames[i], true); err != nil {
			return err
		}
	}
	return nil
}

// Duration binds a time.Duration flag to target.
func Duration(target *time.Duration, flags string, defaultVal time.Duration, usage string) Option {
	opt := durationOption(target, flags, defaultVal, usage)
	opt.scratch = durationOption(new(time.Duration), flags, defaultVal, usage).Binder
	return opt
}

func durationOption(target *time.Duration, flags string, defaultVal time.Duration, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	defaultText := ""
	if defaultVal != 0 {
		defaultText = defaultVal.String()
	}
	return Option{
		arity:       arityValue,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultText,
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
	opt := stringSliceOption(target, flags, defaultVal, usage)
	opt.scratch = stringSliceOption(new([]string), flags, defaultVal, usage).Binder
	return opt
}

func stringSliceOption(target *[]string, flags string, defaultVal []string, usage string) Option {
	// Copy so the caller's slice backing array is not aliased by pflag.
	*target = append([]string{}, defaultVal...)
	spec := parseFlagSpec(flags)
	defaultText := ""
	if len(defaultVal) > 0 {
		defaultText = strings.Join(defaultVal, ",")
	}
	return Option{
		arity:       arityValue,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultText,
		Binder: func(fs *pflag.FlagSet) error {
			return bindHelper(fs, spec, func(long, short string) {
				init := append([]string{}, defaultVal...)
				if long != "" && short != "" {
					fs.StringSliceVarP(target, long, short, init, usage)
				} else if long != "" {
					fs.StringSliceVar(target, long, init, usage)
				} else if short != "" {
					fs.StringSliceVarP(target, "flag-"+short, short, init, usage)
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
	opt := enumOption(target, flags, allowed, defaultVal, usage)
	opt.scratch = enumOption(new(string), flags, allowed, defaultVal, usage).Binder
	return opt
}

func enumOption(target *string, flags string, allowed []string, defaultVal string, usage string) Option {
	*target = defaultVal
	spec := parseFlagSpec(flags)
	return Option{
		arity:       arityValue,
		Flags:       flags,
		Description: usage,
		DefaultText: defaultVal,
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
			valid := false
			for _, a := range allowed {
				if a == defaultVal {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("flag spec %q: default value %q is not one of [%s]", flags, defaultVal, strings.Join(allowed, ", "))
			}
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

// Value is the interface a custom option's target implements: exactly
// pflag.Value's method set, named here so that a consumer writing one never has
// to import pflag. Any pflag.Value satisfies it and vice versa.
type Value interface {
	String() string
	Set(string) error
	Type() string
}

// optionalBinder wraps a binder so that every spelling the option was registered
// under accepts the flag without a value. pflag decides that from NoOptDefVal,
// and the option's spellings are only known once bind has run.
func optionalBinder(inner func(fs *pflag.FlagSet) error, flags, whenBare string) func(fs *pflag.FlagSet) error {
	if inner == nil {
		return nil
	}
	return func(fs *pflag.FlagSet) error {
		if err := checkOptionalSpec(flags, whenBare); err != nil {
			return err
		}
		if err := inner(fs); err != nil {
			return err
		}
		primary := parseFlagSpec(flags).primaryFlagName()
		fs.VisitAll(func(f *pflag.Flag) {
			if optionGroup(f) == primary {
				f.NoOptDefVal = whenBare
			}
		})
		return nil
	}
}

// checkOptionalSpec keeps the two halves of an optional value from disagreeing:
// the brackets in the spec string are what the user sees, and whenBare is what
// the flag does. A spec that promises one thing while the binding does another is
// the drift this library has been bitten by before, so it is refused at bind
// time rather than discovered from a help page.
func checkOptionalSpec(flags, whenBare string) error {
	spec := parseFlagSpec(flags)
	if whenBare == "" {
		return fmt.Errorf("flag spec %q: Optional needs a value for the bare form; it is what tells %q apart from the flag being absent",
			flags, spec.primaryFlagName())
	}
	if !spec.optionalValue {
		return fmt.Errorf("flag spec %q: Optional requires the placeholder in brackets, as in %q, because that is how the help page says the value may be omitted",
			flags, "--"+spec.primaryFlagName()+" ["+strings.Trim(spec.placeholder, "<>")+"]")
	}
	return nil
}

// Var binds a custom option whose target implements Value. Only the caller's
// value can parse the flag, so example validation binds a permissive stand-in
// rather than writing through it.
func Var(target Value, flags string, usage string) Option {
	opt := varOption(target, flags, usage)
	opt.scratch = varOption(&scratchValue{}, flags, usage).Binder
	return opt
}

func varOption(target Value, flags string, usage string) Option {
	spec := parseFlagSpec(flags)
	return Option{
		arity:       arityValue,
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

// scratchValue stands in for an option's real value while an example line is
// checked. It accepts anything, because what static validation asks of an
// example is that its flags exist and are shaped right — and the alternative,
// binding the option for real, writes both the declared default and the parsed
// value straight through the consumer's own pointer.
type scratchValue struct {
	typeName string
	val      string
}

func (s *scratchValue) String() string { return s.val }

func (s *scratchValue) Set(v string) error {
	s.val = v
	return nil
}

func (s *scratchValue) Type() string {
	if s.typeName == "" {
		return "string"
	}
	return s.typeName
}

// bindScratch registers opt on fs without touching the memory the application
// runs on. The typed constructors supply a binder over freshly allocated storage
// of the right type, so their parsing and their value checks still apply; an
// Option assembled by hand gets a stand-in derived from its flag spec, which
// checks the names and the arity but not the values.
func bindScratch(fs *pflag.FlagSet, opt Option) error {
	if opt.scratch != nil {
		return opt.scratch(fs)
	}
	if opt.Binder == nil {
		return nil
	}
	spec := parseFlagSpec(opt.Flags)
	takesValue := flagTakesValue(opt, spec)
	return bindHelper(fs, spec, func(long, short string) {
		name := long
		if name == "" {
			name = "flag-" + short
		}
		f := fs.VarPF(&scratchValue{}, name, short, opt.Description)
		if !takesValue {
			f.NoOptDefVal = "true"
		}
	})
}

// bindScratchAll binds every option in opts onto fs with bindScratch.
func bindScratchAll(fs *pflag.FlagSet, opts []Option) error {
	for _, opt := range opts {
		if err := bindScratch(fs, opt); err != nil {
			return err
		}
	}
	return nil
}
