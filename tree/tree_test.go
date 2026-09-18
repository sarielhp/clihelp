package tree_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
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
	out := clihelp.StripANSI(buf.String())

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

// The tree measures its own indentation, and every box-drawing glyph it draws
// with is three bytes wide and one column wide. Counting bytes — the
// 2026-09-17 review's tree.visualLen finding — indented and wrapped every
// description as if the tree were two and a half times as wide as it is.
func TestRenderMeasuresWidthInColumns(t *testing.T) {
	app := &clihelp.App{
		Name: "podctl",
		Commands: []clihelp.Command{{
			Name:        "config",
			Description: "Manage settings.",
			Subcommands: []clihelp.Command{{
				Name:        "get",
				Description: "Display one or all configuration values for this podcast, with defaults.",
			}},
		}},
	}

	const width = 80
	var buf bytes.Buffer
	tree.Render(&buf, app, tree.Options{Width: width})

	var descLine string
	for _, line := range strings.Split(clihelp.StripANSI(buf.String()), "\n") {
		if runewidth.StringWidth(line) > width {
			t.Errorf("line is %d columns wide, over the %d requested:\n%q", runewidth.StringWidth(line), width, line)
		}
		if strings.Contains(line, "Display one") {
			descLine = line
		}
	}
	if descLine == "" {
		t.Fatalf("the nested description was not rendered:\n%s", buf.String())
	}
	// "└── podctl config get  Display ..." fits on one line at width 80; a
	// byte-counted indent pushed the description onto a line of its own.
	if !strings.Contains(descLine, "get") {
		t.Errorf("the description was split away from its command:\n%q", descLine)
	}
}
