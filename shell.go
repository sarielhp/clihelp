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
// only as far as supportedShells, which callers print.

// supportedShells names every shell this library can generate for.
var supportedShells = []string{"bash", "zsh", "fish"}

// detectShell names the shell the user is typing into: the interactive shell
// that started this program when there is one, and $SHELL otherwise. It returns
// "" when neither answers.
//
// $SHELL alone is the login shell, not the running one. A user whose login
// shell is zsh but who works in fish was told to paste zsh into fish, and an
// install wrote the integration for a shell they were not using.
//
// It does not translate an unknown shell into "bash": a dash, ksh or nushell
// user was silently given a bash script in their home directory, which their
// shell cannot read and which nothing ever removes.
func detectShell() string {
	if sh := interactiveParentShell(); sh != "" {
		return sh
	}
	sh := os.Getenv("SHELL")
	if strings.TrimSpace(sh) == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(sh))
}

// isSupportedShell reports whether a completion script exists for shell.
func isSupportedShell(shell string) bool {
	for _, s := range supportedShells {
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
		return "", errors.New("cannot detect the active shell: $SHELL is not set; name one of " + strings.Join(supportedShells, ", "))
	}
	if !isSupportedShell(shell) {
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(supportedShells, ", "))
	}
	return shell, nil
}

// interactiveParentShell names the parent process when it is a supported shell
// running interactively, and returns "" otherwise — including wherever /proc
// does not exist, which leaves $SHELL to answer.
//
// A shell running a script or a -c string is not the one the user types into:
// a dotfiles script under bash that runs "app __clihelp install" means the
// user's shell, which $SHELL names, and not bash.
func interactiveParentShell() string {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err != nil {
		return ""
	}
	return interactiveShellName(strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00"))
}

// interactiveShellName names the shell argv starts when it is a supported shell
// with no script and no -c string to run.
func interactiveShellName(argv []string) string {
	// A login shell is started as "-zsh".
	name := strings.ToLower(strings.TrimPrefix(filepath.Base(argv[0]), "-"))
	if !isSupportedShell(name) {
		return ""
	}
	for _, arg := range argv[1:] {
		if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--command") ||
			(!strings.HasPrefix(arg, "--") && strings.Contains(arg, "c")) {
			return ""
		}
	}
	return name
}
