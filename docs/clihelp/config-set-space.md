---
title: 'podctl config set space'
parent: 'podctl config set'
---

# podctl config set space

Set maximum disk space allocation for temporary cache and build artifacts. Configurable in megabytes or gigabytes with an optional automatic cleanup policy.

## Usage

```
podctl config set space <megabytes> [options]
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
| `--unit SIZE` | Space allocation unit (default: MB) |
| `--auto-cleanup` | Purge oldest temporary cache files |

---

[↑ podctl](index.md) — [nav](nav.md)
