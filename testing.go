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
	SkipExampleValidation bool
}

// Audit traverses the app's command tree to statically verify documentation and
// consistency. It is what the README recommends running in CI.
//
// The options are variadic so that the common call is Audit(app) and the
// customised one is Audit(app, AuditOptions{...}); only the first is read. This
// was two functions, Audit and AuditWithOptions, which is two entry points for
// one job and a default that had to be documented rather than shown.
func Audit(app *App, opts ...AuditOptions) error {
	var o AuditOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return audit(app, o)
}

type commandPathInfo struct {
	path    []string
	wordSet string
}

func auditCommandNameUniqueness(seenNames map[string]bool, cmd Command, currentPath []string) error {
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
	if cmd.Description == "" {
		return fmt.Errorf("command %q under path %q is missing a Description", cmd.Name, strings.Join(currentPath, " "))
	}
	return nil
}

func checkPathPermutation(allPaths []commandPathInfo, cmdPath []string, allowedPermutations [][]string) (string, error) {
	words := append([]string(nil), cmdPath...)
	sort.Strings(words)
	wordSetKey := strings.Join(words, " ")

	for _, allowed := range allowedPermutations {
		allowedWords := append([]string(nil), allowed...)
		sort.Strings(allowedWords)
		if strings.Join(allowedWords, " ") == wordSetKey {
			return wordSetKey, nil
		}
	}

	for _, prev := range allPaths {
		if prev.wordSet == wordSetKey {
			return "", fmt.Errorf("inconsistent path permutation detected: path %q and path %q use the same set of words in a different order", strings.Join(prev.path, " "), strings.Join(cmdPath, " "))
		}
	}
	return wordSetKey, nil
}

// flagOwners records the scope that claimed each flag name, so a collision can
// name both ends of it. The scopes it spans mirror what setupFlagSet binds into
// one flag set: the app's persistent and global options, every ancestor's
// persistent options, and the command's own options.
type flagOwners map[string]string

func (o flagOwners) clone() flagOwners {
	c := make(flagOwners, len(o))
	for name, scope := range o {
		c[name] = scope
	}
	return c
}

// checkOptionScope validates each spec the way the binder will, then claims
// every name it declares.
func checkOptionScope(owners flagOwners, scope string, options []Option) error {
	for _, opt := range options {
		spec := parseFlagSpec(opt.Flags)
		if err := validateOptionSpec(opt, spec); err != nil {
			return fmt.Errorf("%s: %w", scope, err)
		}
		names := make([]string, 0, len(spec.longNames)+len(spec.shortNames))
		for _, l := range spec.longNames {
			names = append(names, "--"+l)
		}
		for _, sh := range spec.shortNames {
			names = append(names, "-"+sh)
		}
		for _, name := range names {
			if prev, taken := owners[name]; taken {
				return fmt.Errorf("duplicate option %s declared in %s (already declared in %s)", name, scope, prev)
			}
			owners[name] = scope
		}
	}
	return nil
}

func validateOptionSpec(opt Option, spec flagSpec) error {
	if opt.toggle || spec.isToggle {
		return spec.validateToggle()
	}
	return spec.validate()
}

func auditCommandOptions(inherited flagOwners, cmd Command) (flagOwners, error) {
	scope := fmt.Sprintf("command %q", cmd.Name)
	persistent := inherited.clone()
	if err := checkOptionScope(persistent, scope, cmd.PersistentOptions); err != nil {
		return nil, err
	}
	local := persistent.clone()
	if err := checkOptionScope(local, scope, cmd.Options); err != nil {
		return nil, err
	}
	// Only persistent options reach the subcommands.
	return persistent, nil
}

func auditCommandTree(cmds []Command, currentPath []string, allPaths *[]commandPathInfo, inherited flagOwners, opts AuditOptions) error {
	seenNames := make(map[string]bool)
	for _, cmd := range cmds {
		if err := auditCommandNameUniqueness(seenNames, cmd, currentPath); err != nil {
			return err
		}

		cmdPath := append(append([]string(nil), currentPath...), cmd.Name)
		wordSetKey, err := checkPathPermutation(*allPaths, cmdPath, opts.AllowPathPermutations)
		if err != nil {
			return err
		}
		*allPaths = append(*allPaths, commandPathInfo{path: cmdPath, wordSet: wordSetKey})

		persistent, err := auditCommandOptions(inherited, cmd)
		if err != nil {
			return err
		}

		if err := auditCommandTree(cmd.Subcommands, cmdPath, allPaths, persistent, opts); err != nil {
			return err
		}
	}
	return nil
}

func audit(app *App, opts AuditOptions) error {
	if !opts.SkipExampleValidation {
		if err := app.ValidateAllExamples(); err != nil {
			return err
		}
	}

	owners := make(flagOwners)
	if err := checkOptionScope(owners, "the app's persistent options", app.PersistentOptions); err != nil {
		return err
	}
	if err := checkOptionScope(owners, "the app's global flags", app.GlobalFlags); err != nil {
		return err
	}

	var allPaths []commandPathInfo
	return auditCommandTree(app.Commands, nil, &allPaths, owners, opts)
}
