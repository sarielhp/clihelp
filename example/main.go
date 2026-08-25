// Package main demonstrates how to integrate the 'clihelp' package into a
// production Go command-line application with declarative routing and flag parsing.
package main

import (
	"fmt"
	"os"
	"strings"
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

	deepCmd := buildDeepTree()

	app := &clihelp.App{
		Name:        "podctl",
		Description: "[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.",
		Version:     "0.2.17",
		GlobalNote:  "Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)\nRun 'podctl <command> --help' for command-specific options.",
		PersistentOptions: []clihelp.Option{
			clihelp.Bool(&globals.Verbose, "-v, --verbose", false, "Enable verbose output logs"),
			clihelp.Bool(&globals.Silent, "-s, --silent", false, "Suppress non-error output"),
		},
		Commands: []clihelp.Command{
			{
				Name:        "build",
				Description: "Compile, encode, and package raw audio source files into fully tagged MP3 podcast episodes with configurable bitrate, loudness normalization, and embedded ID3 metadata tags for distribution across multiple platforms and aggregators like Apple Podcasts, Spotify, and Google Podcasts.",
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
					fmt.Fprintf(ctx.Stdout, "Compiling audio episode '%s' (bitrate: %d kbps, output: %q)...\nSuccess.\n",
						ctx.Args[0], buildBitrate, buildOutput)
					return nil
				},
			},
			{
				Name:        "serve",
				Description: "Start a local development HTTP server that serves your podcast RSS feed and episode audio files with live-reload support, CORS headers for cross-origin testing, and a built-in web dashboard for previewing feed metadata before deploying to your production CDN and cloud storage endpoints.",
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
					fmt.Fprintf(ctx.Stdout, "Starting local RSS development server on http://%s:%d (live reload: %v)...\n",
						serveHost, servePort, serveLiveReload)
					return nil
				},
			},
			{
				Name:        "config",
				Description: "View, inspect, set, modify, and manage your application configuration settings including storage locations, disk space allocation limits, CDN bucket names, API keys, RSS feed metadata defaults, and podcast episode publishing preferences via a hierarchy of dedicated subcommands.",
				UsageLine:   "podctl config <subcommand> [options]",
				Subcommands: []clihelp.Command{
					{
						Name:        "set",
						Description: "Assign, update, or override one or more configuration attribute values in the persistent application configuration store, supporting nested key paths and bulk operations for efficient environment setup across development, staging, and production deployment targets.",
						UsageLine:   "podctl config set <attribute> <value> [options]",
						Subcommands: []clihelp.Command{
							{
								Name:        "space",
								Description: "Set the maximum disk space allocation for temporary cache and episode build artifacts in megabytes or gigabytes, with an optional automatic cleanup policy that purges the oldest temporary cache files when the configured limit is exceeded.",
								UsageLine:   "podctl config set space <megabytes> [options]",
								Args:        clihelp.ExactArgs(1),
								Options: []clihelp.Option{
									clihelp.Enum(&configSpaceUnit, "--unit SIZE", []string{"MB", "GB"}, "MB", "Space allocation unit"),
									clihelp.Bool(&configSpaceAuto, "--auto-cleanup", false, "Purge oldest temporary cache files"),
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
						Description: "Display, inspect, and print the current configured value for one or more application configuration attributes, reading from the persistent configuration store or falling back to built-in defaults when no explicit user configuration value has been set for that attribute.",
						UsageLine:   "podctl config get <attribute>",
						Args:        clihelp.ExactArgs(1),
						Run: func(ctx *clihelp.Context) error {
							fmt.Fprintf(ctx.Stdout, "%s = 5 (default)\n", ctx.Args[0])
							return nil
						},
					},
				},
			},
			{
				Name:        "deploy",
				Description: "Publish your compiled podcast RSS feed XML and encoded audio episode MP3 files to cloud storage providers like Amazon S3 and Google Cloud Storage with optional CDN cache invalidation, dry-run simulation mode, configurable timeout, and multi-stage deployment to staging and production environments.",
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
					fmt.Fprintf(ctx.Stdout, "Deploying podcast feed to environment '%s' (dry-run: %v, timeout: %v)...\nSuccess.\n",
						deployStage, deployDryRun, deployTimeout)
					return nil
				},
			},
			{
				Name:        "status",
				Description: "Check and display comprehensive health metrics for your podcast RSS feed, CDN edge cache status, episode download statistics over configurable time windows, origin server connectivity, and real-time feed validation status across all configured deployment stages and environments.",
				UsageLine:   "podctl status [options]",
				Options: []clihelp.Option{
					clihelp.Enum(&statusStage, "-S, --stage STAGE", []string{"staging", "production"}, "production", "Environment to inspect"),
					clihelp.Bool(&statusJSON, "--json", false, "Output metrics and status in JSON format"),
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
			{
				Name:        "completion",
				Description: "Generate and print shell autocompletion scripts for bash or zsh shells that enable tab-completion for all podctl commands, subcommands, flags, and option values to improve command-line efficiency and reduce typing errors during daily usage of the tool.",
				UsageLine:   "podctl completion <bash|zsh>",
				Args:        clihelp.ExactArgs(1),
				Run: func(ctx *clihelp.Context) error {
					switch ctx.Args[0] {
					case "bash":
						return clihelp.GenBashCompletion(ctx.App, ctx.Stdout)
					case "zsh":
						return clihelp.GenZshCompletion(ctx.App, ctx.Stdout)
					default:
						return fmt.Errorf("unsupported shell: %s (choose bash or zsh)", ctx.Args[0])
					}
				},
			},
			deepCmd,
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
		Description: fmt.Sprintf("**%s** — This is the [%s command](https://example.com/%s) at depth %d with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.", strings.Join(path, " "), name, strings.Join(path, "/"), depth),
		UsageLine:   fmt.Sprintf("podctl %s [options] [arguments...] — This is a **very long usage line** for the [%s command](https://example.com/%s) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.", strings.Join(path, " "), name, strings.Join(path, "/")),
	}

	if depth < 5 {
		suffixes := levelSuffixes[depth]
		child1 := name + "-" + suffixes[0]
		child2 := name + "-" + suffixes[1]
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
