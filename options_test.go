package clihelp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func newFlagSetForTest(name string) *pflag.FlagSet {
	return pflag.NewFlagSet(name, pflag.ContinueOnError)
}

type customValue struct {
	val string
}

func (c *customValue) String() string     { return c.val }
func (c *customValue) Set(s string) error { c.val = s; return nil }
func (c *customValue) Type() string       { return "custom" }

func TestOptionsParsing(t *testing.T) {
	var (
		strVal      string
		intVal      int
		boolVal     bool
		toggleTrue  bool
		toggleFalse bool
		durVal      time.Duration
		sliceVal    []string
		enumVal     string
		customVal   customValue
	)

	app := &App{
		Name: "testopt",
		Commands: []Command{
			{
				Name: "run",
				Options: []Option{
					String(&strVal, "-s, -S, --str <val>", "default_str", "string flag"),
					Int(&intVal, "-i, --int <num>", 42, "int flag"),
					Bool(&boolVal, "-b, --bool", false, "bool flag"),
					BoolToggle(&toggleTrue, "--[no-]feature-a", true, "toggle feature a"),
					BoolToggle(&toggleFalse, "--[no-]feature-b", false, "toggle feature b"),
					Duration(&durVal, "-d, --duration <time>", 5*time.Second, "duration flag"),
					StringSlice(&sliceVal, "-t, --tag <tag>", []string{"init"}, "string slice"),
					Enum(&enumVal, "-e, --env <env>", []string{"dev", "staging", "prod"}, "dev", "target env"),
					Var(&customVal, "--custom <c>", "custom value"),
				},
				Run: func(ctx *Context) error {
					return nil
				},
			},
		},
	}

	// Test default values
	err := app.ExecuteContext(context.Background(), []string{"run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal != "default_str" || intVal != 42 || boolVal != false || toggleTrue != true || toggleFalse != false || durVal != 5*time.Second || enumVal != "dev" {
		t.Fatalf("defaults not set correctly")
	}

	// Test passing flags
	args := []string{
		"run",
		"--str", "custom_str",
		"-i", "100",
		"-b",
		"--no-feature-a",
		"--feature-b",
		"-d", "10m",
		"--tag", "foo,bar",
		"--tag", "baz",
		"-e", "prod",
		"--custom", "hello",
	}
	err = app.ExecuteContext(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error with flags: %v", err)
	}

	if strVal != "custom_str" {
		t.Errorf("got strVal %q, want custom_str", strVal)
	}
	if intVal != 100 {
		t.Errorf("got intVal %d, want 100", intVal)
	}
	if !boolVal {
		t.Errorf("got boolVal false, want true")
	}
	if toggleTrue {
		t.Errorf("got toggleTrue true, want false after --no-feature-a")
	}
	if !toggleFalse {
		t.Errorf("got toggleFalse false, want true after --feature-b")
	}
	if durVal != 10*time.Minute {
		t.Errorf("got durVal %v, want 10m", durVal)
	}
	if len(sliceVal) != 3 || sliceVal[0] != "foo" || sliceVal[1] != "bar" || sliceVal[2] != "baz" {
		t.Errorf("got sliceVal %v, want [foo, bar, baz]", sliceVal)
	}
	if enumVal != "prod" {
		t.Errorf("got enumVal %q, want prod", enumVal)
	}
	if customVal.val != "hello" {
		t.Errorf("got customVal %q, want hello", customVal.val)
	}

	// Test second shorthand flag alias (-S)
	err = app.ExecuteContext(context.Background(), []string{"run", "-S", "aliased"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal != "aliased" {
		t.Errorf("got strVal %q, want aliased", strVal)
	}

	// Test invalid enum
	err = app.ExecuteContext(context.Background(), []string{"run", "-e", "invalid"})
	if err == nil {
		t.Fatalf("expected error for invalid enum value, got nil")
	}
	if !strings.Contains(err.Error(), "invalid value \"invalid\"") {
		t.Errorf("unexpected error string: %v", err)
	}
}

func TestParseFlagSpec(t *testing.T) {
	spec := parseFlagSpec("-p, -P, --podcast <name>")
	if len(spec.shortNames) != 2 || spec.shortNames[0] != "p" || spec.shortNames[1] != "P" {
		t.Errorf("unexpected short names: %v", spec.shortNames)
	}
	if len(spec.longNames) != 1 || spec.longNames[0] != "podcast" {
		t.Errorf("unexpected long names: %v", spec.longNames)
	}
	if spec.placeholder != "<name>" {
		t.Errorf("unexpected placeholder: %v", spec.placeholder)
	}

	toggleSpec := parseFlagSpec("--[no-]check-new")
	if !toggleSpec.isToggle || toggleSpec.baseToggle != "check-new" {
		t.Errorf("unexpected toggle spec: %+v", toggleSpec)
	}
}

func TestParseFlagSpecEqualsForm(t *testing.T) {
	spec := parseFlagSpec("-o, --output=PATH")
	if len(spec.longNames) != 1 || spec.longNames[0] != "output" {
		t.Errorf("--output=PATH should register long name 'output', got %v", spec.longNames)
	}
	if len(spec.shortNames) != 1 || spec.shortNames[0] != "o" {
		t.Errorf("unexpected short names: %v", spec.shortNames)
	}
}

func TestEnumRejectsInvalidDefault(t *testing.T) {
	var env string
	opt := Enum(&env, "--env <env>", []string{"dev", "prod"}, "qa", "target env")
	err := opt.Binder(newFlagSetForTest("test"))
	if err == nil {
		t.Fatal("expected error for enum default outside allowed set, got nil")
	}
	if !strings.Contains(err.Error(), "qa") {
		t.Errorf("error should mention the bad default, got: %v", err)
	}
}

func TestStringSliceDoesNotAliasCallerSlice(t *testing.T) {
	defaults := []string{"a", "b"}
	var got []string
	opt := StringSlice(&got, "--tag <t>", defaults, "tags")
	fs := newFlagSetForTest("test")
	if err := opt.Binder(fs); err != nil {
		t.Fatal(err)
	}
	defaults[0] = "MUTATED"
	if got[0] == "MUTATED" {
		t.Error("StringSlice aliased the caller's backing array")
	}
}

func TestHelpFlagRejection(t *testing.T) {
	var helpVal bool
	app := &App{
		Name: "testhelp",
		Commands: []Command{
			{
				Name: "sub",
				Options: []Option{
					Bool(&helpVal, "-h, --help", false, "explicit help"),
				},
				Run: func(ctx *Context) error { return nil },
			},
		},
	}

	err := app.ExecuteContext(context.Background(), []string{"sub"})
	if err == nil {
		t.Fatalf("expected error when declaring -h/--help option, got nil")
	}
	if !strings.Contains(err.Error(), "automatically managed by clihelp") {
		t.Errorf("expected automatic help management error, got: %v", err)
	}
}
