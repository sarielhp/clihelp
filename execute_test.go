package clihelp

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecuteLifecycleAndHooks(t *testing.T) {
	var callOrder []string
	var globalFlag string
	var localFlag string
	var outBuf bytes.Buffer

	app := &App{
		Name:    "lifecycle_app",
		Version: "2.1.0",
		Stdout:  &outBuf,
		PersistentOptions: []Option{
			String(&globalFlag, "-g, --global <g>", "init_g", "global flag"),
		},
		BeforeRun: func(ctx *Context) error {
			callOrder = append(callOrder, "BeforeRun")
			return nil
		},
		AfterRun: func(ctx *Context) error {
			callOrder = append(callOrder, "AfterRun")
			return nil
		},
		Commands: []Command{
			{
				Name:    "sync",
				Aliases: []string{"s", "fetch"},
				Options: []Option{
					String(&localFlag, "-l, --local <l>", "init_l", "local flag"),
				},
				Args: ExactArgs(1),
				PreRun: func(ctx *Context) error {
					callOrder = append(callOrder, "PreRun")
					return nil
				},
				Run: func(ctx *Context) error {
					callOrder = append(callOrder, "Run")
					if len(ctx.Args) != 1 || ctx.Args[0] != "target" {
						t.Errorf("unexpected ctx.Args: %v", ctx.Args)
					}
					return nil
				},
				PostRun: func(ctx *Context) error {
					callOrder = append(callOrder, "PostRun")
					return nil
				},
			},
		},
	}

	// 1. Run using alias 'fetch'
	err := app.ExecuteContext(context.Background(), []string{"fetch", "-g", "val_g", "-l", "val_l", "target"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedOrder := []string{"BeforeRun", "PreRun", "Run", "PostRun", "AfterRun"}
	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("call order mismatch, got: %v", callOrder)
	}
	for i, name := range expectedOrder {
		if callOrder[i] != name {
			t.Errorf("step %d: got %s, want %s", i, callOrder[i], name)
		}
	}

	if globalFlag != "val_g" || localFlag != "val_l" {
		t.Errorf("flags not updated: g=%s, l=%s", globalFlag, localFlag)
	}

	// 2. Test positional arg validation failure
	err = app.ExecuteContext(context.Background(), []string{"sync"})
	if err == nil {
		t.Fatalf("expected ExactArgs validation error, got nil")
	}

	// 3. Test lifecycle error aborts sequence
	app.BeforeRun = func(ctx *Context) error {
		return errors.New("before run failed")
	}
	err = app.ExecuteContext(context.Background(), []string{"sync", "target"})
	if err == nil || err.Error() != "before run failed" {
		t.Fatalf("expected error 'before run failed', got %v", err)
	}
}

func TestExecuteTypoSuggestion(t *testing.T) {
	app := &App{
		Name: "testcli",
		Commands: []Command{
			{Name: "scan", Aliases: []string{"rescan"}},
			{Name: "download"},
		},
	}

	err := app.ExecuteContext(context.Background(), []string{"scna"})
	if err == nil {
		t.Fatalf("expected typo error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown command "scna" for "testcli". Did you mean "scan"?`) {
		t.Errorf("unexpected error message: %v", err)
	}

	err = app.ExecuteContext(context.Background(), []string{"completelyunknown"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown command "completelyunknown" for "testcli"`) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecuteVersionAndHelpInterception(t *testing.T) {
	var outBuf bytes.Buffer
	app := &App{
		Name:    "versionapp",
		Version: "1.2.3",
		Stdout:  &outBuf,
		Commands: []Command{
			{
				Name:        "info",
				Description: "Print info",
				Run: func(ctx *Context) error {
					return nil
				},
			},
		},
	}

	// Test --version
	outBuf.Reset()
	err := app.ExecuteContext(context.Background(), []string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "versionapp 1.2.3") {
		t.Errorf("version output missing, got: %q", outBuf.String())
	}

	// Test -h
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"-h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Usage of versionapp:") {
		t.Errorf("help output missing, got: %q", outBuf.String())
	}

	// Test help subcommand
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"help", "info"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "info") {
		t.Errorf("command help output missing, got: %q", outBuf.String())
	}
}

func TestExecuteNestedPersistentOptions(t *testing.T) {
	var (
		appGlobal string
		parentOpt string
		childOpt  string
	)

	app := &App{
		Name: "nestcli",
		PersistentOptions: []Option{
			String(&appGlobal, "--app-global <val>", "default_app", "app global"),
		},
		Commands: []Command{
			{
				Name: "parent",
				PersistentOptions: []Option{
					String(&parentOpt, "--parent-opt <val>", "default_parent", "parent persistent"),
				},
				Subcommands: []Command{
					{
						Name: "child",
						Options: []Option{
							String(&childOpt, "--child-opt <val>", "default_child", "child option"),
						},
						Run: func(ctx *Context) error {
							return nil
						},
					},
				},
			},
		},
	}

	err := app.ExecuteContext(context.Background(), []string{
		"parent", "child",
		"--app-global", "set_app",
		"--parent-opt", "set_parent",
		"--child-opt", "set_child",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if appGlobal != "set_app" || parentOpt != "set_parent" || childOpt != "set_child" {
		t.Errorf("options not set properly: app=%s, parent=%s, child=%s", appGlobal, parentOpt, childOpt)
	}
}

func TestExecuteCustomHelpCommand(t *testing.T) {
	customHelpCalled := false
	app := &App{
		Name: "customhelpapp",
		Commands: []Command{
			{
				Name: "help",
				Run: func(ctx *Context) error {
					customHelpCalled = true
					return nil
				},
			},
		},
	}

	err := app.ExecuteContext(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !customHelpCalled {
		t.Errorf("expected custom help command to be executed")
	}
}

func TestPrintError(t *testing.T) {
	// Ensure calling PrintError with nil or an error does not panic
	PrintError(nil)
	PrintError(errors.New("sample error"))
}
