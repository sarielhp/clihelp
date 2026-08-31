---
title: 'mail_cli archive'
---

# mail\_cli archive

Move message(s) by ID from their current folder to the Archive or Received folder. Or archive all messages in Inbox (default) or the specified label (by prefix).

## Usage

```
mail_cli archive <all [label] | message-id...>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `all [label]` | Archive all emails in the Inbox or specified label prefix. |
| `<message-id...>` | One or more message IDs to archive (short 8-char or full). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli archive abc123de`
- `mail_cli archive all`
- `mail_cli archive all receipts`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
