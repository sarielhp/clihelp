---
title: mail_cli labels search
---

# mail\_cli labels search

Search labels whose full path contains the given substring (case-insensitive). Uses the cached labels list; refreshes asynchronously if the cache is older than 24 hours.

## Usage

```
mail_cli labels search <substring>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<substring>` | Substring to search for in label paths. |

## Examples

- `mail_cli labels search work`
- `mail_cli labels search sort`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
