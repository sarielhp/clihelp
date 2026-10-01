package clihelp

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestListingArithmetic(t *testing.T) {
	rows := listing{{Name: "build", Description: "Build"}, {Name: "deploy", Description: "Deploy"}}
	// The column is the widest name that fits, plus four.
	if got := rows.indent(80); got != len("deploy")+4 {
		t.Errorf("indent = %d, want %d", got, len("deploy")+4)
	}
	// Descriptions wrap at indent+maxContent but never past the terminal.
	if got := rows.wrapTo(80, 80); got != 80 {
		t.Errorf("wrapTo(80, 80) = %d, want the terminal width", got)
	}
	if got := rows.wrapTo(200, 80); got != rows.indent(200)+80 {
		t.Errorf("wrapTo(200, 80) = %d, want indent+80", got)
	}
	if got := rows.textWidth(80, 80); got != 80-rows.indent(80) {
		t.Errorf("textWidth = %d", got)
	}
}

func TestListingFits(t *testing.T) {
	rows := listing{{Name: "build", Description: "x"}}
	room := rows.textWidth(80, 80)
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"exactly the room", strings.Repeat("x", room), true},
		{"one past", strings.Repeat("x", room+1), false},
		{"a line break", "two\nlines", false},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rows.fits(tt.text, 80, 80); got != tt.want {
				t.Errorf("fits(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestListingWriteAlignsEveryRow(t *testing.T) {
	rows := listing{{Name: "a", Description: "first"}, {Name: "longer", Description: "second"}}
	var buf bytes.Buffer
	rows.write(&buf, Options{NoColor: true}, 80, defaultTheme().Body)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %q", lines)
	}
	col := strings.Index(lines[0], "first")
	if col < 0 || strings.Index(lines[1], "second") != col {
		t.Errorf("descriptions are not aligned:\n%s", buf.String())
	}
}

func TestRowBuildersAgreeWithThePages(t *testing.T) {
	cmds := []Command{
		{Name: "build", Aliases: []string{"b"}, Description: "Build it. Then more.", Group: "Main"},
		{Name: "secret", Description: "Hidden", Hidden: true},
		{Name: "ship", Description: "Ship it", Group: "Main"},
	}
	rows, groups := commandRows(cmds)
	if len(rows) != 2 || rows[0].Name != "build (b)" || rows[0].Description != "Build it." || groups[1] != "Main" {
		t.Errorf("commandRows = %v %v", rows, groups)
	}

	opts := []Option{
		{Flags: "--a", Description: "A", DefaultText: "1"},
		{Flags: "--hidden", Description: "H", Hidden: true},
		{Flags: "--req", Description: "R", Required: true, Group: "G"},
	}
	orows, ogroups := optionRows(opts)
	if len(orows) != 2 || orows[0].Description != "A (default: 1)" || orows[1].Description != "R (required)" || ogroups[1] != "G" {
		t.Errorf("optionRows = %v %v", orows, ogroups)
	}

	entries := &Command{SubcommandEntries: []Param{{Name: "x", Description: "X in full. Both sentences."}}, Subcommands: cmds}
	if got := subcommandRows(entries); len(got) != 1 || got[0].Description != "X in full. Both sentences." {
		t.Errorf("explicit entries must be drawn in full and replace the tree: %v", got)
	}
	tree := &Command{Subcommands: cmds}
	if got := subcommandRows(tree); len(got) != 2 || got[0].Description != "Build it." {
		t.Errorf("without entries the visible subcommands are drawn by first sentence: %v", got)
	}
}

// The renderer and Audit disagreed whenever each did its own arithmetic: the
// name column and the room for a description were computed in a dozen places, and
// every drift was a page that Audit passed and that wrapped. They now ask
// listing. This keeps it that way: the two files whose agreement matters may not
// reach for the column functions themselves.
func TestRendererAndAuditShareTheLayoutArithmetic(t *testing.T) {
	forbidden := map[string]bool{"colIndentFor": true, "colIndent": true, "clampIndent": true}
	for _, file := range []string{"render.go", "audit_layout.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && forbidden[id.Name] {
				t.Errorf("%s calls %s directly; ask listing (listing.go) so the renderer and Audit cannot disagree", file, id.Name)
			}
			return true
		})
	}
}
