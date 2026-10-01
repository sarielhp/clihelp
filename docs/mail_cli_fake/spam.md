---
title: 'mail_cli spam'
---

# mail\_cli spam

Manage Spam folder, train filters, and unsubscribe from political mail.

## Usage

```
mail_cli spam <subcommand> [args...]
mail_cli spam <message_id...>           Mark one or more messages as spam by ID.
```

## Subcommands

| Command | Description |
|---------|-------------|
| del | Permanently purge all emails in the Spam folder |
| pol audit | Score political fundraising emails in Spam |
| pol unsub | Unsubscribe from political mail and delete it |
| bye | Unsubscribe political spam, train on the rest, purge Spam |
| learn \[force\] | Train Bogofilter on the Spam folder ('force' retrains) |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |

## Examples

- `mail_cli spam del`
- `mail_cli spam pol audit`
- `mail_cli spam pol unsub`
- `mail_cli spam bye`
- `mail_cli spam learn`
- `mail_cli spam learn force`
- `mail_cli spam abc123de`

## Unsubscribing

Unsubscribing from political mail is safe because PACs and campaigns are registered entities that respect opt-out requests. For regular spam, unsubscribing is unsafe: it confirms to malicious actors that your address is active.

---

[↑ mail\_cli](index.md) — [nav](nav.md)
