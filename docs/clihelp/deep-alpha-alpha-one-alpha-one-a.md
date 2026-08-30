---
title: 'podctl deep alpha alpha_one alpha_one_a'
has_children: true
parent: 'podctl deep alpha alpha_one'
---

# podctl deep alpha alpha\_one alpha\_one\_a

This is the [alpha_one_a command](https://example.com/deep/alpha/alpha_one/alpha_one_a) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep alpha alpha_one alpha_one_a [options] [arguments...] — This is a **very long usage line** for the [alpha_one_a command](https://example.com/deep/alpha/alpha_one/alpha_one_a) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| alpha\_one\_a\_i \[arguments...\] | This is the [alpha_one_a_i command](https://example.com/deep/alpha/alpha_one/alpha_one_a/alpha_one_a_i) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| alpha\_one\_a\_ii \[arguments...\] | This is the [alpha_one_a_ii command](https://example.com/deep/alpha/alpha_one/alpha_one_a/alpha_one_a_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

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
