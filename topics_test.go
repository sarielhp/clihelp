package clihelp

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderFlags(t *testing.T) {
	app := &App{
		Name:        "podctl",
		Description: "Container manager",
		Version:     "1.2.3",
		PersistentOptions: []Option{
			{
				Flags:       "--token <string>",
				Description: "API auth token",
				Group:       "Authentication",
			},
			{
				Flags:       "-o, --output <format>",
				Description: "Output format",
				DefaultText: "table",
				Group:       "Output & Logging",
			},
		},
		Commands: []Command{
			{Name: "run", Description: "Run a pod"},
		},
	}

	o, buf := captureOptions(80)
	app.renderFlagsPage(o)
	out := strip(buf.String())

	for _, want := range []string{
		"Usage:  podctl [flags] <command> [args]",
		"Global flags available to all commands:",
		"Authentication:",
		"--token <string>",
		"API auth token",
		"Output & Logging:",
		"-o, --output <format>",
		"Output format (default: table)",
		"Help & Information:",
		"-h, --help",
		"-v, --version",
		"Run 'podctl <command> -h' for command-specific flags.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderFlags missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestRenderMan(t *testing.T) {
	app := &App{
		Name:        "podctl",
		Description: "Container manager",
		GlobalNote:  "For issues visit https://example.com/bugs",
		PersistentOptions: []Option{
			{Flags: "--token <string>", Description: "Auth token"},
		},
		Commands: []Command{
			{
				Name:        "run",
				Description: "Run a container",
				Parameters: []Param{
					{Name: "<image>", Description: "Docker image to run"},
				},
				Options: []Option{
					{Flags: "-p, --port <int>", Description: "Port mapping"},
				},
				Examples: []Example{
					{Line: "podctl run nginx -p 80", Description: "Expose port 80"},
				},
				Notes: []Note{
					{Heading: "Warning", Text: "Requires root privileges"},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	app.renderManPage(o)
	out := strip(buf.String())

	for _, want := range []string{
		"NAME",
		"podctl - Container manager",
		"SYNOPSIS",
		"DESCRIPTION",
		"For issues visit https://example.com/bugs",
		"GLOBAL FLAGS",
		"--token <string>",
		"COMMANDS",
		"podctl run",
		"Run a container",
		"Parameters:",
		"<image>",
		"Flags:",
		"-p, --port <int>",
		"Examples:",
		"podctl run nginx -p 80",
		"Warning:",
		"Requires root privileges",
		"HELP TOPICS",
		"flags",
		"man",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderMan missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestRenderHelpTopics(t *testing.T) {
	app := testApp()
	o, buf := captureOptions(80)
	app.renderTopicsPage(o)
	out := strip(buf.String())

	for _, want := range []string{
		"Help Topics:",
		"help <command>",
		"help flags",
		"help man",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderHelpTopics missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestExecuteHelpTopics(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantSubstr string
	}{
		{
			name:       "help flags",
			args:       []string{"help", "flags"},
			wantSubstr: "Global flags available to all commands:",
		},
		{
			name:       "help options alias",
			args:       []string{"help", "options"},
			wantSubstr: "Global flags available to all commands:",
		},
		{
			name:       "help man",
			args:       []string{"help", "man"},
			wantSubstr: "NAME",
		},
		{
			name:       "help all alias",
			args:       []string{"help", "all"},
			wantSubstr: "NAME",
		},
		{
			name:       "help topics",
			args:       []string{"help", "topics"},
			wantSubstr: "Help Topics:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			app := testApp()
			app.Stdout = &stdout
			err := app.Execute(tt.args)
			if err != nil {
				t.Fatalf("Execute(%v) unexpected error: %v", tt.args, err)
			}
			out := strip(stdout.String())
			if !strings.Contains(out, tt.wantSubstr) {
				t.Errorf("Execute(%v) missing %q\nGot:\n%s", tt.args, tt.wantSubstr, out)
			}
		})
	}
}

func TestOmitGlobalFlagsInCommands(t *testing.T) {
	app := &App{
		Name: "podctl",
		PersistentOptions: []Option{
			{Flags: "--token <string>", Description: "Auth token"},
			{Flags: "--kubeconfig <path>", Description: "Kubeconfig path"},
		},
		Commands: []Command{
			{
				Name:        "run",
				Description: "Run a pod",
				Options: []Option{
					{Flags: "-p, --port <int>", Description: "Expose port"},
				},
			},
		},
		OmitGlobalFlagsInCommands: true,
	}

	o, buf := captureOptions(80)
	if !app.RenderCommand(o, "run") {
		t.Fatal("RenderCommand(\"run\") failed")
	}
	out := strip(buf.String())

	if !strings.Contains(out, "Run 'podctl help flags' for flags available to all commands.") {
		t.Errorf("expected 1-line global flags reference, got:\n%s", out)
	}
	if strings.Contains(out, "--kubeconfig") {
		t.Errorf("expected global flags table to be omitted when OmitGlobalFlagsInCommands is true")
	}
}

func TestRenderManLongDescriptionAndRawNotes(t *testing.T) {
	app := &App{
		Name:             "podctl",
		ExtendedHelpFlag: true,
		Commands: []Command{
			{
				Name:            "build",
				Description:     "Short description",
				LongDescription: "Exhaustive long build description for man page.",
				Notes: []Note{
					{
						Heading: "Environment",
						Text:    "  PODCTL_BUILD_FAST=1\n    PODCTL_BUILD_DEBUG=1",
						Raw:     true,
					},
				},
			},
		},
	}

	o, buf := captureOptions(80)
	app.renderManPage(o)
	out := strip(buf.String())

	if !strings.Contains(out, "Exhaustive long build description for man page.") {
		t.Errorf("expected LongDescription in man page, got:\n%s", out)
	}
	if strings.Contains(out, "Short description") {
		t.Errorf("Short description should be replaced by LongDescription in man page, got:\n%s", out)
	}
	if !strings.Contains(out, "Environment:") {
		t.Errorf("expected note heading in man page, got:\n%s", out)
	}
	if !strings.Contains(out, "          PODCTL_BUILD_FAST=1") {
		t.Errorf("expected raw note with 8 base indent + 2 text indent, got:\n%s", out)
	}
}

// "help <command>" shows one command's examples among its usage, parameters and
// flags. "help examples" is all of them and nothing else — the view for "how do
// I use this?" rather than "what does this flag do?". A second application built
// on this library had written about a hundred and fifty lines to get it.
func TestHelpExamplesCollectsTheWholeTree(t *testing.T) {
	app := &App{
		Name:     "demo",
		Examples: []Example{{Line: "demo build x", Description: "The common case."}},
		Commands: []Command{
			{
				Name: "build", Description: "Build it.",
				Examples: []Example{{Line: "demo build --fast x", Description: "Quickly."}},
				Run:      func(*Context) error { return nil },
				Subcommands: []Command{{
					Name: "all", Description: "Build everything.",
					Examples: []Example{{Line: "demo build all", Description: "The lot."}},
					Run:      func(*Context) error { return nil },
				}},
			},
			{Name: "quiet", Description: "No examples here.", Run: func(*Context) error { return nil }},
			{
				Name: "secret", Description: "Hidden.", Hidden: true,
				Examples: []Example{{Line: "demo secret", Description: "Should not appear."}},
				Run:      func(*Context) error { return nil },
			},
		},
	}
	out := silentApp(app)
	if err := app.Execute([]string{"help", "examples"}); err != nil {
		t.Fatalf("`demo help examples` failed: %v", err)
	}
	body := StripANSI(out.String())

	for _, want := range []string{
		"demo build x", "demo build --fast x", "demo build all",
		"The common case.", "Quickly.", "The lot.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the page is missing %q:\n%s", want, body)
		}
	}
	// A hidden command's examples are not advertised, and a command with none
	// gets no empty heading.
	if strings.Contains(body, "demo secret") {
		t.Errorf("a hidden command's examples were shown:\n%s", body)
	}
	if strings.Contains(body, "quiet") {
		t.Errorf("a command with no examples was given a heading:\n%s", body)
	}
	// Grouped under the command that declares them, deepest included.
	if !strings.Contains(body, "demo build all") || !strings.Contains(body, "demo build\n") {
		t.Errorf("the groups are not headed by their command:\n%s", body)
	}
}

// An application with no examples at all says so rather than printing a bare
// heading over nothing.
func TestHelpExamplesWithNoneDeclared(t *testing.T) {
	app := &App{Name: "bare", Commands: []Command{
		{Name: "go", Description: "Go.", Run: func(*Context) error { return nil }},
	}}
	out := silentApp(app)
	if err := app.Execute([]string{"help", "examples"}); err != nil {
		t.Fatal(err)
	}
	if body := StripANSI(out.String()); !strings.Contains(body, "No examples are declared") {
		t.Errorf("expected a plain statement, got:\n%s", body)
	}
}
