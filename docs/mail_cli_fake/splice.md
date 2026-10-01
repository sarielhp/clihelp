---
title: 'mail_cli splice'
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
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |
| `-n, --n <int>` | Number of messages to process (default 10) |
| `-f, --folder <name>` | Destination folder name, without suffix |
| `-F, --folder-suffix <name>` | Destination folder name, with year/month suffix |
| `--move` | Actually move the messages instead of dry run |

## Examples

- `mail_cli splice research/cfps`
- `mail_cli splice research/cfps -f archive` — Moves into keep/YYYY/MM/archive
- `mail_cli splice research/cfps -F wuna` — Moves into keep/YYYY/MM/wuna-YYYY-MM
- `mail_cli splice research/cfps -n 20 --move`

The destination folder/label is created on the server automatically if it does not exist.

## When dry run only (no --move)

messages are not moved - this shows where they would go.

---

[↑ mail\_cli](index.md) — [nav](nav.md)
