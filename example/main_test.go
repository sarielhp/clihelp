package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sarielhp/clihelp"
)

// TestPodctl_CommandDescriptions verifies that all commands in podctl
// have a non-empty Description.
func TestPodctl_CommandDescriptions(t *testing.T) {
	app := buildApp()

	err := app.Walk(func(path []string, cmd *clihelp.Command) error {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			if strings.TrimSpace(cmd.Description) == "" {
				t.Errorf("command %q has empty Description", strings.Join(path, " "))
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
}

// TestPodctl_LeafUsageLines verifies that all leaf commands in podctl
// have a non-empty UsageLine.
func TestPodctl_LeafUsageLines(t *testing.T) {
	app := buildApp()

	err := app.Walk(func(path []string, cmd *clihelp.Command) error {
		if len(cmd.Subcommands) == 0 {
			t.Run(strings.Join(path, " "), func(t *testing.T) {
				if strings.TrimSpace(cmd.UsageLine) == "" {
					t.Errorf("leaf command %q has empty UsageLine", strings.Join(path, " "))
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
}

// TestPodctl_ValidateAllExamples verifies that all defined examples
// pass app.ValidateAllExamples() static check.
func TestPodctl_ValidateAllExamples(t *testing.T) {
	app := buildApp()

	if err := app.ValidateAllExamples(); err != nil {
		t.Fatalf("ValidateAllExamples failed: %v", err)
	}
}

// TestPodctl_RenderCommandSmoke smoke-tests that every command help page
// renders via RenderCommand without error or empty output.
func TestPodctl_RenderCommandSmoke(t *testing.T) {
	app := buildApp()

	err := app.Walk(func(path []string, cmd *clihelp.Command) error {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			var buf bytes.Buffer
			opts := clihelp.Options{
				Writer: &buf,
				Width:  80,
			}
			ok := app.RenderCommand(opts, path...)
			if !ok {
				t.Fatalf("RenderCommand returned false for path %v", path)
			}
			if strings.TrimSpace(buf.String()) == "" {
				t.Errorf("RenderCommand produced empty output for path %v", path)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
}
