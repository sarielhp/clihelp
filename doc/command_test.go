package doc

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

func TestMarkdownCommand(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		page bool
	}{
		{"generate", []string{"docs"}, true},
		{"help", []string{"docs", "--help"}, false},
		{"unexpected argument", []string{"docs", "extra"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLIHELP_GEN", "")
			dir := filepath.Join(t.TempDir(), "pages")
			app := testApp()
			app.Commands = append(app.Commands, MarkdownCommand(MarkdownOptions{Dir: dir}))
			var out, report bytes.Buffer
			app.Stdout, app.Stderr = &out, &report
			err := app.Execute(tt.args)
			if tt.name == "unexpected argument" {
				if !clihelp.IsUsageError(err) {
					t.Fatalf("expected usage error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Stat(filepath.Join(dir, "index.md"))
			if tt.page != (statErr == nil) {
				t.Fatalf("generated page: %v", statErr)
			}
			if !tt.page {
				return
			}
			if out.Len() != 0 || !strings.Contains(report.String(), "help pages generated") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), report.String())
			}
			report.Reset()
			if err := app.Execute(tt.args); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(report.String(), "up to date") {
				t.Fatalf("unchanged generation: %q", report.String())
			}
		})
	}
}

func TestMarkdownCommandOutputOverride(t *testing.T) {
	for _, flag := range []string{"--output", "-o"} {
		t.Run(flag, func(t *testing.T) {
			root := t.TempDir()
			original, override := filepath.Join(root, "original"), filepath.Join(root, "override")
			app := testApp()
			cmd := MarkdownCommand(MarkdownOptions{Dir: original})
			cmd.Name = "markdown"
			app.Commands = append(app.Commands, clihelp.Command{Name: "dev", Subcommands: []clihelp.Command{cmd}})
			var out, report bytes.Buffer
			app.Stdout, app.Stderr = &out, &report
			if err := app.Execute([]string{"dev", "markdown", flag, override}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(override, "index.md")); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(original); !os.IsNotExist(err) {
				t.Fatalf("default directory was touched: %v", err)
			}
		})
	}
}

func TestMarkdownCommandErrors(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "file")
	if err := os.WriteFile(blocked, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		dir    string
		cancel bool
	}{
		{"not a directory", blocked, false},
		{"cancelled", filepath.Join(root, "cancelled"), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := testApp()
			app.Commands = append(app.Commands, MarkdownCommand(MarkdownOptions{Dir: tt.dir}))
			var out, report bytes.Buffer
			app.Stdout, app.Stderr = &out, &report
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			err := app.ExecuteContext(ctx, []string{"docs"})
			if err == nil || !strings.Contains(err.Error(), "generate markdown") {
				t.Fatalf("expected contextual error, got %v", err)
			}
			if tt.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation was lost: %v", err)
			}
			if out.Len() != 0 || report.Len() != 0 {
				t.Fatalf("failed command reported success: stdout=%q stderr=%q", out.String(), report.String())
			}
		})
	}
}
