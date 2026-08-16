# deploy

Publish RSS feed & audio files to cloud storage / CDN

## Usage

```
podctl deploy [options] <stage>
```

## Flags

- `-s, --stage STAGE` — Target deployment environment (staging | production)
- `--dry-run` — Simulate publishing without uploading files
- `--purge-cdn` — Invalidate CDN cache for feed and updated audio files
- `--timeout SEC` — Maximum upload timeout in seconds (default: 300)

## Examples

- `podctl deploy --dry-run staging`
- `podctl deploy -s production --purge-cdn`
