---
title: 'podctl deploy'
---

# podctl deploy

Publish compiled podcast RSS feeds and MP3 files to cloud storage. Supports Amazon S3, Google Cloud Storage, CDN cache invalidation, dry-run simulation, and multi-stage deployments.

## Usage

```
podctl deploy [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `--token TOKEN` | Bearer token for cluster authentication |
| `--api-key KEY` | API key for cloud provider access |
| `-c, --config PATH` | Path to configuration file (default: ~/.config/podctl.yaml) |
| `--endpoint URL` | API service endpoint URL (default: https://api.podctl.example.com) |
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--no-color` | Disable ANSI color output |
| `-b, --bucket NAME` | Target cloud storage bucket name |
| `-S, --stage STAGE` | Target deployment environment (default: staging) |
| `--dry-run` | Simulate publishing without uploading files |
| `--purge-cdn` | Invalidate CDN cache for feed and updated audio files |
| `--timeout SEC` | Maximum upload timeout (default: 5m0s) |

## Safety Precaution

Always test with `--dry-run` before ~~overwriting~~ publishing to **production** (see [Deploy Docs](https://podctl.example.com/docs/deploy)).

---

[↑ podctl](index.md) — [nav](nav.md)
