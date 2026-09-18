package clihelp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Which shell, asked once.
//
// "Is this a shell we can write for, and which one is the user running?" was
// answered in six places across this surface, with two different error messages
// — one of which did not say what the supported shells were. resolveShell is
// the single answer; detectShell and isSupportedShell are its parts, exported
// only as far as SupportedShells, which callers print.

// SupportedShells names every shell this library can generate for.
var SupportedShells = []string{"bash", "zsh", "fish"}

// detectShell names the shell from $SHELL, or returns "" when $SHELL is unset.
// It does not translate an unknown shell into "bash": a dash, ksh or nushell
// user was silently given a bash script in their home directory, which their
// shell cannot read and which nothing ever removes.
func detectShell() string {
	sh := os.Getenv("SHELL")
	if strings.TrimSpace(sh) == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(sh))
}

// isSupportedShell reports whether a completion script exists for shell.
func isSupportedShell(shell string) bool {
	for _, s := range SupportedShells {
		if s == shell {
			return true
		}
	}
	return false
}

// resolveShell turns a caller's shell argument into a supported shell name. An
// empty argument means "whichever shell the user is running", which is how every
// entry point in this library spells the default.
func resolveShell(shell string) (string, error) {
	if strings.TrimSpace(shell) == "" {
		shell = detectShell()
	}
	shell = strings.ToLower(strings.TrimSpace(shell))
	// "" deserves its own sentence: "unsupported shell \"\"" told five of the
	// six call sites' users nothing about what had gone wrong.
	if shell == "" {
		return "", errors.New("cannot detect the active shell: $SHELL is not set; name one of " + strings.Join(SupportedShells, ", "))
	}
	if !isSupportedShell(shell) {
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(SupportedShells, ", "))
	}
	return shell, nil
}
