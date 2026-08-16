# blacklist (alias: blist)

Manage the personal sender blacklist to instantly classify messages as spam.

## Usage

```
mail_cli blacklist <subcommand> [args...]
```

## Subcommands

| Command | Description |
|---------|-------------|
| add \<email> | Add an email address to the blacklist. |
| del \<email> | Remove an email address from the blacklist. |
| list | List all blacklisted email addresses. |

## Examples

- `mail_cli blacklist list`
- `mail_cli blacklist add spammer@example.com`
