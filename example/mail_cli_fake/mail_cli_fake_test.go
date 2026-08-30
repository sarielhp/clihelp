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
	oFlag   = color.New(color.FgCyan)
)

// oracleWidth pins the layout width to the same 70 columns the tests inject
// into clihelp, so the comparison is deterministic regardless of TTY state.
func oracleWidth() int {
	return 70
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
	words := strings.Fields(text)

	if prefix != "" {
		prefixDisplay := "  " + prefix
		if prefixColor != nil {
			prefixDisplay = prefixColor.Sprint(prefixDisplay)
		}
		var prefixStr string
		var curLen int
		if oracleVisualLen(prefixDisplay)+2 > indent {
			c.Fprintln(out, prefixDisplay)
			prefixStr = strings.Repeat(" ", indent)
			curLen = indent
		} else {
			prefixStr = "  " + padRight(prefix, indent-2)
			if prefixColor != nil {
				prefixStr = prefixColor.Sprint(prefixStr)
			}
			curLen = oracleVisualLen(prefixStr)
		}
		if len(words) == 0 {
			if oracleVisualLen(prefixDisplay)+2 <= indent {
				c.Fprintln(out, prefixDisplay)
			}
			return
		}
		indentStr := strings.Repeat(" ", indent)
		var current strings.Builder
		current.WriteString(prefixStr)
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

	words = strings.Fields(text)
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
	if prefix != "" && indent < 2 {
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

func oracleColIndent(params []clihelp.Param) int {
	maxW := 0
	for _, p := range params {
		l := oracleVisualLen(p.Name)
		if l+4 <= clihelp.DefaultMaxColIndent && l > maxW {
			maxW = l
		}
	}
	if maxW == 0 {
		return clihelp.DefaultMaxColIndent
	}
	return maxW + 4
}

func oracleDetailedUsage(out io.Writer, a *clihelp.App, path []string, cmd *clihelp.Command) {
	usage := cmd.UsageLine
	if usage == "" {
		fullPath := strings.Join(append([]string{a.Name}, path...), " ")
		hasFlags := len(cmd.Options) > 0
		hasSubs := len(cmd.Subcommands) > 0
		switch {
		case hasSubs && hasFlags:
			usage = fmt.Sprintf("%s [flags] <subcommand> [args]", fullPath)
		case hasSubs:
			usage = fmt.Sprintf("%s <subcommand> [args]", fullPath)
		case hasFlags:
			usage = fmt.Sprintf("%s [flags] [args]", fullPath)
		default:
			usage = fmt.Sprintf("%s [args]", fullPath)
		}
	}
	oHdr.Fprint(out, "Usage:  ")
	io.WriteString(out, clihelp.Inline(usage)+"\n")
	if cmd.Description != "" {
		io.WriteString(out, "\n")
		oracleReflow(out, oBody, nil, 0, "", cmd.Description)
	}
	if subs := oracleSubs(cmd); len(subs) > 0 {
		oHdr.Fprintln(out, "\nSubcommands:")
		indent := oracleColIndent(subs)
		for _, s := range subs {
			oracleReflow(out, oBody, oSub, indent, s.Name, s.Description)
		}
	}
	if len(cmd.Parameters) > 0 {
		oHdr.Fprintln(out, "\nParameters:")
		indent := oracleColIndent(cmd.Parameters)
		for _, pp := range cmd.Parameters {
			oracleReflow(out, oBody, nil, indent, pp.Name, pp.Description)
		}
	}

	// Flags mirror clihelp.collectOptions: app persistent + global flags,
	// each ancestor's persistent options, then the target's persistent and
	// local options.
	var localOpts []clihelp.Option
	for _, o := range cmd.Options {
		if !o.Hidden {
			localOpts = append(localOpts, o)
		}
	}

	var globalOpts []clihelp.Option
	addOpts := func(list []clihelp.Option) {
		for _, o := range list {
			if !o.Hidden {
				globalOpts = append(globalOpts, o)
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

	if len(localOpts) > 0 {
		oHdr.Fprintln(out, "\nFlags:")
		params := make([]clihelp.Param, 0, len(localOpts))
		for _, o := range localOpts {
			params = append(params, clihelp.Param{Name: o.Flags, Description: o.Description})
		}
		indent := oracleColIndent(params)
		for _, p := range params {
			oracleReflow(out, oBody, oFlag, indent, p.Name, p.Description)
		}
	}

	if len(globalOpts) > 0 {
		oHdr.Fprintln(out, "\nGlobal Flags:")
		params := make([]clihelp.Param, 0, len(globalOpts))
		for _, o := range globalOpts {
			params = append(params, clihelp.Param{Name: o.Flags, Description: o.Description})
		}
		indent := oracleColIndent(params)
		for _, p := range params {
			oracleReflow(out, oBody, oFlag, indent, p.Name, p.Description)
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
	io.WriteString(out, "\n")
}

func oracleGlobalUsage(out io.Writer, a *clihelp.App) {
	oHdr.Fprint(out, "Usage:  ")
	io.WriteString(out, a.Name+" [flags] <command> [args]\n\n")
	if a.Description != "" {
		oracleReflow(out, oBody, nil, 0, "", a.Description)
		io.WriteString(out, "\n")
	}
	oAccent.Fprintln(out, "Commands:")
	{
		params := make([]clihelp.Param, 0, len(a.Commands))
		for _, c := range a.Commands {
			params = append(params, clihelp.Param{Name: clihelp.DisplayNameWithArgs(c), Description: clihelp.FirstSentence(c.Description)})
		}
		indent := oracleColIndent(params)
		anyMultiLine := false
		for _, p := range params {
			if oracleVisualLen(p.Name)+4 > indent || oracleVisualLen(p.Description) > 70-indent || strings.Contains(p.Description, "\n") {
				anyMultiLine = true
				break
			}
		}
		for i, p := range params {
			if anyMultiLine && i > 0 {
				io.WriteString(out, "\n")
			}
			oracleReflow(out, oBody, oSub, indent, p.Name, p.Description)
		}
	}
	io.WriteString(out, "\n")
	if len(a.Shortcuts) > 0 {
		oAccent.Fprintln(out, "Shortcut Commands:")
		params := make([]clihelp.Param, 0, len(a.Shortcuts))
		for _, s := range a.Shortcuts {
			params = append(params, clihelp.Param{Name: clihelp.DisplayNameWithArgs(s), Description: clihelp.FirstSentence(s.Description)})
		}
		indent := oracleColIndent(params)
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
			desc := f.Description
			if f.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
				desc = desc + " (default: " + f.DefaultText + ")"
			}
			params = append(params, clihelp.Param{Name: f.Flags, Description: desc})
		}
		indent := oracleColIndent(params)
		for _, p := range params {
			oracleReflow(out, oBody, oFlag, indent, p.Name, p.Description)
		}
		io.WriteString(out, "\n")
	}
	oracleReflow(out, oBody, nil, 0, "", fmt.Sprintf("Run '%s <command> -h' for command help, or '%s help [flags|man]'.", a.Name, a.Name))
	if a.ConfigPath != "" {
		io.WriteString(out, "\n")
		oHdr.Fprint(out, "Config: ")
		io.WriteString(out, a.ConfigPath+"\n")
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

func TestMailCLIPagerEnabled(t *testing.T) {
	app := buildApp()
	if !app.Pager {
		t.Errorf("expected buildApp().Pager to be true, got false")
	}
}
