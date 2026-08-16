---
title: podctl config set time
---

# podctl config set time

Set max execution timeout or timestamp window

## Usage

```
podctl config set time <seconds> [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `--unit SEC` | Time unit format (s: seconds, m: minutes, h: hours) |
| `--persist` | Save setting to configuration file |

## Examples

- `podctl config set time 120`
- `podctl config set time 2 --unit h --persist`

---

[↑ podctl](index.md) — [nav](nav.md)
