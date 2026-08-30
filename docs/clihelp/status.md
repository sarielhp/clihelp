---
title: 'podctl status'
---

# podctl status

Check and display comprehensive health and validation metrics. Monitors RSS feed status, CDN edge cache, episode download statistics, and origin server connectivity across environments.

## Usage

```
podctl status [options]
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
| `-S, --stage STAGE` | Environment to inspect (default: production) |
| `--json` | Output metrics and status in JSON format |

---

[↑ podctl](index.md) — [nav](nav.md)
