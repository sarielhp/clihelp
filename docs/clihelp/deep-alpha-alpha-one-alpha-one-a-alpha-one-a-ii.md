---
title: 'podctl deep alpha alpha_one alpha_one_a alpha_one_a_ii'
parent: 'podctl deep alpha alpha_one alpha_one_a'
---

# podctl deep alpha alpha\_one alpha\_one\_a alpha\_one\_a\_ii

This is the [alpha_one_a_ii command](https://example.com/deep/alpha/alpha_one/alpha_one_a/alpha_one_a_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep alpha alpha_one alpha_one_a alpha_one_a_ii [options] [arguments...] — This is a **very long usage line** for the [alpha_one_a_ii command](https://example.com/deep/alpha/alpha_one/alpha_one_a/alpha_one_a_ii) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
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
