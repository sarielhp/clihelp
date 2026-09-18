package tree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	renderTreeTo(w, th, width, a.Commands, "", nil)
	if len(a.Shortcuts) > 0 {
		fmt.Fprintln(w, "\nShortcut Commands:")
		renderTreeTo(w, th, width, a.Shortcuts, "", nil)
	}
}

func computeContBase(prefix string, isLastCmd, hasSubcommands bool) string {
	if hasSubcommands {
		if !isLastCmd {
			return prefix + "│   │   "
		}
		return prefix + "    │   "
	}
	if !isLastCmd {
		return prefix + "│   "
	}
	return prefix + "    "
}

func renderTreeNode(w io.Writer, th clihelp.Theme, width int, cmd clihelp.Command, treeColor, cmdColor *color.Color, prefix, branch string, currentPath []string, isLastCmd bool) {
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

	contBase := computeContBase(prefix, isLastCmd, len(cmd.Subcommands) > 0)

	if cmd.Description == "" {
		fmt.Fprintln(w, firstLineCmd)
		return
	}

	if firstWidth > 24 || remainingWidth < 45 {
		fmt.Fprintln(w, firstLineCmd)
		descPrefix := treeColor.Sprint(contBase) + "  "
		descIndent := visualLen(contBase) + 2
		reflowTree(w, th.Body, descIndent, maxWidth, descPrefix, descPrefix, firstSentence(cmd.Description))
		return
	}

	firstPrefixFormatted := firstLineCmd + "  "
	totalWidth := firstWidth
	contPrefixFormatted := treeColor.Sprint(contBase)
	if rem := totalWidth - visualLen(contBase); rem > 0 {
		contPrefixFormatted += strings.Repeat(" ", rem)
	}
	reflowTree(w, th.Body, totalWidth, maxWidth, firstPrefixFormatted, contPrefixFormatted, firstSentence(cmd.Description))
}

func renderTreeTo(w io.Writer, th clihelp.Theme, width int, commands []clihelp.Command, prefix string, path []string) {
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
		renderTreeNode(w, th, width, cmd, treeColor, cmdColor, prefix, branch, currentPath, isLastCmd)

		if len(cmd.Subcommands) > 0 {
			nextPrefix := prefix + "│   "
			if isLastCmd {
				nextPrefix = prefix + "    "
			}
			renderTreeTo(w, th, width, cmd.Subcommands, nextPrefix, currentPath)
		}
	}
}

// visualLen returns the display column width of s, ignoring ANSI escapes.
//
// It defers to clihelp rather than measuring here. This file used to strip
// escapes with a third-party stripper that does not understand OSC sequences,
// so an OSC 8 hyperlink — which this library emits — measured 22 columns
// instead of 4, and every column beside it was placed wrong.
// Counting bytes here measured every box-drawing glyph the tree draws with as
// three columns instead of one, so descriptions were indented and wrapped as if
// the tree were far wider than it is.
func visualLen(s string) int {
	return clihelp.VisualWidth(s)
}

// firstSentence is clihelp's own, not a copy of it. The copy that used to live
// here checked for ". " before truncating at a line break, so a description
// whose first full stop fell on a later line yielded a string containing
// newlines — and reflowTree below assumes one line.
func firstSentence(s string) string {
	return clihelp.FirstSentence(s)
}

func colorizeTreeWord(c *color.Color, word string) string {
	if c == nil {
		return word
	}
	return c.Sprint(word)
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
		colored := colorizeTreeWord(bodyColor, word)
		space := 0
		if lineHasWords {
			space = 1
		}
		if lineHasWords && curLen+space+wlen > width {
			fmt.Fprintln(w, cur.String())
			cur.Reset()
			cur.WriteString(contPrefixFormatted)
			cur.WriteString(colored)
			curLen = indent + wlen
			lineHasWords = true
		} else {
			if space > 0 {
				cur.WriteString(" ")
				curLen++
			}
			cur.WriteString(colored)
			curLen += wlen
			lineHasWords = true
		}
	}
	if lineHasWords {
		fmt.Fprintln(w, cur.String())
	}
}
