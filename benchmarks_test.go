package clihelp

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func benchApp() *App {
	var verbose bool
	var format string
	var output string

	return &App{
		Name:        "benchcli",
		Description: "A benchmark CLI application",
		GlobalFlags: []Option{
			Bool(&verbose, "-v, --verbose", false, "Verbose logging"),
		},
		PersistentOptions: []Option{
			String(&format, "--format FMT", "text", "Output format (text, json, yaml)"),
		},
		Commands: []Command{
			{
				Name:        "build",
				Aliases:     []string{"b"},
				Description: "Compile and package artifacts",
				Options: []Option{
					String(&output, "-o, --output PATH", "dist", "Output directory"),
				},
				Parameters: []Param{
					{
						Name:        "<source>",
						Description: "Source directory or file",
						Complete: func(toComplete string) []string {
							candidates := []string{"src", "pkg", "cmd", "internal"}
							var matched []string
							for _, c := range candidates {
								if strings.HasPrefix(c, toComplete) {
									matched = append(matched, c)
								}
							}
							return matched
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
			{
				Name:        "deploy",
				Description: "Deploy artifacts to cluster",
				Subcommands: []Command{
					{
						Name:        "staging",
						Description: "Deploy to staging cluster",
						Run:         func(*Context) error { return nil },
					},
					{
						Name:        "production",
						Description: "Deploy to production cluster",
						Run:         func(*Context) error { return nil },
					},
				},
			},
			{
				Name:        "config",
				Description: "Configuration management",
				Subcommands: []Command{
					{
						Name:        "get",
						Description: "Read configuration value",
						Parameters: []Param{
							{
								Name:        "<key>",
								Description: "Configuration key",
								Complete: func(toComplete string) []string {
									keys := []string{"endpoint\tAPI endpoint", "token\tAuth token", "timeout\tTimeout in ms"}
									var matches []string
									for _, k := range keys {
										if strings.HasPrefix(k, toComplete) {
											matches = append(matches, k)
										}
									}
									return matches
								},
							},
						},
						Run: func(*Context) error { return nil },
					},
					{
						Name:        "set",
						Description: "Set configuration value",
						Parameters: []Param{
							{Name: "<key>", Description: "Configuration key"},
							{Name: "<val>", Description: "Configuration value"},
						},
						Run: func(*Context) error { return nil },
					},
				},
			},
		},
	}
}

func BenchmarkResolveCommand(b *testing.B) {
	app := benchApp()
	args := []string{"--format", "json", "deploy", "production", "--verbose"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := app.resolveCommand(args)
		if err != nil || res.cmd == nil {
			b.Fatalf("resolve failed: %v", err)
		}
	}
}

func BenchmarkCompleteSubcommand(b *testing.B) {
	app := benchApp()
	app.Stdout = io.Discard
	ctx := context.Background()
	args := []string{"__complete", "deploy", "st"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := app.ExecuteContext(ctx, args); err != nil {
			b.Fatalf("completion failed: %v", err)
		}
	}
}

func BenchmarkCompletePositional(b *testing.B) {
	app := benchApp()
	app.Stdout = io.Discard
	ctx := context.Background()
	args := []string{"__complete", "config", "get", "to"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := app.ExecuteContext(ctx, args); err != nil {
			b.Fatalf("completion failed: %v", err)
		}
	}
}

func BenchmarkSuggestCommand(b *testing.B) {
	app := benchApp()
	typo := "deplo"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := suggestCommand(typo, app.Commands)
		if s != "deploy" {
			b.Fatalf("expected deploy, got: %s", s)
		}
	}
}

func BenchmarkRenderGlobal(b *testing.B) {
	app := benchApp()
	var buf bytes.Buffer
	opts := Options{
		Writer:  &buf,
		Width:   80,
		NoColor: true,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		app.RenderGlobal(opts)
	}
}

func BenchmarkRenderCommand(b *testing.B) {
	app := benchApp()
	var buf bytes.Buffer
	opts := Options{
		Writer:  &buf,
		Width:   80,
		NoColor: true,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if !app.RenderCommand(opts, "config", "get") {
			b.Fatal("RenderCommand failed")
		}
	}
}
