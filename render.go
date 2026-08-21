package clihelp

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// Theme controls the colors, separators, and header wording used by the
// renderer. A Theme's zero value is safe: any nil color field falls back to
// the default mail_cli palette when applied via App.Theme or Options.Theme.
type Theme struct {
	// Hdr colors section labels (e.g. "Description:", "Usage:").
	Hdr *color.Color
	// Body colors description/usage prose.
	Body *color.Color
	// Accent colors the help header line, separators, and global command groups.
	Accent *color.Color
	// Separator toggles the horizontal rule drawn around the header block.
	Separator bool
	// TitlePrefix is prepended to the command help header line
	// (e.g. "Detailed Usage: ").
	TitlePrefix string
}

func defaultTheme() Theme {
	return Theme{
		Hdr:         color.New(color.FgYellow, color.Bold),
		Body:        color.New(color.FgWhite),
		Accent:      color.New(color.FgCyan, color.Bold),
		Separator:   true,
		TitlePrefix: "Detailed Usage: ",
	}
}

// Options controls a single render operation.
type Options struct {
	// Writer is the output destination. When nil, os.Stdout is used.
	Writer io.Writer
	// Width is the target terminal width in columns. When zero it is
	// auto-detected with a 70-column fallback for non-terminal output.
	Width int
	// Theme overrides the App.Theme and the package default. When nil the
	// App's theme (or the default) applies.
	Theme *Theme
}

func (o Options) out() io.Writer {
	if o.Writer != nil {
		return o.Writer
	}
	return os.Stdout
}

func (o Options) theme(a *App) Theme {
	th := defaultTheme()
	src := o.Theme
	if src == nil && a != nil {
		src = a.Theme
	}
	if src == nil {
		return th
	}
	if src.Hdr != nil {
		th.Hdr = src.Hdr
	}
	if src.Body != nil {
		th.Body = src.Body
	}
	if src.Accent != nil {
		th.Accent = src.Accent
	}
	if src.TitlePrefix != "" {
		th.TitlePrefix = src.TitlePrefix
	}
	th.Separator = src.Separator
	return th
}

// width resolves the layout width: an explicit Width wins, otherwise stdout's
// terminal width is used with a 70-column fallback for non-terminals.
func (o Options) width() int {
	if o.Width > 0 {
		return o.Width
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 70
	}
	return w
}

// visualLen calculates the visible character length of s, ignoring ANSI codes.
func visualLen(s string) int {
	return len([]rune(stripansi.Strip(s)))
}

// splitLines splits text on '\n', preserving empty segments so consecutive
// newlines produce blank lines.
func splitLines(text string) []string {
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

// reflowSegment word-wraps a single paragraph (no newlines) so that no visual
// line exceeds width columns. An optional prefix is placed in its own first-line
// column and following lines are indented to align it.
func reflowSegment(w io.Writer, c *color.Color, width, indent int, prefix, text string) {
	words := strings.Fields(text)

	if prefix != "" {
		prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
		curLen := visualLen(prefixStr)
		if curLen > indent {
			c.Fprintln(w, prefixStr)
			prefixStr = strings.Repeat(" ", indent)
			curLen = indent
		}
		if len(words) == 0 {
			if curLen > 0 {
				c.Fprintln(w, prefixStr)
			}
			return
		}
		indentStr := strings.Repeat(" ", indent)
		var cur strings.Builder
		cur.WriteString(prefixStr)
		for _, word := range words {
			wlen := visualLen(word)
			space := 0
			if curLen > indent {
				space = 1
			}
			if curLen+space+wlen > width {
				c.Fprintln(w, cur.String())
				cur.Reset()
				cur.WriteString(indentStr)
				cur.WriteString(word)
				curLen = indent + wlen
			} else {
				if space > 0 {
					cur.WriteString(" ")
					curLen++
				}
				cur.WriteString(word)
				curLen += wlen
			}
		}
		if curLen > indent {
			c.Fprintln(w, cur.String())
		}
		return
	}

	if len(words) == 0 {
		return
	}
	indentStr := strings.Repeat(" ", indent)
	var cur strings.Builder
	cur.WriteString(indentStr)
	curLen := indent
	for _, word := range words {
		wlen := visualLen(word)
		space := 0
		if curLen > indent {
			space = 1
		}
		if curLen+space+wlen > width {
			c.Fprintln(w, cur.String())
			cur.Reset()
			cur.WriteString(indentStr)
			cur.WriteString(word)
			curLen = indent + wlen
		} else {
			if space > 0 {
				cur.WriteString(" ")
				curLen++
			}
			cur.WriteString(word)
			curLen += wlen
		}
	}
	if curLen > indent {
		c.Fprintln(w, cur.String())
	}
}

// reflow word-wraps text so that no visual line exceeds width columns. It
// preserves intentional newlines in text by processing each line segment
// independently. An optional prefix is placed in its own first-line column
// and following lines are indented to align it. Width is measured in visible
// characters, so ANSI escape codes and multi-byte runes are ignored when
// deciding where to wrap.
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string) {
	if indent < 2 {
		indent = 2
	}
	segments := splitLines(strings.TrimSpace(text))
	for i, seg := range segments {
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
				c.Fprintln(w, prefixStr)
				prefix = ""
			} else {
				c.Fprintln(w, strings.Repeat(" ", indent))
			}
			continue
		}
		if seg == "" {
			continue
		}
		reflowSegment(w, c, width, indent, prefix, seg)
		prefix = ""
	}
}

// separator writes a horizontal rule in the accent color.
func separator(w io.Writer, th Theme, width int) {
	th.Accent.Fprintln(w, strings.Repeat("=", width))
}

// displayName renders a command name followed by its aliases in parentheses.
func displayName(c Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + " (" + strings.Join(c.Aliases, ", ") + ")"
}

// title returns the explicit help title of cmd, falling back to its name.
func title(c *Command) string {
	if c.Title != "" {
		return c.Title
	}
	return c.Name
}

// subcommandEntries returns the display list for the Subcommands section,
// preferring explicit entries over the structural Subcommands tree.
func subcommandEntries(c *Command) []Param {
	if len(c.SubcommandEntries) > 0 {
		return c.SubcommandEntries
	}
	var out []Param
	for i := range c.Subcommands {
		if !c.Subcommands[i].Hidden {
			out = append(out, Param{Name: c.Subcommands[i].Name, Description: c.Subcommands[i].Description})
		}
	}
	return out
}

// RenderGlobal writes the top-level application overview: a "Usage of <name>:"
// header, the command list with aliases, shortcut commands, global flags, and
// (when set) config path and version.
func (a *App) RenderGlobal(o Options) {
	th := o.theme(a)
	w := o.out()
	wrapW := min(o.width(), 80)

	th.Hdr.Fprintf(w, "Usage of %s:\n\n", a.Name)

	if a.Description != "" {
		reflow(w, th.Body, wrapW, 2, "", inline(a.Description))
		fmt.Fprintln(w)
	}

	th.Accent.Fprintln(w, "Commands:")
	{
		params := make([]Param, 0, len(a.Commands))
		for _, c := range a.Commands {
			if !c.Hidden {
				params = append(params, Param{Name: displayName(c), Description: c.Description})
			}
		}
		indent := colIndent(params)
		for _, p := range params {
			reflow(w, th.Body, wrapW, indent, p.Name, inline(p.Description))
		}
	}
	fmt.Fprintln(w)

	if len(a.Shortcuts) > 0 {
		th.Accent.Fprintln(w, "Shortcut Commands:")
		params := make([]Param, 0, len(a.Shortcuts))
		for _, s := range a.Shortcuts {
			if !s.Hidden {
				params = append(params, Param{Name: displayName(s), Description: s.Description})
			}
		}
		indent := colIndent(params)
		for _, p := range params {
			reflow(w, th.Body, wrapW, indent, p.Name, inline(p.Description))
		}
		fmt.Fprintln(w)
	}

	var globalFlags []Option
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
		th.Accent.Fprintln(w, "Global Flags:")
		params := make([]Param, 0, len(globalFlags))
		for _, f := range globalFlags {
			params = append(params, Param{Name: f.Flags, Description: f.Description})
		}
		indent := colIndent(params)
		for _, p := range params {
			reflow(w, th.Body, wrapW, indent, p.Name, inline(p.Description))
		}
		fmt.Fprintln(w)
	}

	th.Hdr.Fprintln(w, "Detailed Help:")
	fmt.Fprintf(w, "  To see more details and usage for any command, run:\n")
	fmt.Fprintf(w, "  %s <command> --help\n\n", a.Name)

	if a.ConfigPath != "" {
		th.Hdr.Fprintln(w, "Config file location:")
		fmt.Fprintf(w, "  %s\n\n", a.ConfigPath)
	}

	if a.Version != "" {
		th.Hdr.Fprintln(w, "Version:")
		fmt.Fprintf(w, "  %s\n", a.Version)
	}

	if a.GlobalNote != "" {
		fmt.Fprintln(w)
		reflow(w, th.Body, wrapW, 2, "", inline(a.GlobalNote))
	}
}

// RenderCommand writes help for the command at path (e.g. "config" "set"),
// rendering any of these present sections in order: Description, Usage,
// Subcommands, Parameters, Flags, Examples, Notes. Returns true if the path
// exists.
func (a *App) RenderCommand(o Options, path ...string) bool {
	cmd := a.LookupCommand(path...)
	if cmd == nil {
		return false
	}
	th := o.theme(a)
	w := o.out()

	sepW := min(o.width(), 70)
	wrapW := min(o.width(), 80)

	fmt.Fprintln(w)
	if th.Separator {
		separator(w, th, sepW)
	}
	th.Accent.Fprintln(w, th.TitlePrefix+title(cmd))
	if th.Separator {
		separator(w, th, sepW)
	}

	if cmd.Description != "" {
		th.Hdr.Fprintln(w, "Description:")
		reflow(w, th.Body, wrapW, 2, "", inline(cmd.Description))
	}

	if cmd.UsageLine != "" {
		th.Hdr.Fprintln(w, "\nUsage:")
		reflow(w, th.Body, wrapW, 2, "", cmd.UsageLine)
	}

	if subs := subcommandEntries(cmd); len(subs) > 0 {
		th.Hdr.Fprintln(w, "\nSubcommands:")
		indent := colIndent(subs)
		for _, s := range subs {
			reflow(w, th.Body, wrapW, indent, s.Name, inline(s.Description))
		}
	}

	if len(cmd.Parameters) > 0 {
		th.Hdr.Fprintln(w, "\nParameters:")
		indent := colIndent(cmd.Parameters)
		for _, p := range cmd.Parameters {
			reflow(w, th.Body, wrapW, indent, p.Name, inline(p.Description))
		}
	}

	var allOptions []Option
	// Collect ancestor persistent options
	for _, anc := range a.ancestorsForPath(path...) {
		for _, opt := range anc.PersistentOptions {
			if !opt.Hidden {
				allOptions = append(allOptions, opt)
			}
		}
	}
	for _, opt := range cmd.PersistentOptions {
		if !opt.Hidden {
			allOptions = append(allOptions, opt)
		}
	}
	for _, opt := range cmd.Options {
		if !opt.Hidden {
			allOptions = append(allOptions, opt)
		}
	}

	if len(allOptions) > 0 {
		th.Hdr.Fprintln(w, "\nFlags:")
		optParams := make([]Param, 0, len(allOptions))
		for _, o0 := range allOptions {
			desc := o0.Description
			if o0.DefaultText != "" {
				desc = desc + " (default: " + o0.DefaultText + ")"
			}
			optParams = append(optParams, Param{Name: o0.Flags, Description: desc})
		}
		indent := colIndent(optParams)
		for _, p := range optParams {
			reflow(w, th.Body, wrapW, indent, p.Name, inline(p.Description))
		}
	}

	if len(cmd.Examples) > 0 {
		th.Hdr.Fprintln(w, "\nExamples:")
		for _, ex := range cmd.Examples {
			reflow(w, th.Body, wrapW, 2, "", ex.Line)
			if ex.Description != "" {
				reflow(w, th.Body, wrapW, 4, "", inline(ex.Description))
			}
		}
	}

	for _, note := range cmd.Notes {
		if note.Heading != "" {
			th.Hdr.Fprintln(w, "\n"+note.Heading+":")
		}
		reflow(w, th.Body, wrapW, 2, "", inline(note.Text))
	}

	if th.Separator {
		separator(w, th, sepW)
	}
	fmt.Fprintln(w)
	return true
}

// Render writes global help when path is empty, or command help for a path.
func (a *App) Render(o Options, path ...string) bool {
	if len(path) == 0 {
		a.RenderGlobal(o)
		return true
	}
	return a.RenderCommand(o, path...)
}

// colIndent returns the indent (max visible width + 4) so that every entry in
// params lines up its name column with the longest name.
func colIndent(params []Param) int {
	maxW := 0
	for _, p := range params {
		if l := visualLen(p.Name); l > maxW {
			maxW = l
		}
	}
	return maxW + 4
}

// inline renders inline markdown in s to a string with ANSI/OSC8 sequences.
func inline(s string) string {
	var buf strings.Builder
	renderInline(&buf, s)
	return buf.String()
}
