---
title: 'mail_cli blacklist add'
parent: 'mail_cli blacklist'
---

# mail\_cli blacklist add

Add a sender email address to your personal blacklist. Senders on the blacklist are immediately marked as spam without querying Bogofilter.

## Usage

```
mail_cli blacklist add <email>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The sender email address to blacklist (e.g. spammer@gmail.com). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli blacklist add spammer@gmail.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
