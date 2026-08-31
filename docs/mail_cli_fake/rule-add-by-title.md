---
title: 'mail_cli rule add_by_title'
parent: 'mail_cli rule'
---

# mail\_cli rule add\_by\_title

Add an auto-labeling rule by subject prefix. Emails with subjects starting with the specified title prefix will automatically be labeled with the target label and archived (the "received" label will be removed).

## Usage

```
mail_cli rule add_by_title <title> <lbl>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<title>` | The subject prefix to match (e.g. "[Alert]"). |
| `<lbl>` | The target label hierarchy (e.g. "Sort/Alerts"). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli rule add_by_title "[Alert]" "Sort/Alerts"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
