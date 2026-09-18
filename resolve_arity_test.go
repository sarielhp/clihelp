package clihelp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// H1 — bindToggle derives a "--no-" spelling for the toggle base and for every
// positive long name; addFlagArity registered only the names the spec string
// spells out. So pflag bound --no-cache and the arity probe did not: resolution
// stopped at the flag, the command name after it survived as a positional, and
// the global help rendered with exit 0.
func TestToggleNegativeSpellingResolvesBeforeTheCommand(t *testing.T) {
	for _, spec := range []string{
		"--[no-]color",
		"--cache",
		"--[no-]color, --colour",
		"--verbose, -V",
	} {
		for _, flag := range negativeSpellings(spec) {
			t.Run(spec+" "+flag, func(t *testing.T) {
				var on bool
				ran := false
				app := &App{
					Name:              "specapp",
					PersistentOptions: []Option{BoolToggle(&on, spec, true, "A toggle.")},
					Commands: []Command{{
						Name: "run", Description: "Run it.",
						Run: func(*Context) error { ran = true; return nil },
					}},
				}
				var out bytes.Buffer
				app.Stdout, app.Stderr = &out, &out

				if err := app.Execute([]string{flag, "run"}); err != nil {
					t.Fatalf("`specapp %s run` failed: %v", flag, err)
				}
				if !ran {
					t.Errorf("`specapp %s run` printed %d bytes of help instead of running", flag, out.Len())
				}
				if on {
					t.Errorf("the toggle was not applied")
				}
			})
		}
	}
}

// negativeSpellings names the "--no-" forms bindToggle derives for a spec.
func negativeSpellings(spec string) []string {
	var out []string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "--") {
			continue
		}
		name := strings.TrimPrefix(part, "--")
		name = strings.ReplaceAll(name, "[no-]", "")
		if name != "" && !strings.HasPrefix(name, "no-") {
			out = append(out, "--no-"+name)
		}
	}
	return out
}

// The invariant the arity map exists to satisfy, asserted directly: every long
// flag the real FlagSet binds must be known to the probe, with the same arity.
// Nothing tested this, which is why one toggle shape worked and three did not.
func TestArityProbeKnowsEveryFlagPflagBinds(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts []Option
	}{
		{"marked toggle", []Option{BoolToggle(new(bool), "--[no-]color", true, "T.")}},
		{"unmarked toggle", []Option{BoolToggle(new(bool), "--cache", true, "T.")}},
		{"toggle with an alias", []Option{BoolToggle(new(bool), "--[no-]color, --colour", true, "T.")}},
		{"toggle with a shorthand", []Option{BoolToggle(new(bool), "--verbose, -V", true, "T.")}},
		{"short-only string", []Option{String(new(string), "-S <v>", "", "S.")}},
		{"several shorthands", []Option{String(new(string), "--tag <v>, -t, -T", "", "T.")}},
		{"plain bool", []Option{Bool(new(bool), "--force, -f", false, "F.")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{Name: "app", PersistentOptions: tt.opts,
				Commands: []Command{{Name: "run", Run: func(*Context) error { return nil }}}}

			fs, _, err := app.setupFlagSet(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			probe := app.leadingFlagArity(nil)

			fs.VisitAll(func(f *pflag.Flag) {
				takesValue, known := probe["--"+f.Name]
				if !known {
					t.Errorf("pflag binds --%s and the arity probe does not know it", f.Name)
					return
				}
				if want := f.NoOptDefVal == ""; takesValue != want {
					t.Errorf("--%s: probe says takesValue=%v, pflag says %v", f.Name, takesValue, want)
				}
			})
		})
	}
	_ = fmt.Sprint()
}
