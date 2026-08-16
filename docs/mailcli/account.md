# mail\_cli account

Manage and list configured mail accounts.

## Usage

```
mail_cli account <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| list | List all configured mail accounts with status. |
| new \<jmap|gmail|outlook> \[name\] | Add a new JMAP, Gmail, or Outlook account template to config.json. |
| associate \[account\_name\] \<prog> | Associate a program/symlink name with an account. |
| rename \[old\_name\] \[new\_name\] | Rename an existing account and update cache/tokens. |
| delete \<account\_name> | Delete an existing account and its credentials. |
| test \[account\_name\] | Test validation and server connection for an account. |
| calendar \[account\_name\] | Designate or show the calendar manager account. |
| login \[account\_name\] | Perform interactive OAuth login for a Gmail or Outlook account. |

## Examples

- `mail_cli account list`
- `mail_cli account new outlook outlook-personal`
- `mail_cli account login outlook-personal`
- `mail_cli account test outlook-personal`
- `mail_cli account delete outlook-personal`
- `mail_cli account associate outlook-personal personal-mail`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
