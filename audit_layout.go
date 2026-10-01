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
		rendered := renderInlineTo(p.Description, true)
		if w := visualLen(rendered); w > textWidth {
			errs = append(errs, fmt.Errorf("%s: description of %q is %d columns, but only %d fit on one row at width %d (shorten it, or move the detail to LongDescription)",
				scope, p.Name, w, textWidth, width))
		}
	}
	return errs
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

func visibleOptionParams(groups ...[]Option) []Param {
	var params []Param
	for _, options := range groups {
		for _, opt := range options {
			if !opt.Hidden {
				params = append(params, Param{Name: opt.Flags, Description: decorateOptionDescription(opt)})
			}
		}
	}
	return params
}

// auditLayout checks the layout of the whole tree and reports every violation,
// not just the first: an application with ten long descriptions should learn
// that in one run. It is independent of the structural checks, which stop at
// their first error, and audit joins the two.
func auditLayout(app *App, opts AuditOptions) error {
	errs := auditRows("the app's commands", commandParams(app.Commands), opts)
	errs = append(errs, auditRows("the app's shortcuts", commandParams(app.Shortcuts), opts)...)
	errs = append(errs, auditRows("the app's flags", visibleOptionParams(app.PersistentOptions, app.GlobalFlags, app.Options), opts)...)
	warnLongText("the app", app.Examples, nil, opts)
	errs = append(errs, auditCommandsLayout(app.Commands, nil, opts)...)
	errs = append(errs, auditCommandsLayout(app.Shortcuts, nil, opts)...)
	return errors.Join(errs...)
}

// auditCommandsLayout checks what each command contributes to its own help: a
// one-line Description, its subcommand listing and its flag lists. A command's
// own row belongs to its parent's listing and is checked there.
func auditCommandsLayout(cmds []Command, parent []string, opts AuditOptions) []error {
	var errs []error
	for _, cmd := range cmds {
		path := append(append([]string(nil), parent...), cmd.Name)
		scope := "command " + strings.Join(path, " ")
		if strings.Contains(strings.TrimSpace(cmd.Description), "\n") {
			errs = append(errs, fmt.Errorf("%s: Description must be one line; put the rest in LongDescription", scope))
		}
		errs = append(errs, auditRows(scope+" subcommands", commandParams(cmd.Subcommands), opts)...)
		errs = append(errs, auditRows(scope+" flags", visibleOptionParams(cmd.PersistentOptions, cmd.Options), opts)...)
		warnLongText(scope, cmd.Examples, cmd.Notes, opts)
		errs = append(errs, auditCommandsLayout(cmd.Subcommands, path, opts)...)
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
			if w := VisualWidth(line); w > width {
				opts.warn("%s: verbatim note %q has a %d-column line, wider than %d; it will not be wrapped", scope, n.Heading, w, width)
				break
			}
		}
	}
	for _, ex := range examples {
		if w := VisualWidth(ex.Line); w > width {
			opts.warn("%s: example %q is %d columns, wider than %d", scope, ex.Line, w, width)
		}
	}
}
