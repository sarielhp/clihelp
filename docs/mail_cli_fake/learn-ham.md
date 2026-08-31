---
title: 'mail_cli learn-ham'
---

# mail\_cli learn-ham

Train Bogofilter on ham (non-spam) emails in a folder. The folder must be an exact match and cannot have subfolders.

## Usage

```
mail_cli learn-ham <label> [flags]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<label>` | The folder name containing ham emails to train on. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |
| `--force` | Bypass trained message database and re-train all emails. |

## Examples

- `mail_cli learn-ham receipts --force`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
