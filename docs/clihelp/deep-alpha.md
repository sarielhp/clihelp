---
title: 'podctl deep alpha'
has_children: true
parent: 'podctl deep'
---

# podctl deep alpha

This is the [alpha command](https://example.com/deep/alpha) at depth 2 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep alpha [options] [arguments...] — This is a **very long usage line** for the [alpha command](https://example.com/deep/alpha) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| alpha\_one \[arguments...\] | This is the [alpha_one command](https://example.com/deep/alpha/alpha_one) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| alpha\_two \[arguments...\] | This is the [alpha_two command](https://example.com/deep/alpha/alpha_two) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

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
