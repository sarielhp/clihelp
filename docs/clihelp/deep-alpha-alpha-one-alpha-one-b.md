---
title: 'podctl deep alpha alpha_one alpha_one_b'
has_children: true
parent: 'podctl deep alpha alpha_one'
---

# podctl deep alpha alpha\_one alpha\_one\_b

This is the [alpha_one_b command](https://example.com/deep/alpha/alpha_one/alpha_one_b) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep alpha alpha_one alpha_one_b [options] [arguments...] — This is a **very long usage line** for the [alpha_one_b command](https://example.com/deep/alpha/alpha_one/alpha_one_b) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| alpha\_one\_b\_i \[arguments...\] | This is the [alpha_one_b_i command](https://example.com/deep/alpha/alpha_one/alpha_one_b/alpha_one_b_i) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| alpha\_one\_b\_ii \[arguments...\] | This is the [alpha_one_b_ii command](https://example.com/deep/alpha/alpha_one/alpha_one_b/alpha_one_b_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

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
