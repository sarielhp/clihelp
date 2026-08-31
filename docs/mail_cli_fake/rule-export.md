---
title: 'mail_cli rule export'
parent: 'mail_cli rule'
---

# mail\_cli rule export

Export local auto-labeling rules from config.json to mail server filters. If the 'force' keyword is supplied, conflicting remote filters are overwritten. For JMAP accounts (e.g. FastMail), server-side filters are not supported; use '--sieve <file>' to export as a Sieve script.

## Usage

```
mail_cli rule export [force]
mail_cli rule export --sieve <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Examples

- `mail_cli rule export`
- `mail_cli rule export force`
- `mail_cli rule export --sieve rules.sieve`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
