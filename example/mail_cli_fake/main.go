// Command mail_cli_fake regenerates the complete mail_cli usage/help interface
// using clihelp's data model and unified renderer. The global overview is
// written to stderr and every detailed usage page to stdout, mirroring
// mail_cli's writer routing.
package main

import (
	"os"

	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/doc"
)

func main() {
	app := buildApp()
	cmd := doc.MarkdownCommand(doc.MarkdownOptions{Dir: "docs/mail_cli_fake"})
	cmd.Hidden = true
	app.Commands = append(app.Commands, cmd)

	if len(os.Args) > 1 {
		if err := app.Execute(os.Args[1:]); err != nil {
			app.PrintError(err)
			os.Exit(1)
		}
		return
	}

	app.RenderGlobal(clihelp.Options{Writer: os.Stderr})

	for _, path := range detailedPaths(app) {
		app.RenderCommand(clihelp.Options{Writer: os.Stdout}, path...)
	}
}
