package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

func silentApp(app *App) *bytes.Buffer {
	var b bytes.Buffer
	app.Stdout, app.Stderr = &b, &b
	return &b
}

// M1 — an ambiguity among real commands used to be discarded whenever a shortcut
// also matched the prefix, so the user who typed "d" meaning "deploy" ran a
// third, unrelated command with no warning at all.
func TestShortcutDoesNotBreakACommandAmbiguity(t *testing.T) {
	ran := ""
	mk := func(name string) func(*Context) error {
		return func(*Context) error { ran = name; return nil }
	}
	app := &App{
		Name: "app", AbbrevCommands: true,
		Commands: []Command{
			{Name: "deploy", Description: "D.", Run: mk("deploy")},
			{Name: "destroy", Description: "D.", Run: mk("destroy")},
		},
		Shortcuts: []Command{{Name: "dance", Description: "Dance.", Run: mk("dance")}},
	}
	silentApp(app)

	err := app.Execute([]string{"d"})
	if err == nil {
		t.Fatalf("`app d` is ambiguous between deploy and destroy but ran %q", ran)
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected an ambiguity error, got: %v", err)
	}
	if ran != "" {
		t.Errorf("something ran anyway: %q", ran)
	}
}

// M2 — lookupCommandPath and resolveCommandPath ordered exact-match,
// abbreviation and shortcut differently, so the help a user reads described a
// different command from the one that runs.
func TestHelpAndExecutionResolveTheSameName(t *testing.T) {
	for _, tt := range []struct {
		name      string
		commands  []Command
		shortcuts []Command
		arg       string
		want      string
	}{
		{"an exact shortcut beats an abbreviated command", []Command{{Name: "deploy"}},
			[]Command{{Name: "dep"}}, "dep", "dep"},
		{"an exact command beats an exact shortcut", []Command{{Name: "run"}},
			[]Command{{Name: "run"}}, "run", "run"},
		{"an abbreviated command beats an abbreviated shortcut", []Command{{Name: "deploy"}},
			[]Command{{Name: "depart"}}, "deplo", "deploy"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mark := func(cmds []Command, into *string, label string) []Command {
				out := make([]Command, len(cmds))
				for i, c := range cmds {
					c.Description = label
					name := c.Name
					c.Run = func(*Context) error { *into = name; return nil }
					out[i] = c
				}
				return out
			}
			var ran string
			app := &App{Name: "app", AbbrevCommands: true}
			app.Commands = mark(tt.commands, &ran, "from commands")
			app.Shortcuts = mark(tt.shortcuts, &ran, "from shortcuts")
			silentApp(app)

			if err := app.Execute([]string{tt.arg}); err != nil {
				t.Fatalf("running %q failed: %v", tt.arg, err)
			}
			if ran != tt.want {
				t.Errorf("`app %s` ran %q, want %q", tt.arg, ran, tt.want)
			}

			// And the help path has to agree about the same name.
			cmd, path := app.lookupCommandPath([]string{tt.arg})
			if cmd == nil {
				t.Fatalf("`app help %s` resolved nothing while running it resolved %q", tt.arg, ran)
			}
			if cmd.Name != tt.want || len(path) != 1 || path[0] != tt.want {
				t.Errorf("`app help %s` resolved %q (path %v) while running it resolved %q",
					tt.arg, cmd.Name, path, ran)
			}
		})
	}
}

// The help path must refuse an ambiguity for the same reason execution does.
func TestHelpRefusesAnAmbiguousAbbreviation(t *testing.T) {
	app := &App{Name: "app", AbbrevCommands: true,
		Commands:  []Command{{Name: "deploy"}, {Name: "destroy"}},
		Shortcuts: []Command{{Name: "dance"}},
	}
	if cmd, _ := app.lookupCommandPath([]string{"d"}); cmd != nil {
		t.Errorf("`app help d` resolved %q for an ambiguous prefix", cmd.Name)
	}
}
