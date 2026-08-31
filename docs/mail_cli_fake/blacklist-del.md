---
title: 'mail_cli blacklist del'
parent: 'mail_cli blacklist'
---

# mail\_cli blacklist del

Remove a sender email address from your personal blacklist.

## Usage

```
mail_cli blacklist del <email>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The blacklisted email address to remove. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli blacklist del spammer@gmail.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
