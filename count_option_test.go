package clihelp

import (
	"strings"
	"testing"
)

// "-v", "-vv", "-vvv" is the standard spelling for verbosity, and two of the
// four applications built on this library reached past it for a pflag feature it
// did not wrap — this one, through a hand-written Option.Binder.
func TestCountCountsItsRepetitions(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want int
	}{
		{"absent", []string{"build"}, 0},
		{"once, short", []string{"build", "-v"}, 1},
		{"twice, bundled", []string{"build", "-vv"}, 2},
		{"three times, bundled", []string{"build", "-vvv"}, 3},
		{"twice, separately", []string{"build", "-v", "-v"}, 2},
		{"long form", []string{"build", "--verbose"}, 1},
		{"long form twice", []string{"build", "--verbose", "--verbose"}, 2},
		{"set directly", []string{"build", "--verbose=3"}, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var n int
			app := &App{Name: "demo", Commands: []Command{{
				Name: "build", Description: "Build it.", Args: NoArgs,
				Options: []Option{Count(&n, "-v, --verbose", "Be louder.")},
				Run:     func(*Context) error { return nil },
			}}}
			silentApp(app)
			if err := app.Execute(tt.args); err != nil {
				t.Fatalf("%v: %v", tt.args, err)
			}
			if n != tt.want {
				t.Errorf("%v: count = %d, want %d", tt.args, n, tt.want)
			}
		})
	}
}

// A counting flag consumes nothing, so the word after it is still available to
// be a command name. Getting this wrong is how a command becomes a positional
// and the program prints help and exits 0.
func TestACountFlagDoesNotEatTheNextArgument(t *testing.T) {
	var n int
	var ran bool
	app := &App{
		Name:              "demo",
		PersistentOptions: []Option{Count(&n, "-v, --verbose", "Be louder.")},
		Commands: []Command{{
			Name: "build", Description: "Build it.", Args: NoArgs,
			Run: func(*Context) error { ran = true; return nil },
		}},
	}
	silentApp(app)
	if err := app.Execute([]string{"-vv", "build"}); err != nil {
		t.Fatalf("`demo -vv build` failed: %v", err)
	}
	if !ran {
		t.Error("the command did not run: its name was taken as the flag's value")
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}

	// And through the arity table command resolution reads.
	opt := Count(&n, "-v, --verbose", "x")
	if flagTakesValue(opt, parseFlagSpec(opt.Flags)) {
		t.Error("a counting flag is recorded as consuming the argument after it")
	}
}

// Example validation binds a stand-in rather than writing through the
// application's own target, as it does for every other constructor.
func TestCountExamplesPassAuditWithoutWriting(t *testing.T) {
	var n int
	app := &App{Name: "demo", Commands: []Command{{
		Name: "build", Description: "Build it.", Args: NoArgs,
		Options:  []Option{Count(&n, "-v, --verbose", "Be louder.")},
		Run:      func(*Context) error { return nil },
		Examples: []Example{{Line: "demo build -vv", Description: "Loudly."}},
	}}}
	if err := Audit(app); err != nil {
		t.Errorf("Audit rejected an example using a counting flag: %v", err)
	}
	if n != 0 {
		t.Errorf("validating examples wrote %d through the application's own target", n)
	}
}

// It appears in help like any other flag.
func TestCountRendersInHelp(t *testing.T) {
	var n int
	app := &App{Name: "demo", Commands: []Command{{
		Name: "build", Description: "Build it.",
		Options: []Option{Count(&n, "-v, --verbose", "Be louder.")},
		Run:     func(*Context) error { return nil },
	}}}
	out := silentApp(app)
	app.RenderCommand(Options{Writer: out, Width: 100}, "build")
	if body := StripANSI(out.String()); !strings.Contains(body, "-v, --verbose") ||
		!strings.Contains(body, "Be louder.") {
		t.Errorf("the counting flag is missing from the help:\n%s", body)
	}
}
