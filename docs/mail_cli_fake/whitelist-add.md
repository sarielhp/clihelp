---
title: 'mail_cli whitelist add'
parent: 'mail_cli whitelist'
---

# mail\_cli whitelist add

Add a sender email address to your personal whitelist. Senders on the whitelist bypass all language, script, and spam filters.

## Usage

```
mail_cli whitelist add <email>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The sender email address to whitelist (e.g. mom@gmail.com). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli whitelist add mom@gmail.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
