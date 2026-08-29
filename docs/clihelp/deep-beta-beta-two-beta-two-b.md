---
title: 'podctl deep beta beta_two beta_two_b'
has_children: true
parent: 'podctl deep beta beta_two'
---

# podctl deep beta beta\_two beta\_two\_b

This is the [beta_two_b command](https://example.com/deep/beta/beta_two/beta_two_b) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta beta_two beta_two_b [options] [arguments...] — This is a **very long usage line** for the [beta_two_b command](https://example.com/deep/beta/beta_two/beta_two_b) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| beta\_two\_b\_i \[arguments...\] | This is the [beta_two_b_i command](https://example.com/deep/beta/beta_two/beta_two_b/beta_two_b_i) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| beta\_two\_b\_ii \[arguments...\] | This is the [beta_two_b_ii command](https://example.com/deep/beta/beta_two/beta_two_b/beta_two_b_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

---

[↑ podctl](index.md) — [nav](nav.md)
