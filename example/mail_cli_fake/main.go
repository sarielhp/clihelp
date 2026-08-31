// Command mail_cli_fake regenerates the complete mail_cli usage/help interface
// using clihelp's data model and unified renderer. The global overview is
// written to stderr and every detailed usage page to stdout, mirroring
// mail_cli's writer routing.
package main

import (
	"fmt"
	"os"

	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/doc"
)

func main() {
	app := buildApp()

	if os.Getenv("CLIHELP_GEN") != "" {
		changed, err := doc.RenderMarkdown(app, doc.MarkdownOptions{Dir: "docs/mail_cli_fake"})
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate docs: %v\n", err)
			os.Exit(1)
		}
		if changed {
			fmt.Fprintln(os.Stderr, "processed under docs/mail_cli_fake/")
		} else {
			fmt.Fprintln(os.Stderr, "docs up to date.")
		}
		return
	}

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
