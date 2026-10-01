package clihelp

import (
	"errors"
	"fmt"
	"strings"
)

// defaultAuditWidth is the terminal width the layout checks assume.
const defaultAuditWidth = 80

func (o AuditOptions) width() int {
	if o.Width > 0 {
		return o.Width
	}
	return defaultAuditWidth
}

func (o AuditOptions) warn(format string, args ...any) {
	if o.Warn != nil {
		o.Warn(fmt.Sprintf(format, args...))
	}
}

// auditRows checks that every row of one two-column listing renders on a single
// line at the audit width. The layout is the renderer's own: the shared
// description column from colIndentFor, and the text width from wrapWidth.
// A name wider than the column puts the description on its own line, so
// the same indent applies either way.
func auditRows(scope string, params []Param, opts AuditOptions) []error {
	if len(params) == 0 {
		return nil
	}
	var errs []error
	width := opts.width()
	indent := colIndentFor(params, width, minTextColumns)
	textWidth := wrapWidth(width, indent, Options{}.maxContent()) - indent
	for _, p := range params {
		if p.Name == examplesFlagSpec {
			continue // the library's own row: it shapes the column but is not the author's to shorten
		}
		rendered := renderInlineTo(p.Description, true)
		if w := visualLen(rendered); w > textWidth {
			errs = append(errs, fmt.Errorf("%s: description of %q is %d columns, but only %d fit on one row at width %d (shorten it, or move the detail to LongDescription)",
				scope, p.Name, w, textWidth, width))
		}
	}
	return errs
}

// styleParams is commandParams without the commands the library supplies: their
// wording is fixed, so it cannot be held to the author's punctuation.
func styleParams(cmds []Command) []Param {
	authored := make([]Command, 0, len(cmds))
	for _, c := range cmds {
		if !c.libraryOwned {
			authored = append(authored, c)
		}
	}
	return commandParams(authored)
}

func commandParams(cmds []Command) []Param {
	params := make([]Param, 0, len(cmds))
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		params = append(params, Param{Name: displayNameWithAliases(c), Description: firstSentence(c.Description)})
	}
	return params
}

// optionParams turns an already-collected option list into the rows the
// renderer draws for it.
func optionParams(options []Option) []Param {
	params := make([]Param, 0, len(options))
	for _, opt := range options {
		if !opt.Hidden {
			params = append(params, Param{Name: opt.Flags, Description: decorateOptionDescription(opt)})
		}
	}
	return params
}

// auditLayout checks the layout of the whole tree and reports every violation,
// not just the first: an application with ten long descriptions should learn
// that in one run. It is independent of the structural checks, which stop at
// their first error, and audit joins the two.
func auditLayout(app *App, opts AuditOptions) error {
	errs := auditListing("the app's commands", commandParams(app.Commands), styleParams(app.Commands), opts)
	errs = append(errs, auditListing("the app's shortcuts", commandParams(app.Shortcuts), styleParams(app.Shortcuts), opts)...)
	errs = append(errs, auditOptionListing("the app's flags", app.rootFlags(), opts)...)
	warnLongText("the app", app.Examples, nil, opts)
	errs = append(errs, auditCommandsLayout(app, app.Commands, nil, opts)...)
	errs = append(errs, auditCommandsLayout(app, app.Shortcuts, nil, opts)...)
	return errors.Join(dedupe(errs)...)
}

// dedupe drops repeated messages. The inherited flags are listed on every
// command page, so one over-long global description would otherwise be reported
// once per command.
func dedupe(errs []error) []error {
	seen := make(map[string]bool, len(errs))
	out := errs[:0:0]
	for _, err := range errs {
		if !seen[err.Error()] {
			seen[err.Error()] = true
			out = append(out, err)
		}
	}
	return out
}

// auditCommandsLayout checks what each command contributes to its own help: a
// one-line Description, its subcommand listing and its flag lists. A command's
// own row belongs to its parent's listing and is checked there.
func auditCommandsLayout(app *App, cmds []Command, parent []string, opts AuditOptions) []error {
	var errs []error
	for _, cmd := range cmds {
		path := append(append([]string(nil), parent...), cmd.Name)
		scope := "command " + strings.Join(path, " ")
		if strings.Contains(strings.TrimSpace(cmd.Description), "\n") {
			errs = append(errs, fmt.Errorf("%s: Description must be one line; put the rest in LongDescription", scope))
		}
		// The page draws explicit SubcommandEntries instead of the real tree when
		// the author supplied any (see SubcommandList), and draws their full text;
		// auditing the tree's rows then would measure something nobody sees.
		subs, subStyle := commandParams(cmd.Subcommands), styleParams(cmd.Subcommands)
		if len(cmd.SubcommandEntries) > 0 {
			subs, subStyle = cmd.SubcommandEntries, cmd.SubcommandEntries
		}
		errs = append(errs, auditListing(scope+" subcommands", subs, subStyle, opts)...)
		errs = append(errs, auditListing(scope+" parameters", cmd.Parameters, cmd.Parameters, opts)...)
		errs = append(errs, auditOptionListing(scope+" flags", app.collectLocalOptions(&cmd), opts)...)
		if !app.OmitGlobalFlagsInCommands {
			errs = append(errs, auditOptionListing("global flags on command pages", app.collectGlobalOptions(path, &cmd), opts)...)
		}
		warnLongText(scope, cmd.Examples, cmd.Notes, opts)
		errs = append(errs, auditCommandsLayout(app, cmd.Subcommands, path, opts)...)
	}
	return errs
}

// warnLongText reports the long-tier text that the renderer cannot wrap:
// verbatim notes and example command lines. Prose is wrapped and has no limit,
// so only these can overflow a terminal, and sometimes they must (a URL, a
// long command), which is why this warns and does not fail.
func warnLongText(scope string, examples []Example, notes []Note, opts AuditOptions) {
	width := opts.width()
	for _, n := range notes {
		if !n.Raw {
			continue
		}
		for _, line := range strings.Split(n.Text, "\n") {
			if w := visualLen(line); w > width {
				opts.warn("%s: verbatim note %q has a %d-column line, wider than %d; it will not be wrapped", scope, n.Heading, w, width)
				break
			}
		}
	}
	for _, ex := range examples {
		if w := visualLen(ex.Line); w > width {
			opts.warn("%s: example %q is %d columns, wider than %d", scope, ex.Line, w, width)
		}
	}
}

// auditListing runs every check on one listing: rows holds all of its entries
// for the one-row rule, and style the ones whose punctuation is the author's.
func auditListing(scope string, rows, style []Param, opts AuditOptions) []error {
	errs := auditRows(scope, rows, opts)
	if err := auditPeriods(scope, style); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// auditPeriods requires one listing to pick a style for its descriptions: all
// end with a full stop, or none do. Which one is the author's choice; a listing
// that mixes them reads as unfinished.
func auditPeriods(scope string, params []Param) error {
	var with, without []string
	for _, p := range params {
		if p.Name == examplesFlagSpec {
			continue
		}
		text := strings.TrimSpace(renderInlineTo(p.Description, true))
		if text == "" {
			continue
		}
		if strings.HasSuffix(text, ".") {
			with = append(with, p.Name)
		} else {
			without = append(without, p.Name)
		}
	}
	if len(with) == 0 || len(without) == 0 {
		return nil
	}
	return fmt.Errorf("%s: descriptions disagree about a trailing period: %s end with one, %s do not (use one style throughout the listing)",
		scope, strings.Join(with, ", "), strings.Join(without, ", "))
}

// auditOptionListing is auditListing for a flag list. The one-row rule measures
// the decorated text the renderer draws; the punctuation rule reads the author's
// own words, since a "(default: x)" suffix would otherwise hide a full stop.
func auditOptionListing(scope string, options []Option, opts AuditOptions) []error {
	raw := make([]Param, 0, len(options))
	for _, opt := range options {
		if !opt.Hidden {
			raw = append(raw, Param{Name: opt.Flags, Description: opt.Description})
		}
	}
	return auditListing(scope, optionParams(options), raw, opts)
}
