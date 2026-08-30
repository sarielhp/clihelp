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
	app.RenderFlags(o)
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
	app.RenderMan(o)
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
		"tree",
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
	app.RenderHelpTopics(o)
	out := strip(buf.String())

	for _, want := range []string{
		"Help Topics:",
		"help <command>",
		"help flags",
		"help tree",
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
			name:       "help tree",
			args:       []string{"help", "tree"},
			wantSubstr: "podctl",
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
