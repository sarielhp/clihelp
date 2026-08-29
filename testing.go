package clihelp

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
)

// TestResult holds the outcome of a TestExecute execution.
type TestResult struct {
	Stdout string
	Stderr string
	Error  error
}

// AssertNoError asserts that the command executed successfully without error.
func (tr *TestResult) AssertNoError(t *testing.T) {
	t.Helper()
	if tr.Error != nil {
		t.Fatalf("expected no error, got: %v", tr.Error)
	}
}

// AssertErrorContains asserts that an error occurred and contains the substring.
func (tr *TestResult) AssertErrorContains(t *testing.T, substring string) {
	t.Helper()
	if tr.Error == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(tr.Error.Error(), substring) {
		t.Fatalf("expected error containing %q, got: %v", substring, tr.Error)
	}
}

// AssertStdoutContains asserts that stdout contains the substring.
func (tr *TestResult) AssertStdoutContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stdout, substring) {
		t.Fatalf("expected stdout containing %q, got:\n%s", substring, tr.Stdout)
	}
}

// AssertStderrContains asserts that stderr contains the substring.
func (tr *TestResult) AssertStderrContains(t *testing.T, substring string) {
	t.Helper()
	if !strings.Contains(tr.Stderr, substring) {
		t.Fatalf("expected stderr containing %q, got:\n%s", substring, tr.Stderr)
	}
}

// TestExecute runs the app with mock buffers and redirected stdout/stderr.
func TestExecute(app *App, args []string) *TestResult {
	return TestExecuteWithStdin(app, args, nil)
}

// TestExecuteWithStdin runs the app redirecting stdout, stderr, and stdin.
func TestExecuteWithStdin(app *App, args []string, stdin io.Reader) *TestResult {
	var stdout, stderr bytes.Buffer
	origStdin := app.Stdin
	origStdout := app.Stdout
	origStderr := app.Stderr

	app.Stdin = stdin
	app.Stdout = &stdout
	app.Stderr = &stderr
	defer func() {
		app.Stdin = origStdin
		app.Stdout = origStdout
		app.Stderr = origStderr
	}()

	err := app.Execute(args)
	return &TestResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Error:  err,
	}
}

// AuditOptions configures the static analysis audit helper.
type AuditOptions struct {
	AllowPathPermutations [][]string
}

// Audit traverses the app's command tree to statically verify documentation and consistency.
func Audit(app *App) error {
	return AuditWithOptions(app, AuditOptions{})
}

// AuditWithOptions traverses the app's command tree using customized options.
func AuditWithOptions(app *App, opts AuditOptions) error {
	type commandPathInfo struct {
		path    []string
		wordSet string // sorted space-separated words
	}
	var allPaths []commandPathInfo

	var walk func(cmds []Command, currentPath []string) error
	walk = func(cmds []Command, currentPath []string) error {
		seenNames := make(map[string]bool)
		for _, cmd := range cmds {
			// Subcommand name/alias collision checks
			if seenNames[cmd.Name] {
				return fmt.Errorf("duplicate subcommand name %q under path %q", cmd.Name, strings.Join(currentPath, " "))
			}
			seenNames[cmd.Name] = true
			for _, alias := range cmd.Aliases {
				if seenNames[alias] {
					return fmt.Errorf("duplicate subcommand alias %q under path %q", alias, strings.Join(currentPath, " "))
				}
				seenNames[alias] = true
			}

			// Description check
			if cmd.Description == "" {
				return fmt.Errorf("command %q under path %q is missing a Description", cmd.Name, strings.Join(currentPath, " "))
			}

			cmdPath := append(currentPath, cmd.Name)

			// Path permutation checks
			words := append([]string(nil), cmdPath...)
			sort.Strings(words)
			wordSetKey := strings.Join(words, " ")

			isAllowed := false
			for _, allowed := range opts.AllowPathPermutations {
				allowedWords := append([]string(nil), allowed...)
				sort.Strings(allowedWords)
				if strings.Join(allowedWords, " ") == wordSetKey {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				for _, prev := range allPaths {
					if prev.wordSet == wordSetKey {
						return fmt.Errorf("inconsistent path permutation detected: path %q and path %q use the same set of words in a different order", strings.Join(prev.path, " "), strings.Join(cmdPath, " "))
					}
				}
			}
			allPaths = append(allPaths, commandPathInfo{path: cmdPath, wordSet: wordSetKey})

			// Duplicate option checks
			seenFlags := make(map[string]bool)
			checkOptions := func(options []Option) error {
				for _, opt := range options {
					spec := parseFlagSpec(opt.Flags)
					if len(spec.longNames) == 0 && len(spec.shortNames) == 0 {
						return fmt.Errorf("option flags spec %q is invalid (missing flag names)", opt.Flags)
					}
					for _, l := range spec.longNames {
						key := "--" + l
						if seenFlags[key] {
							return fmt.Errorf("duplicate option --%s declared in command %q", l, cmd.Name)
						}
						seenFlags[key] = true
					}
					for _, s := range spec.shortNames {
						key := "-" + s
						if seenFlags[key] {
							return fmt.Errorf("duplicate option shorthand -%s declared in command %q", s, cmd.Name)
						}
						seenFlags[key] = true
					}
				}
				return nil
			}

			if err := checkOptions(cmd.PersistentOptions); err != nil {
				return err
			}
			if err := checkOptions(cmd.Options); err != nil {
				return err
			}

			if err := walk(cmd.Subcommands, cmdPath); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(app.Commands, nil); err != nil {
		return err
	}
	return nil
}
