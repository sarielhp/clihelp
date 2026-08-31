---
title: 'mail_cli test'
---

# mail\_cli test

Run system and integration self-tests to verify API credentials and mail flow.

## Usage

```
mail_cli test run
```

## Subcommands

| Command | Description |
|---------|-------------|
| run | Execute connection and integration tests. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli test run`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
