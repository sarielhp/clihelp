---
title: 'podctl completion wrap'
parent: 'podctl completion'
---

# podctl completion wrap

Generate a wrapper script with preset arguments

## Usage

```
completion wrap [--from <path>] <name> [<args>...]
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
| `--endpoint URL` | API service endpoint URL (default: https://api.podctl.example.com) |
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--no-color` | Disable ANSI color output |
| `--from <path>` | Inspect an existing wrapper script to extract preset arguments |

## Examples

- `completion wrap pd deploy` — Generate wrapper 'pd' for '<app> deploy'
- `completion wrap --from ~/bin/mt` — Inspect existing script and generate wrapper

---

[↑ podctl](index.md) — [nav](nav.md)
