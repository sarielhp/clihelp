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

## Examples

- `mail_cli calendar add abc123de`
- `mail_cli calendar add receipts xyz789gh`
- `mail_cli calendar week`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
