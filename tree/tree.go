package tree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
	"github.com/sarielhp/clihelp"
)

// Options controls tree rendering.
type Options struct {
	// Writer is where output is written. Defaults to os.Stdout.
	Writer io.Writer
	// Width is the target column width. Defaults to 80.
	Width int
	// Theme customizes colors.
	Theme clihelp.Theme
}

// Render writes a tree view of the command hierarchy of a to w.
func Render(w io.Writer, a *clihelp.App, opts ...Options) {
	if a == nil || w == nil {
		return
	}
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	width := o.Width
	if width <= 0 {
		width = 80
	}

	name := a.Name
	if name == "" {
		name = filepath.Base(os.Args[0])
	}
	th := o.Theme
	treeColor := th.Subcommand
	if treeColor == nil {
		treeColor = color.New(color.FgGreen)
	}
	treeColor.Fprintln(w, name)
	renderTreeTo(w, th, width, a.Commands, "", nil, false)
	if len(a.Shortcuts) > 0 {
		fmt.Fprintln(w, "\nShortcut Commands:")
		renderTreeTo(w, th, width, a.Shortcuts, "", nil, true)
	}
}

func renderTreeTo(w io.Writer, th clihelp.Theme, width int, commands []clihelp.Command, prefix string, path []string, isLast bool) {
	if len(commands) == 0 {
		return
	}

	var visible []clihelp.Command
	for _, c := range commands {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}

	treeColor := th.Subcommand
	if treeColor == nil {
		treeColor = color.New(color.FgGreen)
	}
	cmdColor := th.Hdr
	if cmdColor == nil {
		cmdColor = color.New(color.FgYellow)
	}

	for i, cmd := range visible {
		isLastCmd := (i == len(visible)-1)
		branch := "├── "
		if isLastCmd {
			branch = "└── "
		}

		currentPath := append(append([]string(nil), path...), cmd.Name)
		label := strings.Join(currentPath, " ")
		if len(cmd.Aliases) > 0 {
			label += fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases, ", "))
		}

		treeBranch := prefix + branch
		firstLineCmd := treeColor.Sprint(treeBranch) + cmdColor.Sprint(label)
		rawFirstPrefix := treeBranch + label + "  "
		firstWidth := visualLen(rawFirstPrefix)
		maxWidth := width
		remainingWidth := maxWidth - firstWidth

		var contBase string
		if len(cmd.Subcommands) > 0 {
			if !isLastCmd {
				contBase = prefix + "│   │   "
			} else {
				contBase = prefix + "    │   "
			}
		} else {
			if !isLastCmd {
				contBase = prefix + "│   "
			} else {
				contBase = prefix + "    "
			}
		}

		if cmd.Description == "" {
			fmt.Fprintln(w, firstLineCmd)
		} else if firstWidth > 24 || remainingWidth < 45 {
			fmt.Fprintln(w, firstLineCmd)
			descPrefix := treeColor.Sprint(contBase) + "  "
			descIndent := visualLen(contBase) + 2
			reflowTree(w, th.Body, descIndent, maxWidth, descPrefix, descPrefix, firstSentence(cmd.Description))
		} else {
			firstPrefixFormatted := firstLineCmd + "  "
			totalWidth := firstWidth
			contPrefixFormatted := treeColor.Sprint(contBase)
			if rem := totalWidth - visualLen(contBase); rem > 0 {
				contPrefixFormatted += strings.Repeat(" ", rem)
			}
			reflowTree(w, th.Body, totalWidth, maxWidth, firstPrefixFormatted, contPrefixFormatted, firstSentence(cmd.Description))
		}

		if len(cmd.Subcommands) > 0 {
			nextPrefix := prefix + "│   "
			if isLastCmd {
				nextPrefix = prefix + "    "
			}
			renderTreeTo(w, th, width, cmd.Subcommands, nextPrefix, currentPath, isLastCmd)
		}
	}
}

func visualLen(s string) int {
	return len(stripansi.Strip(s))
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, ". "); idx != -1 {
		return s[:idx+1]
	}
	if strings.HasSuffix(s, ".") {
		return s
	}
	if idx := strings.IndexByte(s, '\n'); idx != -1 {
		return s[:idx]
	}
	return s
}

func reflowTree(w io.Writer, bodyColor *color.Color, indent, width int, firstPrefixFormatted, contPrefixFormatted, text string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		fmt.Fprintln(w, strings.TrimRight(firstPrefixFormatted, " "))
		return
	}

	var cur strings.Builder
	cur.WriteString(firstPrefixFormatted)
	curLen := indent
	lineHasWords := false

	for _, word := range words {
		wlen := visualLen(word)
		space := 0
		if lineHasWords {
			space = 1
		}
		if lineHasWords && curLen+space+wlen > width {
			fmt.Fprintln(w, cur.String())
			cur.Reset()
			cur.WriteString(contPrefixFormatted)
			cur.WriteString(word)
			curLen = indent + wlen
			lineHasWords = true
		} else {
			if space > 0 {
				cur.WriteString(" ")
				curLen++
			}
			cur.WriteString(word)
			curLen += wlen
			lineHasWords = true
		}
	}
	if lineHasWords {
		fmt.Fprintln(w, cur.String())
	}
}
