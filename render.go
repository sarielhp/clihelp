package clihelp

import (
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
	var fd int
	if o.Writer == nil || o.Writer == os.Stdout {
		fd = int(os.Stdout.Fd())
	} else if f, ok := o.Writer.(*os.File); ok {
		fd = int(f.Fd())
	} else {
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
	var fd int
	if o.Writer == nil || o.Writer == os.Stdout {
		fd = int(os.Stdout.Fd())
	} else if f, ok := o.Writer.(*os.File); ok {
		fd = int(f.Fd())
	} else {
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
				if f.Required {
					desc = desc + " (required)"
				}
				if f.Deprecated != "" {
					desc = desc + " (deprecated: " + f.Deprecated + ")"
				}
				params = append(params, Param{Name: f.Flags, Description: desc})
			}
			indent := colIndent(params)
			for _, p := range params {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
			}
			fmt.Fprintln(w)
		}

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
				if o0.Required {
					desc = desc + " (required)"
				}
				if o0.Deprecated != "" {
					desc = desc + " (deprecated: " + o0.Deprecated + ")"
				}
				optParams = append(optParams, Param{Name: o0.Flags, Description: desc})
			}
			indent := colIndent(optParams)
			for _, p := range optParams {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
			}
		}

		if len(globalOptions) > 0 {
			th.Hdr.Fprintln(w, "\nGlobal Flags:")
			if a.OmitGlobalFlagsInCommands {
				reflow(w, th.Body, wrapWidth(termWidth, 2, o.maxContent()), 2, "", fmt.Sprintf("Run '%s help flags' for flags available to all commands.", appName(a)))
			} else {
				optParams := make([]Param, 0, len(globalOptions))
				for _, o0 := range globalOptions {
					desc := o0.Description
					if o0.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
						desc = desc + " (default: " + o0.DefaultText + ")"
					}
					if o0.Required {
						desc = desc + " (required)"
					}
					if o0.Deprecated != "" {
						desc = desc + " (deprecated: " + o0.Deprecated + ")"
					}
					optParams = append(optParams, Param{Name: o0.Flags, Description: desc})
				}
				indent := colIndent(optParams)
				for _, p := range optParams {
					reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
				}
			}
		} else if len(cmd.Options) > 0 {
			// Presentation fallback for legacy command options
			var optParams []Param
			for _, opt := range cmd.Options {
				if opt.Hidden {
					continue
				}
				desc := opt.Description
				if opt.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
					desc = desc + " (default: " + opt.DefaultText + ")"
				}
				optParams = append(optParams, Param{Name: opt.Flags, Description: desc})
			}
			if len(optParams) > 0 {
				th.Hdr.Fprintln(w, "\nFlags:")
				indent := colIndent(optParams)
				for _, p := range optParams {
					reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
				}
			}
		}

		if len(cmd.Examples) > 0 {
			th.Hdr.Fprintln(w, "\nExamples:")
			renderExamples(w, a, cmd, th, o, termWidth, cmd.Examples, 2, 4)
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
