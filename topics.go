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
		groups = append(groups, f.Group)
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
		reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
	}
}

// RenderFlags writes the dedicated global flags overview: usage template,
// grouped persistent flags, standard help flags, and guidance.
func (a *App) RenderFlags(o Options) {
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		th.Hdr.Fprint(w, "Usage:  ")
		fmt.Fprintln(w, a.usageLine())

		fmt.Fprintln(w)
		th.Body.Fprintln(w, "Global flags available to all commands:")
		fmt.Fprintln(w)

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

		hasHelp := false
		hasVersion := false
		for _, f := range allFlags {
			if strings.Contains(f.Flags, "--help") || strings.Contains(f.Flags, "-h") {
				hasHelp = true
			}
			if strings.Contains(f.Flags, "--version") || strings.Contains(f.Flags, "-v") {
				hasVersion = true
			}
		}

		helpGroup := "Help & Information"
		var stdFlags []Option
		if !hasHelp {
			stdFlags = append(stdFlags, Option{
				Flags:       "-h, --help",
				Description: "Show help for command or application",
				Group:       helpGroup,
			})
		}
		if a.Version != "" && !hasVersion {
			stdFlags = append(stdFlags, Option{
				Flags:       "-v, --version",
				Description: "Show application version",
				Group:       helpGroup,
			})
		}

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

		allFlags = append(allFlags, stdFlags...)

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
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		// 1. NAME
		th.Hdr.Fprintln(w, "NAME")
		nameDesc := appName(a)
		if a.Description != "" {
			nameDesc += " - " + a.Description
		}
		reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", inline(nameDesc))
		fmt.Fprintln(w)

		// 2. SYNOPSIS
		th.Hdr.Fprintln(w, "SYNOPSIS")
		reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", inline(a.usageLine()))
		fmt.Fprintln(w)

		// 3. DESCRIPTION
		if a.Description != "" || a.GlobalNote != "" {
			th.Hdr.Fprintln(w, "DESCRIPTION")
			if a.Description != "" {
				reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", inline(a.Description))
			}
			if a.GlobalNote != "" {
				if a.Description != "" {
					fmt.Fprintln(w)
				}
				reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", inline(a.GlobalNote))
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

		// 5. COMMANDS
		if len(a.Commands) > 0 {
			th.Hdr.Fprintln(w, "COMMANDS")
			a.renderManCommands(w, th, o, termWidth, a.Commands, []string{appName(a)})
			fmt.Fprintln(w)
		}

		// 6. HELP TOPICS
		th.Hdr.Fprintln(w, "HELP TOPICS")
		reflow(w, th.Body, wrapWidth(termWidth, 4, o.maxContent()), 4, "", fmt.Sprintf("Run '%s help <topic>' for specialized documentation topics:", appName(a)))
		fmt.Fprintln(w)
		topics := []Param{
			{Name: "flags", Description: "Show all global flags and persistent options"},
			{Name: "tree", Description: "Display the full command hierarchy tree"},
			{Name: "man", Description: "Display this complete reference manual (paged)"},
		}
		indent := colIndent(topics) + 4
		for _, t := range topics {
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, t.Name, inline(t.Description), th.Subcommand)
		}
	})
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

		if c.Description != "" {
			reflow(w, th.Body, wrapWidth(termWidth, 6, o.maxContent()), 6, "", inline(c.Description))
		}

		if len(c.Parameters) > 0 {
			fmt.Fprintln(w)
			th.Hdr.Fprintln(w, "      Parameters:")
			indent := colIndent(c.Parameters) + 6
			for _, p := range c.Parameters {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description))
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
			indent := colIndent(optParams) + 6
			for _, p := range optParams {
				reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, p.Name, inline(p.Description), th.Flag)
			}
		}

		if len(c.Examples) > 0 {
			fmt.Fprintln(w)
			th.Hdr.Fprintln(w, "      Examples:")
			for _, ex := range c.Examples {
				reflow(w, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 8, "", inline(ex.Line))
				if ex.Description != "" {
					reflow(w, th.Body, wrapWidth(termWidth, 10, o.maxContent()), 10, "", inline(ex.Description))
				}
			}
		}

		if len(c.Notes) > 0 {
			for _, n := range c.Notes {
				fmt.Fprintln(w)
				if n.Heading != "" {
					th.Hdr.Fprintf(w, "      %s:\n", n.Heading)
				}
				reflow(w, th.Body, wrapWidth(termWidth, 8, o.maxContent()), 8, "", inline(n.Text))
			}
		}

		if len(c.Subcommands) > 0 {
			fmt.Fprintln(w)
			a.renderManCommands(w, th, o, termWidth, c.Subcommands, append(parentPath, c.Name))
		}
	}
}

// RenderHelpTopics writes the index of available help topics.
func (a *App) RenderHelpTopics(o Options) {
	a.pageOutput(o, func(w io.Writer) {
		th := o.theme(a)
		termWidth := o.width()

		th.Hdr.Fprintln(w, "Help Topics:")
		topics := []Param{
			{Name: "help <command>", Description: fmt.Sprintf("Show help for a specific command (or '%s <command> -h')", appName(a))},
			{Name: "help flags", Description: "Show all global flags and persistent options"},
			{Name: "help tree", Description: "Display the full command hierarchy tree"},
			{Name: "help man", Description: "Display the complete reference manual (paged)"},
		}
		indent := colIndent(topics)
		for _, t := range topics {
			reflow(w, th.Body, wrapWidth(termWidth, indent, o.maxContent()), indent, t.Name, inline(t.Description), th.Subcommand)
		}
	})
}
