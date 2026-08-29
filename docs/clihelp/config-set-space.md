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
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--unit SIZE` | Space allocation unit (default: MB) |
| `--auto-cleanup` | Purge oldest temporary cache files |

---

[↑ podctl](index.md) — [nav](nav.md)
