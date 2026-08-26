package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
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
			name:     "moar automatically receives -no-linenumbers",
			pagerEnv: "moar",
			want:     []string{"moar", "-no-linenumbers"},
		},
		{
			name:     "moar with full path automatically receives -no-linenumbers",
			pagerEnv: "/home/sariel/.go/bin/moar",
			want:     []string{"/home/sariel/.go/bin/moar", "-no-linenumbers"},
		},
		{
			name:     "moar with existing -no-linenumbers flag does not duplicate",
			pagerEnv: "moar -no-linenumbers -wrap",
			want:     []string{"moar", "-no-linenumbers", "-wrap"},
		},
		{
			name:     "moar with double-dash --no-linenumbers does not duplicate",
			pagerEnv: "moar --no-linenumbers",
			want:     []string{"moar", "--no-linenumbers"},
		},
		{
			name:     "custom pager is untouched",
			pagerEnv: "cat",
			want:     []string{"cat"},
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
