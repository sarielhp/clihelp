// Package main demonstrates how to integrate the 'clihelp' package into a
// production Go command-line application with declarative routing and flag parsing.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sarielhp/clihelp"
	"github.com/sarielhp/clihelp/doc"
)

type GlobalOptions struct {
	Verbose  bool
	Silent   bool
	NoColor  bool
	Config   string
	Endpoint string
	Token    string
	APIKey   string
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

		deployBucket   string
		deployStage    string
		deployDryRun   bool
		deployPurgeCDN bool
		deployTimeout  time.Duration

		statusStage string
		statusJSON  bool

		configSpaceUnit string
		configSpaceAuto bool
	)

	deepCmd := buildDeepTree()

	app := &clihelp.App{
		Name:                      "podctl",
		Description:               "[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.",
		Version:                   "0.3.3",
		GlobalNote:                "Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)",
		AbbrevCommands:            true,
		Pager:                     true,
		OmitGlobalFlagsInCommands: true,
		InteractiveFallback:       true,
		AutoInstallCompletion:     true,
		PersistentOptions: []clihelp.Option{
			clihelp.Group("Authentication", clihelp.String(&globals.Token, "--token TOKEN", "", "Bearer token for cluster authentication")),
			clihelp.Group("Authentication", clihelp.String(&globals.APIKey, "--api-key KEY", "", "API key for cloud provider access")),

			clihelp.Group("Connection & Environment", clihelp.String(&globals.Config, "-c, --config PATH", "~/.config/podctl.yaml", "Path to configuration file")),
			clihelp.Group("Connection & Environment", clihelp.String(&globals.Endpoint, "--endpoint URL", "https://api.podctl.example.com", "API service endpoint URL")),

			clihelp.Group("Output & Logging", clihelp.Bool(&globals.Verbose, "-v, --verbose", false, "Enable verbose output logs")),
			clihelp.Group("Output & Logging", clihelp.Bool(&globals.Silent, "-s, --silent", false, "Suppress non-error output")),
			clihelp.Group("Output & Logging", clihelp.Bool(&globals.NoColor, "--no-color", false, "Disable ANSI color output")),
		},
		Examples: []clihelp.Example{
			{Line: "podctl build episode01.wav", Description: "Compile raw audio into a release-ready podcast episode"},
			{Line: "podctl serve --port 8080", Description: "Start the local development server for previewing feeds"},
		},
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile, encode, and package raw audio into MP3 podcast episodes. Supports configurable bitrate, loudness normalization, and embedded ID3 tags for distribution across Apple Podcasts, Spotify, and Google Podcasts.",
				UsageLine:   "podctl build [options] <source-file>",
				Args:        clihelp.ExactArgs(1),
				Options: []clihelp.Option{
					clihelp.String(&buildOutput, "-o, --output PATH", "", "Write compiled MP3 output to specified PATH"),
					clihelp.Int(&buildBitrate, "-b, --bitrate KBPS", 192, "Set target audio encoding bitrate in kbps"),
					clihelp.BoolToggle(&buildNormalize, "--[no-]normalize", true, "Apply LUFS loudness normalization"),
					clihelp.String(&buildTags, "--tags TAGS", "", "Embed ID3 metadata tags (e.g. title, artist)"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl build episode01.wav", Description: "Compile a single episode from raw WAV audio."},
					{Line: "podctl build episode01.wav -o ep01.mp3 --bitrate 320 --normalize", Description: "Compile with 320 kbps bitrate and LUFS loudness normalization."},
				},
				Notes: []clihelp.Note{
					{
						Heading: "Encoding Guidelines",
						Text:    "Use `--bitrate 320` for *highest quality* or `--bitrate 128` for **voice-only** episodes (see [Audio Encoding Guide](https://podctl.example.com/docs/audio)).",
					},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Fprintf(ctx.Stdout, "Compiling audio episode '%s' (bitrate: %d kbps, output: %q)...\nSuccess.\n",
						ctx.Args[0], buildBitrate, buildOutput)
					return nil
				},
			},
			{
				Name:        "serve",
				Description: "Start a local HTTP development server for RSS feeds and audio files. Includes live-reload support, CORS headers for cross-origin testing, and a built-in web dashboard for previewing feed metadata before deploying to production.",
				UsageLine:   "podctl serve [options]",
				Options: []clihelp.Option{
					clihelp.Int(&servePort, "-p, --port N", 8080, "Listen HTTP port number"),
					clihelp.String(&serveHost, "-H, --host HOST", "127.0.0.1", "Bind IP host address"),
					clihelp.BoolToggle(&serveLiveReload, "--[no-]live-reload", true, "Automatically reload RSS feed"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl serve", Description: "Start the local preview server on default port 8080."},
					{Line: "podctl serve --port 9090 --no-live-reload", Description: "Bind custom port and disable automatic live reload."},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Fprintf(ctx.Stdout, "Starting local RSS development server on http://%s:%d (live reload: %v)...\n",
						serveHost, servePort, serveLiveReload)
					return nil
				},
			},
			{
				Name:        "config",
				Description: "View, inspect, set, and manage application configuration settings. Controls storage locations, disk space limits, CDN bucket names, API keys, and publishing preferences via dedicated subcommands.",
				UsageLine:   "podctl config <subcommand> [options]",
				Subcommands: []clihelp.Command{
					{
						Name:        "set",
						Description: "Assign, update, or override configuration attribute values. Supports nested key paths and bulk operations for efficient setup across development, staging, and production targets.",
						UsageLine:   "podctl config set <attribute> <value> [options]",
						Subcommands: []clihelp.Command{
							{
								Name:        "space",
								Description: "Set maximum disk space allocation for temporary cache and build artifacts. Configurable in megabytes or gigabytes with an optional automatic cleanup policy.",
								UsageLine:   "podctl config set space <megabytes> [options]",
								Args:        clihelp.ExactArgs(1),
								Options: []clihelp.Option{
									clihelp.Enum(&configSpaceUnit, "--unit SIZE", []string{"MB", "GB"}, "MB", "Space allocation unit"),
									clihelp.Bool(&configSpaceAuto, "--auto-cleanup", false, "Purge oldest temporary cache files"),
								},
								Examples: []clihelp.Example{
									{Line: "podctl config set space 500 --unit MB --auto-cleanup", Description: "Set cache allocation limit to 500 MB with auto-cleanup."},
								},
								Run: func(ctx *clihelp.Context) error {
									fmt.Fprintf(ctx.Stdout, "Config space set to %s %s (auto-cleanup: %v)\n", ctx.Args[0], configSpaceUnit, configSpaceAuto)
									return nil
								},
							},
						},
					},
					{
						Name:        "get",
						Description: "Display, inspect, and print configured attribute values. Reads from the persistent store or falls back to built-in defaults when no explicit user configuration value has been set.",
						UsageLine:   "podctl config get <attribute>",
						Args:        clihelp.ExactArgs(1),
						Examples: []clihelp.Example{
							{Line: "podctl config get space", Description: "Inspect current storage space limit."},
						},
						Run: func(ctx *clihelp.Context) error {
							fmt.Fprintf(ctx.Stdout, "%s = 5 (default)\n", ctx.Args[0])
							return nil
						},
					},
				},
			},
			{
				Name:        "deploy",
				Description: "Publish compiled podcast RSS feeds and MP3 files to cloud storage. Supports Amazon S3, Google Cloud Storage, CDN cache invalidation, dry-run simulation, and multi-stage deployments.",
				UsageLine:   "podctl deploy [options]",
				Options: []clihelp.Option{
					clihelp.Required(clihelp.String(&deployBucket, "-b, --bucket NAME", "", "Target cloud storage bucket name")),
					clihelp.Enum(&deployStage, "-S, --stage STAGE", []string{"staging", "production"}, "staging", "Target deployment environment"),
					clihelp.Bool(&deployDryRun, "--dry-run", false, "Simulate publishing without uploading files"),
					clihelp.Bool(&deployPurgeCDN, "--purge-cdn", false, "Invalidate CDN cache for feed and updated audio files"),
					clihelp.Duration(&deployTimeout, "--timeout SEC", 300*time.Second, "Maximum upload timeout"),
				},
				OptionsValidator: clihelp.ValidateOptions(
					clihelp.MutuallyExclusive("--dry-run", "--purge-cdn"),
				),
				Examples: []clihelp.Example{
					{Line: "podctl deploy --bucket my-podcast-s3 --stage production --dry-run", Description: "Simulate publishing without uploading files."},
				},
				Notes: []clihelp.Note{
					{
						Heading: "Safety Precaution",
						Text:    "Always test with `--dry-run` before ~~overwriting~~ publishing to **production** (see [Deploy Docs](https://podctl.example.com/docs/deploy)).",
					},
				},
				Run: func(ctx *clihelp.Context) error {
					fmt.Fprintf(ctx.Stdout, "Deploying podcast feed to bucket '%s' environment '%s' (dry-run: %v, timeout: %v)...\nSuccess.\n",
						deployBucket, deployStage, deployDryRun, deployTimeout)
					return nil
				},
			},
			{
				Name:        "status",
				Description: "Check and display comprehensive health and validation metrics. Monitors RSS feed status, CDN edge cache, episode download statistics, and origin server connectivity across environments.",
				UsageLine:   "podctl status [options]",
				Options: []clihelp.Option{
					clihelp.Enum(&statusStage, "-S, --stage STAGE", []string{"staging", "production"}, "production", "Environment to inspect"),
					clihelp.Bool(&statusJSON, "--json", false, "Output metrics and status in JSON format"),
				},
				Examples: []clihelp.Example{
					{Line: "podctl status --stage production --json", Description: "Output production health and metrics in JSON format."},
				},
				Run: func(ctx *clihelp.Context) error {
					if statusJSON {
						fmt.Fprintln(ctx.Stdout, `{"health": "OK", "feed": "active", "downloads": 12450}`)
					} else {
						fmt.Fprintf(ctx.Stdout, "Stage: %s | Health: OK | Feed: Active | Downloads: 12,450\n", statusStage)
					}
					return nil
				},
			},
			clihelp.CompletionCommand(),
			deepCmd,
		},
	}

	// Developer bootstrap: CLIHELP_GEN=1 generates documentation markdown
	if os.Getenv("CLIHELP_GEN") != "" {
		changed, gerr := doc.RenderMarkdown(app, doc.MarkdownOptions{Dir: "docs/clihelp"})
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
		app.PrintError(err)
		os.Exit(1)
	}
}

// levelSuffixes maps depth to the two suffixes used for subcommand naming.
var levelSuffixes = [][]string{
	2: {"one", "two"},
	3: {"a", "b"},
	4: {"i", "ii"},
}

// buildDeepTree creates the "deep" command with a binary tree of subcommands
// up to depth 5, as specified in the project todo.
func buildDeepTree() clihelp.Command {
	return clihelp.Command{
		Name:        "deep",
		Description: "**deep** — This is the [deep command](https://example.com/deep) at the root of the demonstration hierarchy with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines for testing purposes.",
		UsageLine:   "podctl deep [options] <subcommand> — This is a **very long usage line** for the [deep command](https://example.com/deep) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.",
		Subcommands: []clihelp.Command{
			buildSubTree("alpha", []string{"deep", "alpha"}, 2),
			buildSubTree("beta", []string{"deep", "beta"}, 2),
		},
	}
}

// buildSubTree recursively builds a command node and its binary subcommand tree.
func buildSubTree(name string, path []string, depth int) clihelp.Command {
	cmd := clihelp.Command{
		Name:        name,
		Description: fmt.Sprintf("This is the [%s command](https://example.com/%s) at depth %d with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.", name, strings.Join(path, "/"), depth),
		UsageLine:   fmt.Sprintf("podctl %s [options] [arguments...] — This is a **very long usage line** for the [%s command](https://example.com/%s) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.", strings.Join(path, " "), name, strings.Join(path, "/")),
	}

	if depth < 5 {
		suffixes := levelSuffixes[depth]
		child1 := name + "_" + suffixes[0]
		child2 := name + "_" + suffixes[1]
		cmd.Subcommands = []clihelp.Command{
			buildSubTree(child1, append(path, child1), depth+1),
			buildSubTree(child2, append(path, child2), depth+1),
		}
	} else {
		cmd.Run = func(ctx *clihelp.Context) error {
			fmt.Fprintf(ctx.Stdout, "Executing leaf command: %s\n", strings.Join(path, " "))
			return nil
		}
	}

	return cmd
}
