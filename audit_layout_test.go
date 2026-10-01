package clihelp

import (
	"strings"
	"testing"
)

func TestAuditShortDescriptionRows(t *testing.T) {
	long := strings.Repeat("word ", 20)
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{"fits", &App{Name: "a", Commands: []Command{{Name: "build", Description: "Build the thing"}}}, ""},
		{"too long", &App{Name: "a", Commands: []Command{{Name: "build", Description: long}}}, "columns"},
		{"multi-line", &App{Name: "a", Commands: []Command{{Name: "build", Description: "one\ntwo"}}}, "one line"},
		{"first sentence only", &App{Name: "a", Commands: []Command{{Name: "build", Description: "Build it. " + long}}}, ""},
		{"markup not counted", &App{Name: "a", Commands: []Command{{Name: "build", Description: "[" + strings.Repeat("x", 30) + "](https://example.com/a/very/long/url/that/is/not/shown)"}}}, ""},
		{"subcommand too long", &App{Name: "a", Commands: []Command{{Name: "g", Description: "Group", Subcommands: []Command{{Name: "s", Description: long}}}}}, "g subcommands"},
		{"flag too long", &App{Name: "a", GlobalFlags: []Option{{Flags: "--x", Description: long}}}, "flags"},
		{"hidden skipped", &App{Name: "a", Commands: []Command{{Name: "h", Description: long, Hidden: true}, {Name: "ok", Description: "fine"}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(tt.app, AuditOptions{SkipExampleValidation: true})
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Audit = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Audit = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestAuditWidthOption(t *testing.T) {
	app := &App{Name: "a", Commands: []Command{{Name: "build", Description: strings.Repeat("x", 50)}}}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true}); err != nil {
		t.Fatalf("width 80: %v", err)
	}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: 50}); err == nil {
		t.Error("width 50: want error, got nil")
	}
}

func TestAuditLongTierIsUnlimitedButWarnsOnUnwrappable(t *testing.T) {
	prose := strings.Repeat("A sentence of ordinary prose. ", 100)
	wide := strings.Repeat("x", 120)
	tests := []struct {
		name     string
		cmd      Command
		wantWarn int
	}{
		{"long prose", Command{Name: "c", Description: "ok", LongDescription: prose, Notes: []Note{{Heading: "h", Text: prose}}}, 0},
		{"wide raw note", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: "short\n" + wide, Raw: true}}}, 1},
		{"narrow raw note", Command{Name: "c", Description: "ok", Notes: []Note{{Heading: "h", Text: "short", Raw: true}}}, 0},
		{"wide example", Command{Name: "c", Description: "ok", Examples: []Example{{Line: "a c " + wide}}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warns []string
			app := &App{Name: "a", Commands: []Command{tt.cmd}}
			err := Audit(app, AuditOptions{SkipExampleValidation: true, Warn: func(m string) { warns = append(warns, m) }})
			if err != nil {
				t.Fatalf("Audit = %v, want nil", err)
			}
			if len(warns) != tt.wantWarn {
				t.Errorf("warnings = %v, want %d", warns, tt.wantWarn)
			}
		})
	}
}
