// Command mailcli regenerates the complete mail_cli usage/help interface
// using clihelp's data model and unified renderer. The global overview is
// written to stderr and every detailed usage page to stdout, mirroring
// mail_cli's writer routing.
package main

import (
	"os"

	"github.com/sarielhp/clihelp"
)

func main() {
	app := buildApp()

	app.RenderGlobal(clihelp.Options{Writer: os.Stderr})

	for _, path := range detailedPaths(app) {
		app.RenderCommand(clihelp.Options{Writer: os.Stdout}, path...)
	}
}
