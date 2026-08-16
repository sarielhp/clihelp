---
title: mail_cli config
---

# mail\_cli config

Show or manage configuration options.

## Usage

```
mail_cli config <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| show | Show the current configuration (download directory, limits, accounts, and browser). |
| set \<key> \<value> | Set configuration parameters (spam_learn, unspam_learn, browser). |
| reset \<key> | Reset configuration parameters to system default (browser). |
| validate | Validate configurations, account parameters, DNS reachability, and Bogofilter service. |

## Examples

- `mail_cli config show`
- `mail_cli config set spam_learn [Gmail]/Spam`
- `mail_cli config set browser brave-browser`
- `mail_cli config reset browser`
- `mail_cli config validate`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
