package clihelp

import (
	"strings"
	"testing"
)

func nopRun(*Context) error { return nil }

// M3 — checkUnknownCommand's gate is "this level has commands", and at the root
// that meant App.Commands only. An application that puts its verbs in Shortcuts
// therefore never ran the check: any typo printed the global help and exited 0,
// so a calling script saw success.
func TestShortcutOnlyAppRejectsATypo(t *testing.T) {
	app := &App{Name: "app", Shortcuts: []Command{
		{Name: "go", Description: "Go.", Run: nopRun},
	}}
	silentApp(app)

	if err := app.Execute([]string{"typo"}); err == nil {
		t.Error("`app typo` exited 0 instead of reporting an unknown command")
	}
	// And the suggestion has to see shortcuts too.
	err := app.Execute([]string{"gp"})
	if err == nil || !strings.Contains(err.Error(), `"go"`) {
		t.Errorf("`app gp` did not suggest the shortcut it mistypes: %v", err)
	}
	// The shortcut itself still runs.
	if err := app.Execute([]string{"go"}); err != nil {
		t.Errorf("the shortcut stopped working: %v", err)
	}
}

// M8 — the unknown-command check lived only on the path where a command failed
// to match. When the loop exited through the flag branch instead — "--" is never
// a known flag, nor is an unrecognised one — the next word was never examined.
func TestDashDashDoesNotExcuseATypo(t *testing.T) {
	app := &App{Name: "app", Commands: []Command{{Name: "deploy", Description: "D.", Run: nopRun}}}
	silentApp(app)

	plain := app.Execute([]string{"junk"})
	if plain == nil {
		t.Fatal("the fixture is wrong: `app junk` should error")
	}
	if dashed := app.Execute([]string{"--", "junk"}); dashed == nil {
		t.Errorf("`app junk` errors but `app -- junk` exits 0")
	}
}

// M4 — a command's own (non-persistent) options are kept out of the arity probe
// because they cannot precede their own command. True, but the loop kept using
// that probe for flags written AFTER the command name, where they can — so the
// subcommand became a positional the grouping command cannot use, and the page
// exited 0 having done nothing.
func TestGroupingCommandReportsALeftoverSubcommand(t *testing.T) {
	var level string
	ran := false
	newApp := func() *App {
		level, ran = "", false
		return &App{Name: "app", Commands: []Command{{
			Name: "remote", Description: "Manage remotes.",
			Options: []Option{String(&level, "--level <n>", "", "A level.")},
			Subcommands: []Command{{
				Name: "add", Description: "Add one.",
				Run: func(*Context) error { ran = true; return nil },
			}},
		}}}
	}

	app := newApp()
	silentApp(app)
	err := app.Execute([]string{"remote", "--level", "2", "add"})
	if err == nil && !ran {
		t.Errorf("`app remote --level 2 add` neither ran the subcommand nor reported anything")
	}

	// A bare grouping command still renders its help, and the subcommand still
	// runs — the guard must not fire on either.
	app = newApp()
	silentApp(app)
	if err := app.Execute([]string{"remote"}); err != nil {
		t.Errorf("`app remote` should render help, got: %v", err)
	}
	app = newApp()
	silentApp(app)
	if err := app.Execute([]string{"remote", "add"}); err != nil || !ran {
		t.Errorf("`app remote add` stopped working: err=%v ran=%v", err, ran)
	}
}

// An app with its own Run still receives its positional arguments.
func TestRootRunStillReceivesPositionals(t *testing.T) {
	var got []string
	app := &App{Name: "app", Run: func(ctx *Context) error { got = ctx.Args; return nil },
		Commands: []Command{{Name: "sub", Run: nopRun}}}
	silentApp(app)
	if err := app.Execute([]string{"--", "anything"}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "anything" {
		t.Errorf("positional arguments were rejected: %v", got)
	}
}
