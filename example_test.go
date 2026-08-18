package clihelp_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/sarielhp/clihelp"
)

func ExampleApp_Execute() {
	var verbose bool
	var output string

	app := &clihelp.App{
		Name:        "demo",
		Description: "Demonstration CLI tool",
		Version:     "1.0.0",
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
		Name: "soundctl",
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
		Name: "tagger",
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
