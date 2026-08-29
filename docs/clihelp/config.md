---
title: 'podctl config'
has_children: true
---

# podctl config

View, inspect, set, and manage application configuration settings. Controls storage locations, disk space limits, CDN bucket names, API keys, and publishing preferences via dedicated subcommands.

## Usage

```
podctl config <subcommand> [options]
```

## Subcommands

| Command | Description |
|---------|-------------|
| set \<attribute> \<value> | Assign, update, or override configuration attribute values. Supports nested key paths and bulk operations for efficient setup across development, staging, and production targets. |
| get \<attribute> | Display, inspect, and print configured attribute values. Reads from the persistent store or falls back to built-in defaults when no explicit user configuration value has been set. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

---

[↑ podctl](index.md) — [nav](nav.md)
