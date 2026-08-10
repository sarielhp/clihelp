package main

import (
	"github.com/sarielhp/clihelp"
)

func main() {
	app := &clihelp.App{
		Name:        "myapp",
		Description: "A sample CLI application",
		GlobalNote:  "Run 'myapp <command> --help' for command-specific options.",
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile the project",
				UsageLine:   "myapp build [options] <target>",
				Options: []clihelp.Option{
					{Flags: "-o, --output PATH", Description: "Write output to PATH"},
					{Flags: "--verbose", Description: "Enable verbose logging"},
					{Flags: "--tags TAGS", Description: "Build tags to enable"},
				},
				Examples: []clihelp.Example{
					{Line: "myapp build"},
					{Line: "myapp build -o myapp.bin"},
					{Line: "myapp build --verbose --tags netgo"},
				},
			},
			{
				Name:        "run",
				Description: "Run the project",
				UsageLine:   "myapp run [options]",
				Options: []clihelp.Option{
					{Flags: "--port N", Description: "Listen port (default 8080)"},
					{Flags: "--verbose", Description: "Enable verbose logging"},
				},
				Examples: []clihelp.Example{
					{Line: "myapp run"},
					{Line: "myapp run --port 9090"},
				},
			},
		},
	}

	clihelp.PrintGlobalUsage(app)
}
