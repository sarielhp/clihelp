---
title: 'podctl deep beta beta_two beta_two_a beta_two_a_ii'
parent: 'podctl deep beta beta_two beta_two_a'
---

# podctl deep beta beta\_two beta\_two\_a beta\_two\_a\_ii

This is the [beta_two_a_ii command](https://example.com/deep/beta/beta_two/beta_two_a/beta_two_a_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta beta_two beta_two_a beta_two_a_ii [options] [arguments...] — This is a **very long usage line** for the [beta_two_a_ii command](https://example.com/deep/beta/beta_two/beta_two_a/beta_two_a_ii) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
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
