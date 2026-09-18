package clihelp

import (
	"fmt"
	"io"
	"strings"
)

// renderOptionsGrouped writes a grouped, aligned option list to w. A group
// heading (in the accent color) is emitted whenever an option's Group value
// changes from the previous visible option's. Options with an empty Group
// render without a heading. Hidden options are skipped.
func renderOptionsGrouped(w io.Writer, th Theme, o Options, termWidth int, opts []Option) {
	var params []Param
	var groups []string
	for _, f := range opts {
		if f.Hidden {
			continue
		}
		params = append(params, Param{Name: f.Flags, Description: decorateOptionDescription(f)})
		groups = append(groups, f.Group)
	}
	if len(params) == 0 {
		return
	}
	// See normalizeGroups: RenderMan reaches here with the raw list, so an
	// ungrouped flag used to be printed under the previous group's heading.
	groups = normalizeGroups(groups, "Other Flags")
	indent := colIndentFor(params, termWidth, minTextColumns)
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
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, o.inline(p.Description), th.Flag)
	}
}

func (a *App) collectVisibleFlags() []Option {
	var allFlags []Option
	for _, f := range a.PersistentOptions {
		if !f.Hidden {
			allFlags = append(allFlags, f)
		}
	}
	for _, f := range a.GlobalFlags {
		if !f.Hidden {
			allFlags = append(allFlags, f)
		}
	}
	return allFlags
}

// specDeclares reports whether spec declares the given long or short name.
// Testing the raw spec for a substring read "--verbose" as declaring "-v" and
// "--host" as declaring "-h", which deleted the --version and --help rows from
// "help flags".
func specDeclares(spec flagSpec, long, short string) bool {
	if long != "" {
		for _, l := range spec.longNames {
			if l == long {
				return true
			}
		}
	}
	if short != "" {
		for _, s := range spec.shortNames {
			if s == short {
				return true
			}
		}
	}
	return false
}

func (a *App) synthesizeStandardFlags(existing []Option) []Option {
	var hasHelp, hasVersion, tookH, tookV bool
	for _, f := range existing {
		spec := parseFlagSpec(f.Flags)
		hasHelp = hasHelp || specDeclares(spec, "help", "")
		hasVersion = hasVersion || specDeclares(spec, "version", "")
		tookH = tookH || specDeclares(spec, "", "h")
		tookV = tookV || specDeclares(spec, "", "v")
	}

	helpGroup := "Help & Information"
	var stdFlags []Option
	if !hasHelp {
		flagsStr := "-h, --help"
		if tookH {
			flagsStr = "--help"
		}
		if a.ExtendedHelpFlag {
			flagsStr += ", -H"
		}
		stdFlags = append(stdFlags, Option{
			Flags:       flagsStr,
			Description: "Show help for command or application",
			Group:       helpGroup,
		})
	}
	if a.Version != "" && !hasVersion {
		versionFlags := "-v, --version"
		if tookV {
			// Something else already answers to -v, so the row offers only the
			// long name rather than disappearing altogether.
			versionFlags = "--version"
		}
		stdFlags = append(stdFlags, Option{
			Flags:       versionFlags,
			Description: "Show application version",
			Group:       helpGroup,
		})
	}
	return stdFlags
}

func normalizeFlagGroups(allFlags, stdFlags []Option) {
	hasAnyGroup := false
	for _, f := range allFlags {
		if f.Group != "" {
			hasAnyGroup = true
			break
		}
	}

	if hasAnyGroup {
		for i := range allFlags {
			if allFlags[i].Group == "" {
				allFlags[i].Group = "General Flags"
			}
		}
	} else if len(allFlags) > 0 {
		for i := range stdFlags {
			stdFlags[i].Group = ""
		}
	}
}

func (a *App) collectRenderFlags() []Option {
	allFlags := a.collectVisibleFlags()
	stdFlags := a.synthesizeStandardFlags(allFlags)
	normalizeFlagGroups(allFlags, stdFlags)
	return append(allFlags, stdFlags...)
}

// RenderFlags writes the dedicated global flags overview: usage template,
// grouped persistent flags, standard help flags, and guidance.
func (a *App) RenderFlags(o Options) {
	o = o.withApp(a)
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		reflowMargin(w, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 0, 8,
			"Usage:", o.inline(a.usageLine()), th.Hdr) // see RenderGlobal

		fmt.Fprintln(w)
		reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", "Global flags available to all commands:")
		fmt.Fprintln(w)

		allFlags := a.collectRenderFlags()
		renderOptionsGrouped(w, th, o, termWidth, allFlags)

		if len(a.Commands) > 0 || len(a.Shortcuts) > 0 {
			fmt.Fprintln(w)
			reflow(w, th.Body, wrapWidth(termWidth, 0, o.maxContent()), 0, "", fmt.Sprintf("Run '%s <command> -h' for command-specific flags.", appName(a)))
		}
	})
}

// RenderGlobalFlags is an alias for RenderFlags.
func (a *App) RenderGlobalFlags(o Options) {
	a.RenderFlags(o)
}

// RenderMan writes an exhaustive, Unix manual-style reference containing the
// full application overview, grouped global options, all command hierarchies,
// parameters, local flags, examples, notes, and help topics.
func (a *App) RenderMan(o Options) {
	o = o.withApp(a)
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		// 1. NAME
		th.Hdr.Fprintln(w, "NAME")
		nameDesc := appName(a)
		if a.Description != "" {
			nameDesc += " - " + a.Description
		}
		reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", o.inline(nameDesc))
		fmt.Fprintln(w)

		// 2. SYNOPSIS
		th.Hdr.Fprintln(w, "SYNOPSIS")
		reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", o.inline(a.usageLine()))
		fmt.Fprintln(w)

		// 3. DESCRIPTION
		if a.Description != "" || a.GlobalNote != "" {
			th.Hdr.Fprintln(w, "DESCRIPTION")
			if a.Description != "" {
				reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", o.inline(a.Description))
			}
			if a.GlobalNote != "" {
				if a.Description != "" {
					fmt.Fprintln(w)
				}
				reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", o.inline(a.GlobalNote))
			}
			fmt.Fprintln(w)
		}

		// 4. GLOBAL FLAGS
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
			th.Hdr.Fprintln(w, "GLOBAL FLAGS")
			renderOptionsGrouped(w, th, o, termWidth, globalFlags)
			fmt.Fprintln(w)
		}

		// 4b. EXAMPLES
		if len(a.Examples) > 0 {
			th.Hdr.Fprintln(w, "EXAMPLES")
			renderExamples(w, a, nil, th, o, termWidth, a.Examples, 4, 6)
			fmt.Fprintln(w)
		}

		// 5. COMMANDS
		if len(a.Commands) > 0 {
			th.Hdr.Fprintln(w, "COMMANDS")
			a.renderManCommands(w, th, o, termWidth, a.Commands, []string{appName(a)})
			fmt.Fprintln(w)
		}

		// 6. HELP TOPICS
		th.Hdr.Fprintln(w, "HELP TOPICS")
		a.renderManTopics(w, th, o, termWidth)
	})
}

// renderManTopics closes the manual with the list of specialised help topics.
func (a *App) renderManTopics(w io.Writer, th Theme, o Options, termWidth int) {
	reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "",
		fmt.Sprintf("Run '%s help <topic>' for specialized documentation topics:", appName(a)))
	fmt.Fprintln(w)
	topics := []Param{
		{Name: "flags", Description: "Show all global flags and persistent options"},
		{Name: "man", Description: "Display this complete reference manual (paged)"},
	}
	// margin 4: the topic list sits under the sentence introducing it.
	indent := clampIndent(colIndent(topics)+2, termWidth, minTextColumns)
	for _, t := range topics {
		reflowMargin(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), 4, indent, t.Name, o.inline(t.Description), th.Subcommand)
	}
}

func (a *App) renderManCommands(w io.Writer, th Theme, o Options, termWidth int, cmds []Command, parentPath []string) {
	for i, c := range cmds {
		if c.Hidden {
			continue
		}
		if i > 0 {
			fmt.Fprintln(w)
		}
		cmdPath := strings.Join(append(parentPath, c.Name), " ")
		th.Subcommand.Fprintf(w, "  %s\n", cmdPath)

		desc := c.LongDescription
		if desc == "" {
			desc = c.Description
		}
		if desc != "" {
			reflow(w, th.Body, wrapWidth(termWidth, 6, o.maxContent()), 6, "", o.inline(desc))
		}

		if len(c.Parameters) > 0 {
			fmt.Fprintln(w)
			th.Hdr.Fprintln(w, "      Parameters:")
			// margin 6: these sit inside the command they belong to, level with
			// the "Parameters:" heading above them.
			indent := clampIndent(colIndent(c.Parameters)+4, termWidth, minTextColumns)
			for _, p := range c.Parameters {
				reflowMargin(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), 6, indent, p.Name, o.inline(p.Description))
			}
		}

		localOpts := a.collectLocalOptions(&c)
		if len(localOpts) > 0 {
			fmt.Fprintln(w)
			th.Hdr.Fprintln(w, "      Flags:")
			optParams := make([]Param, 0, len(localOpts))
			for _, opt := range localOpts {
				desc := opt.Description
				if opt.DefaultText != "" && !strings.Contains(desc, "(default") && !strings.Contains(desc, "[default") {
					desc = desc + " (default: " + opt.DefaultText + ")"
				}
				if opt.Required {
					desc = desc + " (required)"
				}
				if opt.Deprecated != "" {
					desc = desc + " (deprecated: " + opt.Deprecated + ")"
				}
				optParams = append(optParams, Param{Name: opt.Flags, Description: desc})
			}
			indent := clampIndent(colIndent(optParams)+4, termWidth, minTextColumns)
			for _, p := range optParams {
				reflowMargin(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), 6, indent, p.Name, o.inline(p.Description), th.Flag)
			}
		}

		if len(c.Examples) > 0 {
			fmt.Fprintln(w)
			th.Hdr.Fprintln(w, "      Examples:")
			renderExamples(w, a, &c, th, o, termWidth, c.Examples, 8, 10)
		}

		if len(c.Notes) > 0 {
			a.renderManNotes(w, th, o, termWidth, c.Notes)
		}

		if len(c.Subcommands) > 0 {
			fmt.Fprintln(w)
			a.renderManCommands(w, th, o, termWidth, c.Subcommands, append(parentPath, c.Name))
		}
	}
}

func (a *App) renderManNotes(w io.Writer, th Theme, o Options, termWidth int, notes []Note) {
	for _, n := range notes {
		fmt.Fprintln(w)
		if n.Heading != "" {
			th.Hdr.Fprintf(w, "      %s:\n", o.inline(n.Heading))
		}
		renderNoteContent(w, th, o, termWidth, 8, n)
	}
}

// RenderHelpTopics writes the index of available help topics.
func (a *App) RenderHelpTopics(o Options) {
	o = o.withApp(a)
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		th.Hdr.Fprintln(w, "Help Topics:")
		topics := []Param{
			{Name: "help <command>", Description: fmt.Sprintf("Show help for a specific command (or '%s <command> -h')", appName(a))},
			{Name: "help flags", Description: "Show all global flags and persistent options"},
			{Name: "help man", Description: "Display the complete reference manual (paged)"},
		}
		indent := colIndentFor(topics, termWidth, minTextColumns)
		for _, t := range topics {
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, t.Name, o.inline(t.Description), th.Subcommand)
		}
	})
}
