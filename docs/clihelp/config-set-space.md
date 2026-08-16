# podctl config set space

Set maximum disk space allocation or cache budget in MB

## Usage

```
podctl config set space <megabytes> [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `--unit SIZE` | Space allocation unit (MB | GB) |
| `--auto-cleanup` | Purge oldest temporary cache files when limit reached |
| `--persist` | Save setting to configuration file |

## Examples

- `podctl config set space 500`
- `podctl config set space 2 --unit GB --auto-cleanup`

---

[↑ podctl](index.md) — [nav](nav.md)
