package clihelp

import (
	"github.com/spf13/pflag"
	"strings"
	"testing"
)

// The example validator bound its own idea of the help flags — "--help" with
// shorthand "-h" — while the real flag set binds "--help-concise" with "-h" and
// "--help" with "-H". So an example line using a help flag the application
// really accepts failed validation with "invalid flag in example", and Audit,
// which the README recommends running in CI, reported a defect in an example
// that works.
func TestExamplesMayUseTheHelpFlagsTheAppAccepts(t *testing.T) {
	for _, tt := range []struct {
		name     string
		extended bool
		line     string
	}{
		{"concise shorthand", false, "myapp build -h"},
		{"concise long name", false, "myapp build --help-concise"},
		{"extended long name", false, "myapp build --help"},
		{"concise shorthand, extended enabled", true, "myapp build -h"},
		{"concise long name, extended enabled", true, "myapp build --help-concise"},
		{"extended long name, extended enabled", true, "myapp build --help"},
		{"extended shorthand, extended enabled", true, "myapp build -H"},
		{"help flag at the root", false, "myapp --help-concise"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				Name:             "myapp",
				ExtendedHelpFlag: tt.extended,
				Commands: []Command{{
					Name: "build", Description: "Build it.", Run: nopRun,
					Examples: []Example{{Line: tt.line, Description: "An example."}},
				}},
			}
			if errs := app.validateExamples(); len(errs) > 0 {
				t.Errorf("`%s` is accepted at run time but rejected by the validator: %v", tt.line, errs)
			}
		})
	}
}

// ...and a shorthand the application does NOT bind is still rejected: -H only
// exists when ExtendedHelpFlag is set, and the validator must not invent it.
func TestExamplesCannotUseAHelpFlagTheAppDoesNotBind(t *testing.T) {
	app := &App{
		Name: "myapp",
		Commands: []Command{{
			Name: "build", Description: "Build it.", Run: nopRun,
			Examples: []Example{{Line: "myapp build -H", Description: "An example."}},
		}},
	}
	errs := app.validateExamples()
	if len(errs) == 0 {
		t.Fatal("`-H` was accepted although ExtendedHelpFlag is not set")
	}
	if !strings.Contains(errs[0].Error(), "-H") {
		t.Errorf("the error does not name the flag: %v", errs[0])
	}
}

// The same line really does run, which is what makes the rejection wrong rather
// than a matter of taste.
func TestHelpFlagsInExamplesActuallyRun(t *testing.T) {
	for _, tt := range []struct {
		extended bool
		args     []string
	}{
		{false, []string{"build", "-h"}},
		{false, []string{"build", "--help-concise"}},
		{false, []string{"build", "--help"}},
		{true, []string{"build", "-H"}},
	} {
		app := &App{Name: "myapp", ExtendedHelpFlag: tt.extended,
			Commands: []Command{{Name: "build", Description: "Build it.", Run: nopRun}}}
		out := silentApp(app)
		if err := app.Execute(tt.args); err != nil {
			t.Errorf("`myapp %s` failed at run time: %v", strings.Join(tt.args, " "), err)
		}
		// "Did not error" was the whole assertion, so a help flag that quietly
		// printed nothing passed as working. What the user asked for is the help
		// page, so that is what is checked.
		if body := StripANSI(out.String()); !strings.Contains(body, "Build it.") {
			t.Errorf("`myapp %s` printed no help:\n%s", strings.Join(tt.args, " "), body)
		}
	}
}

// helpFlagNames and bindHelpFlags are two lists of the same thing: one is what
// resolution skips, the other is what pflag binds. This is the pair the example
// validator drifted from, so pin them to each other.
func TestHelpFlagNamesMatchWhatIsBound(t *testing.T) {
	for _, extended := range []bool{false, true} {
		app := &App{Name: "app", ExtendedHelpFlag: extended,
			Commands: []Command{{Name: "run", Run: nopRun}}}

		fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
		app.bindHelpFlags(fs, "app")

		bound := map[string]bool{}
		fs.VisitAll(func(f *pflag.Flag) {
			bound["--"+f.Name] = true
			if f.Shorthand != "" {
				bound["-"+f.Shorthand] = true
			}
		})

		named := map[string]bool{}
		for _, n := range app.helpFlagNames() {
			named[n] = true
			if !bound[n] {
				t.Errorf("ExtendedHelpFlag=%v: helpFlagNames lists %q, which bindHelpFlags does not bind", extended, n)
			}
		}
		for n := range bound {
			if !named[n] {
				t.Errorf("ExtendedHelpFlag=%v: bindHelpFlags binds %q, which helpFlagNames does not list", extended, n)
			}
		}
	}
}
