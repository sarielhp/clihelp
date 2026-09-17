package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestPageOutputNonTerminal(t *testing.T) {
	app := &App{
		Name:  "testapp",
		Pager: true,
	}

	var buf bytes.Buffer
	o := Options{
		Writer: &buf,
		Pager:  true,
	}

	app.pageOutput(o, func(w io.Writer) {
		fmt.Fprintln(w, "line 1")
		fmt.Fprintln(w, "line 2")
		fmt.Fprintln(w, "line 3")
	})

	got := buf.String()
	want := "line 1\nline 2\nline 3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOptionsHeightNonTerminal(t *testing.T) {
	var buf bytes.Buffer
	o := Options{Writer: &buf}
	if h := o.height(); h != 0 {
		t.Errorf("expected 0 for non-terminal writer, got %d", h)
	}
}

func TestPageOutputWithCustomPager(t *testing.T) {
	// Pager uses $PAGER if set
	origPager := os.Getenv("PAGER")
	defer os.Setenv("PAGER", origPager)

	os.Setenv("PAGER", "cat")

	app := &App{Name: "testapp", Pager: true}
	var buf bytes.Buffer
	o := Options{Writer: &buf, Pager: true}

	app.pageOutput(o, func(w io.Writer) {
		for i := 0; i < 50; i++ {
			fmt.Fprintf(w, "row %d\n", i)
		}
	})

	if !strings.Contains(buf.String(), "row 0") || !strings.Contains(buf.String(), "row 49") {
		t.Errorf("expected output to contain all rows, got:\n%s", buf.String())
	}
}

func TestBuildPagerArgs(t *testing.T) {
	tests := []struct {
		name     string
		pagerEnv string
		want     []string
	}{
		{
			name:     "default empty uses less with -R -F -X",
			pagerEnv: "",
			want:     []string{"less", "-R", "-F", "-X"},
		},
		{
			name:     "less without -R appends -R",
			pagerEnv: "less -F -X",
			want:     []string{"less", "-F", "-X", "-R"},
		},
		{
			name:     "less with full path and -R preserved",
			pagerEnv: "/usr/bin/less -R",
			want:     []string{"/usr/bin/less", "-R"},
		},
		{
			name:     "custom pager is untouched",
			pagerEnv: "cat",
			want:     []string{"cat"},
		},
		{
			name:     "custom pager with args preserved",
			pagerEnv: "moar -no-linenumbers",
			want:     []string{"moar", "-no-linenumbers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPagerArgs(tt.pagerEnv)
			if len(got) != len(tt.want) {
				t.Fatalf("buildPagerArgs(%q) returned %d args %v, want %d args %v", tt.pagerEnv, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildPagerArgs(%q)[%d] = %q, want %q", tt.pagerEnv, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConcurrentRender(t *testing.T) {
	app := &App{
		Name:        "testapp",
		Description: "A concurrent test app",
		Commands: []Command{
			{
				Name:        "run",
				Description: "Run the task",
			},
			{
				Name:        "status",
				Description: "Show status",
			},
		},
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			var buf bytes.Buffer
			opts := Options{Writer: &buf}
			if id%2 == 0 {
				app.RenderGlobal(opts)
			} else {
				app.RenderCommand(opts, "run")
			}
			if buf.Len() == 0 {
				t.Errorf("goroutine %d got empty render output", id)
			}
		}(i)
	}

	wg.Wait()
}

// TestRunPagerFallbackKeepsOutput covers the failure the 2026-09-17 review
// measured: os/exec drains an io.Reader stdin as soon as the child starts, so a
// buffer that doubles as the fallback copy is empty by the time a failing pager
// returns.
func TestRunPagerFallbackKeepsOutput(t *testing.T) {
	small := []byte(strings.Repeat("help line\n", 60))
	large := []byte(strings.Repeat("help line\n", 20000)) // past the pipe buffer

	for _, tt := range []struct {
		name    string
		pager   []string
		data    []byte
		want    bool
		wantOut string
	}{
		{"pager fails immediately", []string{"false"}, small, false, ""},
		{"pager fails on a large payload", []string{"false"}, large, false, ""},
		{"pager succeeds", []string{"cat"}, small, true, string(small)},
		{"pager writes then fails", []string{"sh", "-c", "head -c 9; exit 3"}, small, true, "help line"},
		{"pager does not exist", []string{"no-such-pager-binary"}, small, false, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := runPager(tt.pager, tt.data, &buf)
			if got != tt.want {
				t.Errorf("runPager = %v, want %v", got, tt.want)
			}
			if buf.String() != tt.wantOut {
				t.Errorf("pager wrote %d bytes, want %d", buf.Len(), len(tt.wantOut))
			}
			if !got && buf.Len() != 0 {
				t.Errorf("a failed pager must leave the fallback the whole output, wrote %d bytes", buf.Len())
			}
		})
	}
}

func TestBuildPagerArgsRawDetection(t *testing.T) {
	for _, tt := range []struct {
		pagerEnv string
		wantLast string
	}{
		{"less --clear-screen", "-R"},
		{"less --raw-control-chars", "--raw-control-chars"},
		{"less -RF", "-RF"},
		{"less -N", "-R"},
	} {
		t.Run(tt.pagerEnv, func(t *testing.T) {
			got := buildPagerArgs(tt.pagerEnv)
			if last := got[len(got)-1]; last != tt.wantLast {
				t.Errorf("buildPagerArgs(%q) = %v, want it to end in %q", tt.pagerEnv, got, tt.wantLast)
			}
		})
	}
}
