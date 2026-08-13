// Package main demonstrates how to integrate the 'clihelp' package into a
// production Go command-line application with multi-level subcommands.
package main

import (
	"fmt"
	"os"

	"github.com/sarielhp/clihelp"
)

func main() {
	// Step 1: Define your application structure using clihelp.App.
	// An App contains the program name, description, global note, and top-level commands.
	app := &clihelp.App{
		Name:        "podctl",
		Description: "A podcast distribution & audio processing tool",
		GlobalNote:  "Run 'podctl <command> --help' or 'podctl help <command>' for command-specific options.",
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile & package audio episodes with metadata",
				UsageLine:   "podctl build [options] <source-file>",
				Options: []clihelp.Option{
					{Flags: "-o, --output PATH", Description: "Write compiled MP3 output to specified PATH"},
					{Flags: "-b, --bitrate KBPS", Description: "Set target audio encoding bitrate in kbps (default: 192)"},
					{Flags: "--normalize", Description: "Apply LUFS loudness normalization filter across tracks"},
					{Flags: "--tags TAGS", Description: "Embed ID3 metadata tags (e.g. title, artist, album, year)"},
					{Flags: "-v, --verbose", Description: "Enable verbose ffmpeg build output logs"},
				},
				Examples: []clihelp.Example{
					{Line: "podctl build episode01.wav"},
					{Line: "podctl build -o ep01.mp3 --bitrate 320 --normalize"},
					{Line: "podctl build -o dist/ep01.mp3 --tags 'title=Ep1,artist=Podcast' episode01.wav"},
				},
			},
			{
				Name:        "serve",
				Description: "Start local development RSS feed server",
				UsageLine:   "podctl serve [options]",
				Options: []clihelp.Option{
					{Flags: "-p, --port N", Description: "Listen HTTP port number (default: 8080)"},
					{Flags: "-H, --host HOST", Description: "Bind IP host address (default: 127.0.0.1)"},
					{Flags: "--tls-cert PATH", Description: "Path to TLS public certificate file for HTTPS"},
					{Flags: "--tls-key PATH", Description: "Path to TLS private key file for HTTPS"},
					{Flags: "--live-reload", Description: "Automatically reload RSS feed on XML or audio updates"},
				},
				Examples: []clihelp.Example{
					{Line: "podctl serve"},
					{Line: "podctl serve --port 9090 --live-reload"},
					{Line: "podctl serve -H 0.0.0.0 -p 8443 --tls-cert cert.pem --tls-key key.pem"},
				},
			},
			// Nested Subcommands Example:
			// Commands can define their own Subcommands slice to form tree hierarchies
			// such as: podctl config -> set -> location, time, space.
			{
				Name:        "config",
				Description: "View and manage application configuration settings",
				UsageLine:   "podctl config <subcommand> [options]",
				Examples: []clihelp.Example{
					{Line: "podctl config set location 5"},
					{Line: "podctl config set time 120"},
					{Line: "podctl config set space 500"},
				},
				Subcommands: []clihelp.Command{
					{
						Name:        "set",
						Description: "Set configuration attribute values",
						UsageLine:   "podctl config set <attribute> <value> [options]",
						Options: []clihelp.Option{
							{Flags: "--global", Description: "Apply setting across all system profiles"},
							{Flags: "--persist", Description: "Save attribute setting permanently to config.json"},
						},
						Examples: []clihelp.Example{
							{Line: "podctl config set location 5"},
							{Line: "podctl config set time 120"},
							{Line: "podctl config set space 500"},
						},
						Subcommands: []clihelp.Command{
							{
								Name:        "time",
								Description: "Set max execution timeout or timestamp window",
								UsageLine:   "podctl config set time <seconds> [options]",
								Options: []clihelp.Option{
									{Flags: "--unit SEC", Description: "Time unit format (s: seconds, m: minutes, h: hours)"},
									{Flags: "--persist", Description: "Save setting to configuration file"},
								},
								Examples: []clihelp.Example{
									{Line: "podctl config set time 120"},
									{Line: "podctl config set time 2 --unit h --persist"},
								},
							},
							{
								Name:        "space",
								Description: "Set maximum disk space allocation or cache budget in MB",
								UsageLine:   "podctl config set space <megabytes> [options]",
								Options: []clihelp.Option{
									{Flags: "--unit SIZE", Description: "Space allocation unit (MB | GB)"},
									{Flags: "--auto-cleanup", Description: "Purge oldest temporary cache files when limit reached"},
									{Flags: "--persist", Description: "Save setting to configuration file"},
								},
								Examples: []clihelp.Example{
									{Line: "podctl config set space 500"},
									{Line: "podctl config set space 2 --unit GB --auto-cleanup"},
								},
							},
							{
								Name:        "location",
								Description: "Set geographic storage region or default output zone ID",
								UsageLine:   "podctl config set location <id> [options]",
								Options: []clihelp.Option{
									{Flags: "--zone NAME", Description: "Specify datacenter or cloud availability zone"},
									{Flags: "--persist", Description: "Save setting to configuration file"},
								},
								Examples: []clihelp.Example{
									{Line: "podctl config set location 5"},
									{Line: "podctl config set location 12 --zone us-east-1 --persist"},
								},
							},
						},
					},
					{
						Name:        "get",
						Description: "Display current value for a configuration attribute",
						UsageLine:   "podctl config get <attribute>",
						Examples: []clihelp.Example{
							{Line: "podctl config get location"},
							{Line: "podctl config get time"},
						},
					},
				},
			},
			{
				Name:        "deploy",
				Description: "Publish RSS feed & audio files to cloud storage / CDN",
				UsageLine:   "podctl deploy [options] <stage>",
				Options: []clihelp.Option{
					{Flags: "-s, --stage STAGE", Description: "Target deployment environment (staging | production)"},
					{Flags: "--dry-run", Description: "Simulate publishing without uploading files"},
					{Flags: "--purge-cdn", Description: "Invalidate CDN cache for feed and updated audio files"},
					{Flags: "--timeout SEC", Description: "Maximum upload timeout in seconds (default: 300)"},
				},
				Examples: []clihelp.Example{
					{Line: "podctl deploy --dry-run staging"},
					{Line: "podctl deploy -s production --purge-cdn"},
				},
			},
			{
				Name:        "status",
				Description: "Check RSS feed health, CDN metrics, and download stats",
				UsageLine:   "podctl status [options]",
				Options: []clihelp.Option{
					{Flags: "-s, --stage STAGE", Description: "Environment to inspect (staging | production)"},
					{Flags: "--json", Description: "Output metrics and status in JSON format"},
				},
				Examples: []clihelp.Example{
					{Line: "podctl status"},
					{Line: "podctl status -s production --json"},
				},
			},
		},
	}

	rawArgs := os.Args[1:]

	// Step 2: Handle global help when no arguments are supplied.
	if len(rawArgs) == 0 {
		app.PrintGlobalUsage()
		return
	}

	// Step 3: Inspect command-line flags and help requests.
	var helpRequested bool
	var cleanArgs []string

	for _, arg := range rawArgs {
		if arg == "--help" || arg == "-h" {
			helpRequested = true
		} else {
			cleanArgs = append(cleanArgs, arg)
		}
	}

	if len(cleanArgs) > 0 && cleanArgs[0] == "help" {
		helpRequested = true
		cleanArgs = cleanArgs[1:]
	}

	// Step 4: Extract command path (e.g. ["config", "set", "location"]) and positional arguments.
	var cmdPath []string
	var posArgs []string

	for _, arg := range cleanArgs {
		if isFlag(arg) {
			continue
		}
		// Match against app's command tree
		if app.LookupCommand(append(cmdPath, arg)...) != nil {
			cmdPath = append(cmdPath, arg)
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	// Step 5: If help was explicitly requested or no command path was matched, render help text.
	if helpRequested || len(cmdPath) == 0 {
		if len(cmdPath) > 0 {
			// Print command help for nested path (e.g. app.PrintCommandUsage("config", "set", "location"))
			app.PrintCommandUsage(cmdPath...)
		} else {
			// Print top-level global help
			app.PrintGlobalUsage()
		}
		return
	}

	// Step 6: Dispatch valid command execution when arguments are complete.
	switch cmdPath[0] {
	case "build":
		if len(posArgs) == 0 {
			app.PrintCommandUsage("build")
			return
		}
		fmt.Printf("Compiling audio episode '%s'...\nSuccess.\n", posArgs[0])

	case "serve":
		fmt.Println("Starting local RSS development server on http://127.0.0.1:8080...")

	case "deploy":
		stage := "staging"
		if len(posArgs) > 0 {
			stage = posArgs[0]
		}
		fmt.Printf("Deploying podcast feed to environment '%s'...\nDeployment successful.\n", stage)

	case "status":
		fmt.Println("Health: OK | Feed: Active | Downloads: 12,450 | CDN Hit Ratio: 99.4%")

	case "config":
		if len(cmdPath) < 3 {
			// Incomplete subcommand (e.g. "podctl config" or "podctl config set") -> print help for that level
			app.PrintCommandUsage(cmdPath...)
			return
		}

		if cmdPath[1] == "set" {
			attr := cmdPath[2]
			if len(posArgs) == 0 {
				// Missing required value -> print help for attribute command
				app.PrintCommandUsage(cmdPath...)
				return
			}
			val := posArgs[0]
			fmt.Printf("Updated configuration: %s = %s\n", attr, val)
		} else if cmdPath[1] == "get" {
			attr := cmdPath[2]
			fmt.Printf("%s = 5 (default)\n", attr)
		} else {
			app.PrintCommandUsage(cmdPath...)
		}

	default:
		fmt.Printf("Unknown command: %s\n", cmdPath[0])
		app.PrintGlobalUsage()
	}
}

// isFlag returns true if string starts with '-' or '--'.
func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
