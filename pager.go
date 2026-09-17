package clihelp

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	data := buf.Bytes()
	if len(parts) == 0 || !runPager(parts, data, outWriter) {
		_, _ = outWriter.Write(data)
	}
}

// countingWriter records how much of the pager's output reached the user, which
// is what tells a pager that never ran apart from one that printed the help and
// then exited non-zero.
type countingWriter struct {
	w io.Writer
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += n
	return n, err
}

// runPager pipes data through the pager and reports whether the output reached
// the user; when it returns false the caller must print data itself.
//
// The pager reads from its own reader over data rather than from the buffer the
// help was rendered into: os/exec drains an io.Reader stdin in a copy goroutine
// as soon as the child starts, so by the time a failing pager returned, the
// buffer was empty and the fallback printed nothing at all. A pager that failed
// after writing is not repeated, since its output is already on screen.
func runPager(parts []string, data []byte, out io.Writer) bool {
	counted := &countingWriter{w: out}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = counted
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if os.Getenv("LESS") == "" {
		cmd.Env = append(cmd.Env, "LESS=RFX")
	}

	if err := cmd.Run(); err != nil {
		return counted.n > 0
	}
	return true
}

// buildPagerArgs parses the PAGER environment string into executable command parts,
// ensuring -R is present for less to preserve ANSI color formatting.
func buildPagerArgs(pager string) []string {
	if pager == "" {
		pager = "less -R -F -X"
	}

	parts := strings.Fields(pager)
	if len(parts) == 0 {
		return nil
	}

	binName := filepath.Base(parts[0])
	if binName == "less" && !hasRawControlChars(parts[1:]) {
		parts = append(parts, "-R")
	}

	return parts
}

// hasRawControlChars reports whether less is already told to pass ANSI sequences
// through. Only option letters count: searching the whole argument for an "r"
// found one in "--clear-screen" and left the colors to be printed as escapes.
func hasRawControlChars(args []string) bool {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			if strings.EqualFold(arg, "--raw-control-chars") {
				return true
			}
			continue
		}
		if strings.ContainsAny(arg, "Rr") {
			return true
		}
	}
	return false
}
