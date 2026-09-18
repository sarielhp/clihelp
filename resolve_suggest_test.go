package clihelp

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// M6 — the library's policy is that a hidden command is never named in a
// suggestion; suggestCommand says so and TestSuggestionsSkipHiddenCommands
// enforces it. findSubcommandPaths checked Hidden on the visited node only, so a
// visible subcommand under a hidden parent was offered — with the hidden
// parent's name spelled out, which is the exact invocation.
func TestSuggestionsNeverDiscloseAHiddenAncestor(t *testing.T) {
	app := &App{Name: "testcli", Commands: []Command{
		{Name: "visible", Description: "V.", Subcommands: []Command{
			{Name: "run", Description: "R.", Run: nopRun},
		}},
		{Name: "internal", Description: "Internal.", Hidden: true, Subcommands: []Command{
			{Name: "wipe", Description: "Destroy everything.", Run: nopRun},
			{Name: "deeper", Description: "D.", Subcommands: []Command{
				{Name: "nuke", Description: "N.", Run: nopRun},
			}},
		}},
	}}
	silentApp(app)

	for _, arg := range []string{"wipe", "nuke"} {
		err := app.Execute([]string{arg})
		if err == nil {
			t.Fatalf("`testcli %s` did not error", arg)
		}
		if strings.Contains(err.Error(), "internal") {
			t.Errorf("the error discloses a hidden command and its path: %v", err)
		}
	}

	// A visible deep command is still suggested.
	err := app.Execute([]string{"run"})
	if err == nil || !strings.Contains(err.Error(), "visible run") {
		t.Errorf("a visible deep command lost its suggestion: %v", err)
	}
}

// M9 — the edit-distance matrix was built against every command and alias with
// no bound on the typed word, and the word comes from argv. suggestCommand
// accepts only distances below 3, so a rune-length difference of 3 or more
// already decides the answer.
func TestSuggestionCostIsBoundedByLength(t *testing.T) {
	var cmds []Command
	for i := 0; i < 50; i++ {
		cmds = append(cmds, Command{
			Name:    fmt.Sprintf("command%02d", i),
			Aliases: []string{fmt.Sprintf("c%02d", i), fmt.Sprintf("cc%02d", i)},
		})
	}
	typed := strings.Repeat("a", 128<<10) // Linux caps one argv word at 128 KB
	start := time.Now()
	got := suggestCommand(typed, cmds)
	elapsed := time.Since(start)

	if got != "" {
		t.Errorf("a 128 KB word suggested %q", got)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("suggesting against a 128 KB word took %v; the length bound decides it in O(1) per candidate", elapsed)
	}
}

// The bound must not change any answer, and it must count runes: a byte-length
// bound would wrongly skip 日本語 against 日本.
func TestSuggestionsAreUnchangedByTheBound(t *testing.T) {
	cmds := []Command{
		{Name: "build", Aliases: []string{"compile"}},
		{Name: "status"},
		{Name: "日本語"},
		{Name: "secret", Hidden: true},
	}
	for _, tt := range []struct{ typed, want string }{
		{"buildd", "build"}, // one insertion
		{"buil", "build"},   // one deletion
		{"BUILD", "build"},  // case folding
		{"compil", "build"}, // through an alias
		{"日本", "日本語"},       // one rune, three bytes
		{"rn", ""},          // far from everything
		{"bldu", ""},        // three edits away
		{"", ""},            // an empty word names nothing
		{"secref", ""},      // hidden commands are never suggested
	} {
		if got := suggestCommand(tt.typed, cmds); got != tt.want {
			t.Errorf("suggestCommand(%q) = %q, want %q", tt.typed, got, tt.want)
		}
	}
}

// The distance function itself, with typos of unequal length — every typo in the
// suite was a same-length edit, which is why four different ways of breaking
// this went unnoticed.
func TestLevenshtein(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"", "", 0}, {"", "ab", 2}, {"ab", "", 2},
		{"scna", "scan", 2}, {"secref", "secret", 1},
		{"buil", "build", 1}, {"buildd", "build", 1},
		{"rn", "build", 5}, {"lst", "status", 5},
		{"BUILD", "build", 0}, {"Deploy", "deploy", 0},
		{"héllo", "hello", 1}, {"日本語", "日本", 1}, {"caf", "café", 1},
		{"bldu", "build", 3},
	} {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// min3 has 100% statement coverage and had no assertion: a mutant returning a
// non-minimum passed the whole suite.
func TestMin3(t *testing.T) {
	for _, tt := range []struct{ a, b, c, want int }{
		{1, 2, 3, 1}, {2, 1, 3, 1}, {3, 2, 1, 1},
		{5, 5, 9, 5}, {9, 5, 5, 5}, {5, 9, 5, 5}, {4, 4, 4, 4},
	} {
		if got := min3(tt.a, tt.b, tt.c); got != tt.want {
			t.Errorf("min3(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
		}
	}
}

// filterCommandsByPrefix carries two policy decisions and neither was asserted:
// a hidden command must not win an abbreviation, and an alias prefix must.
func TestFilterCommandsByPrefix(t *testing.T) {
	cmds := []Command{
		{Name: "deploy"},
		{Name: "destroy", Hidden: true},
		{Name: "zebra", Aliases: []string{"apple", "apricot"}},
	}
	for _, tt := range []struct {
		prefix string
		want   []string
	}{
		{"de", []string{"deploy"}}, // the hidden sibling is not a match
		{"ap", []string{"zebra"}},  // matched by alias, and only once
		{"z", []string{"zebra"}},   // matched by name
		{"nothing", nil},           //
	} {
		got := filterCommandsByPrefix(cmds, tt.prefix)
		if len(got) != len(tt.want) {
			t.Errorf("prefix %q matched %d commands, want %d", tt.prefix, len(got), len(tt.want))
			continue
		}
		for i, cmd := range got {
			if cmd.Name != tt.want[i] {
				t.Errorf("prefix %q matched %q at %d, want %q", tt.prefix, cmd.Name, i, tt.want[i])
			}
		}
	}
}

// A hidden command must not be reachable by abbreviation, only by its exact
// name — the same rule as a hidden ordinary command.
func TestHiddenCommandIsNotAbbreviated(t *testing.T) {
	ran := ""
	app := &App{Name: "app", AbbrevCommands: true, Commands: []Command{
		{Name: "secret", Hidden: true, Run: func(*Context) error { ran = "secret"; return nil }},
	}}
	silentApp(app)
	if err := app.Execute([]string{"sec"}); err == nil && ran == "secret" {
		t.Error("an abbreviation reached a hidden command")
	}
	if err := app.Execute([]string{"secret"}); err != nil || ran != "secret" {
		t.Errorf("the exact name stopped reaching it: err=%v ran=%q", err, ran)
	}
}

// Shortcuts are top-level commands. Matching them at every depth would let a
// root shortcut resolve underneath an unrelated subcommand.
func TestShortcutsAreMatchedOnlyAtTheRoot(t *testing.T) {
	ran := ""
	app := &App{Name: "app",
		Commands: []Command{{
			Name: "db", Description: "Database.",
			Subcommands: []Command{{Name: "migrate", Run: func(*Context) error { ran = "migrate"; return nil }}},
		}},
		Shortcuts: []Command{{Name: "quick", Description: "Q.", Run: func(*Context) error { ran = "quick"; return nil }}},
	}
	silentApp(app)

	err := app.Execute([]string{"db", "quick"})
	if err == nil || ran == "quick" {
		t.Errorf("a root shortcut resolved under a subcommand: err=%v ran=%q", err, ran)
	}
	ran = ""
	if err := app.Execute([]string{"quick"}); err != nil || ran != "quick" {
		t.Errorf("the shortcut stopped working at the root: err=%v ran=%q", err, ran)
	}
}
