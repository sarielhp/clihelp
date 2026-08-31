---
title: 'mail_cli labels del'
parent: 'mail_cli labels'
---

# mail\_cli labels del

Delete an existing label by its name.

## Usage

```
mail_cli labels del <lbl_name>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<lbl_name>` | The name of the label to delete (e.g. "temp-label"). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli labels del "temp-label"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
