package clihelp

import (
	"strings"
	"testing"
)

// M5 — isHelpToken ran at every depth, and its abbreviation guard
// (`no command starts with this prefix`) is trivially satisfied at a node with
// no subcommands — which is exactly a leaf command taking positional arguments.
// The word was unreachable, with no escape, because this runs before the "--"
// branch.
func TestLeafCommandReceivesTheWordHelp(t *testing.T) {
	for _, tt := range []struct {
		name   string
		abbrev bool
		arg    string
	}{
		{"the word itself, abbreviation off", false, "help"},
		{"the word itself, abbreviation on", true, "help"},
		{"a prefix, abbreviation on", true, "hel"},
		{"one letter, abbreviation on", true, "h"},
		{"one letter, abbreviation off", false, "h"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			app := &App{Name: "app", AbbrevCommands: tt.abbrev,
				Commands: []Command{{
					Name: "echo", Description: "Echo it.",
					Run: func(ctx *Context) error { got = ctx.Args; return nil },
				}}}
			silentApp(app)
			if err := app.Execute([]string{"echo", tt.arg}); err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0] != tt.arg {
				t.Errorf("`app echo %s` gave the command %v, want [%q]", tt.arg, got, tt.arg)
			}
		})
	}
}

// ...while `help` stays universal where it can be one: at a node that has
// subcommands, "app <group> help" is the git-like convention worth keeping.
func TestHelpStillWorksWhereItCanMeanHelp(t *testing.T) {
	app := &App{Name: "app", Commands: []Command{{
		Name: "remote", Description: "Manage remotes.",
		Subcommands: []Command{{Name: "add", Description: "Add one.", Run: nopRun}},
	}}}
	out := silentApp(app)
	if err := app.Execute([]string{"remote", "help"}); err != nil {
		t.Fatalf("`app remote help` failed: %v", err)
	}
	if !strings.Contains(StripANSI(out.String()), "add") {
		t.Errorf("`app remote help` did not render the group's help:\n%s", out.String())
	}
	out.Reset()
	if err := app.Execute([]string{"help"}); err != nil {
		t.Fatalf("`app help` failed: %v", err)
	}
	if out.Len() == 0 {
		t.Error("`app help` rendered nothing")
	}
}

// ...and an abbreviation that names a command is that command, not help.
func TestAbbreviationBeatsTheHelpPrefix(t *testing.T) {
	ran := false
	app := &App{Name: "app", AbbrevCommands: true, Commands: []Command{
		{Name: "hello", Description: "Greet.", Run: func(*Context) error { ran = true; return nil }},
	}}
	silentApp(app)
	if err := app.Execute([]string{"h"}); err != nil {
		t.Fatalf("`app h` failed: %v", err)
	}
	if !ran {
		t.Error("`app h` rendered help although `hello` is the only command it abbreviates")
	}
}

// M7 — helpPath took the raw tail, so a flag after `help` became part of the
// path. Flags before it worked; the order is the user's arbitrary choice.
func TestFlagsAfterHelpAreNotPartOfThePath(t *testing.T) {
	var verbose bool
	newApp := func() *App {
		return &App{Name: "app",
			GlobalFlags: []Option{Bool(&verbose, "--verbose, -v", false, "Be loud.")},
			Commands:    []Command{{Name: "deploy", Description: "Deploy it.", Run: nopRun}}}
	}
	for _, args := range [][]string{
		{"help", "deploy"},
		{"--verbose", "help", "deploy"},
		{"help", "--verbose", "deploy"},
		{"help", "--", "deploy"},
	} {
		app := newApp()
		out := silentApp(app)
		if err := app.Execute(args); err != nil {
			t.Errorf("`app %s` failed: %v", strings.Join(args, " "), err)
			continue
		}
		if !strings.Contains(StripANSI(out.String()), "Deploy it.") {
			t.Errorf("`app %s` did not render deploy's help:\n%s", strings.Join(args, " "), out.String())
		}
	}
}

// At the root, help and its abbreviations are always help — the application's
// own page and its topic list exist whether or not it has any commands.
func TestRootHelpWorksWithoutCommands(t *testing.T) {
	app := &App{Name: "mycli", Description: "A CLI tool", Version: "2.4.6",
		Run: func(*Context) error { return nil }}
	for _, arg := range []string{"help", "h"} {
		out := silentApp(app)
		if err := app.Execute([]string{arg}); err != nil {
			t.Errorf("`mycli %s` failed: %v", arg, err)
		}
		if out.Len() == 0 {
			t.Errorf("`mycli %s` rendered nothing", arg)
		}
	}
}

// A lone token at the root is kept as written, because -v and --version are
// themselves help topics.
func TestRootHelpTopicsSurvive(t *testing.T) {
	app := &App{Name: "app", Version: "1.0",
		Commands: []Command{{Name: "deploy", Description: "D.", Run: nopRun}}}
	for _, topic := range []string{"-v", "--version", "flags"} {
		out := silentApp(app)
		if err := app.Execute([]string{"help", topic}); err != nil {
			t.Errorf("`app help %s` failed: %v", topic, err)
		}
		if out.Len() == 0 {
			t.Errorf("`app help %s` rendered nothing", topic)
		}
	}
}

// L1 — an empty topic matched "flags" by prefix under AbbrevCommands, so the
// behaviour flipped on an unrelated switch.
func TestEmptyHelpTopicIsAnError(t *testing.T) {
	for _, abbrev := range []bool{false, true} {
		app := &App{Name: "app", AbbrevCommands: abbrev,
			Commands: []Command{{Name: "deploy", Description: "D.", Run: nopRun}}}
		silentApp(app)
		if err := app.Execute([]string{"help", ""}); err == nil {
			t.Errorf("AbbrevCommands=%v: `app help \"\"` rendered something instead of erroring", abbrev)
		}
	}
}
