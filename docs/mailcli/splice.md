---
title: mail_cli splice
---

# mail\_cli splice

Move messages from a folder into the keep/YYYY/MM/<folder> structure. The root "keep" is fixed. Use -f to change the target folder name, or -F to change the target folder name and automatically suffix it with the year and month.

## Usage

```
mail_cli splice <folder> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-n, --n <int>` | Number of messages to process (default 10). |
| `--folder, -f` | Folder name to use for the destination path without suffix. |
| `--folder-suffix, -F` | Folder name to use for the destination path with year/month suffix attached. |
| `--move` | Actually move the messages instead of dry run. |

## Examples

- `mail_cli splice research/cfps`
- `mail_cli splice research/cfps -f archive (keep/YYYY/MM/archive)`
- `mail_cli splice research/cfps -F wuna (keep/YYYY/MM/wuna-YYYY-MM)`
- `mail_cli splice research/cfps -n 20 --move`

The destination folder/label is created on the server automatically if it does not exist.

## When dry run only (no --move)

messages are not moved - this shows where they would go.

---

[↑ mail\_cli](index.md) — [nav](nav.md)
