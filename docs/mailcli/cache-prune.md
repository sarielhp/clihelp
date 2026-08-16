# cache prune \[days\]

Prune cached emails and scores older than a certain number of days.

## Usage

```
mail_cli cache prune [days] [--wipe]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `[days]` | Number of days (default: 30). |

## Flags

| Flag | Description |
|------|-------------|
| `--wipe` | Wipe the entire cache immediately. |

## Examples

- `mail_cli cache prune 7`
