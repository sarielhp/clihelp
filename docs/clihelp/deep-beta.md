---
title: 'podctl deep beta'
has_children: true
parent: 'podctl deep'
---

# podctl deep beta

This is the [beta command](https://example.com/deep/beta) at depth 2 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta [options] [arguments...] — This is a **very long usage line** for the [beta command](https://example.com/deep/beta) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| beta\_one \[arguments...\] | This is the [beta_one command](https://example.com/deep/beta/beta_one) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| beta\_two \[arguments...\] | This is the [beta_two command](https://example.com/deep/beta/beta_two) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

---

[↑ podctl](index.md) — [nav](nav.md)
