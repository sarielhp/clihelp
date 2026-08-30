---
title: 'podctl deep'
has_children: true
---

# podctl deep

**deep** — This is the [deep command](https://example.com/deep) at the root of the demonstration hierarchy with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines for testing purposes.

## Usage

```
podctl deep [options] <subcommand> — This is a **very long usage line** for the [deep command](https://example.com/deep) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| alpha \[arguments...\] | This is the [alpha command](https://example.com/deep/alpha) at depth 2 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| beta \[arguments...\] | This is the [beta command](https://example.com/deep/beta) at depth 2 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

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
