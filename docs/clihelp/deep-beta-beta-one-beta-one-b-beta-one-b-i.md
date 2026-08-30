---
title: 'podctl deep beta beta_one beta_one_b beta_one_b_i'
parent: 'podctl deep beta beta_one beta_one_b'
---

# podctl deep beta beta\_one beta\_one\_b beta\_one\_b\_i

This is the [beta_one_b_i command](https://example.com/deep/beta/beta_one/beta_one_b/beta_one_b_i) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta beta_one beta_one_b beta_one_b_i [options] [arguments...] — This is a **very long usage line** for the [beta_one_b_i command](https://example.com/deep/beta/beta_one/beta_one_b/beta_one_b_i) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
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

---

[↑ podctl](index.md) — [nav](nav.md)
