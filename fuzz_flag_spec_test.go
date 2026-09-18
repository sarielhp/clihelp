package clihelp

import (
	"io"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

// FuzzBindFlagSpec drives arbitrary flag specs through every typed constructor's
// binder. pflag answers a malformed name with a panic, so the property under test
// is that a spec never reaches it unchecked: a binder either registers the option
// or returns an error.
func FuzzBindFlagSpec(f *testing.F) {
	for _, seed := range []string{
		"-o, --output PATH",
		"-p, -P, --port, --listen-port <num>",
		"--[no-]cache",
		"--[no-]color, --colour, -c, -C",
		"out <F>",
		"-out <F>",
		"-é",
		"--a, --a",
		"-a, -a",
		"--x, --x-alias-y, -a, -y",
		"--[no-]",
		"--=",
		"",
		"-",
		"--",
		"--no-cache, --[no-]cache",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, spec string) {
		var (
			str    string
			num    int
			flag   bool
			toggle bool
			dur    time.Duration
			list   []string
			choice string
		)
		options := []Option{
			String(&str, spec, "", "usage"),
			Int(&num, spec, 0, "usage"),
			Bool(&flag, spec, false, "usage"),
			BoolToggle(&toggle, spec, false, "usage"),
			Duration(&dur, spec, 0, "usage"),
			StringSlice(&list, spec, nil, "usage"),
			Enum(&choice, spec, []string{""}, "", "usage"),
		}
		for _, opt := range options {
			fs := pflag.NewFlagSet("fuzz", pflag.ContinueOnError)
			fs.SetOutput(io.Discard)
			if err := opt.Binder(fs); err != nil {
				continue // the spec was rejected, which is a legitimate answer
			}
			// A second bind of the same spec must be refused, not panic — and
			// this was only a comment: both results were discarded, so every
			// name-collision check in options.go could have been removed with
			// the fuzzer still green.
			if err := opt.Binder(fs); err == nil {
				t.Errorf("binding %q twice was accepted; the second registration "+
					"silently shadows or duplicates the first", spec)
			}
		}
	})
}
