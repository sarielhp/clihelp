// Package main demonstrates how to integrate the 'clihelp' package into a
// production Go command-line application with declarative routing and flag parsing.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sarielhp/clihelp"
)

type GlobalOptions struct {
	Verbose bool
	Silent  bool
}

func main() {
	var globals GlobalOptions

	// Define command-specific option targets
	var (
		buildOutput    string
		buildBitrate   int
		buildNormalize bool
		buildTags      string

		servePort       int
		serveHost       string
		serveLiveReload bool

		deployStage    string
		deployDryRun   bool
		deployPurgeCDN bool
		deployTimeout  time.Duration

		statusStage string
		statusJSON  bool

		configSpaceUnit string
		configSpaceAuto bool
	)

	app := &clihelp.App{
		Name:        "podctl",
		Description: "[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.",
		Version:     "0.2.9",
		GlobalNote:  "Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)\nRun 'podctl <command> --help' for command-specific options.",
		PersistentOptions: []clihelp.Option{
			clihelp.Bool(&globals.Verbose, "-v, --verbose", false, "Enable verbose output logs"),
			clihelp.Bool(&globals.Silent, "-s, --silent", false, "Suppress non-error output"),
		},
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile & package audio episodes with **metadata** and *ID3 tags*",
				UsageLine:   "podctl build [options] <source-file>",
				Args:        clihelp.ExactArgs(1),
				Options: []clihelp.Option{
					clihelp.String(&buildOutput, "-o, --output PATH", "", "Write compiled MP3 output to specified PATH"),
					clihelp.Int(&buildBitrate, "-b, --bitrate KBPS", 192, "Set target audio encoding bitrate in kbps"),
					clihelp.BoolToggle(&buildNormalize, "--[no-]normalize", true, "Apply LUFS loudness normalization"),
					clihelp.String(&buildTags, "--tags TAGS", "", "Embed ID3 metadata tags (e.g. title, artist)"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl build episode01.wav"},
					{Line: "podctl build -o ep01.mp3 --bitrate 320 --normalize"},
				},
				Notes: []clihelp.Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use `--bitrate 320` for *highest quality* or `--bitrate 128` for **voice-only** episodes (see [Audio Encoding Guide](https://podctl.example.com/docs/audio)).",
					},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("Compiling audio episode '%s' (bitrate: %d kbps, output: %q)...\nSuccess.\n",
						ctx.Args[0], buildBitrate, buildOutput)
					return nil
				},
			},
			{
				Name:        "serve",
				Description: "Start local development RSS feed server",
				UsageLine:   "podctl serve [options]",
				Options: []clihelp.Option{
					clihelp.Int(&servePort, "-p, --port N", 8080, "Listen HTTP port number"),
					clihelp.String(&serveHost, "-H, --host HOST", "127.0.0.1", "Bind IP host address"),
					clihelp.BoolToggle(&serveLiveReload, "--[no-]live-reload", true, "Automatically reload RSS feed"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl serve"},
					{Line: "podctl serve --port 9090 --no-live-reload"},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("Starting local RSS development server on http://%s:%d (live reload: %v)...\n",
						serveHost, servePort, serveLiveReload)
					return nil
				},
			},
			{
				Name:        "config",
				Description: "View and manage application configuration settings",
				UsageLine:   "podctl config <subcommand> [options]",
				Subcommands: []clihelp.Command{
					{
						Name:        "set",
						Description: "Set configuration attribute values",
						UsageLine:   "podctl config set <attribute> <value> [options]",
						Subcommands: []clihelp.Command{
							{
								Name:        "space",
								Description: "Set maximum disk space allocation in MB",
								UsageLine:   "podctl config set space <megabytes> [options]",
								Args:        clihelp.ExactArgs(1),
								Options: []clihelp.Option{
									clihelp.Enum(&configSpaceUnit, "--unit SIZE", []string{"MB", "GB"}, "MB", "Space allocation unit"),
									clihelp.Bool(&configSpaceAuto, "--auto-cleanup", false, "Purge oldest temporary cache files"),
								},
								Run: func(ctx *clihelp.Context) error {
									fmt.Printf("Config space set to %s %s (auto-cleanup: %v)\n", ctx.Args[0], configSpaceUnit, configSpaceAuto)
									return nil
								},
							},
						},
					},
					{
						Name:        "get",
						Description: "Display current value for a configuration attribute",
						UsageLine:   "podctl config get <attribute>",
						Args:        clihelp.ExactArgs(1),
						Run: func(ctx *clihelp.Context) error {
							fmt.Printf("%s = 5 (default)\n", ctx.Args[0])
							return nil
						},
					},
				},
			},
			{
				Name:        "deploy",
				Description: "Publish RSS feed & audio to cloud storage (e.g. `S3`, `GCS`) and CDN",
				UsageLine:   "podctl deploy [options]",
				Options: []clihelp.Option{
					clihelp.Enum(&deployStage, "-S, --stage STAGE", []string{"staging", "production"}, "staging", "Target deployment environment"),
					clihelp.Bool(&deployDryRun, "--dry-run", false, "Simulate publishing without uploading files"),
					clihelp.Bool(&deployPurgeCDN, "--purge-cdn", false, "Invalidate CDN cache for feed and updated audio files"),
					clihelp.Duration(&deployTimeout, "--timeout SEC", 300*time.Second, "Maximum upload timeout"),
				},
				Notes: []clihelp.Note{
					{
						Heading: "Safety Precaution",
						Text:    "Always test with `--dry-run` before ~~overwriting~~ publishing to **production** (see [Deploy Docs](https://podctl.example.com/docs/deploy)).",
					},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Printf("Deploying podcast feed to environment '%s' (dry-run: %v, timeout: %v)...\nSuccess.\n",
						deployStage, deployDryRun, deployTimeout)
					return nil
				},
			},
			{
				Name:        "status",
				Description: "Check RSS feed health, CDN metrics, and download stats",
				UsageLine:   "podctl status [options]",
				Options: []clihelp.Option{
					clihelp.Enum(&statusStage, "-S, --stage STAGE", []string{"staging", "production"}, "production", "Environment to inspect"),
					clihelp.Bool(&statusJSON, "--json", false, "Output metrics and status in JSON format"),
				},
				Run: func(ctx *clihelp.Context) error {
					if statusJSON {
						fmt.Println(`{"health": "OK", "feed": "active", "downloads": 12450}`)
					} else {
						fmt.Printf("Stage: %s | Health: OK | Feed: Active | Downloads: 12,450\n", statusStage)
					}
					return nil
				},
			},
			{
				Name:        "completion",
				Description: "Generate shell autocompletion script",
				UsageLine:   "podctl completion <bash|zsh|fish>",
				Args:        clihelp.ExactArgs(1),
				Run: func(ctx *clihelp.Context) error {
					switch ctx.Args[0] {
					case "bash":
						return clihelp.GenBashCompletion(ctx.App, ctx.Stdout)
					case "zsh":
						return clihelp.GenZshCompletion(ctx.App, ctx.Stdout)
					case "fish":
						return clihelp.GenFishCompletion(ctx.App, ctx.Stdout)
					default:
						return fmt.Errorf("unsupported shell: %s (choose bash, zsh, or fish)", ctx.Args[0])
					}
				},
			},
		},
	}

	// Developer bootstrap: CLIHELP_GEN=1 generates documentation markdown
	if os.Getenv("CLIHELP_GEN") != "" {
		changed, gerr := clihelp.RenderMarkdown(app, clihelp.MarkdownOptions{Dir: "docs/clihelp"})
		if gerr != nil {
			fmt.Fprintf(os.Stderr, "generate docs: %v\n", gerr)
			os.Exit(1)
		}
		if changed {
			fmt.Fprintln(os.Stderr, "processed under docs/clihelp/")
		} else {
			fmt.Fprintln(os.Stderr, "docs up to date.")
		}
		return
	}

	// Execute application lifecycle
	if err := app.Execute(os.Args[1:]); err != nil {
		clihelp.PrintError(err)
		os.Exit(1)
	}
}
