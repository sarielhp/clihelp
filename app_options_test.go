package clihelp

import (
	"strings"
	"testing"
)

// App.Options are the root's own flags: bound when the application itself runs,
// and not inherited by its commands. Command has had this distinction between
// Options and PersistentOptions from the start; App had only the inherited kind.
//
// It blocks a real shape. An application whose root does the work — takes a
// directory and twenty-one flags — and whose subcommands reuse those flag names
// could not be expressed: as PersistentOptions the names collide, and Audit
// refuses the application.
func TestAppOptionsAreLocalToTheRoot(t *testing.T) {
	newApp := func(rootURL, cmdURL *string, rootRan, cmdRan *bool) *App {
		return &App{
			Name: "demo",
			Args: MinimumNArgs(0),
			Options: []Option{
				String(rootURL, "-u, --url <u>", "", "The root's own URL."),
			},
			Run: func(*Context) error { *rootRan = true; return nil },
			Commands: []Command{{
				Name: "fetch", Description: "Fetch it.", Args: NoArgs,
				// The same names, belonging to something else.
				Options: []Option{String(cmdURL, "-u, --url <u>", "", "The command's URL.")},
				Run:     func(*Context) error { *cmdRan = true; return nil },
			}},
		}
	}

	t.Run("the root binds its own", func(t *testing.T) {
		var rootURL, cmdURL string
		var rootRan, cmdRan bool
		app := newApp(&rootURL, &cmdURL, &rootRan, &cmdRan)
		silentApp(app)
		if err := app.Execute([]string{"--url", "root.example"}); err != nil {
			t.Fatal(err)
		}
		if rootURL != "root.example" || cmdURL != "" {
			t.Errorf("root=%q cmd=%q; the root's flag wrote the wrong target", rootURL, cmdURL)
		}
		if !rootRan {
			t.Error("the root handler did not run")
		}
	})

	t.Run("a command binds its own", func(t *testing.T) {
		var rootURL, cmdURL string
		var rootRan, cmdRan bool
		app := newApp(&rootURL, &cmdURL, &rootRan, &cmdRan)
		silentApp(app)
		if err := app.Execute([]string{"fetch", "--url", "cmd.example"}); err != nil {
			t.Fatal(err)
		}
		if cmdURL != "cmd.example" || rootURL != "" {
			t.Errorf("root=%q cmd=%q; the command's flag wrote the wrong target", rootURL, cmdURL)
		}
		if !cmdRan {
			t.Error("the command did not run")
		}
	})

	t.Run("the names may be reused, so Audit accepts it", func(t *testing.T) {
		var rootURL, cmdURL string
		var rootRan, cmdRan bool
		if err := Audit(newApp(&rootURL, &cmdURL, &rootRan, &cmdRan)); err != nil {
			t.Errorf("Audit refused an application whose root and command share a flag name: %v", err)
		}
	})

	t.Run("but not twice in the root's own scope", func(t *testing.T) {
		var a, b string
		app := &App{
			Name:              "demo",
			PersistentOptions: []Option{String(&a, "--url <u>", "", "Persistent.")},
			Options:           []Option{String(&b, "--url <u>", "", "The root's own.")},
			Run:               func(*Context) error { return nil },
		}
		if err := Audit(app); err == nil {
			t.Error("the same name was declared twice in the root's scope and accepted")
		}
	})
}

// A root flag written before a positional argument still takes its value — the
// arity probe runs before resolution, so it has to know the root's own flags or
// the value is read as the command name.
func TestAppOptionsAreKnownToTheArityProbe(t *testing.T) {
	var url string
	var got []string
	app := &App{
		Name:    "demo",
		Args:    MinimumNArgs(0),
		Options: []Option{String(&url, "-u, --url <u>", "", "URL.")},
		Run:     func(ctx *Context) error { got = ctx.Args; return nil },
		Commands: []Command{{
			Name: "fetch", Description: "Fetch it.", Run: func(*Context) error { return nil },
		}},
	}
	silentApp(app)
	if err := app.Execute([]string{"--url", "x.example", "somedir"}); err != nil {
		t.Fatal(err)
	}
	if url != "x.example" {
		t.Errorf("url = %q, want x.example", url)
	}
	if len(got) != 1 || got[0] != "somedir" {
		t.Errorf("positional arguments = %v, want [somedir]", got)
	}
}

// They appear on the root's help page and on no other.
func TestAppOptionsRenderOnTheRootOnly(t *testing.T) {
	var url string
	app := &App{
		Name:    "demo",
		Options: []Option{String(&url, "--root-only <u>", "", "Only at the root.")},
		Run:     func(*Context) error { return nil },
		Commands: []Command{{
			Name: "fetch", Description: "Fetch it.", Run: func(*Context) error { return nil },
		}},
	}
	out := silentApp(app)
	app.RenderGlobal(Options{Writer: out, Width: 100})
	if !strings.Contains(StripANSI(out.String()), "--root-only") {
		t.Errorf("the root's own flag is missing from its help:\n%s", out.String())
	}

	sub := silentApp(app)
	app.RenderCommand(Options{Writer: sub, Width: 100}, "fetch")
	if strings.Contains(StripANSI(sub.String()), "--root-only") {
		t.Errorf("the root's own flag appeared on a command's help:\n%s", sub.String())
	}
}
