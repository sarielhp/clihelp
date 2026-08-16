---
title: mail_cli whitelist
has_children: true
---

# mail\_cli whitelist

Manage the personal sender whitelist to bypass spam checks.

## Usage

```
mail_cli whitelist <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| add \<email> | Add an email address to the whitelist. |
| del \<email> | Remove an email address from the whitelist. |
| list | List all whitelisted email addresses. |

## Examples

- `mail_cli whitelist list`
- `mail_cli whitelist add friend@example.com`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
