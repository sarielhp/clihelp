package clihelp

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
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

	_, isTerminal := o.termFd()

	var buf bytes.Buffer
	fn(&buf)

	if !isPagerEnabled || !isTerminal {
		_, _ = outWriter.Write(buf.Bytes())
		return
	}

	// Rows, not newlines. A page of twenty logical lines that each wrap to three
	// occupies sixty rows and used to count as twenty, so help that overflowed
	// the screen printed unpaged. "< h" rather than "<= h" leaves a row for the
	// shell prompt, which otherwise scrolls the first line away.
	h := o.height()
	if h <= 0 || screenRows(buf.String(), o.width()) < h {
		_, _ = outWriter.Write(buf.Bytes())
		return
	}

	parts := resolvePagerArgs(os.Getenv("PAGER"))
	data := buf.Bytes()
	if len(parts) == 0 || !runPager(parts, data, outWriter, a.stderr()) {
		_, _ = outWriter.Write(data)
	}
}

// screenRows reports how many terminal rows text occupies at the given width: a
// logical line wider than the terminal fills several, and the last line counts
// even without a trailing newline.
func screenRows(text string, width int) int {
	if text == "" {
		return 0
	}
	if width <= 0 {
		width = 80
	}
	rows := 0
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		w := VisualWidth(line)
		if w <= width {
			rows++
			continue
		}
		rows += (w + width - 1) / width
	}
	return rows
}

// resolvePagerArgs turns the $PAGER value into a command, or nothing when the
// user has said not to page.
//
// An explicitly empty PAGER is the conventional way to disable paging — git,
// man and most CLIs honour it — and os.Getenv cannot tell it from unset, so
// PAGER="" used to launch less anyway. PAGER=cat and PAGER=true are the other
// two spellings of the same intent, and both cost a subprocess to achieve
// nothing.
func resolvePagerArgs(pager string) []string {
	trimmed := strings.TrimSpace(pager)
	if pager != "" && trimmed == "" {
		return nil
	}
	switch trimmed {
	case "cat", "true", "/bin/cat", "/bin/true", "/usr/bin/cat", "/usr/bin/true":
		return nil
	}
	return buildPagerArgs(trimmed)
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
func runPager(parts []string, data []byte, out, errOut io.Writer) bool {
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewReader(data)
	// The pager's diagnostics follow App.Stderr like everything else this
	// library prints. Sending them to the process stderr unconditionally painted
	// them over a TUI, and over a test harness's captured output.
	cmd.Stderr = errOut
	cmd.Env = os.Environ()
	if os.Getenv("LESS") == "" {
		cmd.Env = append(cmd.Env, "LESS=RFX")
	}
	// A $PAGER that backgrounds anything keeps the stdout pipe open after it
	// exits, and Cmd.Wait waits for the copy goroutine, which waits for the last
	// holder of that pipe. Without a bound the program hung with nothing on
	// screen. The *os.File path below has no pipe and no goroutine at all.
	cmd.WaitDelay = 2 * time.Second

	// The pager owns Ctrl-C while it runs; see pagerSignals.
	held := make(chan os.Signal, 1)
	signal.Notify(held, pagerSignals()...)
	defer signal.Stop(held)

	if f, ok := out.(*os.File); ok {
		// Hand the pager the terminal. os/exec passes a descriptor straight
		// through only when Stdout is an *os.File; wrapped in anything else the
		// child gets a pipe, sees a non-terminal stdout and degrades to cat — so
		// less never paged, and the whole feature was a fork and a copy.
		cmd.Stdout = f
		err := cmd.Run()
		if err == nil {
			return true
		}
		// A pager that ran and failed has already drawn its output; only a pager
		// that could not be executed leaves the caller to print the help. 126 and
		// 127 are a wrapper script whose own exec failed.
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() != 126 && exit.ExitCode() != 127 {
			return true
		}
		return false
	}

	counted := &countingWriter{w: out}
	cmd.Stdout = counted
	if err := cmd.Run(); err != nil {
		return counted.n > 0
	}
	// Success is not enough: PAGER=true exits 0 having printed nothing, and
	// reporting that as success meant the caller skipped its fallback and the
	// user saw no help at all.
	return counted.n > 0 || len(data) == 0
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
