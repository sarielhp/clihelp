package clihelp

import (
	"strings"
	"testing"
)

// App.Shortcuts are top-level commands presented under their own heading. The
// 2026-09-17 review found that completion offered them and help listed them, but
// command resolution never consulted them, so running one was "unknown command"
// and asking for its help printed nothing.
func shortcutApp(ran *string) *App {
	return &App{
		Name: "app",
		Commands: []Command{
			{Name: "scan", Description: "Scan things", Run: func(*Context) error { *ran = "scan"; return nil }},
		},
		Shortcuts: []Command{
			{
				Name:        "quick",
				Aliases:     []string{"q"},
				Description: "Quick action",
				Options:     []Option{},
				Run:         func(ctx *Context) error { *ran = "quick:" + strings.Join(ctx.Args, ","); return nil },
			},
		},
	}
}

func TestShortcutCommandsRun(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"by name", []string{"quick"}, "quick:"},
		{"by alias", []string{"q"}, "quick:"},
		{"with arguments", []string{"quick", "now"}, "quick:now"},
		{"a normal command still runs", []string{"scan"}, "scan"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ran := ""
			res := TestExecute(shortcutApp(&ran), tt.args)
			res.AssertNoError(t)
			if ran != tt.want {
				t.Errorf("ran = %q, want %q", ran, tt.want)
			}
		})
	}
}

func TestShortcutCommandHelp(t *testing.T) {
	ran := ""
	app := shortcutApp(&ran)
	for _, args := range [][]string{{"help", "quick"}, {"quick", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := TestExecute(shortcutApp(&ran), args)
			res.AssertNoError(t)
			res.AssertStdoutContains(t, "Quick action")
		})
	}
	if cmd := app.LookupCommand("quick"); cmd == nil {
		t.Errorf("LookupCommand(%q) = nil, want the shortcut", "quick")
	}
}

func TestSuggestionsSkipHiddenCommands(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{Name: "secret", Description: "Hidden", Hidden: true, Run: func(*Context) error { return nil }},
			{Name: "status", Description: "Status", Run: func(*Context) error { return nil }},
		},
	}
	res := TestExecute(app, []string{"secref"})
	if res.Error == nil {
		t.Fatalf("expected an unknown-command error")
	}
	if strings.Contains(res.Error.Error(), "secret") {
		t.Errorf("a hidden command was named in a suggestion: %v", res.Error)
	}
}

// An empty argument is a prefix of every command name. The 2026-09-17 review
// found that "app ”" — which is what "app --filter $UNSET" expands to — ran the
// only command under AbbrevCommands, or was taken for a help request.
func TestEmptyArgumentIsNotACommandOrHelp(t *testing.T) {
	t.Run("does not run the only command", func(t *testing.T) {
		ran := false
		app := &App{
			Name:           "app",
			AbbrevCommands: true,
			Commands: []Command{
				{Name: "build", Description: "Build", Run: func(*Context) error { ran = true; return nil }},
			},
		}
		res := TestExecute(app, []string{""})
		if ran {
			t.Errorf(`app "" ran the only command`)
		}
		if res.Error == nil {
			t.Errorf(`app "" returned no error`)
		}
	})

	t.Run("reaches the app as an argument", func(t *testing.T) {
		var got []string
		app := &App{
			Name:           "app",
			AbbrevCommands: true,
			Run:            func(ctx *Context) error { got = ctx.Args; return nil },
		}
		res := TestExecute(app, []string{""})
		res.AssertNoError(t)
		if len(got) != 1 || got[0] != "" {
			t.Errorf("Run received %q, want one empty argument", got)
		}
	})
}

func TestCategoryCommandCompletesTheLifecycle(t *testing.T) {
	var events []string
	record := func(name string) func(*Context) error {
		return func(*Context) error { events = append(events, name); return nil }
	}
	app := &App{
		Name:      "app",
		BeforeRun: record("before"),
		AfterRun:  record("after"),
		Commands: []Command{{
			Name:        "db",
			Description: "Database commands",
			PreRun:      record("pre"),
			PostRun:     record("post"),
			Subcommands: []Command{
				{Name: "migrate", Description: "Migrate", Run: record("migrate")},
			},
		}},
	}
	res := TestExecute(app, []string{"db"})
	res.AssertNoError(t)
	if strings.Join(events, ",") != "before,pre,post,after" {
		t.Errorf("lifecycle events = %v, want before,pre,post,after", events)
	}
}
