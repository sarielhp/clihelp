---
title: 'mail_cli scan'
---

# mail\_cli scan

Scan all folders starting with the given label prefix (case-insensitive) for spam.

## Usage

```
mail_cli scan <lbl_prefix> [flags]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<lbl_prefix>` | Label/folder prefix to scan (e.g. 'inbox') |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |
| `-m, --move [From]` | Move spam to Spam; a From address moves one sender |
| `--inbox-move <From>` | Move a From address's messages back to the Inbox |
| `-p, --pattern <pattern>` | Only messages whose subject contains this pattern |

## Examples

- `mail_cli scan inbox`
- `mail_cli scan inbox -m`
- `mail_cli scan receipts --move=spammer@example.com` — Move one sender's mail

---

[↑ mail\_cli](index.md) — [nav](nav.md)
