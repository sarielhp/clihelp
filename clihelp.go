// Package clihelp provides a declarative, lightweight CLI application framework
// and width-aware, colorized help text formatter for Go applications.
package clihelp

import (
	"context"
	"io"
	"os"

	"github.com/spf13/pflag"
)

// Option represents a command-line flag or option definition.
type Option struct {
	Flags       string                           // e.g. "-p, --podcast <name>"
	Description string                           // e.g. "Podcast title, index, or ID"
	Group       string                           // Category heading (e.g. "Authentication", "Logging")
	DefaultText string                           // Custom display override for default value
	Hidden      bool                             // Hidden from help and completion output
	Deprecated  string                           // Deprecation notice
	Required    bool                             // Required flag constraint
	Complete    func(toComplete string) []string // Dynamic shell tab-completion callback
	Binder      func(fs *pflag.FlagSet) error    // Registers the flag on fs; returns an error on duplicate/help-flag conflicts
}

// Required marks an Option as required.
func Required(opt Option) Option {
	opt.Required = true
	return opt
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

// ArgsValidator validates positional arguments after flag parsing.
type ArgsValidator func(args []string) error

// OptionsValidator validates command-line flags after parsing.
type OptionsValidator func(fs *pflag.FlagSet) error

// Command represents an executable command or category node.
type Command struct {
	Name              string
	Aliases           []string
	Description       string
	UsageLine         string
	Group             string
	Hidden            bool
	PersistentOptions []Option
	Options           []Option
	Subcommands       []Command
	Examples          []Example
	Args              ArgsValidator
	OptionsValidator  OptionsValidator
	PreRun            func(ctx *Context) error
	Run               func(ctx *Context) error
	PostRun           func(ctx *Context) error

	// Presentation / legacy fields
	Title             string
	Parameters        []Param
	SubcommandEntries []Param
	Notes             []Note
}

// App represents the root CLI application.
type App struct {
	Name              string
	Description       string
	Version           string
	GlobalNote        string
	UsageLine         string
	PersistentOptions []Option
	Commands          []Command
	BeforeRun         func(ctx *Context) error
	AfterRun          func(ctx *Context) error
	Run               func(ctx *Context) error

	// AbbrevCommands enables prefix-based command matching. When true, a unique
	// prefix of a command name (or alias) is accepted as a match. When the prefix
	// is ambiguous, an error listing the candidates is returned.
	AbbrevCommands bool
	// Pager enables automatic paging through $PAGER when output exceeds
	// the terminal height. When true, help output is buffered and piped through
	// the pager only when it doesn't fit on one screen.
	Pager bool
	// OmitGlobalFlagsInCommands when true replaces the full list of persistent/global
	// flags in subcommand help with a one-line reference pointing to 'help flags'.
	OmitGlobalFlagsInCommands bool
	// InteractiveFallback enables prompting for missing inputs/flags interactively.
	InteractiveFallback bool

	// Presentation overrides
	Theme       *Theme
	GlobalFlags []Option
	Shortcuts   []Command
	ConfigPath  string

	// I/O overrides for testing and custom redirection
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (a *App) stdin() io.Reader {
	if a.Stdin != nil {
		return a.Stdin
	}
	return os.Stdin
}

// Context encapsulates execution state passed to command handlers and lifecycle hooks.
type Context struct {
	Context context.Context
	App     *App
	Command *Command
	Args    []string
	RawArgs []string
	Stdout  io.Writer
	Stderr  io.Writer
}

func (a *App) stdout() io.Writer {
	if a.Stdout != nil {
		return a.Stdout
	}
	return os.Stdout
}

func (a *App) stderr() io.Writer {
	if a.Stderr != nil {
		return a.Stderr
	}
	return os.Stderr
}

// findCommand searches cmds for name, matching both Name and Aliases. It
// returns the matching Command pointer and its index (or nil, -1).
func findCommand(cmds []Command, name string) (*Command, int) {
	for i := range cmds {
		if cmds[i].Name == name {
			return &cmds[i], i
		}
		for _, alias := range cmds[i].Aliases {
			if alias == name {
				return &cmds[i], i
			}
		}
	}
	return nil, -1
}

// LookupCommand traverses the command hierarchy and returns the matching
// Command pointer, or nil if not found. Matches both Name and Aliases.
func (a *App) LookupCommand(path ...string) *Command {
	if len(path) == 0 {
		return nil
	}
	currentSlice := a.Commands
	var found *Command
	for _, p := range path {
		found, _ = findCommand(currentSlice, p)
		if found == nil {
			return nil
		}
		currentSlice = found.Subcommands
	}
	return found
}

// ancestorsForPath returns all ancestor Command pointers for the given path,
// excluding the final (target) command. Returns nil for top-level commands.
func (a *App) ancestorsForPath(path ...string) []*Command {
	if len(path) <= 1 {
		return nil
	}
	var ancestors []*Command
	currentSlice := a.Commands
	for i := 0; i < len(path)-1; i++ {
		found, _ := findCommand(currentSlice, path[i])
		if found == nil {
			return ancestors
		}
		ancestors = append(ancestors, found)
		currentSlice = found.Subcommands
	}
	return ancestors
}

// collectLocalOptions returns non-hidden command-specific Options for cmd.
func (a *App) collectLocalOptions(cmd *Command) []Option {
	if cmd == nil {
		return nil
	}
	var opts []Option
	for _, o := range cmd.Options {
		if !o.Hidden {
			opts = append(opts, o)
		}
	}
	return opts
}

// collectGlobalOptions returns non-hidden app and ancestor persistent/global options for path and cmd.
func (a *App) collectGlobalOptions(path []string, cmd *Command) []Option {
	var opts []Option
	appendAll := func(optSlice []Option) {
		for _, o := range optSlice {
			if !o.Hidden {
				opts = append(opts, o)
			}
		}
	}
	appendAll(a.PersistentOptions)
	appendAll(a.GlobalFlags)
	for _, anc := range a.ancestorsForPath(path...) {
		appendAll(anc.PersistentOptions)
	}
	if cmd != nil {
		appendAll(cmd.PersistentOptions)
	}
	return opts
}

// collectOptions returns the ordered option set for a command path: app
// PersistentOptions and GlobalFlags, each ancestor's PersistentOptions, then
// the target's PersistentOptions and Options. Hidden options are skipped.
func (a *App) collectOptions(path []string, cmd *Command) []Option {
	var opts []Option
	opts = append(opts, a.collectGlobalOptions(path, cmd)...)
	opts = append(opts, a.collectLocalOptions(cmd)...)
	return opts
}
