// Package clihelp provides width-aware, colorized CLI help text formatting
// for Go applications with single or multi-level subcommand hierarchies.
//
// Building output is done through a single, theme-driven renderer exposed on
// *App: (Render, RenderGlobal, RenderCommand). Everything writes to a caller
// supplied io.Writer and honors a configurable Theme and terminal width, so
// output is deterministic and easy to test.
package clihelp

// Option represents a command-line flag or option definition (e.g., "-o, --output PATH").
type Option struct {
	Flags       string
	Description string
}

// Example represents a usage line demonstration in command help text.
type Example struct {
	Line        string
	Description string
}

// Param describes a positional argument (or a listable subcommand/flag) with
// a display name and a free-form description.
type Param struct {
	Name        string
	Description string
}

// Note carries an optional heading (rendered as a section label) and a body
// of prose that is reflowed to the available width.
type Note struct {
	Heading string
	Text    string
}

// Command represents a single CLI command or subcommand.
// Commands can contain nested Subcommands recursively (e.g., "config set location").
type Command struct {
	Name        string
	Description string
	UsageLine   string
	Options     []Option
	Examples    []Example
	Subcommands []Command

	// Title is the explicit header shown by the renderer (e.g. "rule export
	// [force]"). When empty, the command name is used.
	Title string
	// Aliases lists alternate names displayed in parentheses, e.g. "archive (arc)".
	Aliases []string
	// Parameters describes positional arguments.
	Parameters []Param
	// SubcommandEntries is the display list used by the Subcommands section.
	// When empty, it falls back to the Subcommands tree.
	SubcommandEntries []Param
	// Notes holds optional prose sections with an optional heading.
	Notes []Note
}

// App represents a top-level CLI application containing application metadata,
// registered commands, and global notes.
type App struct {
	Name        string
	Description string
	GlobalNote  string
	Commands    []Command

	// Theme overrides the renderer colors. When nil a mail_cli-compatible
	// default theme is used.
	Theme *Theme
	// GlobalFlags are shown in the global overview.
	GlobalFlags []Option
	// Shortcuts are alias-level commands shown in their own global section.
	Shortcuts []Command
	// Version is displayed in the global overview.
	Version string
	// ConfigPath is displayed in the global overview.
	ConfigPath string
}

// LookupCommand traverses the command hierarchy and returns the matching
// Command pointer, or nil if not found.
func (a *App) LookupCommand(path ...string) *Command {
	if len(path) == 0 {
		return nil
	}
	currentSlice := a.Commands
	var found *Command
	for _, p := range path {
		found = nil
		for i := range currentSlice {
			if currentSlice[i].Name == p {
				found = &currentSlice[i]
				break
			}
		}
		if found == nil {
			return nil
		}
		currentSlice = found.Subcommands
	}
	return found
}
