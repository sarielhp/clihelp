package clihelp

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"golang.org/x/term"
)

type Option struct {
	Flags       string
	Description string
}

type Example struct {
	Line        string
	Description string
}

type Command struct {
	Name        string
	Description string
	UsageLine   string
	Options     []Option
	Examples    []Example
}

type App struct {
	Name        string
	Description string
	GlobalNote  string
	Commands    []Command
}

var (
	section = color.New(color.FgYellow, color.Bold).Sprint
	label   = color.New(color.FgCyan).Sprint
	emph    = color.New(color.Bold).Sprint
)

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 40 {
		return 80
	}
	return w
}

func stripANCI(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '\033' {
			in = true
			continue
		}
		if in {
			if r == 'm' {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func visualLen(s string) int {
	return len([]rune(stripANCI(s)))
}

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

func PrintSection(name string) {
	fmt.Printf("\n%s\n", section(name))
}

func describeLabel(left, right string, leftWidth int) {
	visLeft := visualLen(left)
	indent := visLeft + 2
	if indent < leftWidth+2 {
		indent = leftWidth + 2
	}

	w := termWidth()
	avail := w - 2
	oneline := fmt.Sprintf("  %-*s%s", leftWidth, left, right)

	if visualLen(oneline) <= avail {
		fmt.Println(oneline)
		return
	}

	descWidth := w - indent
	wrapped := wrapText(right, descWidth)
	fmt.Printf("  %-*s\n%s\n", leftWidth, left, indentLines(wrapped, indent))
}

func describeFlags(flags, desc string) {
	w := termWidth()
	indent := 32
	flagLine := fmt.Sprintf("  %-27s ", label(flags))
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

func PrintGlobalUsage(a *App) {
	fmt.Printf("\n%s\n\n", section("USAGE"))
	fmt.Printf("  %s\n", a.Name+" [command] [options]")

	PrintSection("COMMANDS")
	for _, c := range a.Commands {
		describeLabel(emph(c.Name), c.Description, 36)
	}

	if a.GlobalNote != "" {
		avail := termWidth() - 2
		fmt.Printf("\n%s\n", indentLines(wrapText(a.GlobalNote, avail), 2))
	}
}

func PrintCommandUsage(a *App, cmdName string) bool {
	var cmd *Command
	for i := range a.Commands {
		if a.Commands[i].Name == cmdName {
			cmd = &a.Commands[i]
			break
		}
	}
	if cmd == nil {
		return false
	}

	fmt.Printf("\n%s\n", section(a.Name+" "+cmd.Name+" - "+cmd.Description))

	PrintSection("USAGE")
	fmt.Printf("  %s\n", cmd.UsageLine)

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

func clean(str string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range str {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
