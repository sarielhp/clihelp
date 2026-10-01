package clihelp

import (
	"io"
	"strings"

	"github.com/fatih/color"
)

// A listing is one two-column table the library draws: names down the left,
// descriptions wrapped beside them — a command list, a flag list, a parameter
// list. It is the one place that decides how wide the name column is and how
// much room a description has, and both the renderer and Audit ask it. They used
// to compute the column and the row width separately, in a dozen places, and
// every disagreement between them was a help page that Audit passed and that
// wrapped anyway.
type listing []Param

// indent is the column the descriptions start in.
func (l listing) indent(termWidth int) int {
	return colIndentFor(l, termWidth, minTextColumns)
}

// wrapTo is the width descriptions are wrapped at.
func (l listing) wrapTo(termWidth, maxContent int) int {
	return wrapWidth(termWidth, l.indent(termWidth), maxContent)
}

// textWidth is the columns a description has before it wraps.
func (l listing) textWidth(termWidth, maxContent int) int {
	return l.wrapTo(termWidth, maxContent) - l.indent(termWidth)
}

// fits reports whether a description, already rendered for display, draws on one
// row. Measure what is drawn, not the markdown it came from: rendering only ever
// shrinks the visible width, so measuring the source was a systematic false
// positive — one link in one description put a blank line between every entry in
// the list, none of which wrapped.
func (l listing) fits(rendered string, termWidth, maxContent int) bool {
	room := l.textWidth(termWidth, maxContent)
	if room <= 0 {
		room = 40 // too narrow to say: assume a plausible column over flagging every row
	}
	return visualLen(rendered) <= room && !strings.Contains(rendered, "\n")
}

// write draws every row, aligned to the shared column. name colours the name
// column, if given.
func (l listing) write(w io.Writer, o Options, termWidth int, body *color.Color, name ...*color.Color) {
	indent, wrap := l.indent(termWidth), l.wrapTo(termWidth, o.maxContent())
	for _, p := range l {
		reflow(w, body, wrap, indent, p.Name, o.inline(p.Description), name...)
	}
}

// commandRows is the list of visible commands as the pages show them — display
// name with aliases, first sentence of the description — and, row for row, each
// command's group.
func commandRows(cmds []Command) (listing, []string) {
	var rows listing
	var groups []string
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		rows = append(rows, Param{Name: displayNameWithAliases(c), Description: firstSentence(c.Description)})
		groups = append(groups, c.Group)
	}
	return rows, groups
}

// optionRows is the list of visible flags as the pages show them — the spec, and
// the description with its default, required and deprecated suffixes — and, row
// for row, each flag's group.
func optionRows(options []Option) (listing, []string) {
	var rows listing
	var groups []string
	for _, opt := range options {
		if opt.Hidden {
			continue
		}
		rows = append(rows, Param{Name: opt.Flags, Description: decorateOptionDescription(opt)})
		groups = append(groups, opt.Group)
	}
	return rows, groups
}

// subcommandRows is the Subcommands section of a command's page: the explicit
// SubcommandEntries when the author supplied any, in full, and otherwise the
// visible subcommands. When entries exist the page draws them instead of the
// tree (see SubcommandList), so nothing else is measured.
func subcommandRows(cmd *Command) listing {
	if len(cmd.SubcommandEntries) > 0 {
		return listing(cmd.SubcommandEntries)
	}
	rows, _ := commandRows(cmd.Subcommands)
	return rows
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

// examplesFlagSpec is the built-in flag EnableExamplesFlag adds to the root page.
const examplesFlagSpec = "-E, --examples"

// rootFlags is the list the root page's "Global Flags" section shows: the
// persistent and global options, the application's own options, and the built-in
// examples flag when enabled. Audit measures the same list.
func (a *App) rootFlags() []Option {
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
	if a.EnableExamplesFlag {
		globalFlags = append(globalFlags, Option{
			Flags:       examplesFlagSpec,
			Description: "Show every example in one place; add a command to narrow it",
		})
	}
	return globalFlags
}
