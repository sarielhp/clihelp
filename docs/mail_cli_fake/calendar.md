---
title: 'mail_cli calendar'
---

# mail\_cli calendar

Manage calendar events extracted from email attachments.

## Usage

```
mail_cli calendar <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| add \[label\_prefix\] \<message\_id> | Add a calendar event from an .ics attachment. Default prefix is 'inbox'. |
| week | Show all events in the upcoming week in the default calendar. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli calendar add abc123de`
- `mail_cli calendar add receipts xyz789gh`
- `mail_cli calendar week`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
