package clihelp

import (
	"fmt"
	"strings"
	"testing"
)

func TestAuditShortDescriptionRows(t *testing.T) {
	long := strings.Repeat("word ", 20)
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{"fits", &App{Name: "a", Commands: []Command{{Name: "build", Description: "Build the thing"}}}, ""},
		{"too long", &App{Name: "a", Commands: []Command{{Name: "build", Description: long}}}, "columns"},
		{"multi-line", &App{Name: "a", Commands: []Command{{Name: "build", Description: "one\ntwo"}}}, "one line"},
		{"first sentence only", &App{Name: "a", Commands: []Command{{Name: "build", Description: "Build it. " + long}}}, ""},
		{"markup not counted", &App{Name: "a", Commands: []Command{{Name: "build", Description: "[" + strings.Repeat("x", 30) + "](https://example.com/a/very/long/url/that/is/not/shown)"}}}, ""},
		{"subcommand too long", &App{Name: "a", Commands: []Command{{Name: "g", Description: "Group", Subcommands: []Command{{Name: "s", Description: long}}}}}, "g subcommands"},
		{"flag too long", &App{Name: "a", GlobalFlags: []Option{{Flags: "--x", Description: long}}}, "flags"},
		{"hidden skipped", &App{Name: "a", Commands: []Command{{Name: "h", Description: long, Hidden: true}, {Name: "ok", Description: "fine"}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app, AuditOptions{SkipExampleValidation: true})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Audit = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Audit = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestAuditWidthOption(t *testing.T) {
	app := &App{Name: "a", Commands: []Command{{Name: "build", Description: strings.Repeat("x", 50)}}}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true}); err != nil {
		t.Fatalf("width 80: %v", err)
	}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: 50}); err == nil {
		t.Error("width 50: want error, got nil")
	}
}

func TestAuditLongTierIsUnlimitedButWarnsOnUnwrappable(t *testing.T) {
	prose := strings.Repeat("A sentence of ordinary prose. ", 100)
	wide := strings.Repeat("x", 120)
	tests := []struct {
		name     string
		cmd      Command
		wantWarn int
	}{
		{"long prose", Command{Name: "c", Description: "ok", LongDescription: prose, Notes: []Note{{Heading: "h", Text: prose}}}, 0},
		{"wide raw note", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: "short\n" + wide, Raw: true}}}, 1},
		{"narrow raw note", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: "short", Raw: true}}}, 0},
		{"wide example", Command{Name: "c", Description: "ok", Examples: []Example{{Line: "a c " + wide}}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warns []string
			app := &App{Name: "a", Commands: []Command{tt.cmd}}
			err := Audit(app, AuditOptions{SkipExampleValidation: true, Warn: func(m string) { warns = append(warns, m) }})
			if err != nil {
				t.Fatalf("Audit = %v, want nil", err)
			}
			if len(warns) != tt.wantWarn {
				t.Errorf("warnings = %v, want %d", warns, tt.wantWarn)
			}
		})
	}
}

func TestAuditReportsEveryLayoutViolation(t *testing.T) {
	long := strings.Repeat("word ", 20)
	app := &App{Name: "a", Commands: []Command{
		{Name: "one", Description: long},
		{Name: "two", Description: "fine"},
		{Name: "three", Description: long, Subcommands: []Command{{Name: "sub", Description: long}}},
	}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil, want errors")
	}
	for _, want := range []string{`"one"`, `"three"`, `"sub"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Audit error does not mention %s:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), `"two"`) {
		t.Errorf("Audit error mentions the conforming command:\n%v", err)
	}
}

// A flag row is drawn with its default, required and deprecated suffixes, so
// the audit measures the decorated text the renderer draws.
func TestAuditFlagRowsIncludeDecorations(t *testing.T) {
	desc := strings.Repeat("x", 40)
	tests := []struct {
		name    string
		opt     Option
		wantErr bool
	}{
		{"plain fits", Option{Flags: "--x", Description: desc}, false},
		{"default overflows", Option{Flags: "--x", Description: desc, DefaultText: "/var/lib/some/long/default/path"}, true},
		{"required overflows", Option{Flags: "--x", Description: strings.Repeat("x", 65), Required: true}, true},
		{"deprecated overflows", Option{Flags: "--x", Description: desc, Deprecated: "use --other-flag instead"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{Name: "a", GlobalFlags: []Option{tt.opt}}
			err := Audit(app, AuditOptions{SkipExampleValidation: true})
			if (err != nil) != tt.wantErr {
				t.Errorf("Audit = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditTrailingPeriodsAreConsistent(t *testing.T) {
	cmds := func(descs ...string) []Command {
		var out []Command
		for i, d := range descs {
			out = append(out, Command{Name: fmt.Sprintf("c%d", i), Description: d})
		}
		return out
	}
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{"none end with a period", &App{Name: "a", Commands: cmds("First", "Second")}, ""},
		{"all end with a period", &App{Name: "a", Commands: cmds("First.", "Second.")}, ""},
		{"mixed", &App{Name: "a", Commands: cmds("First.", "Second")}, "trailing period"},
		{"mixed subcommands", &App{Name: "a", Commands: []Command{{Name: "g", Description: "Group", Subcommands: cmds("One.", "Two")}}}, "g subcommands"},
		{"first sentence decides", &App{Name: "a", Commands: cmds("First. More detail follows", "Second.")}, ""},
		{"different listings may differ", &App{Name: "a", Commands: []Command{{Name: "g", Description: "Group", Subcommands: cmds("One.", "Two.")}}}, ""},
		{"flags mixed", &App{Name: "a", GlobalFlags: []Option{{Flags: "--a", Description: "One."}, {Flags: "--b", Description: "Two"}}}, "trailing period"},
		{"a default suffix does not hide the period", &App{Name: "a", GlobalFlags: []Option{{Flags: "--a", Description: "One.", DefaultText: "x"}, {Flags: "--b", Description: "Two."}}}, ""},
		{"library commands are exempt", &App{Name: "a", Commands: append(cmds("First.", "Second."), CompletionCommand())}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app, AuditOptions{SkipExampleValidation: true})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Audit = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Audit = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// A structural error used to end the audit before the layout pass ran, so an
// author fixing a bad example learned about a long description only afterwards,
// on the next run — against the audit's own promise to report everything at
// once. And Audit(nil) dereferenced nil.
func TestAuditReportsLayoutAlongsideStructuralErrors(t *testing.T) {
	long := strings.Repeat("word ", 30)
	tests := []struct {
		name string
		app  *App
		want []string
	}{
		{"bad example and a long description",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: long, Examples: []Example{{Line: "a job --nope"}}}}},
			[]string{"--nope", "columns"}},
		{"duplicate flag and a long description",
			&App{Name: "a", GlobalFlags: []Option{{Flags: "--x", Description: "one"}, {Flags: "--x", Description: long}}},
			[]string{"duplicate option --x", "columns"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app)
			if err == nil {
				t.Fatal("Audit = nil, want errors")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Audit error does not mention %q:\n%v", want, err)
				}
			}
		})
	}
}

func TestAuditOfANilAppIsAnErrorNotAPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Audit(nil) panicked: %v", r)
		}
	}()
	if err := Audit(nil); err == nil {
		t.Error("Audit(nil) = nil, want an error")
	}
}

// The audit has to reach every listing the renderer draws, not just the ones the
// first tests happened to use: nested subcommands, shortcuts, the root's own
// persistent flags — and it must leave hidden options alone.
func TestAuditReachesEveryListing(t *testing.T) {
	long := strings.Repeat("word ", 30)
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{"flag on a nested subcommand",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Subcommands: []Command{
				{Name: "sub", Description: "Sub", Options: []Option{{Flags: "--out", Description: long}}}}}}},
			"command job sub flags"},
		{"description of a doubly nested subcommand",
			&App{Name: "a", Commands: []Command{{Name: "a1", Description: "A", Subcommands: []Command{
				{Name: "b1", Description: "B", Subcommands: []Command{{Name: "c1", Description: long}}}}}}},
			"c1"},
		{"flag on a shortcut",
			&App{Name: "a", Shortcuts: []Command{{Name: "sc", Description: "Shortcut", Options: []Option{{Flags: "--out", Description: long}}}}},
			"command sc flags"},
		{"subcommand of a shortcut",
			&App{Name: "a", Shortcuts: []Command{{Name: "sc", Description: "Shortcut", Subcommands: []Command{{Name: "inner", Description: long}}}}},
			"inner"},
		{"root persistent option",
			&App{Name: "a", PersistentOptions: []Option{{Flags: "--root", Description: long}}},
			`"--root"`},
		{"hidden option is not drawn, so not audited",
			&App{Name: "a", GlobalFlags: []Option{{Flags: "--secret", Description: long, Hidden: true}, {Flags: "--ok", Description: "fine"}}},
			""},
		{"hidden root persistent option",
			&App{Name: "a", PersistentOptions: []Option{{Flags: "--secret", Description: long, Hidden: true}}},
			""},
		{"hidden command option",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Options: []Option{{Flags: "--secret", Description: long, Hidden: true}}}}},
			""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app, AuditOptions{SkipExampleValidation: true})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Audit = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Audit = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

// markLibraryOwned has to reach every level, or the exemption holds for a
// library command and not for its subcommands — and the one test that existed
// only looked at the top.
func TestMarkLibraryOwnedReachesEveryLevel(t *testing.T) {
	tree := markLibraryOwned(Command{Name: "lib", Subcommands: []Command{
		{Name: "a", Subcommands: []Command{{Name: "a1"}, {Name: "a2"}}},
		{Name: "b"},
	}})
	var walk func(c Command, depth int)
	walk = func(c Command, depth int) {
		if !c.libraryOwned {
			t.Errorf("%s (depth %d) is not marked library-owned", c.Name, depth)
		}
		if c.libraryRoot != "lib" || c.libraryDepth != depth {
			t.Errorf("%s: root %q depth %d, want %q %d", c.Name, c.libraryRoot, c.libraryDepth, "lib", depth)
		}
		for _, s := range c.Subcommands {
			walk(s, depth+1)
		}
	}
	walk(tree, 0)

	// And the exemption is observable: library subcommands that end in a full
	// stop beside ones that do not are not the author's inconsistency.
	app := &App{Name: "a", Commands: []Command{markLibraryOwned(Command{Name: "lib", Description: "Library", Subcommands: []Command{
		{Name: "x", Description: "One."}, {Name: "y", Description: "Two"}}})}}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true}); err != nil {
		t.Errorf("Audit = %v, want nil for a library-owned subtree", err)
	}
}

// The warnings are advisory but their edges are the contract: exactly Width is
// fine, one past is not; one note reports once however many lines are long; and
// a caller who passes no Warn gets no panic.
func TestAuditWarningsEdges(t *testing.T) {
	line := func(n int) string { return strings.Repeat("x", n) }
	tests := []struct {
		name string
		cmd  Command
		want int
	}{
		{"raw note line exactly at the width", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: line(80), Raw: true}}}, 0},
		{"raw note line one past", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: line(81), Raw: true}}}, 1},
		{"two long lines in one note warn once", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: line(90) + "\n" + line(95), Raw: true}}}, 1},
		{"example exactly at the width", Command{Name: "c", Description: "ok", Examples: []Example{{Line: line(80)}}}, 0},
		{"example one past", Command{Name: "c", Description: "ok", Examples: []Example{{Line: line(81)}}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warns []string
			app := &App{Name: "a", Commands: []Command{tt.cmd}}
			if err := Audit(app, AuditOptions{SkipExampleValidation: true, Warn: func(m string) { warns = append(warns, m) }}); err != nil {
				t.Fatalf("Audit = %v", err)
			}
			if len(warns) != tt.want {
				t.Errorf("warnings = %v, want %d", warns, tt.want)
			}
			// No Warn at all must be harmless.
			if err := Audit(app, AuditOptions{SkipExampleValidation: true}); err != nil {
				t.Errorf("Audit with no Warn = %v", err)
			}
		})
	}
}

// The period rule's edges: which names it blames, what it ignores, and that the
// library's own `-E` row — which has no full stop — never counts against an
// author whose flags all do.
func TestAuditPeriodRuleEdges(t *testing.T) {
	t.Run("names the right side", func(t *testing.T) {
		app := &App{Name: "a", Commands: []Command{{Name: "c0", Description: "First."}, {Name: "c1", Description: "Second"}, {Name: "c2", Description: "Third"}}}
		err := Audit(app, AuditOptions{SkipExampleValidation: true})
		if err == nil || !strings.Contains(err.Error(), "c0 end with one, c1, c2 do not") {
			t.Errorf("Audit = %v, want c0 under 'end with one' and c1, c2 under 'do not'", err)
		}
	})
	passing := []struct {
		name string
		app  *App
	}{
		{"trailing space after the full stop",
			&App{Name: "a", GlobalFlags: []Option{{Flags: "--a", Description: "One. "}, {Flags: "--b", Description: "Two."}}}},
		{"an empty flag description is ignored",
			&App{Name: "a", GlobalFlags: []Option{{Flags: "--a", Description: "One."}, {Flags: "--b", Description: ""}, {Flags: "--c", Description: "Three."}}}},
		{"the built-in examples row does not count",
			&App{Name: "a", EnableExamplesFlag: true, GlobalFlags: []Option{{Flags: "--a", Description: "One."}, {Flags: "--b", Description: "Two."}},
				Commands: []Command{{Name: "job", Description: "Jobs.", Examples: []Example{{Line: "a job"}}}}}},
	}
	for _, tt := range passing {
		t.Run(tt.name, func(t *testing.T) {
			if err := Audit(tt.app, AuditOptions{SkipExampleValidation: true}); err != nil {
				t.Errorf("Audit = %v, want nil", err)
			}
		})
	}
}

func TestAuditCoversParametersAndExplicitSubcommandEntries(t *testing.T) {
	long := strings.Repeat("word ", 30)
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{"long parameter",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Parameters: []Param{{Name: "file", Description: long}}}}},
			"command job parameters"},
		{"mixed periods among parameters",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Parameters: []Param{{Name: "a", Description: "One."}, {Name: "b", Description: "Two"}}}}},
			"command job parameters: descriptions disagree"},
		{"long explicit entry",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", SubcommandEntries: []Param{{Name: "sub", Description: long}}}}},
			"command job subcommands"},
		{"the tree's long row is not drawn when entries replace it",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs",
				SubcommandEntries: []Param{{Name: "sub", Description: "Short"}},
				Subcommands:       []Command{{Name: "sub", Description: long}}}}},
			""},
		{"parameters that fit",
			&App{Name: "a", Commands: []Command{{Name: "job", Description: "Jobs", Parameters: []Param{{Name: "file", Description: "The input file"}}}}},
			""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app, AuditOptions{SkipExampleValidation: true})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Audit = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Audit = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}
