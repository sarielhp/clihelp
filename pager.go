package clihelp

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// pageOutput buffers the rendering performed by fn, then pipes the output to
// $PAGER if o.Pager (or a.Pager) is enabled, output is directed to a terminal, and
// the line count exceeds the terminal height. Otherwise it writes directly to o.out().
func (a *App) pageOutput(o Options, fn func(w io.Writer)) {
	isPagerEnabled := o.Pager
	if a != nil && a.Pager {
		isPagerEnabled = true
	}

	outWriter := o.out()

	var isTerminal bool
	fd := int(os.Stdout.Fd())
	if o.Writer == nil || o.Writer == os.Stdout {
		isTerminal = term.IsTerminal(fd)
	} else if f, ok := o.Writer.(*os.File); ok {
		fd = int(f.Fd())
		isTerminal = term.IsTerminal(fd)
	}

	// When outputting to a terminal, ensure ANSI color escape sequences are preserved
	// during buffering so that the pager receives full color formatting.
	hadNoColor := color.NoColor
	if isTerminal && os.Getenv("NO_COLOR") == "" {
		color.NoColor = false
	}
	defer func() {
		color.NoColor = hadNoColor
	}()

	var buf bytes.Buffer
	fn(&buf)

	if !isPagerEnabled || !isTerminal {
		_, _ = outWriter.Write(buf.Bytes())
		return
	}

	h := o.height()
	lineCount := strings.Count(buf.String(), "\n")
	if h <= 0 || lineCount <= h {
		_, _ = outWriter.Write(buf.Bytes())
		return
	}

	parts := buildPagerArgs(os.Getenv("PAGER"))
	if len(parts) == 0 {
		_, _ = outWriter.Write(buf.Bytes())
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = &buf
	cmd.Stdout = outWriter
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if os.Getenv("LESS") == "" {
		cmd.Env = append(cmd.Env, "LESS=RFX")
	}

	if err := cmd.Run(); err != nil {
		_, _ = outWriter.Write(buf.Bytes())
	}
}

// buildPagerArgs parses the PAGER environment string into executable command parts,
// automatically injecting sensible flags for known pagers (e.g. -R for less to preserve
// ANSI color formatting, -no-linenumbers for moar to avoid line number column clutter).
func buildPagerArgs(pager string) []string {
	if pager == "" {
		pager = "less -R -F -X"
	}

	parts := strings.Fields(pager)
	if len(parts) == 0 {
		return nil
	}

	binName := filepath.Base(parts[0])
	if binName == "less" {
		hasR := false
		for _, arg := range parts[1:] {
			if strings.Contains(arg, "R") || strings.Contains(arg, "r") {
				hasR = true
				break
			}
		}
		if !hasR {
			parts = append(parts, "-R")
		}
	} else if binName == "moar" {
		hasNoLineNumbers := false
		for _, arg := range parts[1:] {
			if strings.Contains(arg, "no-linenumbers") {
				hasNoLineNumbers = true
				break
			}
		}
		if !hasNoLineNumbers {
			parts = append(parts, "-no-linenumbers")
		}
	}

	return parts
}
