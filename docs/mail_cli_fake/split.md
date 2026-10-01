---
title: 'mail_cli split'
---

# mail\_cli split

Scan messages in the source label. If their subject matches the pattern (which may contain wildcards * and ?), move them to the target label. Runs in dry-run mode by default; use --do to perform actual operations.

## Usage

```
mail_cli split <source_label> <pattern> <target_label> [flags]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<source_label>` | Label (unique name or prefix) holding the messages |
| `<pattern>` | Subject pattern: * matches any run, ? any one character |
| `<target_label>` | Existing target label (unique name or prefix) |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |
| `--do` | Perform the actual move operations on the server instead of dry-run |

## Examples

- `mail_cli split inbox "*invoice*" Work/Billing`
- `mail_cli split inbox "*urgent*" Urgent --do`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
