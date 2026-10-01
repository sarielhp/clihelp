package clihelp

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

var (
	errNoNumbers = errors.New("no \" is N columns\" in the message")
	fmtSscanf    = fmt.Sscanf
)

// What an author sees when the audit fails is the product. These tests read the
// messages the way an author would: does each one say which command, which
// row, what the text is, how far over it is, and what to do?

func TestAuditMessageNamesTheTextAndTheOvershoot(t *testing.T) {
	desc := "Build the episodes from raw audio and package everything for release into the output directory"
	app := &App{Name: "a", Commands: []Command{{Name: "build", Description: desc}}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil")
	}
	msg := err.Error()
	for _, want := range []string{`description of "build"`, `"Build the episodes from raw audio`, "…", "over the", "columns", "width 80"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	// The overshoot is arithmetic the author should not have to do: columns - fit.
	var cols, over, fit int
	if _, scanErr := sscanf(msg, &cols, &over, &fit); scanErr != nil {
		t.Fatalf("cannot read the numbers from the message: %v\n%s", scanErr, msg)
	}
	if cols-over != fit {
		t.Errorf("%d columns, %d over, %d fit: the numbers do not add up\n%s", cols, over, fit, msg)
	}
}

// sscanf pulls "is N columns, M over the K that fit" out of an audit message.
func sscanf(msg string, cols, over, fit *int) (int, error) {
	i := strings.Index(msg, " is ")
	if i < 0 {
		return 0, errNoNumbers
	}
	return fmtSscanf(msg[i:], " is %d columns, %d over the %d that fit", cols, over, fit)
}

func TestAuditMessageSaysFlagSuffixesAreCounted(t *testing.T) {
	app := &App{Name: "a", GlobalFlags: []Option{{
		Flags:       "--path",
		Description: "Where the output goes",
		DefaultText: strings.Repeat("/very/long/default/", 5),
	}}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil")
	}
	if !strings.Contains(err.Error(), "(default: …)") {
		t.Errorf("a flag row measured with its default suffix does not say so:\n%v", err)
	}
	bare := &App{Name: "a", GlobalFlags: []Option{{Flags: "--out", Description: strings.Repeat("word ", 30)}}}
	if err = Audit(bare, AuditOptions{SkipExampleValidation: true}); err == nil || strings.Contains(err.Error(), "(default: …)") {
		t.Errorf("a flag row with no suffix should not mention suffixes:\n%v", err)
	}
	plain := &App{Name: "a", Commands: []Command{{Name: "x", Description: strings.Repeat("word ", 30)}}}
	err = Audit(plain, AuditOptions{SkipExampleValidation: true})
	if err == nil || strings.Contains(err.Error(), "(default: …)") {
		t.Errorf("a command row should not mention flag suffixes:\n%v", err)
	}
}

func TestAuditRejectsAWidthTooNarrowToMeasure(t *testing.T) {
	app := &App{Name: "a", Commands: []Command{{Name: "x", Description: "fine"}}}
	for _, width := range []int{1, 10, minAuditWidth - 1} {
		err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: width})
		if err == nil || !strings.Contains(err.Error(), "at least") {
			t.Errorf("Width %d: Audit = %v, want an error saying the minimum", width, err)
		}
		if err != nil && strings.Contains(err.Error(), "-") && strings.Contains(err.Error(), "fit") {
			t.Errorf("Width %d produced a negative-width message:\n%v", width, err)
		}
	}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: minAuditWidth}); err != nil {
		t.Errorf("Width %d (the minimum) = %v, want nil", minAuditWidth, err)
	}
}

func TestAuditCountsTheLayoutProblems(t *testing.T) {
	long := strings.Repeat("word ", 30)
	app := &App{Name: "a", Commands: []Command{
		{Name: "one", Description: long}, {Name: "two", Description: long}, {Name: "three", Description: long},
	}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil || !strings.HasPrefix(err.Error(), "3 layout problems:") {
		t.Errorf("Audit = %v, want a '3 layout problems:' header", err)
	}
	one := &App{Name: "a", Commands: []Command{{Name: "one", Description: long}}}
	err = Audit(one, AuditOptions{SkipExampleValidation: true})
	if err == nil || !strings.HasPrefix(err.Error(), "1 layout problem:") {
		t.Errorf("Audit = %v, want a '1 layout problem:' header", err)
	}
}

func TestAuditNamesTheTopLevel(t *testing.T) {
	app := &App{Name: "a", Commands: []Command{{Name: "build"}}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil")
	}
	if strings.Contains(err.Error(), `under path ""`) {
		t.Errorf("an empty path is printed as \"\":\n%v", err)
	}
	if !strings.Contains(err.Error(), "top level") {
		t.Errorf("the message does not say the command is at the top level:\n%v", err)
	}
}

func TestAuditPeriodMessageAgreesWithItsNumbers(t *testing.T) {
	tests := []struct {
		name  string
		descs []string
		want  string
	}{
		{"one and one", []string{"A.", "B"}, "c0 ends with one, c1 does not"},
		{"two and one", []string{"A.", "B.", "C"}, "c0, c1 end with one, c2 does not"},
		{"one and two", []string{"A.", "B", "C"}, "c0 ends with one, c1, c2 do not"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmds []Command
			for i, d := range tt.descs {
				cmds = append(cmds, Command{Name: "c" + string(rune('0'+i)), Description: d})
			}
			err := Audit(&App{Name: "a", Commands: cmds}, AuditOptions{SkipExampleValidation: true})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Audit = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

// Two commands each declare a persistent --out whose description is too long
// for the page it is listed on. The inherited-flags scope used to carry no
// command name and the messages merged into one, so the author fixed one, re-ran,
// and met the other with no way to tell where it lived.
func TestAuditNamesTheCommandThatOwnsAnInheritedFlag(t *testing.T) {
	long := strings.Repeat("word ", 20)
	app := &App{Name: "a", Commands: []Command{
		{Name: "one", Description: "One", PersistentOptions: []Option{{Flags: "--out", Description: long + "one"}},
			Subcommands: []Command{{Name: "leaf", Description: "Leaf"}}},
		{Name: "two", Description: "Two", PersistentOptions: []Option{{Flags: "--out", Description: long + "two"}}},
	}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil {
		t.Fatal("Audit = nil")
	}
	msg := err.Error()
	for _, want := range []string{"command one", "command two"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message does not name %q:\n%s", want, msg)
		}
	}
	// A flag declared on one command is reported under that command, once, even
	// though every page beneath it lists the flag.
	if n := strings.Count(msg, "word word word word word word word word word word…"); n < 2 {
		t.Errorf("expected both flags to be reported with their text, saw %d:\n%s", n, msg)
	}
}

// `help flags` is a listing like -h, with its own column: it carries the
// library's -h/--help and -v/--version rows and leaves out the root page's own
// Options. A description can fit the root page and wrap here, and the audit has to
// say so, naming the page.
func TestAuditChecksTheHelpFlagsPage(t *testing.T) {
	desc := strings.Repeat("x", 64)
	app := &App{Name: "a", Version: "1", GlobalFlags: []Option{{Flags: "-x", Description: desc}},
		Commands: []Command{{Name: "job", Description: "Jobs"}}}
	err := Audit(app, AuditOptions{SkipExampleValidation: true})
	if err == nil || !strings.Contains(err.Error(), `help flags: description of "-x"`) {
		t.Errorf("Audit = %v, want the help flags page named", err)
	}
	// Without a version row the column is narrower and the same text fits.
	app.Version = ""
	if err := Audit(app, AuditOptions{SkipExampleValidation: true}); err != nil {
		t.Errorf("Audit = %v, want nil when there is no -v row to widen the column", err)
	}
}

// The library's own rows on that page shape the column but are not the author's
// to shorten, so even at the narrowest supported width they are never reported.
func TestAuditNeverReportsTheLibrarysFlagRows(t *testing.T) {
	app := &App{Name: "a", Version: "1", ExtendedHelpFlag: true, EnableExamplesFlag: true,
		GlobalFlags: []Option{{Flags: "-x", Description: "short"}},
		Commands:    []Command{{Name: "job", Description: "Jobs", Examples: []Example{{Line: "a job"}}}}}
	if err := Audit(app, AuditOptions{SkipExampleValidation: true, Width: minAuditWidth}); err != nil {
		t.Errorf("Audit = %v, want nil: only the author's rows are reported", err)
	}
}
