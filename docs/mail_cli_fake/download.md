---
title: 'mail_cli download'
---

# mail\_cli download

Download all messages in the specified label (which must match a unique label) to a local mbox file.

## Usage

```
mail_cli download <label> <file_name>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<label>` | The name or prefix of the label containing messages to download (must match a unique label). |
| `<file_name>` | Path to the destination local mbox file (e.g. archive.mbox). |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli download inbox my_inbox.mbox`
- `mail_cli download Work/ProjectA project_a.mbox`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
