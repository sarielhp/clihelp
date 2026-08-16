---
title: mail_cli show
---

# mail\_cli show

Show the contents of emails in folders matching a label prefix, or show a specific email's details and body without running spam checks.

## Usage

```
mail_cli show <lbl_prefix> [message_id] [flags]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<lbl_prefix>` | The prefix of the label/folder to view (e.g. 'inbox' or 'receipts'). |
| `[message_id]` | Optional message ID (short 8-char or full) of a specific email to show. |

## Flags

| Flag | Description |
|------|-------------|
| `-w, --web` | Open the HTML body of the email in your configured browser. |

## Examples

- `mail_cli show inbox`
- `mail_cli show inbox abc123de`
- `mail_cli show inbox abc123de -w`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
