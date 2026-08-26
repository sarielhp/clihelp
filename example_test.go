package clihelp_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sarielhp/clihelp"
)

func ExampleApp_Execute() {
	var verbose bool
	var output string

	app := &clihelp.App{
		Name:        "demo",
		Description: "Demonstration CLI tool",
		Version:     "1.0.0",
		Pager:       true,
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile the binary target",
				UsageLine:   "demo build [options] <target>",
				Args:        clihelp.ExactArgs(1),
				Options: []clihelp.Option{
					clihelp.String(&output, "-o, --output PATH", "dist/app", "Output binary path"),
					clihelp.Bool(&verbose, "-v, --verbose", false, "Enable verbose logging"),
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("building %s -> %s (verbose: %v)\n", ctx.Args[0], output, verbose)
					return nil
				},
			},
		},
	}

	// Run with build command arguments
	if err := app.Execute([]string{"build", "-o", "bin/demo", "-v", "main.go"}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	// Output:
	// building main.go -> bin/demo (verbose: true)
}

func ExampleBoolToggle() {
	var normalize bool

	app := &clihelp.App{
		Name:  "soundctl",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "process",
				Options: []clihelp.Option{
					clihelp.BoolToggle(&normalize, "--[no-]normalize", true, "Apply audio normalization"),
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("normalize: %v\n", normalize)
					return nil
				},
			},
		},
	}

	_ = app.Execute([]string{"process", "--no-normalize"})

	// Output:
	// normalize: false
}

func ExampleExactArgs() {
	app := &clihelp.App{
		Name:  "tagger",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "tag",
				Args: clihelp.ExactArgs(2),
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("tagging %s with %s\n", ctx.Args[0], ctx.Args[1])
					return nil
				},
			},
		},
	}

	_ = app.Execute([]string{"tag", "file.txt", "v1.0"})

	// Output:
	// tagging file.txt with v1.0
}

func ExampleApp_Render() {
	var buf strings.Builder

	app := &clihelp.App{
		Name:        "webcli",
		Description: "[webcli](https://example.com) — Modern web utility tool with `fast` execution.",
		Pager:       true,
		Commands: []clihelp.Command{
			{
				Name:        "ping",
				Description: "Ping remote server",
			},
		},
	}

	// Render global help to buffer
	app.RenderGlobal(clihelp.Options{Writer: &buf, Width: 80})

	// Output is formatted with colors and OSC 8 hyperlinks in supported terminals
	fmt.Println(strings.Contains(buf.String(), "webcli"))

	// Output:
	// true
}

func ExampleEnum() {
	var env string

	app := &clihelp.App{
		Name:  "deployer",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "deploy",
				Options: []clihelp.Option{
					clihelp.Enum(&env, "-e, --env ENV", []string{"dev", "staging", "prod"}, "dev", "Target deployment environment"),
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("deploying to: %s\n", env)
					return nil
				},
			},
		},
	}

	_ = app.Execute([]string{"deploy", "--env", "staging"})

	// Output:
	// deploying to: staging
}

func ExampleStringSlice() {
	var tags []string

	app := &clihelp.App{
		Name:  "builder",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "build",
				Options: []clihelp.Option{
					clihelp.StringSlice(&tags, "-t, --tag TAG", []string{"latest"}, "Image tag (repeatable)"),
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("tags: %v\n", tags)
					return nil
				},
			},
		},
	}

	_ = app.Execute([]string{"build", "-t", "v1.0", "-t", "release"})

	// Output:
	// tags: [v1.0 release]
}

func ExampleDuration() {
	var timeout time.Duration

	app := &clihelp.App{
		Name:  "fetcher",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "fetch",
				Options: []clihelp.Option{
					clihelp.Duration(&timeout, "-t, --timeout DURATION", 10*time.Second, "Request timeout"),
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("timeout: %v\n", timeout)
					return nil
				},
			},
		},
	}

	_ = app.Execute([]string{"fetch", "--timeout", "45s"})

	// Output:
	// timeout: 45s
}

func ExampleApp_ExecuteContext() {
	app := &clihelp.App{
		Name:  "runner",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name: "run",
				Run: func(ctx *clihelp.Context) error {
					select {
					case <-ctx.Context.Done():
						return ctx.Context.Err()
					default:
						fmt.Println("job completed successfully")
						return nil
					}
				},
			},
		},
	}

	ctx := context.Background()
	_ = app.ExecuteContext(ctx, []string{"run"})

	// Output:
	// job completed successfully
}

func ExampleApp_RenderTree() {
	var buf strings.Builder

	app := &clihelp.App{
		Name:  "gitcli",
		Pager: true,
		Commands: []clihelp.Command{
			{
				Name:        "remote",
				Description: "Manage set of tracked repositories",
				Subcommands: []clihelp.Command{
					{Name: "add", Description: "Add a remote"},
					{Name: "remove", Description: "Remove a remote"},
				},
			},
			{
				Name:        "status",
				Description: "Show working tree status",
			},
		},
	}

	app.RenderTree(clihelp.Options{Writer: &buf, Width: 80})

	fmt.Println(strings.Contains(buf.String(), "remote"))
	fmt.Println(strings.Contains(buf.String(), "add"))
	fmt.Println(strings.Contains(buf.String(), "status"))

	// Output:
	// true
	// true
	// true
}
