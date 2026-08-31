package tree_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/tree"
)

func TestRender(t *testing.T) {
	app := &clihelp.App{
		Name: "podctl",
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile audio episodes into distribution formats.",
			},
			{
				Name:        "config",
				Description: "Manage podcast settings and feed metadata.",
				Subcommands: []clihelp.Command{
					{Name: "get", Description: "Display one or all configuration values."},
					{Name: "set", Description: "Set a configuration value."},
				},
			},
		},
	}

	var buf bytes.Buffer
	tree.Render(&buf, app, tree.Options{Width: 80})
	out := stripansi.Strip(buf.String())

	if !strings.Contains(out, "build") {
		t.Errorf("Tree output missing build: %q", out)
	}
	if !strings.Contains(out, "config") {
		t.Errorf("Tree output missing config: %q", out)
	}
	if !strings.Contains(out, "get") {
		t.Errorf("Tree output missing get: %q", out)
	}
	if !strings.Contains(out, "Compile audio episodes") {
		t.Errorf("Tree output missing description: %q", out)
	}
}
