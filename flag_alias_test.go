package clihelp

import (
	"strings"
	"testing"
)

// Chain B of the 2026-09-17 review: every spelling of an option has to behave
// as the same option. These tests drive the app through its exported API, which
// is the only place the defects were visible.

func aliasApp(run func(*Context) error, opts ...Option) *App {
	return &App{
		Name: "aliasapp",
		Commands: []Command{{
			Name:        "run",
			Description: "Run it",
			Options:     opts,
			Run:         run,
		}},
	}
}

func TestRequiredOptionAcceptsEveryAlias(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"primary long", []string{"run", "--name", "x"}},
		{"alias long", []string{"run", "--title", "x"}},
		{"primary short", []string{"run", "-n", "x"}},
		{"alias short", []string{"run", "-t", "x"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			app := aliasApp(nil, Required(String(&got, "--name, --title, -n, -t <s>", "", "name")))
			res := TestExecute(app, tt.args)
			res.AssertNoError(t)
			if got != "x" {
				t.Errorf("target = %q, want %q", got, "x")
			}
		})
	}
}

func TestRequiredToggleAcceptsNegativeSpelling(t *testing.T) {
	var color bool
	app := aliasApp(nil, Required(BoolToggle(&color, "--[no-]color", true, "colorize")))
	res := TestExecute(app, []string{"run", "--no-color"})
	res.AssertNoError(t)
	if color {
		t.Errorf("--no-color left color = true")
	}
}

func TestStringSliceAccumulatesAcrossAliases(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want []string
	}{
		{"long then short alias", []string{"run", "--tag", "a", "--tag", "b", "-T", "c"}, []string{"a", "b", "c"}},
		{"long alias", []string{"run", "--tag", "a", "--tags2", "b"}, []string{"a", "b"}},
		{"primary only", []string{"run", "--tag", "a", "--tag", "b"}, []string{"a", "b"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var tags []string
			app := aliasApp(nil, StringSlice(&tags, "--tag, --tags2, -t, -T <tag>", nil, "tags"))
			res := TestExecute(app, tt.args)
			res.AssertNoError(t)
			if strings.Join(tags, ",") != strings.Join(tt.want, ",") {
				t.Errorf("tags = %v, want %v", tags, tt.want)
			}
		})
	}
}

func TestRelationValidatorsResolveShorthands(t *testing.T) {
	var alpha, beta string
	makeApp := func(validator OptionsValidator) *App {
		app := aliasApp(nil,
			String(&alpha, "--alpha, -a <s>", "", "alpha"),
			String(&beta, "--beta, -b <s>", "", "beta"),
		)
		app.Commands[0].OptionsValidator = validator
		return app
	}

	for _, tt := range []struct {
		name      string
		validator OptionsValidator
		args      []string
		wantErr   string
	}{
		{"shorthand names", MutuallyExclusive("-a", "-b"), []string{"run", "-a", "1", "-b", "2"}, "mutually exclusive"},
		{"long names", MutuallyExclusive("--alpha", "--beta"), []string{"run", "-a", "1", "-b", "2"}, "mutually exclusive"},
		{"mixed names", MutuallyExclusive("--alpha", "-b"), []string{"run", "--alpha", "1", "--beta", "2"}, "mutually exclusive"},
		{"required with", RequiredWith("-a", "-b"), []string{"run", "-a", "1"}, "is required when using"},
		{"required if value", RequiredIf("--beta", "--alpha=1"), []string{"run", "-a", "1"}, "is required when"},
		{"satisfied", MutuallyExclusive("-a", "-b"), []string{"run", "-a", "1"}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := TestExecute(makeApp(tt.validator), tt.args)
			if tt.wantErr == "" {
				res.AssertNoError(t)
				return
			}
			res.AssertErrorContains(t, tt.wantErr)
		})
	}
}

func TestRelationValidatorSeesAliasOfTheSameOption(t *testing.T) {
	var alpha, beta string
	app := aliasApp(nil,
		String(&alpha, "--alpha, --first, -a <s>", "", "alpha"),
		String(&beta, "--beta, -b <s>", "", "beta"),
	)
	app.Commands[0].OptionsValidator = MutuallyExclusive("--alpha", "--beta")
	res := TestExecute(app, []string{"run", "--first", "1", "-b", "2"})
	res.AssertErrorContains(t, "mutually exclusive")
}

func TestRelationValidatorReportsUndeclaredFlag(t *testing.T) {
	var alpha string
	app := aliasApp(nil, String(&alpha, "--alpha, -a <s>", "", "alpha"))
	app.Commands[0].OptionsValidator = MutuallyExclusive("--alpha", "--nonesuch")
	res := TestExecute(app, []string{"run", "--alpha", "1"})
	res.AssertErrorContains(t, "--nonesuch")
}

func deprecatedOption(opt Option, notice string) Option {
	opt.Deprecated = notice
	return opt
}

func TestDeprecationWarnsForShortOnlyAndAliasedOptions(t *testing.T) {
	for _, tt := range []struct {
		name string
		spec string
		args []string
	}{
		{"short only, primary", "-a, -b", []string{"run", "-a"}},
		{"short only, alias", "-a, -b", []string{"run", "-b"}},
		{"long alias", "--alpha, --first, -a", []string{"run", "--first"}},
		{"short alias", "--alpha, -a, -A", []string{"run", "-A"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var flag bool
			app := aliasApp(func(*Context) error { return nil },
				deprecatedOption(Bool(&flag, tt.spec, false, "old flag"), "use --new instead"))
			res := TestExecute(app, tt.args)
			res.AssertNoError(t)
			res.AssertStderrContains(t, "use --new instead")
			if !flag {
				t.Errorf("%s did not set the target", strings.Join(tt.args, " "))
			}
		})
	}
}

func TestDeprecationWarnsOnceWhenSeveralAliasesAreUsed(t *testing.T) {
	var flag bool
	app := aliasApp(func(*Context) error { return nil },
		deprecatedOption(Bool(&flag, "--alpha, -a, -A", false, "old flag"), "use --new instead"))
	res := TestExecute(app, []string{"run", "--alpha", "-a", "-A"})
	res.AssertNoError(t)
	if n := strings.Count(res.Stderr, "use --new instead"); n != 1 {
		t.Errorf("deprecation warned %d times, want 1:\n%s", n, res.Stderr)
	}
}

// TestDocumentedMultiAliasSpec exercises the spec advertised in
// docs/flags-and-options.md, which is the shape none of the library's own
// examples used and every chain B defect needed.
func TestDocumentedMultiAliasSpec(t *testing.T) {
	for _, name := range []string{"-p", "-P", "--port", "--listen-port"} {
		t.Run(name, func(t *testing.T) {
			var port int
			app := aliasApp(nil, Required(Int(&port, "-p, -P, --port, --listen-port <num>", 0, "listen port")))
			res := TestExecute(app, []string{"run", name, "8080"})
			res.AssertNoError(t)
			if port != 8080 {
				t.Errorf("%s left port = %d, want 8080", name, port)
			}
		})
	}
}
