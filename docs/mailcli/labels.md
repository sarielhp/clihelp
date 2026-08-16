---
title: mail_cli labels
---

# mail\_cli labels

Manage and organize folders/labels.

## Usage

```
mail_cli labels <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| list \[-a, --all\] | List labels/folders. |
| create \<lbl> | Create a new label. |
| [print](labels-print.md) | Print all labels/folders, one per line (full path only). |
| rename \<old> \<new> | Rename a label and move all its emails. |
| fix | Fix nested folder parent hierarchies. |
| del \<lbl> | Delete a label. |
| search \<str> | Search labels by substring (matches full path). |
| [cache](labels-cache.md) | Manage the labels cache. |

## Examples

- `mail_cli labels list`
- `mail_cli labels list --all`
- `mail_cli labels create "Work/ProjectA"`
- `mail_cli labels print`
- `mail_cli labels rename "sort-coop" "Sort/Services/Coop"`
- `mail_cli labels search work`
- `mail_cli labels cache update`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
