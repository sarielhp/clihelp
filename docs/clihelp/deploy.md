---
title: podctl deploy
---

# podctl deploy

Publish RSS feed & audio to cloud storage (e.g. `S3`, `GCS`) and CDN

## Usage

```
podctl deploy [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `-S, --stage STAGE` | Target deployment environment |
| `--dry-run` | Simulate publishing without uploading files |
| `--purge-cdn` | Invalidate CDN cache for feed and updated audio files |
| `--timeout SEC` | Maximum upload timeout |

## Safety Precaution

Always test with `--dry-run` before ~~overwriting~~ publishing to **production** (see [Deploy Docs](https://podctl.example.com/docs/deploy)).

---

[↑ podctl](index.md) — [nav](nav.md)
