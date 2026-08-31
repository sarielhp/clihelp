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
| del | Permanently purge all emails in the Spam folder. |
| pol audit | Scan Spam folder for political fundraising emails and print heuristic scoring details. |
| pol unsub | Scan the Spam folder for political messages, execute unsubscription opt-outs, and delete matching emails. NOTE: Unsubscribing from political mail is safe because PACs/campaigns are registered entities that respect opt-out requests. For regular spam, unsubscribing is unsafe as it confirms your email is active to malicious actors. |
| bye | Execute a complete sweep: unsubscribe political spam, train the spam classifier on the remaining spam folder, and then permanently purge the spam folder. |
| learn \[force\] | Spam Learning Mode: Connect to Spam folder and train local Bogofilter. If 'force' is specified, bypasses trained message database. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli spam del`
- `mail_cli spam pol audit`
- `mail_cli spam pol unsub`
- `mail_cli spam bye`
- `mail_cli spam learn`
- `mail_cli spam learn force`
- `mail_cli spam abc123de`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
