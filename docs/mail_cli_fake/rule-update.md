---
title: 'mail_cli rule update'
parent: 'mail_cli rule'
---

# mail\_cli rule update

Ensure all blacklisted senders have a corresponding local auto-labeling rule pointing to the SpamLearn folder.

## Usage

```
mail_cli rule update
```

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli rule update`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
