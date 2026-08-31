---
title: 'podctl serve'
---

# podctl serve

Start a local HTTP development server for RSS feeds and audio files. Includes live-reload support, CORS headers for cross-origin testing, and a built-in web dashboard for previewing feed metadata before deploying to production.

## Usage

```
podctl serve [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `--token TOKEN` | Bearer token for cluster authentication |
| `--api-key KEY` | API key for cloud provider access |
| `-c, --config PATH` | Path to configuration file (default: ~/.config/podctl.yaml) |
| `--endpoint URL` | API service endpoint URL (default: https://api.podctl.example.com) |
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--no-color` | Disable ANSI color output |
| `-p, --port N` | Listen HTTP port number (default: 8080) |
| `-H, --host HOST` | Bind IP host address (default: 127.0.0.1) |
| `--[no-]live-reload` | Automatically reload RSS feed (default: true) |

## Examples

- `podctl serve` — Start the local preview server on default port 8080.
- `podctl serve --port 9090 --no-live-reload` — Bind custom port and disable automatic live reload.

---

[↑ podctl](index.md) — [nav](nav.md)
