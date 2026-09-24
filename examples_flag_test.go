package clihelp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestEnableExamplesFlag(t *testing.T) {
	t.Parallel()

	makeApp := func(enabled bool, stdout *bytes.Buffer) *App {
		return &App{
			Name:               "myapp",
			EnableExamplesFlag: enabled,
			Stdout:             stdout,
			Examples: []Example{
				{Line: "myapp --help", Description: "Show general help"},
			},
			Commands: []Command{
				{
					Name:        "scan",
					Description: "Scan mail folders",
					Examples: []Example{
						{Line: "myapp scan inbox", Description: "Scan inbox folder"},
						{Line: "myapp scan archive", Description: "Scan archive folder"},
					},
					Run: func(ctx *Context) error { return nil },
				},
				{
					Name:        "send",
					Description: "Send mail",
					Examples: []Example{
						{Line: "myapp send --to user@example.com", Description: "Send to recipient"},
					},
					Run: func(ctx *Context) error { return nil },
				},
			},
		}
	}

	t.Run("BareFlagShowsAllExamples", func(t *testing.T) {
		var out bytes.Buffer
		app := makeApp(true, &out)

		if err := app.Execute([]string{"-E"}); err != nil {
			t.Fatalf("unexpected error for -E: %v", err)
		}
		got := StripANSI(out.String())
		if !strings.Contains(got, "myapp --help") || !strings.Contains(got, "myapp scan inbox") || !strings.Contains(got, "myapp send --to") {
			t.Errorf("-E output missing expected examples:\n%s", got)
		}

		out.Reset()
		if err := app.Execute([]string{"--examples"}); err != nil {
			t.Fatalf("unexpected error for --examples: %v", err)
		}
		got = StripANSI(out.String())
		if !strings.Contains(got, "myapp --help") || !strings.Contains(got, "myapp scan inbox") || !strings.Contains(got, "myapp send --to") {
			t.Errorf("--examples output missing expected examples:\n%s", got)
		}
	})

	t.Run("ScopedCommandExamples", func(t *testing.T) {
		var out bytes.Buffer
		app := makeApp(true, &out)

		// Top-level flag with command: myapp -E scan
		if err := app.Execute([]string{"-E", "scan"}); err != nil {
			t.Fatalf("unexpected error for -E scan: %v", err)
		}
		got := StripANSI(out.String())
		if !strings.Contains(got, "myapp scan inbox") {
			t.Errorf("-E scan output missing scan example:\n%s", got)
		}
		if strings.Contains(got, "myapp send --to") {
			t.Errorf("-E scan should not contain send example:\n%s", got)
		}

		// Trailing flag: myapp scan -E
		out.Reset()
		if err := app.Execute([]string{"scan", "-E"}); err != nil {
			t.Fatalf("unexpected error for scan -E: %v", err)
		}
		got = StripANSI(out.String())
		if !strings.Contains(got, "myapp scan inbox") {
			t.Errorf("scan -E output missing scan example:\n%s", got)
		}
		if strings.Contains(got, "myapp send --to") {
			t.Errorf("scan -E should not contain send example:\n%s", got)
		}
	})

	t.Run("DisabledByDefault", func(t *testing.T) {
		var out bytes.Buffer
		app := makeApp(false, &out)

		err := app.Execute([]string{"-E"})
		if err == nil {
			t.Fatal("expected error for -E when disabled, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected ErrUsage, got: %v", err)
		}

		err = app.Execute([]string{"--examples"})
		if err == nil {
			t.Fatal("expected error for --examples when disabled, got nil")
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("expected ErrUsage, got: %v", err)
		}
	})

	t.Run("RenderGlobalAdvertisesExamplesFlag", func(t *testing.T) {
		var enabledOut, disabledOut bytes.Buffer
		appEnabled := makeApp(true, &enabledOut)
		appDisabled := makeApp(false, &disabledOut)

		appEnabled.RenderGlobal(Options{Writer: &enabledOut, Width: 80})
		enabledBody := StripANSI(enabledOut.String())
		if !strings.Contains(enabledBody, "-E, --examples") {
			t.Errorf("RenderGlobal() with EnableExamplesFlag=true did not advertise -E, --examples:\n%s", enabledBody)
		}

		appDisabled.RenderGlobal(Options{Writer: &disabledOut, Width: 80})
		disabledBody := StripANSI(disabledOut.String())
		if strings.Contains(disabledBody, "-E, --examples") {
			t.Errorf("RenderGlobal() with EnableExamplesFlag=false unexpectedly advertised -E, --examples:\n%s", disabledBody)
		}
	})
}
