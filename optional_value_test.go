package clihelp

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func optionalApp(move *string) *App {
	return &App{
		Name: "spamcli",
		Commands: []Command{{
			Name: "scan", Description: "Scan a label.", UsageLine: "spamcli scan <prefix>",
			Parameters: []Param{{Name: "<prefix>", Description: "Label prefix."}},
			Args:       ExactArgs(1),
			Options: []Option{
				Optional(String(move, "-m, --move [From]", "", "Move spam. Give a sender to move only theirs."), "true"),
			},
			Examples: []Example{
				{Line: "spamcli scan inbox"},
				{Line: "spamcli scan inbox -m"},
				{Line: "spamcli scan inbox --move=someone@example.com"},
			},
			Run: func(*Context) error { return nil },
		}},
	}
}

// The three states an optional value has to keep apart: absent, given bare, and
// given a value. Before this existed the only way to get them was to set
// Option.Binder by hand and name pflag.NoOptDefVal, which made pflag the
// application's own dependency.
func TestOptionalValueHasThreeStates(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"absent", []string{"scan", "inbox"}, ""},
		{"bare long", []string{"scan", "--move", "inbox"}, "true"},
		{"bare short", []string{"scan", "-m", "inbox"}, "true"},
		{"with a value", []string{"scan", "--move=x@y.z", "inbox"}, "x@y.z"},
		{"short with a value", []string{"scan", "-m=x@y.z", "inbox"}, "x@y.z"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var move string
			app := optionalApp(&move)
			silentApp(app)
			if err := app.Execute(tt.args); err != nil {
				t.Fatalf("%v: %v", tt.args, err)
			}
			if move != tt.want {
				t.Errorf("%v: move = %q, want %q", tt.args, move, tt.want)
			}
		})
	}
}

// The argument after a bare optional flag is a positional argument, not the
// flag's value. Getting this wrong is how a command name gets eaten and the
// program prints help and exits 0 — six instances of that were fixed in v0.3.23.
func TestABareOptionalFlagDoesNotEatTheNextArgument(t *testing.T) {
	var move string
	var got []string
	app := optionalApp(&move)
	app.Commands[0].Run = func(ctx *Context) error { got = ctx.Args; return nil }
	silentApp(app)

	if err := app.Execute([]string{"scan", "-m", "inbox"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "inbox" {
		t.Errorf("positional arguments = %v, want [inbox]; the flag consumed one", got)
	}

	// And the same seen through the arity table that command resolution uses.
	opt := Optional(String(&move, "-m, --move [From]", "", "x"), "true")
	if flagTakesValue(opt, parseFlagSpec(opt.Flags)) {
		t.Error("an optional-value flag is recorded as consuming the argument after it")
	}
}

// The flag written before the command name is the case resolution has to get
// right, because there the skipped argument is the command.
func TestABareOptionalFlagBeforeTheCommandName(t *testing.T) {
	var move string
	var ran bool
	app := &App{
		Name: "spamcli",
		PersistentOptions: []Option{
			Optional(String(&move, "-m, --move [From]", "", "Move spam."), "true"),
		},
		Commands: []Command{{
			Name: "scan", Description: "Scan.", Args: NoArgs,
			Run: func(*Context) error { ran = true; return nil },
		}},
	}
	silentApp(app)
	if err := app.Execute([]string{"-m", "scan"}); err != nil {
		t.Fatalf("`spamcli -m scan` failed: %v", err)
	}
	if !ran {
		t.Error("the command did not run: its name was taken as the flag's value")
	}
	if move != "true" {
		t.Errorf("move = %q, want %q", move, "true")
	}
}

// The spec string and the binding must agree, in both directions, or the help
// page promises something the flag does not do.
func TestOptionalAndItsSpecMustAgree(t *testing.T) {
	t.Run("brackets without Optional", func(t *testing.T) {
		var v string
		app := &App{Name: "a", Commands: []Command{{
			Name: "c", Description: "C.",
			Options: []Option{String(&v, "-x, --ex [V]", "", "x")},
			Run:     func(*Context) error { return nil },
		}}}
		err := Audit(app)
		if err == nil || !strings.Contains(err.Error(), "Optional") {
			t.Errorf("Audit accepted a bracketed placeholder with a mandatory value: %v", err)
		}
	})

	t.Run("Optional without brackets", func(t *testing.T) {
		var v string
		opt := Optional(String(&v, "-x, --ex <V>", "", "x"), "true")
		err := opt.Binder(pflag.NewFlagSet("t", pflag.ContinueOnError))
		if err == nil || !strings.Contains(err.Error(), "brackets") {
			t.Errorf("a mandatory-looking spec was bound as optional: %v", err)
		}
	})

	t.Run("Optional with no bare value", func(t *testing.T) {
		var v string
		opt := Optional(String(&v, "-x, --ex [V]", "", "x"), "")
		err := opt.Binder(pflag.NewFlagSet("t", pflag.ContinueOnError))
		if err == nil || !strings.Contains(err.Error(), "bare") {
			t.Errorf("an empty bare value was accepted: %v", err)
		}
	})
}

// Audit validates examples by binding a stand-in for every option, so the
// optional form has to survive that path too — otherwise an example that runs is
// reported as broken, which is the defect v0.3.24 fixed for the help flags.
func TestOptionalValueExamplesPassAudit(t *testing.T) {
	var move string
	if err := Audit(optionalApp(&move)); err != nil {
		t.Errorf("Audit rejected examples that run: %v", err)
	}
	if move != "" {
		t.Errorf("validating examples wrote %q through the author's own target", move)
	}
}

// The help shows the author's spelling, so the brackets are what the user reads.
func TestOptionalValueRendersItsBrackets(t *testing.T) {
	var move string
	app := optionalApp(&move)
	out := silentApp(app)
	app.RenderCommand(Options{Writer: out, Width: 100}, "scan")
	if body := StripANSI(out.String()); !strings.Contains(body, "--move [From]") {
		t.Errorf("the optional value is not shown as optional:\n%s", body)
	}
}
