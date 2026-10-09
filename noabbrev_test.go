package clihelp

import (
	"strings"
	"testing"
)

// noAbbrevApp is mail_cli's hazard in miniature: "spam" takes message IDs, and
// "spam de" must not become "spam delete", which empties the folder.
func noAbbrevApp(ran *string) *App {
	record := func(name string) func(*Context) error {
		return func(ctx *Context) error {
			*ran = strings.TrimSpace(name + " " + strings.Join(ctx.Args, " "))
			return nil
		}
	}
	return &App{Name: "mc", AbbrevCommands: true, Commands: []Command{
		{Name: "spam", Description: "Mark spam.", Args: MinimumNArgs(0), Run: record("spam"), Subcommands: []Command{
			{Name: "delete", Aliases: []string{"del"}, Description: "Delete all spam.", NoAbbrev: true, Run: record("delete")},
			{Name: "allow", Description: "Allow a sender.", Args: MinimumNArgs(0), Run: record("allow")},
		}},
		{Name: "deploy", Description: "Deploy.", NoAbbrev: true, Run: record("deploy")},
		{Name: "debug", Description: "Debug.", Run: record("debug")},
		{Name: "purge", Description: "Purge.", NoAbbrev: true, Run: record("purge")},
	}}
}

func TestNoAbbrevCommandsNeedTheirFullName(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		ran     string
		errText string
	}{
		{"a prefix falls to the parent as an argument", []string{"spam", "de"}, "spam de", ""},
		{"the full name runs it", []string{"spam", "delete"}, "delete", ""},
		{"an alias runs it", []string{"spam", "del"}, "delete", ""},
		{"other commands still abbreviate", []string{"spam", "al", "x"}, "allow x", ""},
		{"a help request does not reach it by prefix", []string{"spam", "de", "-h"}, "", ""},
		{"no positional parent: unknown, and suggested", []string{"pu"}, "", `Did you mean "purge"?`},
		{"still counted for ambiguity", []string{"de"}, "", "ambiguous"},
		{"an ordinary prefix beside it still runs", []string{"deb"}, "debug", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ran string
			res := testExecute(noAbbrevApp(&ran), tt.args)
			if tt.errText != "" {
				res.AssertErrorContains(t, tt.errText)
			} else {
				res.AssertNoError(t)
			}
			if ran != tt.ran {
				t.Errorf("ran %q, want %q", ran, tt.ran)
			}
		})
	}
}

// Alt-H rewrites abbreviations to full names on the user's prompt; turning
// "spam de" into "spam delete" there would hand them the dangerous line.
func TestNoAbbrevIsNotExpandedByExplain(t *testing.T) {
	var ran string
	app := noAbbrevApp(&ran)
	if got, _ := app.expandCommandLine("mc spam de"); got != "mc spam de" {
		t.Errorf("expandCommandLine rewrote a NoAbbrev prefix: %q", got)
	}
	if got, _ := app.expandCommandLine("mc spam al x"); got != "mc spam allow x" {
		t.Errorf("expandCommandLine stopped expanding ordinary prefixes: %q", got)
	}
}
