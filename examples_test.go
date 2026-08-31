package clihelp

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

func TestSplitExampleCommandLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "simple command",
			input:   "podctl build episode01.wav",
			want:    []string{"podctl", "build", "episode01.wav"},
			wantErr: false,
		},
		{
			name:    "prompt prefix stripped",
			input:   "$ podctl build ep01.wav --bitrate 320",
			want:    []string{"podctl", "build", "ep01.wav", "--bitrate", "320"},
			wantErr: false,
		},
		{
			name:    "greater than prompt stripped",
			input:   "> podctl serve --port 8080",
			want:    []string{"podctl", "serve", "--port", "8080"},
			wantErr: false,
		},
		{
			name:    "double quotes preserved",
			input:   `mail_cli rule add newsletter@example.com "Sort/Newsletters"`,
			want:    []string{"mail_cli", "rule", "add", "newsletter@example.com", "Sort/Newsletters"},
			wantErr: false,
		},
		{
			name:    "single quotes preserved",
			input:   `podctl build 'my episode.wav' -o 'out file.mp3'`,
			want:    []string{"podctl", "build", "my episode.wav", "-o", "out file.mp3"},
			wantErr: false,
		},
		{
			name:    "pipeline takes primary command",
			input:   "podctl export --format json | jq .",
			want:    []string{"podctl", "export", "--format", "json"},
			wantErr: false,
		},
		{
			name:    "chained command takes primary",
			input:   "podctl build ep.wav && podctl deploy --dry-run",
			want:    []string{"podctl", "build", "ep.wav"},
			wantErr: false,
		},
		{
			name:    "inline comment stripped",
			input:   "podctl build ep.wav # compile raw audio",
			want:    []string{"podctl", "build", "ep.wav"},
			wantErr: false,
		},
		{
			name:    "escaped spaces",
			input:   `podctl\ build my\ episode.wav`,
			want:    []string{"podctl build", "my episode.wav"},
			wantErr: false,
		},
		{
			name:    "unclosed double quote",
			input:   `podctl build "episode.wav`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unclosed single quote",
			input:   `podctl build 'episode.wav`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "trailing backslash",
			input:   `podctl build episode.wav \`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitExampleCommandLine(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SplitExampleCommandLine(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Fatalf("SplitExampleCommandLine(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("SplitExampleCommandLine(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestColorizeExampleLine(t *testing.T) {
	had := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = had }()

	th := defaultTheme()

	t.Run("full comment line", func(t *testing.T) {
		line := "# Compile episode with high bitrate"
		colored := ColorizeExampleLine(line, th)
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("expected ANSI colors in comment line, got %q", colored)
		}
		if stripANSI(colored) != line {
			t.Errorf("stripANSI(%q) = %q, want %q", colored, stripANSI(colored), line)
		}
	})

	t.Run("command line with flags and args", func(t *testing.T) {
		line := "podctl build episode01.wav -o ep01.mp3 --bitrate 320"
		colored := ColorizeExampleLine(line, th)
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("expected ANSI colors in command line, got %q", colored)
		}
		if stripANSI(colored) != line {
			t.Errorf("stripANSI(%q) = %q, want %q", colored, stripANSI(colored), line)
		}
	})

	t.Run("command line with app context", func(t *testing.T) {
		app := testExampleApp()
		line := "podctl build episode01.wav -o ep01.mp3 --bitrate 320"
		colored := ColorizeExampleLineWithApp(app, nil, line, th)
		if !strings.Contains(colored, "\x1b[") {
			t.Errorf("expected ANSI colors in command line, got %q", colored)
		}
		if stripANSI(colored) != line {
			t.Errorf("stripANSI(%q) = %q, want %q", colored, stripANSI(colored), line)
		}
	})

	t.Run("prompt prefix and inline comment", func(t *testing.T) {
		line := "$ podctl serve --port 8080 # start preview server"
		colored := ColorizeExampleLine(line, th)
		if stripANSI(colored) != line {
			t.Errorf("stripANSI(%q) = %q, want %q", colored, stripANSI(colored), line)
		}
	})

	t.Run("pipeline command", func(t *testing.T) {
		line := "podctl status --json | jq .health"
		colored := ColorizeExampleLine(line, th)
		if stripANSI(colored) != line {
			t.Errorf("stripANSI(%q) = %q, want %q", colored, stripANSI(colored), line)
		}
	})

	t.Run("no color mode", func(t *testing.T) {
		color.NoColor = true
		defer func() { color.NoColor = false }()

		line := "podctl build episode01.wav --bitrate 320"
		colored := ColorizeExampleLine(line, th)
		if strings.Contains(colored, "\x1b[") {
			t.Errorf("expected no ANSI codes when NoColor is true, got %q", colored)
		}
		if colored != line {
			t.Errorf("got %q, want %q", colored, line)
		}
	})
}

func testExampleApp() *App {
	var (
		bitrate  int
		output   string
		dryRun   bool
		purgeCDN bool
		timeout  time.Duration
	)

	return &App{
		Name:    "podctl",
		Version: "1.0.0",
		Examples: []Example{
			{Line: "podctl build ep01.wav", Description: "Compile an audio file"},
			{Line: "podctl serve --port 8080", Description: "Start the preview server"},
		},
		PersistentOptions: []Option{
			String(&output, "-c, --config PATH", "", "Path to config"),
		},
		Commands: []Command{
			{
				Name:        "build",
				Description: "Compile podcast episodes",
				UsageLine:   "podctl build [options] <file>",
				Args:        ExactArgs(1),
				Options: []Option{
					Int(&bitrate, "-b, --bitrate KBPS", 192, "Audio bitrate in kbps"),
					String(&output, "-o, --output PATH", "", "Output path"),
				},
				Examples: []Example{
					{Line: "podctl build episode01.wav", Description: "Basic build"},
					{Line: "podctl build episode01.wav -o ep01.mp3 --bitrate 320", Description: "High quality"},
				},
			},
			{
				Name:        "serve",
				Description: "Start preview server",
				Options: []Option{
					Int(&bitrate, "-p, --port PORT", 8080, "Port number"),
				},
				Examples: []Example{
					{Line: "podctl serve", Description: "Default port"},
					{Line: "podctl serve --port 9090", Description: "Custom port"},
				},
			},
			{
				Name:        "deploy",
				Description: "Deploy to cloud storage",
				Options: []Option{
					Bool(&dryRun, "--dry-run", false, "Simulate deploy"),
					Bool(&purgeCDN, "--purge-cdn", false, "Purge CDN"),
					Duration(&timeout, "--timeout SEC", 60*time.Second, "Timeout"),
				},
				OptionsValidator: MutuallyExclusive("--dry-run", "--purge-cdn"),
				Examples: []Example{
					{Line: "podctl deploy --dry-run", Description: "Simulate deploy"},
					{Line: "podctl deploy --timeout 30s", Description: "With timeout"},
				},
			},
		},
	}
}

func TestValidateExample(t *testing.T) {
	app := testExampleApp()
	buildCmd := &app.Commands[0]
	deployCmd := &app.Commands[2]

	tests := []struct {
		name    string
		ex      Example
		cmd     *Command
		wantErr bool
		errSub  string
	}{
		{
			name:    "valid build example with app name",
			ex:      Example{Line: "podctl build episode01.wav -o ep.mp3 --bitrate 320"},
			cmd:     buildCmd,
			wantErr: false,
		},
		{
			name:    "valid build example without app name",
			ex:      Example{Line: "build episode01.wav -o ep.mp3"},
			cmd:     buildCmd,
			wantErr: false,
		},
		{
			name:    "valid build example with shell prompt and comment",
			ex:      Example{Line: "$ podctl build episode01.wav # compile raw audio"},
			cmd:     buildCmd,
			wantErr: false,
		},
		{
			name:    "invalid build missing required positional argument",
			ex:      Example{Line: "podctl build -o ep.mp3 --bitrate 320"},
			cmd:     buildCmd,
			wantErr: true,
			errSub:  "accepts 1 arg(s)",
		},
		{
			name:    "invalid build with too many positional arguments",
			ex:      Example{Line: "podctl build ep1.wav ep2.wav"},
			cmd:     buildCmd,
			wantErr: true,
			errSub:  "accepts 1 arg(s)",
		},
		{
			name:    "invalid flag name in example",
			ex:      Example{Line: "podctl build episode01.wav --unknown-flag"},
			cmd:     buildCmd,
			wantErr: true,
			errSub:  "unknown flag",
		},
		{
			name:    "invalid flag value type in example",
			ex:      Example{Line: "podctl build episode01.wav --bitrate not_an_int"},
			cmd:     buildCmd,
			wantErr: true,
			errSub:  "invalid",
		},
		{
			name:    "mutually exclusive flags violation in deploy example",
			ex:      Example{Line: "podctl deploy --dry-run --purge-cdn"},
			cmd:     deployCmd,
			wantErr: true,
			errSub:  "mutually exclusive",
		},
		{
			name:    "unknown subcommand in example",
			ex:      Example{Line: "podctl nonexistent_command"},
			cmd:     nil,
			wantErr: true,
			errSub:  "invalid command",
		},
		{
			name:    "unclosed quote in example",
			ex:      Example{Line: `podctl build "ep01.wav`},
			cmd:     buildCmd,
			wantErr: true,
			errSub:  "unclosed quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExample(app, tt.ex, tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateExample() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errSub != "" && !strings.Contains(err.Error(), tt.errSub) {
				t.Errorf("ValidateExample() error = %q, want substring %q", err.Error(), tt.errSub)
			}
		})
	}
}

func TestAppValidateExamples(t *testing.T) {
	t.Run("valid app passes all example validations", func(t *testing.T) {
		app := testExampleApp()
		errs := app.ValidateExamples()
		if len(errs) != 0 {
			t.Fatalf("expected 0 validation errors, got %d: %v", len(errs), errs)
		}
		if err := app.ValidateAllExamples(); err != nil {
			t.Fatalf("ValidateAllExamples() error = %v", err)
		}
	})

	t.Run("invalid example in app-level examples caught", func(t *testing.T) {
		app := testExampleApp()
		app.Examples = append(app.Examples, Example{Line: "podctl invalidcmd"})
		errs := app.ValidateExamples()
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "invalid command") {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("invalid example in subcommand caught", func(t *testing.T) {
		app := testExampleApp()
		app.Commands[0].Examples = append(app.Commands[0].Examples, Example{Line: "podctl build --missing-arg"})
		err := app.ValidateAllExamples()
		if err == nil {
			t.Fatal("expected ValidateAllExamples() error, got nil")
		}
		if !strings.Contains(err.Error(), "build") {
			t.Errorf("expected error to mention command 'build', got %v", err)
		}
	})
}

func TestAuditWithExampleValidation(t *testing.T) {
	app := testExampleApp()

	// Good app should pass Audit
	if err := Audit(app); err != nil {
		t.Fatalf("Audit(goodApp) error = %v", err)
	}

	// Bad app with invalid example flag should fail Audit
	badApp := testExampleApp()
	badApp.Commands[0].Examples = []Example{{Line: "podctl build ep.wav --badflag"}}
	err := Audit(badApp)
	if err == nil || !strings.Contains(err.Error(), "badflag") {
		t.Fatalf("expected Audit to fail on invalid example flag, got: %v", err)
	}

	// SkipExampleValidation should allow bad app to pass
	err = AuditWithOptions(badApp, AuditOptions{SkipExampleValidation: true})
	if err != nil {
		t.Fatalf("expected SkipExampleValidation to pass audit, got: %v", err)
	}
}

func TestRenderGlobalWithExamples(t *testing.T) {
	app := testExampleApp()
	var buf bytes.Buffer
	app.RenderGlobal(Options{Writer: &buf, Width: 80})
	out := stripANSI(buf.String())

	if !strings.Contains(out, "Examples:") {
		t.Errorf("expected 'Examples:' section in RenderGlobal output:\n%s", out)
	}
	if !strings.Contains(out, "podctl build ep01.wav") {
		t.Errorf("expected example line in RenderGlobal output:\n%s", out)
	}
	if !strings.Contains(out, "Compile an audio file") {
		t.Errorf("expected example description in RenderGlobal output:\n%s", out)
	}
}

func TestRenderCommandWithRichExamples(t *testing.T) {
	app := testExampleApp()
	var buf bytes.Buffer
	app.RenderCommand(Options{Writer: &buf, Width: 80}, "build")
	out := stripANSI(buf.String())

	if !strings.Contains(out, "Examples:") {
		t.Errorf("expected 'Examples:' section in RenderCommand output:\n%s", out)
	}
	if !strings.Contains(out, "podctl build episode01.wav -o ep01.mp3 --bitrate 320") {
		t.Errorf("expected example line in RenderCommand output:\n%s", out)
	}
	if !strings.Contains(out, "High quality") {
		t.Errorf("expected example description in RenderCommand output:\n%s", out)
	}
}
