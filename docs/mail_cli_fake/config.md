---
title: 'mail_cli config'
---

# mail\_cli config

Show or manage configuration options

## Usage

```
mail_cli config <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| show | Show the current configuration |
| set \<key> \<value> | Set spam_learn, unspam_learn or browser |
| reset \<key> | Reset configuration parameters to system default (browser) |
| validate | Validate configuration, accounts, DNS and Bogofilter |

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3` | Shorthand flags to select configured accounts |

## Examples

- `mail_cli config show`
- `mail_cli config set spam_learn [Gmail]/Spam`
- `mail_cli config set browser brave-browser`
- `mail_cli config reset browser`
- `mail_cli config validate`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
