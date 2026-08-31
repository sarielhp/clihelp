---
title: 'podctl config get'
parent: 'podctl config'
---

# podctl config get

Display, inspect, and print configured attribute values. Reads from the persistent store or falls back to built-in defaults when no explicit user configuration value has been set.

## Usage

```
podctl config get <attribute>
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

## Examples

- `podctl config get space` — Inspect current storage space limit.

---

[↑ podctl](index.md) — [nav](nav.md)
