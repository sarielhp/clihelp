---
title: 'mail_cli blacklist'
has_children: true
---

# mail\_cli blacklist

Manage the personal sender blacklist to instantly classify messages as spam.

## Usage

```
mail_cli blacklist <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| add \<email> | Add an email address to the blacklist. |
| del \<email> | Remove an email address from the blacklist. |
| list | List all blacklisted email addresses. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli blacklist list`
- `mail_cli blacklist add spammer@example.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
