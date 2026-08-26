package clihelp

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestMaxContentWidthCapsWrapping(t *testing.T) {
	long := strings.Repeat("word ", 60)
	o, buf := captureOptions(200)
	o.MaxContentWidth = 40
	reflow(buf, defaultTheme().Body, wrapWidth(o.width(), 2, o.maxContent()), 2, "", long)
	max := 0
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if l := len(strip(line)); l > max {
			max = l
		}
	}
	if max > 42 {
		t.Errorf("MaxContentWidth=40 not honored, max line = %d", max)
	}
}

func TestRenderGlobalGroupedCommands(t *testing.T) {
	app := &App{
		Name: "grp",
		Commands: []Command{
			{Name: "alpha", Group: "Core", Description: "first"},
			{Name: "beta", Group: "Core", Description: "second"},
			{Name: "gamma", Group: "Extras", Description: "third"},
			{Name: "delta", Description: "ungrouped"},
		},
	}
	o, buf := captureOptions(80)
	app.RenderGlobal(o)
	out := strip(buf.String())
	for _, want := range []string{"Core:", "Extras:", "alpha", "delta"} {
		if !strings.Contains(out, want) {
			t.Errorf("global grouped output missing %q:\n%s", want, out)
		}
	}
	coreIdx := strings.Index(out, "Core:")
	extrasIdx := strings.Index(out, "Extras:")
	if coreIdx > extrasIdx {
		t.Errorf("expected Core heading before Extras heading")
	}
}

func TestRenderCommandSubcommandsGroupedAndAliased(t *testing.T) {
	app := &App{
		Name: "g",
		Commands: []Command{
			{
				Name: "parent",
				Subcommands: []Command{
					{Name: "run", Aliases: []string{"r"}, Group: "Ops", Description: "Run it"},
					{Name: "stop", Aliases: []string{"s"}, Group: "Ops", Description: "Stop it"},
				},
			},
		},
	}
	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "parent") {
		t.Fatal("RenderCommand returned false")
	}
	out := strip(buf.String())
	if !strings.Contains(out, "run (r)") {
		t.Errorf("expected subcommand alias display 'run (r)':\n%s", out)
	}
	if !strings.Contains(out, "Ops:") {
		t.Errorf("expected group heading 'Ops:':\n%s", out)
	}
}

func TestRenderGlobalEmptyApp(t *testing.T) {
	app := &App{}
	o, buf := captureOptions(80)
	app.RenderGlobal(o)
	out := strip(buf.String())
	if !strings.Contains(out, "Usage:  app") {
		t.Errorf("expected 'Usage:  app' fallback, got:\n%s", out)
	}
	if strings.Contains(out, "Commands:") {
		t.Errorf("empty app should not render a Commands section:\n%s", out)
	}
}

func TestSplitLinesStripsCRLF(t *testing.T) {
	got := splitLines("one\r\ntwo\r\nthree")
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("splitLines length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestVisualLenWideCJK(t *testing.T) {
	green := color.New(color.FgGreen).Sprint("界界")
	if visualLen(green) != 4 {
		t.Errorf("visualLen of two wide CJK chars = %d, want 4", visualLen(green))
	}
}
