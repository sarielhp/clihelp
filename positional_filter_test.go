package clihelp

import (
	"errors"
	"strings"
	"testing"
)

func TestPositionalFilter(t *testing.T) {
	t.Parallel()

	t.Run("RootPositionalFilter", func(t *testing.T) {
		var rootArg string
		var scanRun bool

		app := &App{
			Name: "mail_cli",
			Commands: []Command{
				{
					Name: "scan",
					Run: func(ctx *Context) error {
						scanRun = true
						return nil
					},
				},
				{
					Name: "search",
					Run:  func(ctx *Context) error { return nil },
				},
			},
			Args: MaximumNArgs(1),
			PositionalFilter: func(arg string) bool {
				return strings.HasPrefix(arg, "%")
			},
			Run: func(ctx *Context) error {
				if len(ctx.Args) > 0 {
					rootArg = ctx.Args[0]
				}
				return nil
			},
		}

		// 1. Valid positional matching filter passes to root Run
		if err := app.Execute([]string{"%inbox"}); err != nil {
			t.Fatalf("unexpected error for %%inbox: %v", err)
		}
		if rootArg != "%inbox" {
			t.Errorf("expected rootArg to be %%inbox, got %q", rootArg)
		}

		// 2. Real subcommand executes normally
		if err := app.Execute([]string{"scan"}); err != nil {
			t.Fatalf("unexpected error for scan: %v", err)
		}
		if !scanRun {
			t.Error("expected scan command to run")
		}

		// 3. Typo in subcommand rejected by PositionalFilter and triggers suggestion
		err := app.Execute([]string{"scann"})
		if err == nil {
			t.Fatal("expected error for mistyped subcommand scann, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected errors.Is(err, ErrUsage), got: %v", err)
		}
		if !strings.Contains(err.Error(), `unknown command "scann" for "mail_cli". Did you mean "scan"?`) {
			t.Errorf("expected suggestion for scan in error, got: %v", err)
		}
	})

	t.Run("CommandPositionalFilter", func(t *testing.T) {
		var tagArg string
		app := &App{
			Name: "app",
			Commands: []Command{
				{
					Name: "tag",
					Subcommands: []Command{
						{Name: "add", Run: func(ctx *Context) error { return nil }},
						{Name: "list", Run: func(ctx *Context) error { return nil }},
					},
					Args: MaximumNArgs(1),
					PositionalFilter: func(arg string) bool {
						return strings.HasPrefix(arg, "v")
					},
					Run: func(ctx *Context) error {
						if len(ctx.Args) > 0 {
							tagArg = ctx.Args[0]
						}
						return nil
					},
				},
			},
		}

		// Valid positional
		if err := app.Execute([]string{"tag", "v1.2.3"}); err != nil {
			t.Fatalf("unexpected error for tag v1.2.3: %v", err)
		}
		if tagArg != "v1.2.3" {
			t.Errorf("expected tagArg to be v1.2.3, got %q", tagArg)
		}

		// Typo in subcommand
		err := app.Execute([]string{"tag", "ad"})
		if err == nil {
			t.Fatal("expected error for tag ad, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected errors.Is(err, ErrUsage), got: %v", err)
		}
		if !strings.Contains(err.Error(), `unknown command "ad" for "tag". Did you mean "add"?`) {
			t.Errorf("expected suggestion for add in error, got: %v", err)
		}
	})
}

func TestSuggestCommand(t *testing.T) {
	t.Parallel()

	candidates := []Command{
		{Name: "scan"},
		{Name: "search", Aliases: []string{"find"}},
		{Name: "status"},
		{Name: "hidden", Hidden: true},
	}

	tests := []struct {
		input string
		want  string
	}{
		{"scann", "scan"},
		{"statu", "status"},
		{"fnid", "search"},
		{"hiddn", ""},
		{"xyz123", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SuggestCommand(tt.input, candidates)
			if got != tt.want {
				t.Errorf("SuggestCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindNearestCommands(t *testing.T) {
	t.Parallel()

	app := &App{
		Name: "testtool",
		Commands: []Command{
			{Name: "scan"},
			{Name: "search"},
			{
				Name: "remote",
				Subcommands: []Command{
					{Name: "add"},
					{Name: "remove"},
				},
			},
			{Name: "internal", Hidden: true},
		},
		Shortcuts: []Command{
			{Name: "quickscan"},
		},
	}

	t.Run("exact or close match", func(t *testing.T) {
		res := app.FindNearestCommands("scann")
		if len(res) == 0 || res[0] != "scan" {
			t.Errorf("FindNearestCommands(scann) = %v, want [scan]", res)
		}
	})

	t.Run("nested command match", func(t *testing.T) {
		res := app.FindNearestCommands("ad")
		if len(res) == 0 || res[0] != "remote add" {
			t.Errorf("FindNearestCommands(ad) = %v, want [remote add]", res)
		}
	})

	t.Run("empty or distant", func(t *testing.T) {
		if res := app.FindNearestCommands(""); len(res) != 0 {
			t.Errorf("FindNearestCommands('') = %v, want empty", res)
		}
		if res := app.FindNearestCommands("completelyunrelatedword"); len(res) != 0 {
			t.Errorf("FindNearestCommands(distant) = %v, want empty", res)
		}
	})
}
