package clihelp

import (
	"errors"
	"fmt"
	"strings"
)

// safeAppName returns the name under which an application owns files and shell
// symbols, or an error when it has none it can own them under.
//
// appName is a *display* fallback: it answers "app" so that help output always
// has something to print. That is the wrong answer for a file name, a shell
// function name, an rc-file marker or a registry key — two nameless programs
// would share all of them, and a name carrying shell metacharacters becomes code
// the moment it is interpolated into a generated script. This is the one gate
// for every such use; appName keeps its meaning for rendering.
func safeAppName(a *App) (string, error) {
	if a == nil {
		return "", errors.New("app is nil")
	}
	name := strings.TrimSpace(a.Name)
	if name == "" {
		return "", errors.New("this program has no App.Name: set one before generating or installing shell integration")
	}
	if name == "." || name == ".." || strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("App.Name %q cannot be used as a file name or a command name", name)
	}
	for _, r := range name {
		if !isSafeNameRune(r) {
			return "", fmt.Errorf("App.Name %q cannot be used in a generated shell script or a file path: %q is not allowed (letters, digits, and - _ . + only)", name, r)
		}
	}
	return name, nil
}

func isSafeNameRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == '.' || r == '+':
		return true
	}
	return false
}

// shellFuncName turns a validated application name into a shell function name.
// Only the characters safeAppName allows can reach it, and of those only "." and
// "+" are illegal in a function name.
func shellFuncName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '.' || r == '+' {
			return '_'
		}
		return r
	}, name)
}

// escapeFishArg quotes a string as a single fish word. fish's single quotes are
// literal except for the backslash and the quote itself, so its rule differs
// from POSIX and needs its own quoter.
func escapeFishArg(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

// safeWrapperName holds a wrapper's name to the same rule as an application's:
// it becomes a command name, a file name, and a token in the registration line
// the user is told to paste into their shell.
func safeWrapperName(name string) error {
	_, err := safeAppName(&App{Name: name})
	if err != nil {
		return fmt.Errorf("wrapper name: %w", err)
	}
	return nil
}
