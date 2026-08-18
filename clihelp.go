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
	DefaultText string                           // Custom display override for default value
	Hidden      bool                             // Hidden from help and completion output
	Deprecated  string                           // Deprecation notice
	Complete    func(toComplete string) []string // Dynamic shell tab-completion callback
	Binder      func(fs *pflag.FlagSet) error    // Internal pflag flag binder
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
	PersistentOptions []Option
	Commands          []Command
	BeforeRun         func(ctx *Context) error
	AfterRun          func(ctx *Context) error
	Run               func(ctx *Context) error

	// Presentation overrides
	Theme       *Theme
	GlobalFlags []Option
	Shortcuts   []Command
	ConfigPath  string

	// I/O overrides for testing and custom redirection
	Stdout io.Writer
	Stderr io.Writer
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

// LookupCommand traverses the command hierarchy and returns the matching
// Command pointer, or nil if not found. Matches both Name and Aliases.
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
			for _, alias := range currentSlice[i].Aliases {
				if alias == p {
					found = &currentSlice[i]
					break
				}
			}
			if found != nil {
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
