package clihelp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// `make commit ARGS="..."` hands the message to tools/commit.sh. make expands
// `$` in a variable's value when it exports it, so a message containing $5,
// $HOME or $(x) arrived with those parts silently removed ("cost $5 $HOME"
// became "cost  OME"): nothing failed, and the wrong text was committed. The
// message has to reach the script exactly as it was typed.
func TestMakeCommitPassesTheMessageUnchanged(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	messages := []struct{ name, msg string }{
		{"plain", "feat: add a thing"},
		{"dollar digits and a variable", "cost $5 $HOME 100%"},
		{"doubled dollar", "price $$ and $$$$"},
		{"command substitution", "fix: $(touch PWNED) and `touch PWNED2`"},
		{"make variable reference", "chore: ${MAKE} $(ARGS) $@"},
		{"quotes", `say "hi" and it's fine`},
		{"semicolon and ampersand", "a; b && c | d"},
		{"multi-line with a body", "fix: subject\n\nBody with $5 and 'quotes'.\n\nNot done: $(x)."},
		{"leading dash", "-m weird"},
		{"trailing backslash", `path C:\dir\`},
	}
	for _, tt := range messages {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			got := filepath.Join(dir, "got")
			if err := os.MkdirAll(filepath.Join(dir, "tools"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "Makefile"), makefile, 0o644); err != nil {
				t.Fatal(err)
			}
			// A stub in place of the gate-and-commit script: it records the single
			// argument it receives, byte for byte, and does nothing else.
			stub := "#!/bin/sh\nprintf '%s' \"$1\" > " + got + "\nprintf '%s' \"$#\" > " + got + ".argc\n"
			if err := os.WriteFile(filepath.Join(dir, "tools", "commit.sh"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command("make", "commit", "ARGS="+tt.msg)
			cmd.Dir = dir
			cmd.Env = withoutMakeState(os.Environ())
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("make commit: %v\n%s", err, out)
			}
			body, err := os.ReadFile(got)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != tt.msg {
				t.Errorf("the script received a different message:\n got %q\nwant %q", body, tt.msg)
			}
			if argc, _ := os.ReadFile(got + ".argc"); string(argc) != "1" {
				t.Errorf("the script received %s arguments, want exactly 1", argc)
			}
			for _, leaked := range []string{"PWNED", "PWNED2"} {
				if _, err := os.Stat(filepath.Join(dir, leaked)); err == nil {
					t.Errorf("the message was executed: %s was created", leaked)
				}
			}
		})
	}
}

// The same message supplied through the environment, the other way the README
// and the AGENTS.md workflow invite people to call it.
func TestMakeCommitPassesAnEnvironmentMessageUnchanged(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	got := filepath.Join(dir, "got")
	if err := os.MkdirAll(filepath.Join(dir, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), makefile, 0o644); err != nil {
		t.Fatal(err)
	}
	stub := "#!/bin/sh\nprintf '%s' \"$1\" > " + got + "\n"
	if err := os.WriteFile(filepath.Join(dir, "tools", "commit.sh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	const msg = "cost $5 $HOME $(x) and $$"
	cmd := exec.Command("make", "commit")
	cmd.Dir = dir
	cmd.Env = append(withoutMakeState(os.Environ()), "ARGS="+msg)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make commit: %v\n%s", err, out)
	}
	body, _ := os.ReadFile(got)
	if string(body) != msg {
		t.Errorf("the script received %q, want %q", body, msg)
	}
}

// withoutMakeState drops the variables an enclosing make leaves in the
// environment. The suite runs inside `make commit`, whose MAKEFLAGS carries the
// outer ARGS= definition; a command-line definition beats the environment, so
// without this the inner make would see the outer message and not the one the
// test supplies.
func withoutMakeState(env []string) []string {
	drop := map[string]bool{"MAKEFLAGS": true, "MFLAGS": true, "MAKELEVEL": true, "MAKEOVERRIDES": true, "ARGS": true, "COMMIT_MESSAGE": true}
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !drop[name] {
			out = append(out, kv)
		}
	}
	return out
}
