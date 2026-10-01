package clihelp

import (
	"fmt"
	"os"
	"testing"

	"github.com/mattn/go-runewidth"
)

// skipper is the part of *testing.T the missing-tool helpers use, so they can be
// tested against a fake.
type skipper interface {
	Helper()
	Skip(args ...any)
	Fatal(args ...any)
}

// skipMissingTool skips a test that needs a program that is not installed. About
// thirty tests drive a real bash, zsh, fish, man or sh; on a machine without one
// they used to skip in silence, so a runner missing zsh quietly lost all of the
// live-zsh coverage and still reported green. With CLIHELP_REQUIRE_SHELLS=1
// (tools/check.sh sets it) a missing tool fails the test instead.
func skipMissingTool(t skipper, reason string) {
	t.Helper()
	if os.Getenv("CLIHELP_REQUIRE_SHELLS") == "1" {
		t.Fatal("required tool is missing: " + reason)
		return
	}
	t.Skip(reason)
}

func skipMissingToolf(t skipper, format string, args ...any) {
	t.Helper()
	skipMissingTool(t, fmt.Sprintf(format, args...))
}

type fakeSkipper struct{ skipped, failed string }

func (f *fakeSkipper) Helper()        {}
func (f *fakeSkipper) Skip(a ...any)  { f.skipped = fmt.Sprint(a...) }
func (f *fakeSkipper) Fatal(a ...any) { f.failed = fmt.Sprint(a...) }

func TestSkipMissingToolFailsOnlyWhenShellsAreRequired(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		wantSkip   bool
		wantFailed bool
	}{
		{"unset skips", "", true, false},
		{"zero skips", "0", true, false},
		{"one fails", "1", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLIHELP_REQUIRE_SHELLS", tt.env)
			f := &fakeSkipper{}
			skipMissingToolf(f, "%s not found", "zsh")
			if (f.skipped != "") != tt.wantSkip || (f.failed != "") != tt.wantFailed {
				t.Errorf("skipped=%q failed=%q, want skip=%v fail=%v", f.skipped, f.failed, tt.wantSkip, tt.wantFailed)
			}
			if f.skipped != "" && f.skipped != "zsh not found" {
				t.Errorf("skip reason %q", f.skipped)
			}
		})
	}
}

// narrowWidthRules pins go-runewidth to the narrow rules for one test, whatever
// the environment says, for a test whose expected numbers are narrow-rule
// numbers. The suite also runs once under RUNEWIDTH_EASTASIAN=1 (tools/check.sh),
// where an ambiguous-width character such as a bullet is two columns wide.
func narrowWidthRules(t *testing.T) {
	t.Helper()
	was := runewidth.DefaultCondition.EastAsianWidth
	runewidth.DefaultCondition.EastAsianWidth = false
	t.Cleanup(func() { runewidth.DefaultCondition.EastAsianWidth = was })
}
