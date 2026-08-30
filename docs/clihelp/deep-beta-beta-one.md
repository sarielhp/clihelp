---
title: 'podctl deep beta beta_one'
has_children: true
parent: 'podctl deep beta'
---

# podctl deep beta beta\_one

This is the [beta_one command](https://example.com/deep/beta/beta_one) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta beta_one [options] [arguments...] — This is a **very long usage line** for the [beta_one command](https://example.com/deep/beta/beta_one) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| beta\_one\_a \[arguments...\] | This is the [beta_one_a command](https://example.com/deep/beta/beta_one/beta_one_a) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| beta\_one\_b \[arguments...\] | This is the [beta_one_b command](https://example.com/deep/beta/beta_one/beta_one_b) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

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
