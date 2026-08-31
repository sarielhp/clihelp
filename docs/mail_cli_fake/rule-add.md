---
title: 'mail_cli rule add'
parent: 'mail_cli rule'
---

# mail\_cli rule add

Add an auto-labeling rule by sender. Emails from the specified sender address will automatically be labeled with the target label and archived (the "received" label will be removed).

## Usage

```
mail_cli rule add <email> <lbl>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The sender email address (e.g. newsletter@example.com). |
| `<lbl>` | The target label hierarchy (e.g. "Sort/Newsletters"). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli rule add newsletter@example.com "Sort/Newsletters"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
