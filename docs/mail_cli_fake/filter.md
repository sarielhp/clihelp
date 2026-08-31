---
title: 'mail_cli filter'
---

# mail\_cli filter

Manage remote filters on Gmail.

## Usage

```
mail_cli filter <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| list | List all remote filters on Gmail with detailed action descriptions. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli filter list`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
