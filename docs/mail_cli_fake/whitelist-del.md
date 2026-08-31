---
title: 'mail_cli whitelist del'
parent: 'mail_cli whitelist'
---

# mail\_cli whitelist del

Remove a sender email address from your personal whitelist.

## Usage

```
mail_cli whitelist del <email>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The whitelisted email address to remove. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli whitelist del mom@gmail.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
