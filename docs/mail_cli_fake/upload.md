---
title: 'mail_cli upload'
---

# mail\_cli upload

Upload all email messages from a local mbox file to the specified target label/folder on the server.

## Usage

```
mail_cli upload <label> <file_name>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<label>` | Target label/folder (unique name or prefix) |
| `<file_name>` | Path to the local mbox file containing emails to upload |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |

## Examples

- `mail_cli upload archive archive.mbox`
- `mail_cli upload Work/ProjectA project_a.mbox`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
