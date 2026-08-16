# mail\_cli cache

Manage the local email download cache.

## Usage

```
mail_cli cache <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| prune \[days\] | Prune cached emails and scores older than [days] (default: 30). |
| [reset](cache-reset.md) | Reset per-account cache — removes all cached emails, scores, labels, and indexes for the current account. |

## Flags

| Flag | Description |
|------|-------------|
| `--wipe` | Wipe the entire cache (equivalent to prune with 0 days). |

## Examples

- `mail_cli cache prune`
- `mail_cli cache prune 15`
- `mail_cli cache prune --wipe`
- `mail_cli cache reset`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
