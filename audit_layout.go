package clihelp

import (
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
func auditRows(scope string, params []Param, opts AuditOptions) error {
	if len(params) == 0 {
		return nil
	}
	width := opts.width()
	indent := colIndentFor(params, width, minTextColumns)
	textWidth := wrapWidth(width, indent, Options{}.maxContent()) - indent
	for _, p := range params {
		rendered := renderInlineTo(p.Description, true)
		if strings.Contains(rendered, "\n") {
			return fmt.Errorf("%s: description of %q spans several lines; the short description must be one line", scope, p.Name)
		}
		if w := visualLen(rendered); w > textWidth {
			return fmt.Errorf("%s: description of %q is %d columns, but only %d fit on one row at width %d (shorten it, or move the detail to LongDescription)",
				scope, p.Name, w, textWidth, width)
		}
	}
	return nil
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
				params = append(params, Param{Name: opt.Flags, Description: firstSentence(opt.Description)})
			}
		}
	}
	return params
}

// auditCommandLayout checks the rows a command contributes to its own help: its
// subcommand listing and its flag lists. The command's own row belongs to its
// parent's listing and is checked there.
func auditCommandLayout(cmd Command, path []string, opts AuditOptions) error {
	scope := "command " + strings.Join(path, " ")
	if err := auditRows(scope+" subcommands", commandParams(cmd.Subcommands), opts); err != nil {
		return err
	}
	if err := auditRows(scope+" flags", visibleOptionParams(cmd.PersistentOptions, cmd.Options), opts); err != nil {
		return err
	}
	warnLongText(scope, cmd.Examples, cmd.Notes, opts)
	return nil
}

// auditAppLayout checks the application-level listings.
func auditAppLayout(app *App, opts AuditOptions) error {
	if err := auditRows("the app's commands", commandParams(app.Commands), opts); err != nil {
		return err
	}
	if err := auditRows("the app's shortcuts", commandParams(app.Shortcuts), opts); err != nil {
		return err
	}
	if err := auditRows("the app's flags", visibleOptionParams(app.PersistentOptions, app.GlobalFlags, app.Options), opts); err != nil {
		return err
	}
	warnLongText("the app", app.Examples, nil, opts)
	return nil
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
