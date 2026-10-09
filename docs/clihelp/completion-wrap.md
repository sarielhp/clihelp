---
title: 'podctl completion wrap'
parent: 'podctl completion'
---

# podctl completion wrap

Generate a wrapper script with preset arguments

## Usage

```
podctl completion wrap [--completion-from <path>] <name> [--] [<args>...]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<name>` | Name of the wrapper script |
| `[<args>...]` | Preset arguments prepended to wrapped command |

## Flags

| Flag | Description |
|------|-------------|
| `--token TOKEN` | Bearer token for cluster authentication |
| `--api-key KEY` | API key for cloud provider access |
| `-c, --config PATH` | Path to configuration file (default: ~/.config/podctl.yaml) |
| `--endpoint URL` | API endpoint URL (default: https://api.podctl.example.com) |
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--no-color` | Disable ANSI color output |
| `--completion-from <path>` | Inspect an existing wrapper to extract its arguments |

## Examples

- `podctl completion wrap pd deploy` — Generate wrapper 'pd' for 'podctl deploy'
- `podctl completion wrap --completion-from ~/bin/mt` — Inspect existing script and generate wrapper

---

[↑ podctl](index.md) — [nav](nav.md)
