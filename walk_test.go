package clihelp

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestApp_Walk(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{
				Name:        "alpha",
				Description: "Alpha command",
				Subcommands: []Command{
					{
						Name:        "one",
						Description: "Alpha one",
					},
					{
						Name:        "two",
						Description: "Alpha two",
					},
				},
			},
			{
				Name:        "beta",
				Description: "Beta command",
			},
		},
	}

	var visited []string
	var paths [][]string
	err := app.Walk(func(path []string, cmd *Command) error {
		visited = append(visited, cmd.Name)
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected Walk error: %v", err)
	}

	expectedVisited := []string{"alpha", "one", "two", "beta"}
	if !reflect.DeepEqual(visited, expectedVisited) {
		t.Errorf("visited = %v, expected %v", visited, expectedVisited)
	}

	expectedPaths := [][]string{
		{"alpha"},
		{"alpha", "one"},
		{"alpha", "two"},
		{"beta"},
	}
	if !reflect.DeepEqual(paths, expectedPaths) {
		t.Errorf("paths = %v, expected %v", paths, expectedPaths)
	}
}

func TestApp_Walk_EarlyExit(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{
				Name: "c1",
				Subcommands: []Command{
					{Name: "c1_sub"},
				},
			},
			{Name: "c2"},
		},
	}

	expectedErr := errors.New("halt walk")
	var count int
	err := app.Walk(func(path []string, cmd *Command) error {
		count++
		if cmd.Name == "c1_sub" {
			return expectedErr
		}
		return nil
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if count != 2 {
		t.Errorf("expected walk to halt after 2 visits, visited %d", count)
	}
}

func TestApp_Walk_NilEdgeCases(t *testing.T) {
	var nilApp *App
	if err := nilApp.Walk(func(path []string, cmd *Command) error { return nil }); err != nil {
		t.Errorf("expected nil error on nil app Walk, got %v", err)
	}

	app := &App{}
	if err := app.Walk(nil); err != nil {
		t.Errorf("expected nil error on nil fn Walk, got %v", err)
	}
}

func TestApp_Walk_PathIsolation(t *testing.T) {
	app := &App{
		Name: "app",
		Commands: []Command{
			{
				Name: "parent",
				Subcommands: []Command{
					{Name: "child1"},
					{Name: "child2"},
				},
			},
		},
	}

	err := app.Walk(func(path []string, cmd *Command) error {
		path[0] = "mutated"
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if app.Commands[0].Name != "parent" {
		t.Errorf("parent command name was modified: %s", app.Commands[0].Name)
	}
}

func TestUVStyleCommandListing(t *testing.T) {
	app := &App{
		Name: "tool",
		Commands: []Command{
			{
				Name:        "run",
				UsageLine:   "tool run <target> [flags]",
				Description: "Execute the target.",
			},
			{
				Name:        "test",
				Aliases:     []string{"t"},
				UsageLine:   "tool test [pkg]",
				Description: "Run test suite.",
			},
		},
	}

	var sb strings.Builder
	app.RenderGlobal(Options{Writer: &sb, Width: 80})
	out := sb.String()

	if !strings.Contains(out, "run ") || !strings.Contains(out, "test (t)") {
		t.Errorf("expected 'run' and 'test (t)' in global help, got:\n%s", out)
	}
	if strings.Contains(out, "run <target>") || strings.Contains(out, "test (t) [pkg]") {
		t.Errorf("command listing should be UV-style (no positional args or flags), got:\n%s", out)
	}
}
