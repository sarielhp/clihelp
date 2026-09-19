package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

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
	// Subcommand colors subcommand names.
	Subcommand *color.Color
	// Flag colors flag/option names (e.g. "--verbose, -v").
	Flag *color.Color
	// Separator toggles the horizontal rule drawn around the header block.
	Separator bool
	// TitlePrefix is prepended to the command help header line
	// (e.g. "Detailed Usage: ").
	TitlePrefix string

	// ExampleCmd colors command and subcommand names in examples.
	ExampleCmd *color.Color
	// ExampleFlag colors flags and options in examples.
	ExampleFlag *color.Color
	// ExampleArg colors arguments, positional values, and paths in examples.
	ExampleArg *color.Color
	// ExampleComment colors shell comments (# ...) in examples.
	ExampleComment *color.Color
	// ExampleDesc colors example descriptions beneath the command line.
	ExampleDesc *color.Color
}

func defaultTheme() Theme {
	return Theme{
		Hdr:            color.New(color.FgYellow, color.Bold),
		Body:           color.New(color.FgWhite),
		Accent:         color.New(color.FgCyan, color.Bold),
		Subcommand:     color.New(color.FgGreen),
		Flag:           color.New(color.FgCyan),
		Separator:      false,
		TitlePrefix:    "",
		ExampleCmd:     color.New(color.FgGreen, color.Bold),
		ExampleFlag:    color.New(color.FgCyan),
		ExampleArg:     color.New(color.FgWhite),
		ExampleComment: color.New(color.FgHiBlack),
		ExampleDesc:    color.New(color.FgHiBlack),
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
	// Concise requests concise command help (-h), suppressing notes and
	// displaying a footer hint pointing to extended help. The result is held to
	// ConciseMaxLines.
	Concise bool
	// NoColor renders this one call without colour, whatever the process-wide
	// setting is. An application offering a --no-color flag had no thread-safe
	// way to honour it: fatih/color's switch is global and decided from stdout at
	// package init, and mutating it per render was removed because it raced.
	NoColor bool
	// ConciseMaxLines bounds concise (-h) output. Zero means the documented
	// default of 24 lines; a negative value means no bound, which is what the
	// concise tier did before the bound existed.
	//
	// The number is the promise -h has always made in README.md and llms.txt and
	// in the comparison with cobra, where it is the stated differentiator. It was
	// a convention with nothing enforcing it: a command with thirty flags
	// rendered 92 lines, identical to --help.
	ConciseMaxLines int
	// Extended requests full command help (--help / -H / help <cmd>),
	// displaying LongDescription, all Notes, and Examples.
	Extended bool
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

// theme resolves the theme for one render: the defaults, then the app's theme,
// then the caller's on top.
//
// Options.Theme used to *replace* App.Theme rather than layer onto it, so an
// application that set a theme once and then passed any per-render Options.Theme
// silently lost every field it had not repeated — including Separator, and with
// it the whole title block.
func (o Options) theme(a *App) Theme {
	th := defaultTheme()
	if a != nil {
		th = applyTheme(th, a.Theme)
	}
	th = applyTheme(th, o.Theme)
	if o.NoColor {
		th = withoutColor(th)
	}
	return th
}

// withApp folds the application's own settings into the caller's options, so
// that every render entry point below can read one struct.
func (o Options) withApp(a *App) Options {
	if a != nil && a.NoColor {
		o.NoColor = true
	}
	return o
}

// noColor reports whether this render should emit no escapes at all.
func (o Options) noColor() bool { return o.NoColor || color.NoColor }

// inline renders inline markdown for this render, honouring Options.NoColor as
// well as the global setting.
func (o Options) inline(s string) string { return renderInlineTo(s, o.noColor()) }

// withoutColor returns th with every colour disabled, leaving the structural
// fields — Separator, TitlePrefix — alone.
func withoutColor(th Theme) Theme {
	for _, c := range []**color.Color{
		&th.Hdr, &th.Body, &th.Accent, &th.Subcommand, &th.Flag,
		&th.ExampleCmd, &th.ExampleFlag, &th.ExampleArg, &th.ExampleComment, &th.ExampleDesc,
	} {
		if *c == nil {
			continue
		}
		plain := *(*c)
		plain.DisableColor()
		*c = &plain
	}
	return th
}

func applyTheme(th Theme, src *Theme) Theme {
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
	if src.Flag != nil {
		th.Flag = src.Flag
	}
	if src.TitlePrefix != "" {
		th.TitlePrefix = src.TitlePrefix
	}
	if src.ExampleCmd != nil {
		th.ExampleCmd = src.ExampleCmd
	}
	if src.ExampleFlag != nil {
		th.ExampleFlag = src.ExampleFlag
	}
	if src.ExampleArg != nil {
		th.ExampleArg = src.ExampleArg
	}
	if src.ExampleComment != nil {
		th.ExampleComment = src.ExampleComment
	}
	if src.ExampleDesc != nil {
		th.ExampleDesc = src.ExampleDesc
	}
	if src.Separator {
		th.Separator = true
	}
	return th
}

// width resolves the layout width: an explicit Width wins, otherwise the
// Writer's terminal width is used when it is a terminal file, falling back to
// stdout, then a 70-column fallback for non-terminals.
// termFd resolves the descriptor the output goes to and whether it is a
// terminal.
//
// It asks for the capability rather than the concrete type: an application that
// sets App.Stdout to a bufio.Writer or a colorable wrapper — a very ordinary
// setup — used to be treated as a non-terminal by all three callers, so it never
// paged and its width collapsed to the 70-column fallback on a 200-column
// screen. This was written out three times with three different fallbacks.
func (o Options) termFd() (int, bool) {
	w := o.out()
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return -1, false
	}
	fd := int(f.Fd())
	return fd, term.IsTerminal(fd)
}

func (o Options) width() int {
	if o.Width > 0 {
		return o.Width
	}
	fd, isTerm := o.termFd()
	if !isTerm {
		return 70
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
	fd, isTerm := o.termFd()
	if !isTerm {
		return 0
	}
	_, h, err := term.GetSize(fd)
	if err != nil || h <= 0 {
		return 0
	}
	return h
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
			Name:        displayNameWithAliases(c),
			Description: firstSentence(c.Description),
		})
		groups = append(groups, c.Group)
	}
	if len(params) == 0 {
		return
	}
	groups = normalizeGroups(groups, "Other Commands")
	indent := colIndentFor(params, termWidth, minTextColumns)
	prev := ""

	isMultiLine := func(p Param) bool {
		textWidth := wrapWidth(termWidth, indent, o.maxContent()) - indent
		if textWidth <= 0 {
			textWidth = 40
		}
		// Measure what is drawn, not the markdown it came from. o.inline() only
		// ever shrinks visible width, so measuring the source was a systematic
		// false positive: one link in one description put a blank line between
		// every entry in the list, none of which wrapped.
		rendered := o.inline(p.Description)
		return visualLen(rendered) > textWidth || strings.Contains(rendered, "\n")
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
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, o.inline(p.Description), th.Subcommand)
	}
}

// usageLine returns the command-line usage template for the app.
func (a *App) usageLine() string {
	if a.UsageLine != "" {
		return a.UsageLine
	}
	name := appName(a)
	hasFlags := len(a.PersistentOptions) > 0 || len(a.GlobalFlags) > 0 || len(a.Options) > 0
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

// decorateOptionDescription appends the suffixes an option's description
// carries in every listing: its default, whether it is required, and whether it
// is deprecated.
//
// These twelve lines existed twice — here and in renderOptionsGrouped — and only
// the other copy was tested, so the default value could vanish from every -h and
// --help flag table while `help flags` went on showing it. The cross-package
// guard cannot see a copy made inside one package.
func decorateOptionDescription(opt Option) string {
	desc := opt.Description
	if opt.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
		desc += " (default: " + opt.DefaultText + ")"
	}
	if opt.Required {
		desc += " (required)"
	}
	if opt.Deprecated != "" {
		desc += " (deprecated: " + opt.Deprecated + ")"
	}
	return desc
}

func optionsToParams(options []Option) []Param {
	params := make([]Param, 0, len(options))
	for _, opt := range options {
		params = append(params, Param{Name: opt.Flags, Description: decorateOptionDescription(opt)})
	}
	return params
}

func renderOptionList(w io.Writer, th Theme, o Options, termWidth int, options []Option) {
	params := optionsToParams(options)
	indent := colIndentFor(params, termWidth, minTextColumns)
	for _, p := range params {
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, o.inline(p.Description), th.Flag)
	}
}

// renderGlobalShortcuts lists the shortcut commands, if any are visible.
//
// The heading used to be printed from the unfiltered count, so an application
// whose every shortcut is hidden printed "Shortcut Commands:" and nothing under
// it. renderGlobalFlagsSection, two functions down, has always had the right
// order: filter, return if empty, then print the heading.
func (a *App) renderGlobalShortcuts(w io.Writer, th Theme, o Options, termWidth int) {
	params := make([]Param, 0, len(a.Shortcuts))
	for _, s := range a.Shortcuts {
		if !s.Hidden {
			params = append(params, Param{
				Name:        displayNameWithAliases(s),
				Description: firstSentence(s.Description),
			})
		}
	}
	if len(params) == 0 {
		return
	}
	th.Accent.Fprintln(w, "Shortcut Commands:")
	indent := colIndentFor(params, termWidth, minTextColumns)
	for _, p := range params {
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, o.inline(p.Description), th.Subcommand)
	}
	fmt.Fprintln(w)
}

func (a *App) renderGlobalFlagsSection(w io.Writer, th Theme, o Options, termWidth int) {
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
	// The application's own options are the root's local flags. They belong on
	// the root's page, and nowhere else — this section renders only there.
	for _, f := range a.Options {
		if !f.Hidden {
			globalFlags = append(globalFlags, f)
		}
	}
	if len(globalFlags) == 0 {
		return
	}
	th.Accent.Fprintln(w, "Global Flags:")
	renderOptionList(w, th, o, termWidth, globalFlags)
	fmt.Fprintln(w)
}

// RenderGlobal writes the top-level application overview: a command-line usage
// template, description, command list with aliases, shortcut commands,
// global flags, and help footer.
func (a *App) RenderGlobal(o Options) {
	o = o.withApp(a)
	a.pageOutput(o, func(out io.Writer) {
		a.budgeted(out, o, nil, func(w io.Writer) {
			th := o.theme(a)
			termWidth := o.width()

			// "Usage:" is the prefix column, so a long usage line wraps under
			// itself instead of overflowing. It was the one line in the page that
			// was never wrapped, which is what made a narrow terminal unreadable.
			//
			// inline(), as RenderCommand and RenderMan already do: App.UsageLine
			// was rendered on two of the four paths, so the same app showed raw
			// ** here and a URL the author had written as a link.
			reflowMargin(w, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 0, 8,
				"Usage:", o.inline(a.usageLine()), th.Hdr)

			if a.Description != "" {
				fmt.Fprintln(w)
				reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", o.inline(a.Description))
			}
			// GlobalNote is the application's own note, and it belongs with the
			// description on the page an author expects it on. It used to appear
			// only in "help docs", "help more" and the manual page — so the two
			// real applications that set one, including this library's own
			// example, put a link in their help that nobody was shown. Extended
			// help only, as Command.Notes are: the concise tier is a prompt, not
			// documentation.
			if a.GlobalNote != "" && a.GlobalNote != a.Description && !o.Concise {
				fmt.Fprintln(w)
				reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", o.inline(a.GlobalNote))
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

			a.renderGlobalShortcuts(w, th, o, termWidth)
			a.renderGlobalFlagsSection(w, th, o, termWidth)

			if len(a.Examples) > 0 {
				th.Accent.Fprintln(w, "Examples:")
				renderExamples(w, a, nil, th, o, termWidth, a.Examples, 2, 4)
				fmt.Fprintln(w)
			}

			if len(visibleCommands) > 0 || len(a.Shortcuts) > 0 {
				reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", fmt.Sprintf("Run '%s <command> -h' for command help, or '%s help [flags|man]'.", appName(a), appName(a)))
			}

			if a.ConfigPath != "" {
				fmt.Fprintln(w)
				th.Hdr.Fprint(w, "Config: ")
				fmt.Fprintln(w, a.ConfigPath)
			}
		})
	})
}

// normalizeGroups gives ungrouped entries a heading of their own, but only when
// something else is grouped.
//
// A heading was printed when the group value changed, and `prev` was updated
// only inside that branch — so an entry declaring no group printed no heading
// and did not reset `prev`, which put it under the previous group's heading and
// then swallowed the heading when that group resumed. The output said something
// false about two entries at once. RenderFlags already avoided this through
// normalizeFlagGroups; the other two call sites never got it.
func normalizeGroups(groups []string, fallback string) []string {
	grouped := false
	for _, g := range groups {
		if g != "" {
			grouped = true
			break
		}
	}
	if !grouped {
		return groups
	}
	out := make([]string, len(groups))
	for i, g := range groups {
		if g == "" {
			g = fallback
		}
		out[i] = g
	}
	return out
}

func renderCommandTitle(w io.Writer, th Theme, o Options, cmd *Command, termWidth, sepW int) {
	if !th.Separator && th.TitlePrefix == "" {
		return
	}
	fmt.Fprintln(w)
	if th.Separator {
		separator(w, th, sepW)
	}
	// The title lives inside the rule, so it gets the rule's width — not the rule's
	// width plus its own indent, which let it overhang by two columns.
	reflow(w, th.Accent, sepW, 2, "", th.TitlePrefix+title(cmd))
	if th.Separator {
		separator(w, th, sepW)
	}
}

func (a *App) buildDefaultUsage(cmd *Command, path []string) string {
	if cmd.UsageLine != "" {
		return cmd.UsageLine
	}
	fullPath := strings.Join(append([]string{appName(a)}, path...), " ")
	hasFlags := len(a.collectOptions(path, cmd)) > 0
	// SubcommandList, not cmd.Subcommands: a command that documents its
	// subcommands through SubcommandEntries used to get "[args]" in its usage
	// line above a populated "Subcommands:" list. The same
	// copy-without-the-preference that SubcommandList's own comment records.
	hasSubs := len(SubcommandList(*cmd)) > 0
	switch {
	case hasSubs && hasFlags:
		return fmt.Sprintf("%s [flags] <subcommand> [args]", fullPath)
	case hasSubs:
		return fmt.Sprintf("%s <subcommand> [args]", fullPath)
	case hasFlags:
		return fmt.Sprintf("%s [flags] [args]", fullPath)
	default:
		return fmt.Sprintf("%s [args]", fullPath)
	}
}

func (a *App) renderCommandSubcommands(w io.Writer, th Theme, o Options, termWidth int, cmd *Command) {
	subs := subcommandEntries(cmd)
	if len(subs) == 0 {
		return
	}
	th.Hdr.Fprintln(w, "\nSubcommands:")
	if len(cmd.SubcommandEntries) > 0 {
		indent := colIndentFor(subs, termWidth, minTextColumns)
		for _, s := range subs {
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, s.Name, o.inline(s.Description), th.Subcommand)
		}
	} else {
		a.renderCommandGrouped(w, th, o, termWidth, cmd.Subcommands)
	}
}

func renderCommandParams(w io.Writer, th Theme, o Options, termWidth int, params []Param) {
	if len(params) == 0 {
		return
	}
	th.Hdr.Fprintln(w, "\nParameters:")
	indent := colIndentFor(params, termWidth, minTextColumns)
	for _, p := range params {
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, o.inline(p.Description))
	}
}

func (a *App) renderCommandFlags(w io.Writer, th Theme, o Options, termWidth int, path []string, cmd *Command) {
	localOptions := a.collectLocalOptions(cmd)
	if len(localOptions) > 0 {
		th.Hdr.Fprintln(w, "\nFlags:")
		renderOptionList(w, th, o, termWidth, localOptions)
	}

	globalOptions := a.collectGlobalOptions(path, cmd)
	if len(globalOptions) > 0 {
		th.Hdr.Fprintln(w, "\nGlobal Flags:")
		if a.OmitGlobalFlagsInCommands {
			reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", fmt.Sprintf("Run '%s help flags' for flags available to all commands.", appName(a)))
		} else {
			renderOptionList(w, th, o, termWidth, globalOptions)
		}
	}
}

func renderRawLines(w io.Writer, th Theme, indent int, text string) {
	indentStr := strings.Repeat(" ", indent)
	// Raw note text goes to the terminal unrendered, so this is one of the four
	// places an author string bypasses renderInline's sanitiser. A lone carriage
	// return here used to reach the screen, where it returns the cursor to
	// column 0 and the rest of the row overwrites what was already drawn.
	lines := splitLines(sanitizeControl(strings.Trim(text, "\r\n")))
	for _, line := range lines {
		if line == "" {
			fmt.Fprintln(w)
		} else {
			th.Body.Fprintln(w, indentStr+line)
		}
	}
}

func renderNoteContent(w io.Writer, th Theme, o Options, termWidth, indent int, note Note) {
	if note.Raw {
		renderRawLines(w, th, indent, note.Text)
		return
	}

	// The fenced branch below prints its lines verbatim, so the author string is
	// sanitised here for the same reason renderRawLines sanitises its own. The
	// prose branch is sanitised again by renderInline, which is harmless.
	lines := splitLines(sanitizeControl(strings.Trim(note.Text, "\r\n")))
	var proseLines []string
	inFence := false

	flushProse := func() {
		if len(proseLines) > 0 {
			prose := strings.Join(proseLines, "\n")
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, "", o.inline(prose))
			proseLines = nil
		}
	}

	indentStr := strings.Repeat(" ", indent)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				flushProse()
				inFence = true
			} else {
				inFence = false
			}
			continue
		}

		if inFence {
			if line == "" {
				fmt.Fprintln(w)
			} else {
				th.Body.Fprintln(w, indentStr+line)
			}
		} else {
			proseLines = append(proseLines, line)
		}
	}
	flushProse()
}

func renderCommandNotes(w io.Writer, th Theme, o Options, termWidth int, notes []Note) {
	for _, note := range notes {
		if note.Heading != "" {
			// inline(), like every other heading: the markdown generator renders
			// a heading's markup (doc/md.go) and the terminal printed it raw, so
			// "**Warning**" came out bold on the page and asterisked on screen.
			// It also sanitises, which is why a heading needs no separate guard.
			th.Hdr.Fprintln(w, "\n"+o.inline(note.Heading)+":")
		}
		renderNoteContent(w, th, o, termWidth, 2, note)
	}
}

// conciseHelpLines is the documented bound on -h output.
const conciseHelpLines = 24

// conciseBudget resolves the line bound for one render: the caller's override,
// then the terminal's own height when it is shorter than the default, then the
// default. A line is left for the shell prompt.
func (o Options) conciseBudget() int {
	return conciseBudgetFor(o.ConciseMaxLines, o.height())
}

// conciseBudgetFor is the concise tier's line budget, given what the caller
// asked for and how tall the terminal is. A terminal shorter than the tier's
// own bound gets one line less than its height, so the shell's next prompt does
// not scroll the first line away.
//
// It takes the height as an argument rather than measuring it, because a
// measurement needs a real terminal: height() returns zero for every writer a
// test can hand it, so the short-terminal branch could be deleted with the suite
// green. This is the same seam doc/'s markdownHashOf uses for its format
// version.
func conciseBudgetFor(configured, height int) int {
	if configured != 0 {
		return configured
	}
	if height > 0 && height-1 < conciseHelpLines {
		return height - 1
	}
	return conciseHelpLines
}

// budgeted renders through fn and holds the result to the concise bound,
// replacing whatever did not fit with one line saying how much was dropped and
// where to read it. Outside the concise tier it renders straight through.
func (a *App) budgeted(w io.Writer, o Options, path []string, fn func(io.Writer)) {
	budget := o.conciseBudget()
	if !o.Concise || budget < 1 {
		fn(w)
		return
	}
	var buf bytes.Buffer
	fn(&buf)
	// Pass it through untouched when it fits: writeWithinBudget trims trailing
	// newlines, and the blank line a help page ends with is deliberate spacing.
	if len(splitLines(strings.TrimRight(buf.String(), "\n"))) <= budget {
		_, _ = w.Write(buf.Bytes())
		return
	}
	writeWithinBudget(w, buf.String(), budget, o.width(), a.explainMoreHint(path))
}

func (a *App) renderCommandConciseFooter(w io.Writer, th Theme, o Options, termWidth int, path []string) {
	helpHint := "(or --help)"
	if a.ExtendedHelpFlag {
		helpHint = "(or --help / -H)"
	}
	footer := fmt.Sprintf("Run '%s help %s' %s for extended documentation and examples.", appName(a), strings.Join(path, " "), helpHint)
	fmt.Fprintln(w)
	reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", footer)
}

// RenderCommand writes help for the command at path (e.g. "config" "set"),
// rendering any of these present sections in order: Usage, Description,
// Subcommands, Parameters, Flags, Examples, Notes. Returns true if the path
// exists.
func (a *App) RenderCommand(o Options, path ...string) bool {
	o = o.withApp(a)
	cmd := a.LookupCommand(path...)
	if cmd == nil {
		return false
	}
	a.pageOutput(o, func(out io.Writer) {
		a.budgeted(out, o, path, func(w io.Writer) {
			th := o.theme(a)
			sepW := min(o.width(), o.maxContent())
			termWidth := o.width()

			renderCommandTitle(w, th, o, cmd, termWidth, sepW)

			usage := a.buildDefaultUsage(cmd, path)
			reflowMargin(w, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 0, 8,
				"Usage:", o.inline(usage), th.Hdr) // see RenderGlobal

			desc := cmd.Description
			if !o.Concise && cmd.LongDescription != "" {
				desc = cmd.LongDescription
			}
			if desc != "" {
				fmt.Fprintln(w)
				reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", o.inline(desc))
			}

			a.renderCommandSubcommands(w, th, o, termWidth, cmd)
			renderCommandParams(w, th, o, termWidth, cmd.Parameters)
			a.renderCommandFlags(w, th, o, termWidth, path, cmd)

			if len(cmd.Examples) > 0 {
				th.Hdr.Fprintln(w, "\nExamples:")
				renderExamples(w, a, cmd, th, o, termWidth, cmd.Examples, 2, 4)
			}

			if !o.Concise {
				renderCommandNotes(w, th, o, termWidth, cmd.Notes)
			} else if cmd.LongDescription != "" || len(cmd.Notes) > 0 {
				a.renderCommandConciseFooter(w, th, o, termWidth, path)
			}

			if th.Separator {
				separator(w, th, sepW)
			}
			fmt.Fprintln(w)
		})
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
