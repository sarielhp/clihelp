---
title: 'podctl'
has_children: true
---

# podctl

[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.

## Commands

| Command | Description |
|---------|-------------|
| [build](build.md) | Compile, encode, and package raw audio into MP3 podcast episodes. Supports configurable bitrate, loudness normalization, and embedded ID3 tags for distribution across Apple Podcasts, Spotify, and Google Podcasts. |
| [serve](serve.md) | Start a local HTTP development server for RSS feeds and audio files. Includes live-reload support, CORS headers for cross-origin testing, and a built-in web dashboard for previewing feed metadata before deploying to production. |
| [config](config.md) | View, inspect, set, and manage application configuration settings. Controls storage locations, disk space limits, CDN bucket names, API keys, and publishing preferences via dedicated subcommands. |
| [deploy](deploy.md) | Publish compiled podcast RSS feeds and MP3 files to cloud storage. Supports Amazon S3, Google Cloud Storage, CDN cache invalidation, dry-run simulation, and multi-stage deployments. |
| [status](status.md) | Check and display comprehensive health and validation metrics. Monitors RSS feed status, CDN edge cache, episode download statistics, and origin server connectivity across environments. |
| [completion](completion.md) | Generate shell autocompletion scripts for bash or zsh. Enables fast tab-completion for all podctl commands, subcommands, flags, and option values. |
| [deep](deep.md) | **deep** — This is the [deep command](https://example.com/deep) at the root of the demonstration hierarchy with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines for testing purposes. |

## Global Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

## Version

0.2.21

## About

Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)

