// Package clihelp provides width-aware, colorized CLI help text formatting
// for Go applications with single or multi-level subcommand hierarchies.
package clihelp

import (
	"fmt"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
	"golang.org/x/term"
)

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

// Command represents a single CLI command or subcommand.
// Commands can contain nested Subcommands recursively (e.g., "config set location").
type Command struct {
	Name        string
	Description string
	UsageLine   string
	Options     []Option
	Examples    []Example
	Subcommands []Command
}

// App represents a top-level CLI application containing application metadata,
// registered commands, and global notes.
type App struct {
	Name        string
	Description string
	GlobalNote  string
	Commands    []Command
}

var (
	section = color.New(color.FgYellow, color.Bold).Sprint
	command = color.New(color.FgGreen, color.Bold).Sprint
	label   = color.New(color.FgCyan).Sprint
)

// termWidth returns the current stdout terminal width, defaulting to 80 cols when non-TTY or small.
func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 40 {
		return 80
	}
	return w
}

// stripANSI removes ANSI control escape codes using the stripansi package.
func stripANSI(s string) string {
	return stripansi.Strip(s)
}

// visualLen calculates the visible character length of a string, ignoring ANSI control codes.
func visualLen(s string) int {
	return len([]rune(stripANSI(s)))
}

// formatPadded pads string s to the target visual width, taking ANSI control codes into account.
func formatPadded(s string, width int) string {
	vis := visualLen(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// wrapText wraps string text into lines of at most avail visible characters.
func wrapText(text string, avail int) string {
	if avail < 20 {
		avail = 20
	}

	var buf strings.Builder
	chars := []rune(text)
	lineStart := 0

	for i := 0; i < len(chars); {
		if chars[i] == '\n' {
			buf.WriteString(string(chars[lineStart : i+1]))
			i++
			lineStart = i
			continue
		}
		segment := string(chars[lineStart : i+1])
		if visualLen(segment) > avail && lineStart < i {
			prevSegment := string(chars[lineStart:i])
			brk := strings.LastIndex(prevSegment, " ")
			if brk < 0 {
				buf.WriteString(string(chars[lineStart:i]))
				buf.WriteByte('\n')
				lineStart = i
			} else {
				buf.WriteString(string(chars[lineStart : lineStart+brk]))
				buf.WriteByte('\n')
				lineStart = lineStart + brk + 1
			}
			continue
		}
		i++
	}
	if lineStart < len(chars) {
		buf.WriteString(string(chars[lineStart:]))
	}
	return buf.String()
}

// indentLines indents each line in s by specified spaces.
func indentLines(s string, indent int) string {
	pad := strings.Repeat(" ", indent)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// PrintSection prints a yellow bold section header (e.g. USAGE, COMMANDS, OPTIONS).
func PrintSection(name string) {
	fmt.Printf("\n%s\n", section(name))
}

func describeLabel(left, right string, leftWidth int) {
	paddedLeft := formatPadded(left, leftWidth)
	visLeftLen := 2 + leftWidth

	w := termWidth()
	oneline := fmt.Sprintf("  %s%s", paddedLeft, right)

	if visualLen(oneline) <= w {
		fmt.Println(oneline)
		return
	}

	descWidth := w - visLeftLen
	if descWidth < 20 {
		fmt.Printf("  %s\n%s\n", paddedLeft, indentLines(wrapText(right, w-4), 4))
		return
	}
	wrapped := wrapText(right, descWidth)
	lines := strings.Split(wrapped, "\n")
	fmt.Printf("  %s%s\n", paddedLeft, lines[0])
	for _, l := range lines[1:] {
		if l != "" {
			fmt.Printf("%s%s\n", strings.Repeat(" ", visLeftLen), l)
		}
	}
}

func describeFlags(flags, desc string) {
	w := termWidth()
	indent := 32
	lbl := label(flags)
	paddedFlags := formatPadded(lbl, 27)
	flagLine := fmt.Sprintf("  %s ", paddedFlags)
	visFlagsLen := visualLen(flagLine)

	oneline := flagLine + desc
	if visualLen(oneline) <= w {
		fmt.Println(oneline)
		return
	}

	descWidth := w - visFlagsLen
	if descWidth < 20 {
		fmt.Printf("%s\n%s", flagLine, indentLines(wrapText(desc, w-indent), indent))
		return
	}
	wrapped := wrapText(desc, descWidth)
	lines := strings.Split(wrapped, "\n")
	fmt.Printf("%s%s\n", flagLine, lines[0])
	for _, l := range lines[1:] {
		if l != "" {
			fmt.Printf("%s%s\n", strings.Repeat(" ", visFlagsLen), l)
		}
	}
}

// PrintGlobalUsage prints the top-level application help overview.
func (a *App) PrintGlobalUsage() {
	fmt.Printf("\n%s\n\n", section("USAGE"))
	fmt.Printf("  %s\n", a.Name+" [command] [options]")

	PrintSection("COMMANDS")
	for _, c := range a.Commands {
		describeLabel(command(c.Name), c.Description, 24)
	}

	if a.GlobalNote != "" {
		avail := termWidth() - 2
		fmt.Printf("\n%s\n", indentLines(wrapText(a.GlobalNote, avail), 2))
	}
}

// LookupCommand traverses the command hierarchy and returns the matching Command pointer, or nil if not found.
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

// PrintCommandUsage prints help text for a specific command or nested subcommand path (e.g. "config", "set", "location").
// Returns true if the command path was found, or false if not found.
func (a *App) PrintCommandUsage(path ...string) bool {
	if len(path) == 0 {
		return false
	}

	var cmd *Command
	currentSlice := a.Commands
	fullPath := []string{a.Name}

	for _, p := range path {
		var found *Command
		for i := range currentSlice {
			if currentSlice[i].Name == p {
				found = &currentSlice[i]
				break
			}
		}
		if found == nil {
			return false
		}
		cmd = found
		fullPath = append(fullPath, cmd.Name)
		currentSlice = cmd.Subcommands
	}

	cmdFullName := strings.Join(fullPath, " ")
	fmt.Printf("\n%s\n", section(cmdFullName+" - "+cmd.Description))

	PrintSection("USAGE")
	fmt.Printf("  %s\n", cmd.UsageLine)

	if len(cmd.Subcommands) > 0 {
		PrintSection("COMMANDS")
		for _, sub := range cmd.Subcommands {
			describeLabel(command(sub.Name), sub.Description, 24)
		}
	}

	if len(cmd.Options) > 0 {
		PrintSection("OPTIONS")
		for _, o := range cmd.Options {
			describeFlags(o.Flags, o.Description)
		}
	}

	if len(cmd.Examples) > 0 {
		PrintSection("EXAMPLES")
		for _, e := range cmd.Examples {
			fmt.Printf("  %s\n", e.Line)
		}
	}

	return true
}

// PrintUsage prints global application help if path is empty, or command help if a command path is provided.
func (a *App) PrintUsage(path ...string) bool {
	if len(path) == 0 {
		a.PrintGlobalUsage()
		return true
	}
	return a.PrintCommandUsage(path...)
}

// PrintGlobalUsage prints top-level application help overview for App a.
func PrintGlobalUsage(a *App) {
	a.PrintGlobalUsage()
}

// PrintCommandUsage prints help text for a command path within App a.
func PrintCommandUsage(a *App, path ...string) bool {
	return a.PrintCommandUsage(path...)
}
