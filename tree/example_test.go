package tree_test

import (
	"fmt"
	"strings"

	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/tree"
)

func ExampleRender() {
	var buf strings.Builder

	app := &clihelp.App{
		Name: "gitcli",
		Commands: []clihelp.Command{
			{
				Name:        "remote",
				Description: "Manage set of tracked repositories",
				Subcommands: []clihelp.Command{
					{Name: "add", Description: "Add a remote"},
					{Name: "remove", Description: "Remove a remote"},
				},
			},
			{
				Name:        "status",
				Description: "Show working tree status",
			},
		},
	}

	tree.Render(&buf, app, tree.Options{Width: 80})

	fmt.Println(strings.Contains(buf.String(), "remote"))
	fmt.Println(strings.Contains(buf.String(), "add"))
	fmt.Println(strings.Contains(buf.String(), "status"))

	// Output:
	// true
	// true
	// true
}
