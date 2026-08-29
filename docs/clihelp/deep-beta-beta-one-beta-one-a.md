---
title: 'podctl deep beta beta_one beta_one_a'
has_children: true
parent: 'podctl deep beta beta_one'
---

# podctl deep beta beta\_one beta\_one\_a

This is the [beta_one_a command](https://example.com/deep/beta/beta_one/beta_one_a) at depth 4 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines.

## Usage

```
podctl deep beta beta_one beta_one_a [options] [arguments...] — This is a **very long usage line** for the [beta_one_a command](https://example.com/deep/beta/beta_one/beta_one_a) that should definitely trigger word-wrapping in the help output because it exceeds typical terminal widths and needs to be reflowed properly by the formatter.
```

## Subcommands

| Command | Description |
|---------|-------------|
| beta\_one\_a\_i \[arguments...\] | This is the [beta_one_a_i command](https://example.com/deep/beta/beta_one/beta_one_a/beta_one_a_i) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |
| beta\_one\_a\_ii \[arguments...\] | This is the [beta_one_a_ii command](https://example.com/deep/beta/beta_one/beta_one_a/beta_one_a_ii) at depth 5 with a very long description that should trigger word-wrapping behavior in the help output formatter to ensure proper text reflow across multiple lines. |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |

---

[↑ podctl](index.md) — [nav](nav.md)
