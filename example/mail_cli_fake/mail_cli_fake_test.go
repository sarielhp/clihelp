package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
	"github.com/sarielhp/clihelp"
)

// --- Oracle: a faithful re-implementation of mail_cli's usage formatter ---

var (
	oHdr    = color.New(color.FgYellow, color.Bold)
	oBody   = color.New(color.FgWhite)
	oAccent = color.New(color.FgCyan, color.Bold)
	oSub    = color.New(color.FgGreen)
)

// oracleWidth pins the layout width to the same 70 columns the tests inject
// into clihelp, so the comparison is deterministic regardless of TTY state.
func oracleWidth() int {
	return 70
}

func oracleSeparator(out io.Writer) {
	width := oracleWidth()
	if width > 70 {
		width = 70
	}
	oAccent.Fprintln(out, strings.Repeat("=", width))
}

func oracleSplitLines(text string) []string {
	if text == "" {
		return nil
	}
	var out []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			out = append(out, text[start:i])
			start = i + 1
		}
	}
	out = append(out, text[start:])
	return out
}

func oracleReflowSegment(out io.Writer, c *color.Color, prefixColor *color.Color, width, indent int, prefix, text string) {
	if prefix != "" {
		prefixStr := "  " + padRight(prefix, indent-2)
		if prefixColor != nil {
			prefixStr = prefixColor.Sprint(prefixStr)
		}
		if oracleVisualLen(prefixStr) > indent {
			c.Fprintln(out, prefixStr)
			prefixStr = strings.Repeat(" ", indent)
		}
		words := strings.Fields(text)
		if len(words) == 0 {
			if len(strings.TrimSpace(prefixStr)) > 0 {
				c.Fprintln(out, prefixStr)
			}
			return
		}
		indentStr := strings.Repeat(" ", indent)
		var current strings.Builder
		current.WriteString(prefixStr)
		curLen := oracleVisualLen(prefixStr)
		for _, word := range words {
			wlen := oracleVisualLen(word)
			space := 0
			if curLen > indent {
				space = 1
			}
			if curLen+space+wlen > width {
				c.Fprintln(out, current.String())
				current.Reset()
				current.WriteString(indentStr)
				current.WriteString(word)
				curLen = indent + wlen
			} else {
				if space > 0 {
					current.WriteString(" ")
					curLen++
				}
				current.WriteString(word)
				curLen += wlen
			}
		}
		if curLen > indent {
			c.Fprintln(out, current.String())
		}
		return
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}
	indentStr := strings.Repeat(" ", indent)
	var current strings.Builder
	current.WriteString(indentStr)
	curLen := indent
	for _, word := range words {
		wlen := oracleVisualLen(word)
		space := 0
		if curLen > indent {
			space = 1
		}
		if curLen+space+wlen > width {
			c.Fprintln(out, current.String())
			current.Reset()
			current.WriteString(indentStr)
			current.WriteString(word)
			curLen = indent + wlen
		} else {
			if space > 0 {
				current.WriteString(" ")
				curLen++
			}
			current.WriteString(word)
			curLen += wlen
		}
	}
	if curLen > indent {
		c.Fprintln(out, current.String())
	}
}

// oracleVisualLen returns the visible width of s, ignoring ANSI escape codes.
func oracleVisualLen(s string) int {
	return len([]rune(stripansi.Strip(s)))
}

func oracleReflow(out io.Writer, c *color.Color, prefixColor *color.Color, indent int, prefix, text string) {
	width := oracleWidth()
	if width > 80 {
		width = 80
	}
	if indent < 2 {
		indent = 2
	}
	segments := oracleSplitLines(strings.TrimSpace(text))
	for i, seg := range segments {
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := "  " + padRight(prefix, indent-2)
				if prefixColor != nil {
					prefixStr = prefixColor.Sprint(prefixStr)
				}
				c.Fprintln(out, prefixStr)
				prefix = ""
			} else {
				c.Fprintln(out, strings.Repeat(" ", indent))
			}
			continue
		}
		if seg == "" {
			continue
		}
		oracleReflowSegment(out, c, prefixColor, width, indent, prefix, seg)
		prefix = ""
	}
}

// oracleSubs mirrors clihelp.subcommandEntries: prefer explicit entries.
func oracleSubs(cmd *clihelp.Command) []clihelp.Param {
	if len(cmd.SubcommandEntries) > 0 {
		return cmd.SubcommandEntries
	}
	var out []clihelp.Param
	for i := range cmd.Subcommands {
		out = append(out, clihelp.Param{Name: cmd.Subcommands[i].Name, Description: cmd.Subcommands[i].Description})
	}
	return out
}

func oracleDetailedUsage(out io.Writer, a *clihelp.App, path []string, cmd *clihelp.Command) {
	title := cmd.Title
	if title == "" {
		title = cmd.Name
	}
	io.WriteString(out, "\n")
	oracleSeparator(out)
	oracleReflow(out, oAccent, nil, 2, "", "Detailed Usage: "+title)
	oracleSeparator(out)
	if cmd.Description != "" {
		oHdr.Fprintln(out, "Description:")
		oracleReflow(out, oBody, nil, 2, "", cmd.Description)
	}
	if cmd.UsageLine != "" {
		oHdr.Fprintln(out, "\nUsage:")
		oracleReflow(out, oBody, nil, 2, "", clihelp.Inline(cmd.UsageLine))
	}
	if subs := oracleSubs(cmd); len(subs) > 0 {
		oHdr.Fprintln(out, "\nSubcommands:")
		maxW := 0
		for _, s := range subs {
			if len(s.Name) > maxW {
				maxW = len(s.Name)
			}
		}
		indent := maxW + 4
		for _, s := range subs {
			oracleReflow(out, oBody, oSub, indent, s.Name, s.Description)
		}
	}
	if len(cmd.Parameters) > 0 {
		oHdr.Fprintln(out, "\nParameters:")
		maxW := 0
		for _, pp := range cmd.Parameters {
			if len(pp.Name) > maxW {
				maxW = len(pp.Name)
			}
		}
		indent := maxW + 4
		for _, pp := range cmd.Parameters {
			oracleReflow(out, oBody, nil, indent, pp.Name, pp.Description)
		}
	}

	// Flags mirror clihelp.collectOptions: app persistent + global flags,
	// each ancestor's persistent options, then the target's persistent and
	// local options.
	var allOpts []clihelp.Option
	addOpts := func(list []clihelp.Option) {
		for _, o := range list {
			if !o.Hidden {
				allOpts = append(allOpts, o)
			}
		}
	}
	addOpts(a.PersistentOptions)
	addOpts(a.GlobalFlags)
	var ancPath []string
	for _, p := range path[:len(path)-1] {
		ancPath = append(ancPath, p)
		if anc := a.LookupCommand(ancPath...); anc != nil {
			addOpts(anc.PersistentOptions)
		}
	}
	addOpts(cmd.PersistentOptions)
	addOpts(cmd.Options)

	if len(allOpts) > 0 {
		oHdr.Fprintln(out, "\nFlags:")
		maxW := 0
		for _, o := range allOpts {
			if len(o.Flags) > maxW {
				maxW = len(o.Flags)
			}
		}
		indent := maxW + 4
		for _, o := range allOpts {
			oracleReflow(out, oBody, nil, indent, o.Flags, o.Description)
		}
	}
	if len(cmd.Examples) > 0 {
		oHdr.Fprintln(out, "\nExamples:")
		for _, e := range cmd.Examples {
			oracleReflow(out, oBody, nil, 2, "", clihelp.Inline(e.Line))
		}
	}
	for _, n := range cmd.Notes {
		if n.Heading != "" {
			oHdr.Fprintln(out, "\n"+n.Heading+":")
		}
		oracleReflow(out, oBody, nil, 2, "", n.Text)
	}
	oracleSeparator(out)
	io.WriteString(out, "\n")
}

func oracleDisplayName(c clihelp.Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + " (" + strings.Join(c.Aliases, ", ") + ")"
}

func oracleGlobalUsage(out io.Writer, a *clihelp.App) {
	oHdr.Fprintf(out, "Usage of %s:\n\n", a.Name)
	if a.Description != "" {
		oracleReflow(out, oBody, nil, 2, "", a.Description)
		io.WriteString(out, "\n")
	}
	oAccent.Fprintln(out, "Commands:")
	{
		params := make([]clihelp.Param, 0, len(a.Commands))
		for _, c := range a.Commands {
			params = append(params, clihelp.Param{Name: oracleDisplayName(c), Description: c.Description})
		}
		maxW := 0
		for _, p := range params {
			if len(p.Name) > maxW {
				maxW = len(p.Name)
			}
		}
		indent := maxW + 4
		for _, p := range params {
			oracleReflow(out, oBody, oSub, indent, p.Name, p.Description)
		}
	}
	io.WriteString(out, "\n")
	if len(a.Shortcuts) > 0 {
		oAccent.Fprintln(out, "Shortcut Commands:")
		params := make([]clihelp.Param, 0, len(a.Shortcuts))
		for _, s := range a.Shortcuts {
			params = append(params, clihelp.Param{Name: oracleDisplayName(s), Description: s.Description})
		}
		maxW := 0
		for _, p := range params {
			if len(p.Name) > maxW {
				maxW = len(p.Name)
			}
		}
		indent := maxW + 4
		for _, p := range params {
			oracleReflow(out, oBody, oSub, indent, p.Name, p.Description)
		}
		io.WriteString(out, "\n")
	}
	var globalFlags []clihelp.Option
	for _, f := range a.PersistentOptions {
		if !f.Hidden {
			globalFlags = append(globalFlags, f)
		}
	}
	for _, f := range a.GlobalFlags {
		if !f.Hidden {
			globalFlags = append(globalFlags, f)
		}
	}
	if len(globalFlags) > 0 {
		oAccent.Fprintln(out, "Global Flags:")
		params := make([]clihelp.Param, 0, len(globalFlags))
		for _, f := range globalFlags {
			params = append(params, clihelp.Param{Name: f.Flags, Description: f.Description})
		}
		maxW := 0
		for _, p := range params {
			if len(p.Name) > maxW {
				maxW = len(p.Name)
			}
		}
		indent := maxW + 4
		for _, p := range params {
			oracleReflow(out, oBody, nil, indent, p.Name, p.Description)
		}
		io.WriteString(out, "\n")
	}
	oHdr.Fprintln(out, "Detailed Help:")
	oracleReflow(out, oBody, nil, 2, "", fmt.Sprintf("Run '%s help <command>' for command-specific options.", a.Name))
	if a.ConfigPath != "" {
		oHdr.Fprintln(out, "Config file location:")
		oracleReflow(out, oBody, nil, 2, "", a.ConfigPath)
	}
	if a.Version != "" {
		oHdr.Fprintln(out, "Version:")
		oracleReflow(out, oBody, nil, 2, "", a.Version)
	}
}

// padRight pads s to width w using fmt's %-*s semantics.
func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// --- Tests ---

func TestMailCLIReconstruction(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	app := buildApp()
	paths := detailedPaths(app)
	if len(paths) != 43 {
		t.Fatalf("expected 43 detailed usage pages, got %d", len(paths))
	}

	for _, path := range paths {
		var got, want bytes.Buffer
		app.RenderCommand(clihelp.Options{Writer: &got, Width: 70}, path...)
		cmd := app.LookupCommand(path...)
		oracleDetailedUsage(&want, app, path, cmd)

		if got.String() != want.String() {
			t.Errorf("detailed page %q mismatch (raw)\n--- got ---\n%s\n--- want ---\n%s",
				strings.Join(path, " "), got.String(), want.String())
		}
		if stripansi.Strip(got.String()) != stripansi.Strip(want.String()) {
			t.Errorf("detailed page %q mismatch (ansi-stripped): %q",
				strings.Join(path, " "), stripansi.Strip(got.String()))
		}
	}
}

func TestMailCLIGlobalOverview(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	app := buildApp()
	var got, want bytes.Buffer
	app.RenderGlobal(clihelp.Options{Writer: &got, Width: 70})
	oracleGlobalUsage(&want, app)

	if got.String() != want.String() {
		t.Errorf("global overview mismatch (raw)\n--- got ---\n%s\n--- want ---\n%s", got.String(), want.String())
	}
	if stripansi.Strip(got.String()) != stripansi.Strip(want.String()) {
		t.Errorf("global overview mismatch (ansi-stripped)")
	}
}
