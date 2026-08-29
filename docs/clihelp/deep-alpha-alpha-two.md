---
title: 'podctl deep alpha alpha_two'
has_children: true
parent: 'podctl deep alpha'
---

# podctl deep alpha alpha\_two

This is the [alpha_two command](https://example.com/deep/alpha/alpha_two) at depth 3 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep alpha alpha_two [options] [arguments...] — This is a **very long usage line** for the [alpha_two command](https://example.com/deep/alpha/alpha_two) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| alpha\_two\_a \[arguments...\] | This is the [alpha_two_a command](https://example.com/deep/alpha/alpha_two/alpha_two_a) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| alpha\_two\_b \[arguments...\] | This is the [alpha_two_b command](https://example.com/deep/alpha/alpha_two/alpha_two_b) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

---

[↑ podctl](index.md) — [nav](nav.md)
