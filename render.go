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
		Separator:   false,
		TitlePrefix: "",
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

// height resolves the layout height: the Writer's terminal height is used
// when it is a terminal file, falling back to stdout. Returns 0 for non-terminals.
func (o Options) height() int {
	fd := int(os.Stdout.Fd())
	if f, ok := o.Writer.(*os.File); ok {
		fd = int(f.Fd())
	}
	_, h, err := term.GetSize(fd)
	if err != nil || h <= 0 {
		return 0
	}
	return h
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
		prefixDisplay := "  " + prefix
		if prefixColor != nil {
			prefixDisplay = prefixColor.Sprint(prefixDisplay)
		}
		var prefixStr string
		var curLen int
		if visualLen(prefixDisplay)+2 > indent {
			c.Fprintln(w, prefixDisplay)
			prefixStr = strings.Repeat(" ", indent)
			curLen = indent
		} else {
			prefixStr = fmt.Sprintf("  %-*s", indent-2, prefix)
			if prefixColor != nil {
				prefixStr = prefixColor.Sprint(prefixStr)
			}
			curLen = visualLen(prefixStr)
		}
		if len(words) == 0 {
			if visualLen(prefixDisplay)+2 <= indent {
				c.Fprintln(w, prefixDisplay)
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
	if prefix != "" && indent < 2 {
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

// FirstSentence returns the first sentence of s, or the first line/paragraph if shorter.
func FirstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "\n\n"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, "\n"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, ". "); idx != -1 {
		return s[:idx+1]
	}
	return s
}

// firstSentence is an internal alias for FirstSentence.
func firstSentence(s string) string {
	return FirstSentence(s)
}

// commandArgs extracts the positional argument signature for cmd, if any.
func commandArgs(cmd Command) string {
	if len(cmd.Parameters) > 0 {
		var parts []string
		for _, p := range cmd.Parameters {
			if p.Name != "" {
				parts = append(parts, p.Name)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	if cmd.UsageLine == "" {
		return ""
	}

	synopsis := strings.Split(cmd.UsageLine, " — ")[0]
	synopsis = strings.Split(synopsis, " - ")[0]
	words := strings.Fields(synopsis)
	if len(words) <= 1 {
		return ""
	}

	var args []string
	for _, w := range words {
		lower := strings.ToLower(w)
		switch lower {
		case "[options]", "[flags]", "[options...]", "[flags...]",
			"<subcommand>", "[subcommand]", "<command>", "[command]",
			cmd.Name, strings.ToLower(cmd.Name):
			continue
		}
		if strings.HasPrefix(w, "<") || strings.HasPrefix(w, "[") {
			args = append(args, w)
		}
	}
	return strings.Join(args, " ")
}

// DisplayName renders a command name followed by its aliases in parentheses.
func DisplayName(c Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + " (" + strings.Join(c.Aliases, ", ") + ")"
}

// displayName is an internal alias for DisplayName.
func displayName(c Command) string {
	return DisplayName(c)
}

// DisplayNameWithArgs renders a command name with aliases and positional argument signature.
func DisplayNameWithArgs(c Command) string {
	name := DisplayName(c)
	args := commandArgs(c)
	if args != "" {
		name += " " + args
	}
	return name
}

// displayNameWithArgs is an internal alias for DisplayNameWithArgs.
func displayNameWithArgs(c Command) string {
	return DisplayNameWithArgs(c)
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
			out = append(out, Param{
				Name:        displayNameWithArgs(c.Subcommands[i]),
				Description: c.Subcommands[i].Description,
			})
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
		params = append(params, Param{
			Name:        displayNameWithArgs(c),
			Description: firstSentence(c.Description),
		})
		groups = append(groups, c.Group)
	}
	if len(params) == 0 {
		return
	}
	indent := colIndent(params)
	prev := ""

	isMultiLine := func(p Param) bool {
		textWidth := wrapWidth(termWidth, indent, o.maxContent()) - indent
		if textWidth <= 0 {
			textWidth = 40
		}
		return visualLen(p.Description) > textWidth || strings.Contains(p.Description, "\n")
	}

	anyMultiLine := false
	for _, p := range params {
		if isMultiLine(p) {
			anyMultiLine = true
			break
		}
	}

	for i, p := range params {
		g := groups[i]
		if g != "" && g != prev {
			if i > 0 {
				fmt.Fprintln(w)
			}
			th.Accent.Fprintln(w, g+":")
			prev = g
		} else if anyMultiLine && i > 0 {
			fmt.Fprintln(w)
		}
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Subcommand)
	}
}

// usageLine returns the command-line usage template for the app.
func (a *App) usageLine() string {
	if a.UsageLine != "" {
		return a.UsageLine
	}
	name := appName(a)
	hasFlags := len(a.PersistentOptions) > 0 || len(a.GlobalFlags) > 0
	hasCmds := false
	for _, c := range a.Commands {
		if !c.Hidden {
			hasCmds = true
			break
		}
	}
	switch {
	case hasCmds && hasFlags:
		return fmt.Sprintf("%s [flags] <command> [args]", name)
	case hasCmds:
		return fmt.Sprintf("%s <command> [args]", name)
	case hasFlags:
		return fmt.Sprintf("%s [flags] [args]", name)
	default:
		return fmt.Sprintf("%s [args]", name)
	}
}

// RenderGlobal writes the top-level application overview: a command-line usage
// template, description, command list with aliases, shortcut commands,
// global flags, and help footer.
func (a *App) RenderGlobal(o Options) {
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		th.Hdr.Fprint(w, "Usage:  ")
		fmt.Fprintln(w, a.usageLine())

		if a.Description != "" {
			fmt.Fprintln(w)
			reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", inline(a.Description))
		}
		fmt.Fprintln(w)

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
					params = append(params, Param{
						Name:        displayNameWithArgs(s),
						Description: firstSentence(s.Description),
					})
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
				desc := f.Description
				if f.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
					desc = desc + " (default: " + f.DefaultText + ")"
				}
				params = append(params, Param{Name: f.Flags, Description: desc})
			}
			indent := colIndent(params)
			for _, p := range params {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
			}
			fmt.Fprintln(w)
		}

		if len(visibleCommands) > 0 || len(a.Shortcuts) > 0 {
			reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", fmt.Sprintf("Run '%s <command> --help' for more information on a command.", appName(a)))
		}

		if a.ConfigPath != "" {
			fmt.Fprintln(w)
			th.Hdr.Fprint(w, "Config: ")
			fmt.Fprintln(w, a.ConfigPath)
		}
	})
}

// RenderCommand writes help for the command at path (e.g. "config" "set"),
// rendering any of these present sections in order: Usage, Description,
// Subcommands, Parameters, Flags, Examples, Notes. Returns true if the path
// exists.
func (a *App) RenderCommand(o Options, path ...string) bool {
	cmd := a.LookupCommand(path...)
	if cmd == nil {
		return false
	}
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		sepW := min(o.width(), o.maxContent())
		termWidth := o.width()

		if th.Separator || th.TitlePrefix != "" {
			fmt.Fprintln(w)
			if th.Separator {
				separator(w, th, sepW)
			}
			reflow(w, th.Accent, wrapWidth(termWidth, 2, o.maxContent()), 2, "", th.TitlePrefix+title(cmd))
			if th.Separator {
				separator(w, th, sepW)
			}
		}

		usage := cmd.UsageLine
		if usage == "" {
			fullPath := strings.Join(append([]string{appName(a)}, path...), " ")
			hasFlags := len(a.collectOptions(path, cmd)) > 0
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

		th.Hdr.Fprint(w, "Usage:  ")
		fmt.Fprintln(w, inline(usage))

		if cmd.Description != "" {
			fmt.Fprintln(w)
			reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", inline(cmd.Description))
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

		localOptions := a.collectLocalOptions(cmd)
		globalOptions := a.collectGlobalOptions(path, cmd)

		if len(localOptions) > 0 {
			th.Hdr.Fprintln(w, "\nFlags:")
			optParams := make([]Param, 0, len(localOptions))
			for _, o0 := range localOptions {
				desc := o0.Description
				if o0.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
					desc = desc + " (default: " + o0.DefaultText + ")"
				}
				optParams = append(optParams, Param{Name: o0.Flags, Description: desc})
			}
			indent := colIndent(optParams)
			for _, p := range optParams {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
			}
		}

		if len(globalOptions) > 0 {
			th.Hdr.Fprintln(w, "\nGlobal Flags:")
			optParams := make([]Param, 0, len(globalOptions))
			for _, o0 := range globalOptions {
				desc := o0.Description
				if o0.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
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
	})
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

// DefaultMaxColIndent defines the standard column threshold for description
// text alignment in two-column command and option listings (GNU standard: 24).
const DefaultMaxColIndent = 24

// colIndent returns the indent (max visible width + 4, capped at DefaultMaxColIndent)
// so that entries line up cleanly without excessive horizontal spacing.
func colIndent(params []Param) int {
	maxW := 0
	for _, p := range params {
		l := visualLen(p.Name)
		if l+4 <= DefaultMaxColIndent && l > maxW {
			maxW = l
		}
	}
	if maxW == 0 {
		return DefaultMaxColIndent
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
	a.pageOutput(o, func(w io.Writer) {
		name := appName(a)
		th := o.theme(a)
		treeColor := th.Subcommand
		if treeColor == nil {
			treeColor = color.New(color.FgGreen)
		}
		treeColor.Fprintln(w, name)
		a.renderTreeTo(w, o, a.Commands, "", nil, false)
		if len(a.Shortcuts) > 0 {
			fmt.Fprintln(w, "\nShortcut Commands:")
			a.renderTreeTo(w, o, a.Shortcuts, "", nil, true)
		}
	})
}

// renderTreeTo recursively renders a command tree with continuous vertical box-drawing connectors.
func (a *App) renderTreeTo(w io.Writer, o Options, commands []Command, prefix string, path []string, isLast bool) {
	if len(commands) == 0 {
		return
	}

	var visible []Command
	for _, c := range commands {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}

	th := o.theme(a)
	treeColor := th.Subcommand
	if treeColor == nil {
		treeColor = color.New(color.FgGreen)
	}
	cmdColor := th.Hdr
	if cmdColor == nil {
		cmdColor = color.New(color.FgYellow)
	}

	for i, cmd := range visible {
		isLastCmd := (i == len(visible)-1)
		branch := "├── "
		if isLastCmd {
			branch = "└── "
		}

		currentPath := append(append([]string(nil), path...), cmd.Name)
		label := strings.Join(currentPath, " ")
		if len(cmd.Aliases) > 0 {
			label += fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases, ", "))
		}

		treeBranch := prefix + branch
		firstLineCmd := treeColor.Sprint(treeBranch) + cmdColor.Sprint(label)
		rawFirstPrefix := treeBranch + label + "  "
		firstWidth := visualLen(rawFirstPrefix)
		maxWidth := wrapWidth(o.width(), 0, o.maxContent())
		remainingWidth := maxWidth - firstWidth

		var contBase string
		if len(cmd.Subcommands) > 0 {
			if !isLastCmd {
				contBase = prefix + "│   │   "
			} else {
				contBase = prefix + "    │   "
			}
		} else {
			if !isLastCmd {
				contBase = prefix + "│   "
			} else {
				contBase = prefix + "    "
			}
		}

		if cmd.Description == "" {
			fmt.Fprintln(w, firstLineCmd)
		} else if firstWidth > 24 || remainingWidth < 45 {
			// Long command line: start description on next line below the command
			fmt.Fprintln(w, firstLineCmd)
			descPrefix := treeColor.Sprint(contBase) + "  "
			descIndent := visualLen(contBase) + 2
			reflowTree(w, th.Body, descIndent, maxWidth, descPrefix, descPrefix, inline(firstSentence(cmd.Description)))
		} else {
			// Short command line: inline description to the right
			firstPrefixFormatted := firstLineCmd + "  "
			totalWidth := firstWidth
			contPrefixFormatted := treeColor.Sprint(contBase)
			if rem := totalWidth - visualLen(contBase); rem > 0 {
				contPrefixFormatted += strings.Repeat(" ", rem)
			}
			reflowTree(w, th.Body, totalWidth, maxWidth, firstPrefixFormatted, contPrefixFormatted, inline(firstSentence(cmd.Description)))
		}

		// Recursively render subcommands
		if len(cmd.Subcommands) > 0 {
			nextPrefix := prefix + "│   "
			if isLastCmd {
				nextPrefix = prefix + "    "
			}
			a.renderTreeTo(w, o, cmd.Subcommands, nextPrefix, currentPath, isLastCmd)
		}
	}
}

// reflowTree word-wraps text across lines where the first line starts with firstPrefixFormatted
// and all continuation lines preserve vertical connectors in contPrefixFormatted.
func reflowTree(w io.Writer, bodyColor *color.Color, indent, width int, firstPrefixFormatted, contPrefixFormatted, text string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		fmt.Fprintln(w, strings.TrimRight(firstPrefixFormatted, " "))
		return
	}

	var cur strings.Builder
	cur.WriteString(firstPrefixFormatted)
	curLen := indent
	lineHasWords := false

	for _, word := range words {
		wlen := visualLen(word)
		space := 0
		if lineHasWords {
			space = 1
		}
		if lineHasWords && curLen+space+wlen > width {
			fmt.Fprintln(w, cur.String())
			cur.Reset()
			cur.WriteString(contPrefixFormatted)
			cur.WriteString(word)
			curLen = indent + wlen
			lineHasWords = true
		} else {
			if space > 0 {
				cur.WriteString(" ")
				curLen++
			}
			cur.WriteString(word)
			curLen += wlen
			lineHasWords = true
		}
	}
	if lineHasWords {
		fmt.Fprintln(w, cur.String())
	}
}
