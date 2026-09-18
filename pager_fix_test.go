package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakePager writes an executable stand-in for $PAGER whose body is given.
func fakePager(t *testing.T, body string) (path, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "log")
	path = filepath.Join(dir, "fakepager")
	script := "#!/bin/sh\n" + strings.ReplaceAll(body, "$LOG", logPath) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, logPath
}

// H1 — os/exec hands the child a descriptor only when Stdout is an *os.File.
// Wrapped in a countingWriter it always got a pipe, so less saw a non-terminal
// stdout and behaved like cat: the pager has never paged.
//
// Proved directly: the pager reports whether its own stdout is the destination
// file or a pipe, and writes a marker that has to land in that file.
func TestPagerIsGivenTheFileNotAPipe(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	f, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pager, logPath := fakePager(t, `if [ -p /dev/stdout ]; then echo PIPE > $LOG; else echo FILE > $LOG; fi; echo MARKER; cat > /dev/null`)
	if !runPager([]string{pager}, []byte("hello\n"), f, io.Discard) {
		t.Fatal("runPager reported failure")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "MARKER") {
		t.Errorf("the pager's stdout did not reach the destination file: %q", got)
	}

	kind, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("the pager never ran: %v", err)
	}
	if strings.TrimSpace(string(kind)) != "FILE" {
		t.Errorf("the pager was handed a %s, not the destination; a real terminal would reach it as a pipe too",
			strings.TrimSpace(string(kind)))
	}
}

// H5 — a pager that exits 0 having written nothing was reported as success, so
// pageOutput skipped its fallback and the user saw no help at all.
func TestSilentSuccessIsNotSuccess(t *testing.T) {
	var sink bytes.Buffer
	if runPager([]string{"true"}, []byte("the help text\n"), &sink, io.Discard) && sink.Len() == 0 {
		t.Errorf("a pager that exited 0 printing nothing was reported as success; the help is lost")
	}
}

// M9 — the paging decision counted newlines, not screen rows. A page whose
// lines wrap occupies more rows than it has newlines, so help that overflows
// the screen printed unpaged.
func TestScreenRowsCountsWrapping(t *testing.T) {
	for _, tt := range []struct {
		name  string
		text  string
		width int
		want  int
	}{
		{"one short line", "hello\n", 80, 1},
		{"no trailing newline still counts", "hello", 80, 1},
		{"a line exactly the width", strings.Repeat("x", 80) + "\n", 80, 1},
		{"a line one column over", strings.Repeat("x", 81) + "\n", 80, 2},
		{"a line three times the width", strings.Repeat("x", 200) + "\n", 80, 3},
		{"escapes cost no rows", "\x1b[31m" + strings.Repeat("x", 40) + "\x1b[0m\n", 80, 1},
		{"empty", "", 80, 0},
	} {
		if got := screenRows(tt.text, tt.width); got != tt.want {
			t.Errorf("%s: screenRows = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// M8 — a $PAGER that backgrounds anything held the stdout pipe open, and
// cmd.Run waited for it with no bound.
func TestPagerDoesNotWaitForAGrandchild(t *testing.T) {
	var sink bytes.Buffer
	start := time.Now()
	runPager([]string{"sh", "-c", "sleep 5 & exit 0"}, []byte("help\n"), &sink, io.Discard)
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("runPager blocked %v on a grandchild holding the pipe", d)
	}
}

// L11 — PAGER="" is the conventional way to say "do not page".
func TestEmptyPagerMeansNoPager(t *testing.T) {
	for _, tt := range []struct {
		name, value string
		wantPager   bool
	}{
		{"unset means the default", "", true},
		{"explicitly empty means none", "   ", false},
		{"cat means none", "cat", false},
		{"true means none", "true", false},
		{"a real pager", "less", true},
	} {
		got := len(resolvePagerArgs(tt.value)) > 0
		if got != tt.wantPager {
			t.Errorf("%s: PAGER=%q resolved to a pager=%v, want %v", tt.name, tt.value, got, tt.wantPager)
		}
	}
	_ = fmt.Sprint()
}
