---
title: podctl serve
---

# podctl serve

Start local development RSS feed server

## Usage

```
podctl serve [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `-p, --port N` | Listen HTTP port number |
| `-H, --host HOST` | Bind IP host address |
| `--[no-]live-reload` | Automatically reload RSS feed |

## Examples

- `podctl serve`
- `podctl serve --port 9090 --no-live-reload`

---

[↑ podctl](index.md) — [nav](nav.md)
