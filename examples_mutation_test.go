package clihelp

import (
	"strings"
	"testing"
)

func auditExampleApp(lines ...string) *App {
	exs := make([]Example, 0, len(lines))
	for _, l := range lines {
		exs = append(exs, Example{Line: l, Description: "An example."})
	}
	var tag string
	return &App{
		Name: "demo",
		Commands: []Command{{
			Name: "build", Description: "Build it.", Aliases: []string{"b"},
			Args:     MaximumNArgs(2),
			Options:  []Option{String(&tag, "-t, --tag <v>", "", "Tag it.")},
			Run:      func(*Context) error { return nil },
			Examples: exs,
		}},
	}
}

// Audit is what the README tells people to run in CI, so an example it accepts
// must run and an example it rejects must not. These are the rules that decide
// which line is even looked at.
func TestExampleLinesThatAreSkipped(t *testing.T) {
	for _, tt := range []struct {
		name string
		line string
		ok   bool
	}{
		{"a hash comment is not a command", "# demo build --bogus", true},
		{"a slash comment is not a command", "// demo build --bogus", true},
		{"an indented hash comment", "   # demo build --bogus", true},
		{"a blank line", "   ", true},
		{"a real line is still checked", "demo build --bogus", false},
		{"a hash inside a line is not a comment", "demo build --tag=#tag", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Audit(auditExampleApp(tt.line))
			if tt.ok && err != nil {
				t.Errorf("%q was rejected: %v", tt.line, err)
			}
			if !tt.ok && err == nil {
				t.Errorf("%q was accepted, but it names a flag the command does not have", tt.line)
			}
		})
	}
}

// The application's own name is stripped however the author wrote it, including
// the "./name" form a reader copies straight out of a terminal.
func TestTheApplicationNameIsStrippedInEveryForm(t *testing.T) {
	for _, line := range []string{
		"demo build --tag x",
		"./demo build --tag x",
		"build --tag x",
	} {
		if err := Audit(auditExampleApp(line)); err != nil {
			t.Errorf("%q was rejected: %v", line, err)
		}
	}

	// And it is only stripped when it really is the name: a command that happens
	// to share a prefix is not.
	tokens, err := splitExampleCommandLine("./demo build")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || tokens[0] != "./demo" {
		t.Errorf("tokenising kept %v; the name is stripped later, by name", tokens)
	}

	// A command that takes no positional arguments is what makes the stripping
	// observable: left in place, "./demo" arrives as one.
	var strict string
	app := &App{Name: "demo", Commands: []Command{{
		Name: "clean", Description: "Clean it.", Args: NoArgs,
		Options: []Option{String(&strict, "--force", "", "Force it.")},
		Run:     func(*Context) error { return nil },
		Examples: []Example{
			{Line: "demo clean", Description: "Plain."},
			{Line: "./demo clean", Description: "As copied from a terminal."},
		},
	}}}
	if err := Audit(app); err != nil {
		t.Errorf("a ./-qualified example was rejected: %v", err)
	}
}

// An example may be written from the command's own point of view, showing only
// what the command takes. Retrying with the command's name prepended is what
// lets an author write "episode01.wav" in the examples of the command that
// processes it.
//
// It is the *positional* form this supports. A line that begins with a flag
// resolves at the root without error, so the retry never fires and the flag is
// judged against the application's options rather than the command's — an
// example written as "--tag x" is rejected. Every example in this repository and
// in the one real application built on it writes the full form, so that is
// recorded here rather than treated as a defect.
func TestAnExampleMayOmitItsOwnCommandName(t *testing.T) {
	if err := Audit(auditExampleApp("source.wav")); err != nil {
		t.Errorf("a positional example written without the command name was rejected: %v", err)
	}

	// The retry must not turn a real mistake into a pass.
	if err := Audit(auditExampleApp("nosuchcommand --nosuchflag")); err == nil {
		t.Error("an example naming a flag that does not exist was accepted")
	}
}

// A help invocation is a legitimate example and is not validated as though the
// help flag were a command's own.
func TestAHelpExampleIsAccepted(t *testing.T) {
	for _, line := range []string{
		"demo build -h",
		"demo build --help",
		"demo help build",
	} {
		if err := Audit(auditExampleApp(line)); err != nil {
			t.Errorf("%q was rejected: %v", line, err)
		}
	}

	// A command that requires an argument is what makes the skip observable:
	// asking for help is not running the command, so its positional rules do not
	// apply to the help line. Both spellings are covered — "help <command>",
	// which resolution recognises, and "<command> --help", which only the flag
	// check below it can.
	var v string
	app := &App{Name: "demo", Commands: []Command{{
		Name: "process", Description: "Process it.", Args: ExactArgs(1),
		Parameters: []Param{{Name: "<file>", Description: "File to process."}},
		Options:    []Option{String(&v, "--mode <m>", "", "Mode.")},
		Run:        func(*Context) error { return nil },
		Examples: []Example{
			{Line: "demo process report.txt", Description: "Ordinary."},
			{Line: "demo process --help", Description: "Read about it, by flag."},
			{Line: "demo help process", Description: "Read about it, by verb."},
		},
	}}}
	if err := Audit(app); err != nil {
		t.Errorf("a help example on a command requiring arguments was rejected: %v", err)
	}

	// And an application with a required option of its own. At run time both
	// help forms print the page and exit 0 without the flag being given, so an
	// audit that demanded it would reject every help example such an application
	// has.
	var cfg string
	strictApp := &App{
		Name:              "demo",
		PersistentOptions: []Option{Required(String(&cfg, "--config <f>", "", "Config file."))},
		Commands: []Command{{
			Name: "process", Description: "Process it.", Args: ExactArgs(1),
			Run: func(*Context) error { return nil },
			Examples: []Example{
				{Line: "demo help process", Description: "By verb."},
				{Line: "demo process --help", Description: "By flag."},
				{Line: "demo --config c.toml process report.txt", Description: "Ordinary."},
			},
		}},
	}
	if err := Audit(strictApp); err != nil {
		t.Errorf("a help example was rejected for want of a required flag it never needs: %v", err)
	}
	if cfg != "" {
		t.Errorf("validating examples wrote %q through the application's own target", cfg)
	}
}

// A multi-line Example is several commands, and every one of them is checked.
// The alternative reading — a command followed by its output — is not what this
// library offers, and a branch implying it sat unreachable in
// cleanExampleCommandLine until this survey found nothing could reach it.
func TestEveryLineOfAMultiLineExampleIsChecked(t *testing.T) {
	good := "demo build --tag x\ndemo build --tag y"
	if err := Audit(auditExampleApp(good)); err != nil {
		t.Errorf("a multi-line example of two good commands was rejected: %v", err)
	}

	bad := "demo build --tag x\ndemo build --nosuchflag"
	err := Audit(auditExampleApp(bad))
	if err == nil {
		t.Fatal("a bad second line was not checked")
	}
	if !strings.Contains(err.Error(), "nosuchflag") {
		t.Errorf("the error does not name the offending line: %v", err)
	}

	// One line in, one line out.
	if got := cleanExampleCommandLine("  $ demo build  "); got != "demo build" {
		t.Errorf("cleanExampleCommandLine(%q) = %q", "  $ demo build  ", got)
	}
}

// A shell operator ends the command clihelp validates: what follows a pipe or a
// semicolon is another program's business.
func TestAShellOperatorEndsTheValidatedCommand(t *testing.T) {
	for _, tt := range []struct {
		line string
		want []string
	}{
		{"demo build | grep --colour x", []string{"demo", "build"}},
		{"demo build ; rm -rf /tmp/x", []string{"demo", "build"}},
		{"demo build && demo build --tag y", []string{"demo", "build"}},
		{"demo build > out.txt", []string{"demo", "build"}},
		{"demo build 2>/dev/null", []string{"demo", "build"}},
	} {
		got, err := splitExampleCommandLine(tt.line)
		if err != nil {
			t.Errorf("%q: %v", tt.line, err)
			continue
		}
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("%q tokenised to %v, want %v", tt.line, got, tt.want)
		}
		// And the whole line passes the audit, because the tail is not ours.
		if err := Audit(auditExampleApp(tt.line)); err != nil {
			t.Errorf("%q was rejected: %v", tt.line, err)
		}
	}
}

// An example declared on a command may be written relative to that command or
// to any of its ancestors, not only from the root.
//
// CompletionCommand() ships the example "completion install --no-keys zsh",
// written for a completion command mounted at the root. An application that
// mounts it under "config" gets the path "config completion install", and Audit
// — which the README tells people to run in CI — rejected the library's own
// example as an unknown flag.
func TestAnExampleMayBeRelativeToAnAncestor(t *testing.T) {
	var deep string
	app := &App{
		Name: "demo",
		Commands: []Command{{
			Name: "config", Description: "Configure it.",
			Run: func(*Context) error { return nil },
			Subcommands: []Command{{
				Name: "remote", Description: "Remotes.",
				Run: func(*Context) error { return nil },
				Subcommands: []Command{{
					Name: "add", Description: "Add one.",
					Args:    ExactArgs(1),
					Options: []Option{String(&deep, "--url <u>", "", "URL.")},
					Run:     func(*Context) error { return nil },
					Examples: []Example{
						{Line: "demo config remote add origin --url u", Description: "From the root."},
						{Line: "remote add origin --url u", Description: "Relative to config."},
						{Line: "add origin --url u", Description: "Relative to remote."},
						{Line: "origin --url u", Description: "Relative to the command itself."},
					},
				}},
			}},
		}},
	}
	if err := Audit(app); err != nil {
		t.Errorf("an example written relative to an ancestor was rejected: %v", err)
	}

	// The tolerance must not swallow a genuine mistake.
	bad := &App{
		Name: "demo",
		Commands: []Command{{
			Name: "config", Description: "Configure it.",
			Run: func(*Context) error { return nil },
			Subcommands: []Command{{
				Name: "set", Description: "Set it.", Args: ExactArgs(1),
				Run:      func(*Context) error { return nil },
				Examples: []Example{{Line: "set key --nosuchflag", Description: "Wrong."}},
			}},
		}},
	}
	if err := Audit(bad); err == nil {
		t.Error("an example naming a flag that does not exist was accepted")
	}
}

// The library's own CompletionCommand, mounted anywhere, audits clean.
func TestCompletionCommandExamplesAuditWhereverMounted(t *testing.T) {
	for _, tt := range []struct {
		name string
		app  *App
	}{
		{"at the root", &App{Name: "demo", Commands: []Command{CompletionCommand()}}},
		{"nested one deep", &App{Name: "demo", Commands: []Command{{
			Name: "config", Description: "Configure it.", Run: func(*Context) error { return nil },
			Subcommands: []Command{CompletionCommand()},
		}}}},
		{"nested two deep", &App{Name: "demo", Commands: []Command{{
			Name: "config", Description: "Configure it.", Run: func(*Context) error { return nil },
			Subcommands: []Command{{
				Name: "shell", Description: "Shell things.", Run: func(*Context) error { return nil },
				Subcommands: []Command{CompletionCommand()},
			}},
		}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := Audit(tt.app); err != nil {
				t.Errorf("Audit rejected the library's own examples: %v", err)
			}
		})
	}
}
