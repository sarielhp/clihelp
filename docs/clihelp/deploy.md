---
title: podctl deploy
---

# podctl deploy

Publish RSS feed & audio files to cloud storage / CDN

## Usage

```
podctl deploy [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `-s, --stage STAGE` | Target deployment environment |
| `--dry-run` | Simulate publishing without uploading files |
| `--purge-cdn` | Invalidate CDN cache for feed and updated audio files |
| `--timeout SEC` | Maximum upload timeout |

---

[↑ podctl](index.md) — [nav](nav.md)
