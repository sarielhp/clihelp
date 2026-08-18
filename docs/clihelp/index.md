---
title: podctl
has_children: true
---

# podctl

[podctl](https://podctl.example.com) — A podcast distribution & audio processing tool.

## Commands

| Command | Description |
|---------|-------------|
| [build](build.md) | Compile & package audio episodes with **metadata** and *ID3 tags* |
| [serve](serve.md) | Start local development RSS feed server |
| [config](config.md) | View and manage application configuration settings |
| [deploy](deploy.md) | Publish RSS feed & audio to cloud storage (e.g. `S3`, `GCS`) and CDN |
| [status](status.md) | Check RSS feed health, CDN metrics, and download stats |
| [completion](completion.md) | Generate shell autocompletion script |

## Global Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

## Version

0.2.9

## About

Documentation & source: [https://github.com/sarielhp/clihelp](https://github.com/sarielhp/clihelp)
Run 'podctl <command> --help' for command-specific options.

