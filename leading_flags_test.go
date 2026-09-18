package clihelp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// leadingState collects the flag targets and the command that ran, so that a
// test can assert both that the right command was resolved and that its flags
// were bound.
type leadingState struct {
	account  string
	readOnly bool
	verbose  bool
	number   int
	dbURL    string
	ran      string
	args     []string
}

// leadingTestApp builds an app covering every shape a leading flag can take:
// global flags that take a value and global flags that do not, a command with
// its own flags, and a nested command carrying a persistent flag.
func leadingTestApp(st *leadingState) *App {
	record := func(name string) func(ctx *Context) error {
		return func(ctx *Context) error {
			st.ran = name
			st.args = ctx.Args
			return nil
		}
	}
	return &App{
		Name: "mail",
		PersistentOptions: []Option{
			String(&st.account, "-A, --account <name>", "", "Account to use"),
			Bool(&st.readOnly, "--read-only", false, "Make no changes"),
		},
		GlobalFlags: []Option{
			Bool(&st.verbose, "-v, --verbose", false, "Verbose output"),
		},
		Run: record("root"),
		Commands: []Command{
			{
				Name:            "search",
				Description:     "Search the mailbox",
				LongDescription: "Full text search over every stored message.",
				Options:         []Option{Int(&st.number, "-n, --number <n>", 0, "How many to show")},
				Run:             record("search"),
			},
			{Name: "last", Description: "Show the last messages", Run: record("last")},
			{
				Name:              "db",
				Description:       "Database commands",
				PersistentOptions: []Option{String(&st.dbURL, "--db-url <url>", "", "Database URL")},
				Subcommands: []Command{
					{Name: "migrate", Description: "Run migrations", Run: record("migrate")},
				},
			},
		},
	}
}

func TestLeadingGlobalFlagsResolveCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantRan  string
		wantArgs []string
		check    func(t *testing.T, st *leadingState)
	}{
		{
			name:     "global bool before command",
			args:     []string{"--read-only", "search", "invoice"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if !st.readOnly {
					t.Error("--read-only was not bound")
				}
			},
		},
		{
			name:     "global bool before a command that has its own flags",
			args:     []string{"--read-only", "search", "invoice", "-n", "2"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if !st.readOnly || st.number != 2 {
					t.Errorf("readOnly = %v, number = %d, want true, 2", st.readOnly, st.number)
				}
			},
		},
		{
			name:     "global flag with a separate value",
			args:     []string{"-A", "fastmail", "last", "2"},
			wantRan:  "last",
			wantArgs: []string{"2"},
			check: func(t *testing.T, st *leadingState) {
				if st.account != "fastmail" {
					t.Errorf("account = %q, want %q", st.account, "fastmail")
				}
			},
		},
		{
			name:     "global flag with an inline value",
			args:     []string{"--account=fastmail", "search", "invoice"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if st.account != "fastmail" {
					t.Errorf("account = %q, want %q", st.account, "fastmail")
				}
			},
		},
		{
			name:     "a flag value that collides with a command name",
			args:     []string{"-A", "last", "search", "invoice"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if st.account != "last" {
					t.Errorf("account = %q, want %q", st.account, "last")
				}
			},
		},
		{
			name:     "shorthand cluster ending in a flag that takes a value",
			args:     []string{"-vA", "fastmail", "search", "invoice"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if !st.verbose || st.account != "fastmail" {
					t.Errorf("verbose = %v, account = %q, want true, %q", st.verbose, st.account, "fastmail")
				}
			},
		},
		{
			name:     "shorthand with an attached value",
			args:     []string{"-Afastmail", "search", "invoice"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if st.account != "fastmail" {
					t.Errorf("account = %q, want %q", st.account, "fastmail")
				}
			},
		},
		{
			name:     "several globals before a command with a long flag",
			args:     []string{"-v", "--read-only", "-A", "work", "search", "invoice", "--number", "3"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if !st.verbose || !st.readOnly || st.account != "work" || st.number != 3 {
					t.Errorf("verbose = %v, readOnly = %v, account = %q, number = %d",
						st.verbose, st.readOnly, st.account, st.number)
				}
			},
		},
		{
			name:     "persistent flag of a parent before its subcommand",
			args:     []string{"--read-only", "db", "--db-url", "pg://x", "migrate"},
			wantRan:  "migrate",
			wantArgs: nil,
			check: func(t *testing.T, st *leadingState) {
				if !st.readOnly || st.dbURL != "pg://x" {
					t.Errorf("readOnly = %v, dbURL = %q", st.readOnly, st.dbURL)
				}
			},
		},
		{
			name:     "flags after the command still work",
			args:     []string{"search", "invoice", "-n", "2", "--read-only"},
			wantRan:  "search",
			wantArgs: []string{"invoice"},
			check: func(t *testing.T, st *leadingState) {
				if !st.readOnly || st.number != 2 {
					t.Errorf("readOnly = %v, number = %d, want true, 2", st.readOnly, st.number)
				}
			},
		},
		{
			name:     "a double dash makes the rest positional",
			args:     []string{"--", "search", "invoice"},
			wantRan:  "root",
			wantArgs: []string{"search", "invoice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st leadingState
			res := TestExecute(leadingTestApp(&st), tt.args)
			res.AssertNoError(t)
			if st.ran != tt.wantRan {
				t.Fatalf("ran %q, want %q (stdout: %q)", st.ran, tt.wantRan, res.Stdout)
			}
			if strings.Join(st.args, " ") != strings.Join(tt.wantArgs, " ") {
				t.Errorf("args = %q, want %q", st.args, tt.wantArgs)
			}
			if tt.check != nil {
				tt.check(t, &st)
			}
		})
	}
}

func TestLeadingUnknownFlagsStillFail(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		errSub string
	}{
		{"unknown long flag", []string{"--bogus", "search"}, "unknown flag: --bogus"},
		{"unknown shorthand", []string{"-z", "search"}, "unknown shorthand flag"},
		{"a command flag cannot precede its command", []string{"-n", "2", "search", "invoice"}, "unknown shorthand flag"},
		{"a subcommand flag cannot precede its subcommand", []string{"--db-url", "pg://x", "db", "migrate"}, "unknown flag: --db-url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st leadingState
			res := TestExecute(leadingTestApp(&st), tt.args)
			res.AssertErrorContains(t, tt.errSub)
			if st.ran != "" {
				t.Errorf("ran %q, want nothing to run", st.ran)
			}
		})
	}
}

func TestLeadingFlagsBeforeHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"extended help after a global", []string{"--read-only", "search", "--help"}},
		{"extended help before the command", []string{"--help", "search"}},
		{"extended help after a global and before the command", []string{"--read-only", "--help", "search"}},
		{"concise help before the command", []string{"-h", "search"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st leadingState
			res := TestExecute(leadingTestApp(&st), tt.args)
			res.AssertNoError(t)
			if st.ran != "" {
				t.Errorf("ran %q, want help instead", st.ran)
			}
			// Command help names the command's own flag and says nothing about
			// its siblings, which only the global listing mentions.
			res.AssertStdoutContains(t, "--number")
			if strings.Contains(res.Stdout, "Show the last messages") {
				t.Errorf("expected help for search alone, got the global listing:\n%s", res.Stdout)
			}
		})
	}
}

func TestLeadingFlagsInCompletion(t *testing.T) {
	t.Run("subcommands complete after a global flag", func(t *testing.T) {
		var st leadingState
		app := leadingTestApp(&st)
		res := TestExecute(app, []string{"__complete", "--read-only", "db", ""})
		res.AssertNoError(t)
		res.AssertStdoutContains(t, "migrate")
		if strings.Contains(res.Stdout, "search") {
			t.Errorf("expected only db's subcommands, got:\n%s", res.Stdout)
		}
	})

	t.Run("command flags complete after a global flag with a value", func(t *testing.T) {
		var st leadingState
		app := leadingTestApp(&st)
		res := TestExecute(app, []string{"__complete", "-A", "work", "search", "-"})
		res.AssertNoError(t)
		res.AssertStdoutContains(t, "--number")
	})
}

func TestLeadingFlagsInExampleColorization(t *testing.T) {
	var st leadingState
	app := leadingTestApp(&st)

	seg := "mail -A work search invoice"
	toks := extractSegmentTokens(seg)
	raw := make([]string, len(toks))
	for i, tok := range toks {
		raw[i] = tok.text
	}
	isCmd, isFlag := identifySegmentTokens(app, nil, toks, raw)

	wantCmd := []bool{true, false, false, true, false} // mail, -A, work, search, invoice
	for i := range toks {
		if isCmd[i] != wantCmd[i] {
			t.Errorf("token %d (%q): isCmd = %v, want %v", i, raw[i], isCmd[i], wantCmd[i])
		}
	}
	if !isFlag[1] {
		t.Errorf("token %q should be marked as a flag", raw[1])
	}
}

func TestLeadingFlagsInExampleValidation(t *testing.T) {
	var st leadingState
	app := leadingTestApp(&st)
	app.Examples = []Example{
		{Line: "mail -A work search invoice -n 2"},
		{Line: "mail --read-only db --db-url pg://x migrate"},
	}
	if errs := app.validateExamples(); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
}

func TestScanLeadingFlag(t *testing.T) {
	var (
		account string
		verbose bool
		color   bool
		num     int
		timeout time.Duration
		tags    []string
		mode    string
	)
	app := &App{
		Name: "x",
		PersistentOptions: []Option{
			String(&account, "-A, --account <name>", "", "Account"),
			Bool(&verbose, "-v, --verbose", false, "Verbose"),
			BoolToggle(&color, "--[no-]color", true, "Color"),
			Int(&num, "-N, --num <n>", 0, "Number"),
			Duration(&timeout, "--timeout <dur>", 0, "Timeout"),
			StringSlice(&tags, "-t, --tag <tag>", nil, "Tags"),
			Enum(&mode, "--mode <mode>", []string{"fast", "slow"}, "fast", "Mode"),
		},
	}
	arity := app.leadingFlagArity(nil)

	tests := []struct {
		args      []string
		wantCount int
		wantKnown bool
	}{
		{[]string{"--account", "x", "cmd"}, 2, true},
		{[]string{"--account=x", "cmd"}, 1, true},
		{[]string{"-A", "x", "cmd"}, 2, true},
		{[]string{"-Ax", "cmd"}, 1, true},
		{[]string{"-A=x", "cmd"}, 1, true},
		{[]string{"--verbose", "cmd"}, 1, true},
		{[]string{"-v", "cmd"}, 1, true},
		{[]string{"-vA", "x", "cmd"}, 2, true},
		{[]string{"-vAx", "cmd"}, 1, true},
		{[]string{"--color", "cmd"}, 1, true},
		{[]string{"--no-color", "cmd"}, 1, true},
		{[]string{"--num", "2", "cmd"}, 2, true},
		{[]string{"--timeout", "3s", "cmd"}, 2, true},
		{[]string{"--tag", "a,b", "cmd"}, 2, true},
		{[]string{"--mode", "slow", "cmd"}, 2, true},
		{[]string{"-h", "cmd"}, 1, true},
		{[]string{"--help", "cmd"}, 1, true},
		{[]string{"-A"}, 1, true}, // missing value; pflag reports it
		{[]string{"--bogus", "cmd"}, 0, false},
		{[]string{"-z", "cmd"}, 0, false},
		{[]string{"-vz", "cmd"}, 0, false},
		{[]string{"--", "cmd"}, 0, false}, // ends flag scanning
		{[]string{"-", "cmd"}, 0, false},  // a positional argument
	}

	for _, tt := range tests {
		count, known := scanLeadingFlag(arity, tt.args)
		if count != tt.wantCount || known != tt.wantKnown {
			t.Errorf("scanLeadingFlag(%q) = (%d, %v), want (%d, %v)",
				tt.args, count, known, tt.wantCount, tt.wantKnown)
		}
	}
}

func TestLeadingFlagsResolutionIndices(t *testing.T) {
	var st leadingState
	app := leadingTestApp(&st)

	res, err := app.resolveCommand([]string{"-A", "work", "db", "--db-url", "pg://x", "migrate", "rest"})
	if err != nil {
		t.Fatalf("resolveCommand: %v", err)
	}
	if got := strings.Join(res.path, " "); got != "db migrate" {
		t.Errorf("path = %q, want %q", got, "db migrate")
	}
	if len(res.indices) != 2 || res.indices[0] != 2 || res.indices[1] != 5 {
		t.Errorf("indices = %v, want [2 5]", res.indices)
	}
	want := "-A work --db-url pg://x rest"
	if got := strings.Join(res.remaining, " "); got != want {
		t.Errorf("remaining = %q, want %q", got, want)
	}
}

func TestLeadingFlagsUnknownCommandAfterGlobal(t *testing.T) {
	var st leadingState
	app := leadingTestApp(&st)
	app.Run = nil // without a root Run, an unknown first word is an error

	res := TestExecute(app, []string{"--read-only", "bogus"})
	res.AssertErrorContains(t, `unknown command "bogus"`)

	// The same invocation still resolves when the word names a command.
	st = leadingState{}
	app2 := leadingTestApp(&st)
	app2.Run = nil
	if err := app2.ExecuteContext(context.Background(), []string{"--read-only", "last"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.ran != "last" {
		t.Errorf("ran %q, want %q", st.ran, "last")
	}
}

// Resolution must not touch the consumer's option variables. Deriving flag arity
// by binding the real options wrote every declared default through the caller's
// pointers, so merely resolving arguments — or rendering help, which resolves
// each example line — reset a running program's state.
func TestResolutionDoesNotBindOptions(t *testing.T) {
	newApp := func(st *leadingState) *App {
		app := leadingTestApp(st)
		app.Commands[0].Examples = []Example{{Line: "mail search invoice -n 2"}}
		return app
	}
	live := func(st *leadingState) {
		st.account, st.number, st.verbose = "USERVALUE", 99, true
	}
	assertLive := func(t *testing.T, st *leadingState, what string) {
		t.Helper()
		if st.account != "USERVALUE" || st.number != 99 || !st.verbose {
			t.Errorf("%s overwrote live option values: account=%q number=%d verbose=%v",
				what, st.account, st.number, st.verbose)
		}
	}

	t.Run("resolveCommand", func(t *testing.T) {
		var st leadingState
		app := newApp(&st)
		live(&st)
		res, err := app.resolveCommand([]string{"-v", "-A", "work", "search"})
		if err != nil {
			t.Fatalf("resolveCommand: %v", err)
		}
		if strings.Join(res.path, " ") != "search" {
			t.Errorf("path = %v, want [search]", res.path)
		}
		assertLive(t, &st, "resolveCommand")
	})

	t.Run("RenderCommand", func(t *testing.T) {
		var st leadingState
		app := newApp(&st)
		var sink bytes.Buffer
		app.Stdout = &sink
		live(&st)
		app.RenderCommand(Options{Writer: &sink, Width: 80}, "search")
		assertLive(t, &st, "RenderCommand")
	})

	t.Run("completion", func(t *testing.T) {
		var st leadingState
		app := newApp(&st)
		live(&st)
		TestExecute(app, []string{"__complete", "search", ""}).AssertNoError(t)
		assertLive(t, &st, "__complete")
	})
}

// TestResolutionMutatesNothing is the purity test the file named for it never
// contained.
//
// It covers the entry points TestResolutionDoesNotBindOptions does not —
// ValidateExamples resolves twice per example line, and the render paths resolve
// too — and it checks the command tree as well as the caller's variables,
// because filterCommandsByPrefix hands out *Command pointers into the caller's
// own slice and nothing stopped a future edit writing through them.
func TestResolutionMutatesNothing(t *testing.T) {
	for _, entry := range []struct {
		name string
		run  func(*App)
	}{
		{"resolveCommand", func(a *App) { _, _ = a.resolveCommand([]string{"-v", "-A", "work", "search"}) }},
		{"ValidateExamples", func(a *App) { _ = a.validateExamples() }},
		{"RenderGlobal", func(a *App) { a.RenderGlobal(Options{Writer: io.Discard, Width: 80}) }},
		{"RenderCommand", func(a *App) { a.RenderCommand(Options{Writer: io.Discard, Width: 80}, "search") }},
		{"RenderMan", func(a *App) { a.RenderMan(Options{Writer: io.Discard, Width: 80}) }},
		{"__complete", func(a *App) { TestExecute(a, []string{"__complete", "search", ""}) }},
		{"__explain", func(a *App) { TestExecute(a, []string{"__explain", "mail search inv"}) }},
	} {
		t.Run(entry.name, func(t *testing.T) {
			var st leadingState
			app := leadingTestApp(&st)
			app.Commands[0].Examples = []Example{{Line: "mail search invoice -n 2"}}
			st = leadingState{account: "USERVALUE", number: 99, verbose: true}
			want := st
			before := snapshotCommandTree(app)

			entry.run(app)

			if st.account != want.account || st.number != want.number || st.verbose != want.verbose ||
				st.readOnly != want.readOnly || st.dbURL != want.dbURL {
				t.Errorf("%s mutated the caller's option variables:\n got %+v\nwant %+v", entry.name, st, want)
			}
			if after := snapshotCommandTree(app); after != before {
				t.Errorf("%s mutated the command tree:\n got %s\nwant %s", entry.name, after, before)
			}
		})
	}
}

// snapshotCommandTree renders every field resolution reads into one string, so a
// write through a returned *Command shows up as a difference.
func snapshotCommandTree(a *App) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app=%q run=%v shortcuts=%d;", a.Name, a.Run != nil, len(a.Shortcuts))
	_ = a.Walk(func(path []string, cmd *Command) error {
		fmt.Fprintf(&b, "[%s name=%q aliases=%v hidden=%v run=%v subs=%d",
			strings.Join(path, " "), cmd.Name, cmd.Aliases, cmd.Hidden, cmd.Run != nil, len(cmd.Subcommands))
		for _, o := range cmd.Options {
			fmt.Fprintf(&b, " opt=%q", o.Flags)
		}
		for _, o := range cmd.PersistentOptions {
			fmt.Fprintf(&b, " popt=%q", o.Flags)
		}
		b.WriteString("]")
		return nil
	})
	return b.String()
}
