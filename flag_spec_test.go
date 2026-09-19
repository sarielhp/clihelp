package clihelp

import (
	"strings"
	"testing"
)

// Chain C of the 2026-09-17 review: a malformed flag spec must become an error
// from the binder (and a finding from Audit), never a pflag panic and never a
// flag that quietly binds nothing.

func specApp(opts ...Option) *App {
	return &App{
		Name: "specapp",
		Commands: []Command{{
			Name:        "run",
			Description: "Run it",
			Options:     opts,
			Run:         func(*Context) error { return nil },
		}},
	}
}

func TestMalformedFlagSpecsAreErrorsNotPanics(t *testing.T) {
	var (
		text   string
		toggle bool
	)
	for _, tt := range []struct {
		name    string
		option  func() Option
		wantErr string
	}{
		{
			name:    "multi-character shorthand",
			option:  func() Option { return String(&text, "-out <F>", "", "output") },
			wantErr: "single ASCII character",
		},
		{
			name:    "non-ascii shorthand",
			option:  func() Option { return String(&text, "-é <F>", "", "output") },
			wantErr: "single ASCII character",
		},
		{
			name:    "no leading dashes",
			option:  func() Option { return String(&text, "out <F>", "", "output") },
			wantErr: "no flag names",
		},
		{
			name:    "toggle without a long name",
			option:  func() Option { return BoolToggle(&toggle, "-c", false, "colorize") },
			wantErr: "long name",
		},
		{
			name:    "toggle with an empty base name",
			option:  func() Option { return BoolToggle(&toggle, "--[no-]", false, "colorize") },
			wantErr: "empty",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := specApp(tt.option())
			res := testExecute(app, []string{"run"})
			res.AssertErrorContains(t, tt.wantErr)
			if err := Audit(app); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Audit error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestBoolToggleBindsAliasLongNames(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want bool
	}{
		{"primary positive", []string{"run", "--color"}, true},
		{"primary negative", []string{"run", "--no-color"}, false},
		{"alias positive", []string{"run", "--colour"}, true},
		{"alias negative", []string{"run", "--no-colour"}, false},
		{"alias shorthand", []string{"run", "-C"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			color := !tt.want
			app := specApp(BoolToggle(&color, "--[no-]color, --colour, -c, -C", !tt.want, "colorize"))
			res := testExecute(app, tt.args)
			res.AssertNoError(t)
			if color != tt.want {
				t.Errorf("color = %v, want %v", color, tt.want)
			}
		})
	}
}

func TestAuditInspectsAppLevelOptions(t *testing.T) {
	var text string
	app := &App{
		Name:        "specapp",
		GlobalFlags: []Option{String(&text, "-out <F>", "", "output")},
		Commands:    []Command{{Name: "run", Description: "Run it", Run: func(*Context) error { return nil }}},
	}
	err := Audit(app)
	if err == nil || !strings.Contains(err.Error(), "single ASCII character") {
		t.Fatalf("Audit error = %v, want one about the shorthand", err)
	}
}

func TestAuditDetectsCollisionsAcrossBindingScopes(t *testing.T) {
	var global, local, persistent, nested string
	for _, tt := range []struct {
		name string
		app  *App
		want string
	}{
		{
			name: "global flag against a command option",
			app: &App{
				Name:        "specapp",
				GlobalFlags: []Option{String(&global, "--out <F>", "", "output")},
				Commands: []Command{{
					Name: "run", Description: "Run it",
					Options: []Option{String(&local, "--out <F>", "", "output")},
					Run:     func(*Context) error { return nil },
				}},
			},
			want: "--out",
		},
		{
			name: "ancestor persistent flag against a subcommand option",
			app: &App{
				Name: "specapp",
				Commands: []Command{{
					Name: "parent", Description: "Parent",
					PersistentOptions: []Option{String(&persistent, "--out, -o <F>", "", "output")},
					Subcommands: []Command{{
						Name: "child", Description: "Child",
						Options: []Option{String(&nested, "--output, -o <F>", "", "output")},
						Run:     func(*Context) error { return nil },
					}},
				}},
			},
			want: "-o",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := Audit(tt.app); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Audit error = %v, want one naming %s", err, tt.want)
			}
		})
	}
}

func TestAuditAcceptsSiblingCommandsReusingAFlag(t *testing.T) {
	var a, b string
	app := &App{
		Name: "specapp",
		Commands: []Command{
			{Name: "one", Description: "One", Options: []Option{String(&a, "--out <F>", "", "output")}, Run: func(*Context) error { return nil }},
			{Name: "two", Description: "Two", Options: []Option{String(&b, "--out <F>", "", "output")}, Run: func(*Context) error { return nil }},
		},
	}
	if err := Audit(app); err != nil {
		t.Fatalf("Audit rejected sibling commands reusing --out: %v", err)
	}
}
