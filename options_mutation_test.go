package clihelp

import (
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// parseFlagSpec decides what every option is called. Four of its rules had no
// assertion: the short "-[no-]" toggle marker, discarding a nameless "--", the
// all-uppercase placeholder form, and which marker wins when a spec carries two.
func TestParseFlagSpecRules(t *testing.T) {
	for _, tt := range []struct {
		name        string
		spec        string
		longNames   []string
		shortNames  []string
		placeholder string
		isToggle    bool
		baseToggle  string
	}{
		{name: "long toggle marker", spec: "--[no-]color",
			longNames: []string{"color", "no-color"}, isToggle: true, baseToggle: "color"},
		{name: "short toggle marker", spec: "-[no-]color",
			longNames: []string{"color", "no-color"}, isToggle: true, baseToggle: "color"},
		{name: "the first marker wins", spec: "--[no-]color --[no-]tint",
			longNames: []string{"color", "no-color", "tint", "no-tint"}, isToggle: true, baseToggle: "color"},
		{name: "an angle placeholder", spec: "--out <file>, -o",
			longNames: []string{"out"}, shortNames: []string{"o"}, placeholder: "<file>"},
		{name: "an uppercase placeholder", spec: "--out PATH, -o",
			longNames: []string{"out"}, shortNames: []string{"o"}, placeholder: "PATH"},
		{name: "a bracket placeholder", spec: "--out [file]",
			longNames: []string{"out"}, placeholder: "[file]"},
		{name: "a lowercase word is not a placeholder", spec: "--out file",
			longNames: []string{"out"}},
		{name: "a bare -- names nothing", spec: "-- --out", longNames: []string{"out"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFlagSpec(tt.spec)
			if strings.Join(got.longNames, ",") != strings.Join(tt.longNames, ",") {
				t.Errorf("longNames = %v, want %v", got.longNames, tt.longNames)
			}
			if strings.Join(got.shortNames, ",") != strings.Join(tt.shortNames, ",") {
				t.Errorf("shortNames = %v, want %v", got.shortNames, tt.shortNames)
			}
			if got.placeholder != tt.placeholder {
				t.Errorf("placeholder = %q, want %q", got.placeholder, tt.placeholder)
			}
			if got.isToggle != tt.isToggle {
				t.Errorf("isToggle = %v, want %v", got.isToggle, tt.isToggle)
			}
			if got.baseToggle != tt.baseToggle {
				t.Errorf("baseToggle = %q, want %q", got.baseToggle, tt.baseToggle)
			}
			if tt.placeholder != "" && !flagTakesValue(Option{Flags: tt.spec}, got) {
				t.Errorf("a spec with placeholder %q does not read as taking a value", tt.placeholder)
			}
		})
	}
}

// toggleBase picks the name the negative spelling is derived from: the one
// inside the marker, and only failing that the first long name. Both halves
// could be removed with the suite green.
func TestToggleBase(t *testing.T) {
	for _, tt := range []struct{ spec, want string }{
		{"--[no-]color", "color"},
		{"--colour --[no-]color", "color"}, // the marker wins over position
		{"--cache", "cache"},               // no marker: the first long name
		{"--cache --tint", "cache"},
		{"-c", ""}, // nothing to derive from
	} {
		if got := parseFlagSpec(tt.spec).toggleBase(); got != tt.want {
			t.Errorf("toggleBase(%q) = %q, want %q", tt.spec, got, tt.want)
		}
	}
}

// Each extra shorthand is registered under a synthetic long name, because pflag
// cannot attach a second shorthand to an existing flag. Those names have to be
// distinct per option, or two options collide at bind time.
func TestAliasFlagNamesAreDistinctPerOption(t *testing.T) {
	seen := map[string]string{}
	for _, tt := range []struct{ primary, short string }{
		{"tag", "t"}, {"tag", "T"}, {"topic", "t"}, {"", "y"}, {"", "Y"},
	} {
		name := aliasFlagName(tt.primary, tt.short)
		if prev, clash := seen[name]; clash {
			t.Errorf("aliasFlagName(%q, %q) = %q, which already belongs to %s",
				tt.primary, tt.short, name, prev)
		}
		seen[name] = tt.primary + "/" + tt.short
		if tt.primary != "" && !strings.Contains(name, tt.primary) {
			t.Errorf("aliasFlagName(%q, %q) = %q does not name its option", tt.primary, tt.short, name)
		}
		if !strings.Contains(name, tt.short) {
			t.Errorf("aliasFlagName(%q, %q) = %q does not name its shorthand", tt.primary, tt.short, name)
		}
	}

	// And two short-only options really do bind together.
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	var a, b string
	for _, opt := range []Option{
		String(&a, "-y, -Y <v>", "", "first"),
		String(&b, "-z, -Z <v>", "", "second"),
	} {
		if err := opt.binder(fs); err != nil {
			t.Fatalf("binding %q failed: %v", opt.Flags, err)
		}
	}
	if err := fs.Parse([]string{"-Y", "one", "-Z", "two"}); err != nil {
		t.Fatalf("parsing failed: %v", err)
	}
	if a != "one" || b != "two" {
		t.Errorf("the second shorthands did not bind independently: a=%q b=%q", a, b)
	}
}

// The synthetic alias names are an implementation detail and must stay out of
// the help; the spellings the author wrote must stay in it.
func TestSyntheticAliasNamesAreHidden(t *testing.T) {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	var tag []string
	if err := StringSlice(&tag, "--tag <v>, -t, -T", nil, "Tag it.").binder(fs); err != nil {
		t.Fatal(err)
	}
	visible := map[string]bool{}
	fs.VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			visible[f.Name] = true
		}
	})
	if !visible["tag"] {
		t.Error("the primary long name is hidden")
	}
	if visible[aliasFlagName("tag", "T")] {
		t.Errorf("the synthetic name %q is visible in the help", aliasFlagName("tag", "T"))
	}
}

// Every spelling of an option is one option: the tag is what ties them together
// for "was this set" and for grouping in the help.
func TestEverySpellingSharesOneIdentity(t *testing.T) {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	var tag []string
	if err := StringSlice(&tag, "--tag <v>, --label, -t, -T", nil, "Tag it.").binder(fs); err != nil {
		t.Fatal(err)
	}
	fs.VisitAll(func(f *pflag.Flag) {
		if got := optionGroup(f); got != "tag" {
			t.Errorf("flag --%s belongs to option %q, want %q", f.Name, got, "tag")
		}
	})

	if optionChanged(fs, "tag") {
		t.Error("nothing was written yet and the option reports as changed")
	}
	if err := fs.Parse([]string{"-T", "x"}); err != nil {
		t.Fatal(err)
	}
	if !optionChanged(fs, "tag") {
		t.Error("the option was set through a shorthand alias and does not report as changed")
	}
	if len(tag) != 1 || tag[0] != "x" {
		t.Errorf("the value did not reach the target: %v", tag)
	}
}

// A toggle is a switch: "--color" must not demand a value, in any of its
// spellings, which is what IsBoolFlag tells pflag.
func TestToggleNeedsNoValue(t *testing.T) {
	for _, args := range [][]string{
		{"--color", "positional"},
		{"--no-color", "positional"},
		{"-c", "positional"},
	} {
		fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
		on := false
		if err := BoolToggle(&on, "--[no-]color, -c", true, "Colour.").binder(fs); err != nil {
			t.Fatal(err)
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if got := fs.Args(); len(got) != 1 || got[0] != "positional" {
			t.Errorf("%v: the toggle consumed the following argument: %v", args, got)
		}
		if want := args[0] != "--no-color"; on != want {
			t.Errorf("%v: toggle is %v, want %v", args, on, want)
		}
	}
}

// optionGroup falls back to the flag's own name, which is a safety net for flags
// clihelp did not bind — so reading the group back cannot tell us whether the
// primary flag was actually tagged. The annotation is the contract; assert it
// directly, or the tagging can be deleted with the suite green.
func TestPrimaryFlagCarriesTheGroupAnnotation(t *testing.T) {
	for _, tt := range []struct{ name, spec, primary string }{
		{"plain option", "--tag <v>, -t", "tag"},
		{"toggle", "--[no-]color, -c", "color"},
		{"toggle without a marker", "--cache", "cache"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
			var tag []string
			var on bool
			opt := StringSlice(&tag, tt.spec, nil, "Tag it.")
			if strings.Contains(tt.name, "toggle") {
				opt = BoolToggle(&on, tt.spec, true, "Colour.")
			}
			if err := opt.binder(fs); err != nil {
				t.Fatal(err)
			}
			f := fs.Lookup(tt.primary)
			if f == nil {
				t.Fatalf("no flag was bound for --%s", tt.primary)
			}
			got := f.Annotations[aliasGroupAnnotation]
			if len(got) != 1 || got[0] != tt.primary {
				t.Errorf("--%s carries group annotation %v, want [%q]", tt.primary, got, tt.primary)
			}
		})
	}
}

// pflag decides whether "--color" consumes the next argument from NoOptDefVal,
// so IsBoolFlag never runs during parsing here. It is part of the pflag Value
// contract that other consumers — cobra, completion generators — do read, so it
// is asserted against the interface rather than through a parse.
func TestToggleValueSatisfiesThePflagBoolContract(t *testing.T) {
	var on bool
	var v pflag.Value = &toggleVal{target: &on, positive: true}
	b, ok := v.(interface{ IsBoolFlag() bool })
	if !ok {
		t.Fatal("toggleVal does not implement pflag's boolFlag interface")
	}
	if !b.IsBoolFlag() {
		t.Error("toggleVal reports that it is not a boolean flag")
	}
	if v.Type() != "bool" {
		t.Errorf("toggleVal.Type() = %q, want %q", v.Type(), "bool")
	}
}

// The arity table negates the toggle's base name as well as every long name the
// spec declared, and today that is belt and braces: parseFlagSpec always puts the
// base among the long names. This records the invariant, so that if it ever stops
// holding, the redundancy is known to have become load-bearing rather than being
// tidied away.
func TestToggleBaseIsAlwaysAmongTheLongNames(t *testing.T) {
	for _, spec := range []string{
		"--[no-]color", "--[no-]color, -c", "--cache", "--cache --tint",
		"--colour --[no-]color", "-c, --[no-]color",
	} {
		s := parseFlagSpec(spec)
		base := s.toggleBase()
		if base == "" {
			t.Errorf("%q: no toggle base", spec)
			continue
		}
		if !slices.Contains(s.longNames, base) {
			t.Errorf("%q: base %q is not among the long names %v; the base's negative "+
				"spelling in addFlagArity is now the only thing binding it",
				spec, base, s.longNames)
		}
	}
}
