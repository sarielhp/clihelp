---
title: 'mail_cli cache prune'
parent: 'mail_cli cache'
---

# mail\_cli cache prune

Prune cached emails and scores older than a certain number of days.

## Usage

```
mail_cli cache prune [days] [--wipe]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `[days]` | Number of days (default: 30). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |
| `--wipe` | Wipe the entire cache immediately. |

## Examples

- `mail_cli cache prune 7`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
