package doc

import (
	"fmt"

	"github.com/sarielhp/clihelp"
)

// MarkdownCommand returns a mountable "docs" command that generates the owning
// application's Markdown help pages. MarkdownOptions.Dir supplies the default
// output directory; --output overrides it. Reports go to the application's
// stderr, and stdout remains empty. Like any mounted command, it runs the
// application's lifecycle hooks.
func MarkdownCommand(o MarkdownOptions) clihelp.Command {
	dir := o.Dir
	if dir == "" {
		dir = defaultMarkdownDir
	}
	return clihelp.Command{
		Name:        "docs",
		Description: "Generate Markdown help pages",
		Args:        clihelp.NoArgs,
		Options: []clihelp.Option{
			clihelp.String(&dir, "-o, --output DIR", dir, "Output directory"),
		},
		Run: func(ctx *clihelp.Context) error {
			if err := ctx.Context.Err(); err != nil {
				return fmt.Errorf("generate markdown: %w", err)
			}
			changed, err := RenderMarkdown(ctx.App, MarkdownOptions{Dir: dir})
			if err != nil {
				return fmt.Errorf("generate markdown: %w", err)
			}
			if changed {
				fmt.Fprintf(ctx.Stderr, "help pages generated under %s\n", dir)
				return nil
			}
			fmt.Fprintf(ctx.Stderr, "help pages up to date under %s\n", dir)
			return nil
		},
	}
}
