package doc

import (
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
			if got[i] != want[i] {
				t.Errorf("%s entry %d: %+v, clihelp says %+v", cmd.Name, i, got[i], want[i])
			}
		}
	}
}
