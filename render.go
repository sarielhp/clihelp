package clihelp

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
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
	// Subcommand colors subcommand names.
	Subcommand *color.Color
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
		Subcommand:  color.New(color.FgGreen),
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
	// MaxContentWidth caps the wrap width to indent+MaxContentWidth columns.
	// When zero it defaults to 80. Set to a larger value to allow content to
	// use more horizontal space than the default 80-column body.
	MaxContentWidth int
	// Theme overrides the App.Theme and the package default. When nil the
	// App's theme (or the default) applies.
	Theme *Theme
	// Pager enables automatic paging through $PAGER when output exceeds
	// the terminal height. When true, output is buffered and piped through
	// the pager only when it doesn't fit on one screen.
	Pager bool
}

// maxContent resolves the content width cap, defaulting to 80.
func (o Options) maxContent() int {
	if o.MaxContentWidth > 0 {
		return o.MaxContentWidth
	}
	return 80
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
	if src.Subcommand != nil {
		th.Subcommand = src.Subcommand
	}
	if src.TitlePrefix != "" {
		th.TitlePrefix = src.TitlePrefix
	}
	th.Separator = src.Separator
	return th
}

// width resolves the layout width: an explicit Width wins, otherwise the
// Writer's terminal width is used when it is a terminal file, falling back to
// stdout, then a 70-column fallback for non-terminals.
func (o Options) width() int {
	if o.Width > 0 {
		return o.Width
	}
	fd := int(os.Stdout.Fd())
	if f, ok := o.Writer.(*os.File); ok {
		fd = int(f.Fd())
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return 70
	}
	return w
}

// stripANSI removes both CSI escape sequences (e.g. \x1b[31m) and OSC
// sequences (e.g. \x1b]8;;url\x1b\ for hyperlinks, \x1b]0;title\x07 for
// window titles) from s, returning only the visible text.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(?:\x1b\\|\x07)`)
	return re.ReplaceAllString(s, "")
}

// visualLen returns the display column width of s, ignoring ANSI escape
// codes. Wide East-Asian characters count as two columns.
func visualLen(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// splitLines splits text on '\n', preserving empty segments so consecutive
// newlines produce blank lines. Trailing '\r' (CRLF line endings) is trimmed.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	var out []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			out = append(out, strings.TrimSuffix(text[start:i], "\r"))
			start = i + 1
		}
	}
	out = append(out, strings.TrimSuffix(text[start:], "\r"))
	return out
}

// reflowSegment word-wraps a single paragraph (no newlines) so that no visual
// line exceeds width columns. An optional prefix is placed in its own first-line
// column and following lines are indented to align it.
func reflowSegment(w io.Writer, c *color.Color, prefixColor *color.Color, width, indent int, prefix, text string) {
	words := strings.Fields(text)

	if prefix != "" {
		prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
		if prefixColor != nil {
			prefixStr = prefixColor.Sprint(prefixStr)
		}
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
func reflow(w io.Writer, c *color.Color, width, indent int, prefix, text string, prefixColors ...*color.Color) {
	var prefixColor *color.Color
	if len(prefixColors) > 0 {
		prefixColor = prefixColors[0]
	}
	if indent < 2 {
		indent = 2
	}
	segments := splitLines(strings.TrimSpace(text))
	for i, seg := range segments {
		if seg == "" && i+1 < len(segments) {
			if prefix != "" {
				prefixStr := fmt.Sprintf("  %-*s", indent-2, prefix)
				if prefixColor != nil {
					prefixStr = prefixColor.Sprint(prefixStr)
				}
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
		reflowSegment(w, c, prefixColor, width, indent, prefix, seg)
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

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wrapWidth calculates the effective wrapping width for a given terminal
// width, indent, and content cap (maxContent).
func wrapWidth(termWidth, indent, maxContent int) int {
	return min(termWidth, indent+maxContent)
}

// appName returns the display name for the app, falling back to "app".
func appName(a *App) string {
	if a.Name != "" {
		return a.Name
	}
	return "app"
}

// renderCommandGrouped writes a grouped, aligned command list to w. A group
// heading (in the accent color) is emitted whenever a command's Group value
// changes from the previous visible command's. Commands with an empty Group
// render without a heading. Hidden commands are skipped.
func (a *App) renderCommandGrouped(w io.Writer, th Theme, o Options, termWidth int, cmds []Command) {
	var params []Param
	var groups []string
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		params = append(params, Param{Name: displayName(c), Description: c.Description})
		groups = append(groups, c.Group)
	}
	if len(params) == 0 {
		return
	}
	indent := colIndent(params)
	prev := ""
	for i, p := range params {
		g := groups[i]
		if g != "" && g != prev {
			if i > 0 {
				fmt.Fprintln(w)
			}
			th.Accent.Fprintln(w, g+":")
			prev = g
		}
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Subcommand)
	}
}

// RenderGlobal writes the top-level application overview: a "Usage of <name>:"
// header, the command list with aliases, shortcut commands, global flags, and
// (when set) config path and version.
func (a *App) RenderGlobal(o Options) {
	th := o.theme(a)
	w := o.out()
	termWidth := o.width()

	th.Hdr.Fprintf(w, "Usage of %s:\n\n", appName(a))

	if a.Description != "" {
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(a.Description))
		fmt.Fprintln(w)
	}

	var visibleCommands []Command
	for _, c := range a.Commands {
		if !c.Hidden {
			visibleCommands = append(visibleCommands, c)
		}
	}
	if len(visibleCommands) > 0 {
		th.Accent.Fprintln(w, "Commands:")
		a.renderCommandGrouped(w, th, o, termWidth, a.Commands)
		fmt.Fprintln(w)
	}

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
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Subcommand)
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
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
		}
		fmt.Fprintln(w)
	}

	if len(a.Commands) > 0 || len(a.Shortcuts) > 0 {
		th.Hdr.Fprintln(w, "Detailed Help:")
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", fmt.Sprintf("Run '%s help <command>' for command-specific options.", appName(a)))
	}

	if a.ConfigPath != "" {
		th.Hdr.Fprintln(w, "Config file location:")
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", a.ConfigPath)
	}

	if a.Version != "" {
		th.Hdr.Fprintln(w, "Version:")
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", a.Version)
	}

	if a.GlobalNote != "" {
		fmt.Fprintln(w)
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(a.GlobalNote))
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

	sepW := min(o.width(), o.maxContent())
	termWidth := o.width()

	fmt.Fprintln(w)
	if th.Separator {
		separator(w, th, sepW)
	}
	reflow(w, th.Accent, wrapWidth(termWidth, 2, o.maxContent()), 2, "", th.TitlePrefix+title(cmd))
	if th.Separator {
		separator(w, th, sepW)
	}

	if cmd.Description != "" {
		th.Hdr.Fprintln(w, "Description:")
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(cmd.Description))
	}

	if cmd.UsageLine != "" {
		th.Hdr.Fprintln(w, "\nUsage:")
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(cmd.UsageLine))
	}

	if subs := subcommandEntries(cmd); len(subs) > 0 {
		th.Hdr.Fprintln(w, "\nSubcommands:")
		if len(cmd.SubcommandEntries) > 0 {
			indent := colIndent(subs)
			for _, s := range subs {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, s.Name, inline(s.Description), th.Subcommand)
			}
		} else {
			a.renderCommandGrouped(w, th, o, termWidth, cmd.Subcommands)
		}
	}

	if len(cmd.Parameters) > 0 {
		th.Hdr.Fprintln(w, "\nParameters:")
		indent := colIndent(cmd.Parameters)
		for _, p := range cmd.Parameters {
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
		}
	}

	allOptions := a.collectOptions(path, cmd)

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
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
		}
	}

	if len(cmd.Examples) > 0 {
		th.Hdr.Fprintln(w, "\nExamples:")
		for _, ex := range cmd.Examples {
			reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(ex.Line))
			if ex.Description != "" {
				reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", inline(ex.Description))
			}
		}
	}

	for _, note := range cmd.Notes {
		if note.Heading != "" {
			th.Hdr.Fprintln(w, "\n"+note.Heading+":")
		}
		reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", inline(note.Text))
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

// RenderTree writes a tree view of the command hierarchy to w.
func (a *App) RenderTree(o Options) {
	a.renderTreeTo(o, a.Commands, "", false)
	if len(a.Shortcuts) > 0 {
		fmt.Fprintln(o.out(), "\nShortcut Commands:")
		a.renderTreeTo(o, a.Shortcuts, "", true)
	}
}

// renderTreeTo recursively renders a command tree with box-drawing characters.
func (a *App) renderTreeTo(o Options, commands []Command, prefix string, isLast bool) {
	if len(commands) == 0 {
		return
	}

	for i, cmd := range commands {
		if cmd.Hidden {
			continue
		}

		isLastCmd := (i == len(commands)-1)
		currentPrefix := prefix
		if prefix != "" {
			if isLastCmd {
				currentPrefix += "└── "
			} else {
				currentPrefix += "├── "
			}
		}

		// Format command name with aliases
		name := cmd.Name
		if len(cmd.Aliases) > 0 {
			name += fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases, ", "))
		}

		// Write the command line
		th := o.theme(a)
		if cmd.Description != "" {
			reflow(o.out(), th.Subcommand, o.width(), 0, currentPrefix+name, inline(cmd.Description))
		} else {
			fmt.Fprintf(o.out(), "%s%s\n", currentPrefix, name)
		}

		// Recursively render subcommands
		if len(cmd.Subcommands) > 0 {
			subPrefix := prefix
			if prefix != "" {
				if isLastCmd {
					subPrefix += "    "
				} else {
					subPrefix += "│   "
				}
			}
			a.renderTreeTo(o, cmd.Subcommands, subPrefix, isLastCmd)
		}
	}
}
