---
title: 'podctl config set'
has_children: true
parent: 'podctl config'
---

# podctl config set

Assign, update, or override configuration attribute values. Supports nested key paths and bulk operations for efficient setup across development, staging, and production targets.

## Usage

```
podctl config set <attribute> <value> [options]
```

## Subcommands

| Command | Description |
|---------|-------------|
| space \<megabytes> | Set maximum disk space allocation for temporary cache and build artifacts. Configurable in megabytes or gigabytes with an optional automatic cleanup policy. |

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

---

[↑ podctl](index.md) — [nav](nav.md)
