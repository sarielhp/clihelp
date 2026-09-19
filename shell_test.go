package clihelp

import (
	"io"
	"strings"
	"testing"
)

func TestResolveShell(t *testing.T) {
	for _, tt := range []struct {
		name, arg, env, want, wantErr string
	}{
		{"named", "zsh", "/bin/bash", "zsh", ""},
		{"named, mixed case and padded", "  ZSH ", "", "zsh", ""},
		{"detected from $SHELL", "", "/usr/bin/fish", "fish", ""},
		{"detected from a path with a version", "", "/opt/homebrew/bin/bash", "bash", ""},
		{"unsupported, named", "ksh", "/bin/bash", "", "unsupported shell \"ksh\""},
		{"unsupported, detected", "", "/bin/ksh", "", "unsupported shell \"ksh\""},
		{"nothing to go on", "", "", "", "cannot detect the active shell"},
		{"blanks are nothing to go on", "   ", "  ", "", "cannot detect the active shell"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.env)
			got, err := resolveShell(tt.arg)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("resolveShell(%q) failed: %v", tt.arg, err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("resolveShell(%q) = %q, wanted an error", tt.arg, got)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("resolveShell(%q) said %q, wanted %q", tt.arg, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveShell(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

// Every entry point that takes a shell has to answer the question the same way.
// It was asked in six places with two different messages, and five of them
// reported an undetectable shell as `unsupported shell ""`, which says nothing
// about $SHELL being unset — the actual problem, and one the user can fix.
func TestEveryEntryPointResolvesTheShellAlike(t *testing.T) {
	// Two of these entry points write files when they succeed. They are only
	// asked to fail here, but a test that can reach a write path sandboxes,
	// because the cost of being wrong about "only asked to fail" is the
	// developer's own ~/.bashrc.
	sandboxHome(t)

	app := &App{Name: "myapp"}
	entries := map[string]func(shell string) error{
		"CompletionPath": func(s string) error {
			_, err := CompletionPath(app, s)
			return err
		},
		"IntegrationPath": func(s string) error {
			_, err := IntegrationPath(app, s)
			return err
		},
		"installCompletion": func(s string) error {
			_, err := installCompletion(app, s)
			return err
		},
		"GenKeyBindings": func(s string) error {
			return GenKeyBindings(app, s, io.Discard)
		},
		"GenShellIntegration": func(s string) error {
			return GenShellIntegration(app, s, true, io.Discard)
		},
		"installShellIntegration": func(s string) error {
			_, err := installShellIntegration(app, s, true)
			return err
		},
		"uninstallShellIntegration": func(s string) error {
			_, err := uninstallShellIntegration(app, s)
			return err
		},
	}

	for name, call := range entries {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SHELL", "/bin/bash")
			err := call("ksh")
			if err == nil {
				t.Fatalf("%s accepted an unsupported shell", name)
			}
			for _, want := range []string{`unsupported shell "ksh"`, "bash, zsh, fish"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%s said %q, which does not contain %q", name, err, want)
				}
			}

			t.Setenv("SHELL", "")
			err = call("")
			if err == nil {
				t.Fatalf("%s accepted an undetectable shell", name)
			}
			if !strings.Contains(err.Error(), "cannot detect the active shell") {
				t.Errorf("%s reported an unset $SHELL as %q", name, err)
			}
		})
	}
}
