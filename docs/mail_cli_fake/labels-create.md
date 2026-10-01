---
title: 'mail_cli labels create'
parent: 'mail_cli labels'
---

# mail\_cli labels create

Create a new label on the server

## Usage

```
mail_cli labels create <lbl_name>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<lbl_name>` | Full name of the new label (e.g. "Work/ProjectA") |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |

## Examples

- `mail_cli labels create "Work/ProjectA"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
