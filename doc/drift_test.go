package doc

import (
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

// The markdown generator's subcommand list has to be clihelp's list. It was a
// copy that had lost the branch preferring an explicit SubcommandEntries, so a
// documented-only entry showed in `--help` and was missing from the page.
func TestSubcommandEntriesAgreesWithClihelp(t *testing.T) {
	for _, cmd := range []clihelp.Command{
		{Name: "leaf"},
		{Name: "parent", Subcommands: []clihelp.Command{
			{Name: "a", Description: "First."},
			{Name: "b", Description: "Second.", Aliases: []string{"bee"}},
			{Name: "hidden", Hidden: true},
		}},
		{Name: "documented", SubcommandEntries: []clihelp.Param{
			{Name: "add <x>", Description: "Add one."},
			{Name: "list", Description: "List them."},
		}, Subcommands: []clihelp.Command{{Name: "add"}}},
	} {
		got, want := subcommandEntries(&cmd), clihelp.SubcommandList(cmd)
		if len(got) != len(want) {
			t.Fatalf("%s: %d entries, clihelp says %d", cmd.Name, len(got), len(want))
		}
		for i := range got {
			if got[i].Name != want[i].Name || got[i].Description != want[i].Description {
				t.Errorf("%s entry %d: %+v, clihelp says %+v", cmd.Name, i, got[i], want[i])
			}
		}
	}
}

// The tautology above is worth keeping — it is what the cross-package guard
// points at — but on its own it compares a one-line delegation against the thing
// it delegates to, so it cannot fail. This asserts the consequence that actually
// regressed: a subcommand documented only through SubcommandEntries used to be
// missing from the generated page while `--help` went on showing it.
func TestDocumentedOnlySubcommandReachesThePage(t *testing.T) {
	cmd := clihelp.Command{
		Name: "whitelist", Description: "Manage the whitelist.",
		UsageLine: "app whitelist <subcommand>",
		SubcommandEntries: []clihelp.Param{
			{Name: "add <email>", Description: "Add an address."},
			{Name: "list", Description: "List every address."},
		},
		Subcommands: []clihelp.Command{{Name: "add", Description: "Add an address."}},
	}
	app := &clihelp.App{Name: "app", Commands: []clihelp.Command{cmd}}
	page := renderCommandPage(app, cmdNode{path: []string{"whitelist"}, cmd: cmd})

	for _, want := range []string{"add \\<email>", "| list |", "List every address."} {
		if !strings.Contains(page, want) {
			t.Errorf("the page is missing %q:\n%s", want, page)
		}
	}
	if !strings.Contains(page, "(whitelist-add.md)") {
		t.Errorf("the entry that does have a page lost its link:\n%s", page)
	}
}
