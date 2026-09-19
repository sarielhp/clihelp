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

// TestPositionalArityKnowsEveryCommandFlagPflagBinds verifies the agreement between
// positionalArity and the FlagSet constructed for a command. Every flag pflag binds
// on the command (including ancestor persistent options, local Options, and Hidden flags)
// must be known to positionalArity with the exact same value-taking arity.
func TestPositionalArityKnowsEveryCommandFlagPflagBinds(t *testing.T) {
	hide := func(o Option) Option { o.Hidden = true; return o }
	app := &App{
		Name: "app",
		PersistentOptions: []Option{
			Bool(new(bool), "--global-dry-run", false, "Dry run."),
		},
		Commands: []Command{
			{
				Name: "parent",
				PersistentOptions: []Option{
					String(new(string), "--parent-opt <val>", "", "Parent opt."),
				},
				Subcommands: []Command{
					{
						Name: "child",
						Options: []Option{
							String(new(string), "--local-opt <val>", "", "Local opt."),
							hide(Bool(new(bool), "--local-hidden", false, "Hidden opt.")),
							hide(String(new(string), "--local-hidden-val <val>", "", "Hidden val.")),
						},
						Run: func(*Context) error { return nil },
					},
				},
			},
		},
	}

	parent := &app.Commands[0]
	child := &parent.Subcommands[0]
	ancestors := []*Command{parent}

	fs, _, err := app.setupFlagSet(child, ancestors)
	if err != nil {
		t.Fatal(err)
	}

	res := resolution{ancestors: ancestors, cmd: child}
	arity := app.positionalArity(res, child)

	fs.VisitAll(func(f *pflag.Flag) {
		takesValue, known := arity["--"+f.Name]
		if !known {
			t.Errorf("pflag binds --%s and positionalArity does not know it", f.Name)
			return
		}
		if want := f.NoOptDefVal == ""; takesValue != want {
			t.Errorf("--%s: positionalArity says takesValue=%v, pflag says %v", f.Name, takesValue, want)
		}
	})
}
