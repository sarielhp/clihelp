---
title: 'mail_cli labels rename'
parent: 'mail_cli labels'
---

# mail\_cli labels rename

Rename an existing label and move all corresponding emails.

## Usage

```
mail_cli labels rename <old_name> <new_name>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<old_name>` | The current label name (e.g. "sort-coop"). |
| `<new_name>` | The new label name (e.g. "Sort/Services/Coop"). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli labels rename "sort-coop" "Sort/Services/Coop"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
