package clihelp

import (
	"errors"
	"testing"
)

func TestIsUsageError(t *testing.T) {
	t.Parallel()

	if IsUsageError(nil) {
		t.Error("IsUsageError(nil) = true, want false")
	}
	if IsUsageError(errors.New("runtime failure")) {
		t.Error("IsUsageError(runtime error) = true, want false")
	}
	if !IsUsageError(ErrUsage) {
		t.Error("IsUsageError(ErrUsage) = false, want true")
	}
}

func TestErrUsageFlags(t *testing.T) {
	t.Parallel()

	var str, req string
	format := "json"
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name: "run",
				Options: []Option{
					String(&str, "-s, --string <val>", "", "a string"),
					Enum(&format, "--format <fmt>", []string{"json", "yaml"}, "json", "output format"),
				},
				Run: func(ctx *Context) error { return nil },
			},
			{
				Name: "deploy",
				Options: []Option{
					Required(String(&req, "--target <name>", "", "target cluster")),
				},
				Run: func(ctx *Context) error { return nil },
			},
		},
	}

	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"run", "--bogus"}},
		{"unknown shorthand flag", []string{"run", "-x"}},
		{"invalid enum value", []string{"run", "--format", "xml"}},
		{"missing required flag", []string{"deploy"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Execute(tt.args)
			if err == nil {
				t.Fatalf("expected error for %v, got nil", tt.args)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("expected errors.Is(err, ErrUsage) for %v, got: %v", tt.args, err)
			}
			if !IsUsageError(err) {
				t.Errorf("expected IsUsageError(err) for %v, got: %v", tt.args, err)
			}
		})
	}
}

func TestErrUsageUnknownCommands(t *testing.T) {
	t.Parallel()

	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name: "scan",
				Run:  func(ctx *Context) error { return nil },
			},
			{
				Name: "group",
				Subcommands: []Command{
					{Name: "sub", Run: func(ctx *Context) error { return nil }},
				},
			},
		},
	}

	tests := []struct {
		name string
		args []string
	}{
		{"unknown root command", []string{"foobar"}},
		{"misspelled root command", []string{"scann"}},
		{"unknown subcommand", []string{"group", "nonexistent"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Execute(tt.args)
			if err == nil {
				t.Fatalf("expected error for %v, got nil", tt.args)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("expected errors.Is(err, ErrUsage) for %v, got: %v", tt.args, err)
			}
			if !IsUsageError(err) {
				t.Errorf("expected IsUsageError(err) for %v, got: %v", tt.args, err)
			}
		})
	}
}

func TestErrUsageArity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validator ArgsValidator
		args      []string
	}{
		{"ExactArgs too few", ExactArgs(2), []string{"one"}},
		{"ExactArgs too many", ExactArgs(2), []string{"one", "two", "three"}},
		{"MinimumNArgs too few", MinimumNArgs(2), []string{"one"}},
		{"MaximumNArgs too many", MaximumNArgs(1), []string{"one", "two"}},
		{"RangeArgs too few", RangeArgs(2, 4), []string{"one"}},
		{"RangeArgs too many", RangeArgs(2, 4), []string{"one", "two", "three", "four", "five"}},
		{"NoArgs with args", NoArgs, []string{"extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.ValidateArgs(tt.args)
			if err == nil {
				t.Fatalf("expected error for validator with %v, got nil", tt.args)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("expected errors.Is(err, ErrUsage) for validator, got: %v", err)
			}
			if !IsUsageError(err) {
				t.Errorf("expected IsUsageError(err) for validator, got: %v", err)
			}
		})
	}
}

func TestErrUsageOptions(t *testing.T) {
	t.Parallel()

	var a, b, c bool
	app := &App{
		Name: "testapp",
		Commands: []Command{
			{
				Name: "check",
				Options: []Option{
					Bool(&a, "--alpha", false, "alpha option"),
					Bool(&b, "--beta", false, "beta option"),
					Bool(&c, "--gamma", false, "gamma option"),
				},
				OptionsValidator: ValidateOptions(
					MutuallyExclusive("--alpha", "--beta"),
					RequiredTogether("--beta", "--gamma"),
					RequiredWith("--gamma", "--alpha"),
				),
				Run: func(ctx *Context) error { return nil },
			},
		},
	}

	tests := []struct {
		name string
		args []string
	}{
		{"mutually exclusive violation", []string{"check", "--alpha", "--beta"}},
		{"required together violation", []string{"check", "--beta"}},
		{"required with violation", []string{"check", "--gamma"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Execute(tt.args)
			if err == nil {
				t.Fatalf("expected error for %v, got nil", tt.args)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("expected errors.Is(err, ErrUsage) for %v, got: %v", tt.args, err)
			}
			if !IsUsageError(err) {
				t.Errorf("expected IsUsageError(err) for %v, got: %v", tt.args, err)
			}
		})
	}
}

func TestErrUsageLeftoverAndRuntime(t *testing.T) {
	t.Parallel()

	t.Run("LeftoverArgument", func(t *testing.T) {
		app := &App{
			Name: "testapp",
			Commands: []Command{
				{
					Name: "group",
					Subcommands: []Command{
						{Name: "sub", Run: func(ctx *Context) error { return nil }},
					},
				},
			},
		}

		// 1. Subcommand following --
		err := app.Execute([]string{"group", "--", "sub"})
		if err == nil {
			t.Fatal("expected error for leftover subcommand after --, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected errors.Is(err, ErrUsage), got: %v", err)
		}

		// 2. Extra argument for grouping command
		err = app.Execute([]string{"group", "bogus"})
		if err == nil {
			t.Fatal("expected error for unknown argument on grouping command, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected errors.Is(err, ErrUsage), got: %v", err)
		}
	})

	t.Run("RuntimeErrorIsNotUsageError", func(t *testing.T) {
		runtimeErr := errors.New("database connection failed")
		app := &App{
			Name: "testapp",
			Commands: []Command{
				{
					Name: "run",
					Run: func(ctx *Context) error {
						return runtimeErr
					},
				},
			},
		}

		err := app.Execute([]string{"run"})
		if err == nil {
			t.Fatal("expected runtime error, got nil")
		}
		if errors.Is(err, ErrUsage) {
			t.Errorf("runtime error must NOT be ErrUsage: %v", err)
		}
		if IsUsageError(err) {
			t.Errorf("IsUsageError(runtime error) must be false: %v", err)
		}
		if !errors.Is(err, runtimeErr) {
			t.Errorf("expected errors.Is(err, runtimeErr), got: %v", err)
		}
	})
}
