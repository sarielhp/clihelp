# split \<source\_label> \<pattern> \<target\_label>

Scan messages in the source label. If their subject matches the pattern (which may contain wildcards * and ?), move them to the target label. Runs in dry-run mode by default; use --do to perform actual operations.

## Usage

```
mail_cli split <source_label> <pattern> <target_label> [flags]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<source_label>` | The name or prefix of the label containing messages to scan (must match a unique label). |
| `<pattern>` | A subject match pattern supporting * (matches any characters) and ? (matches any single character). |
| `<target_label>` | The name or prefix of the target label (must match a unique label and already exist). |

## Flags

| Flag | Description |
|------|-------------|
| `--do` | Perform the actual move operations on the server instead of dry-run. |

## Examples

- `mail_cli split inbox "*invoice*" Work/Billing`
- `mail_cli split inbox "*urgent*" Urgent --do`
