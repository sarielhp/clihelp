---
title: 'mail_cli rule list'
parent: 'mail_cli rule'
---

# mail\_cli rule list

List custom routing and auto-labeling rules for the selected account

## Usage

```
mail_cli rule list [-a, --all]
```

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |
| `-a, --all` | List all rules, including those already exported |

## Examples

- `mail_cli rule list`
- `mail_cli rule list --all`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
