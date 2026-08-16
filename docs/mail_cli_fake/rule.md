---
title: mail_cli rule
has_children: true
---

# mail\_cli rule

Manage auto-labeling rules for matching senders or subject prefixes.

## Usage

```
mail_cli rule <subcommand> [args...]
mail_cli rule -export <file>
mail_cli rule -import <file>
```

## Subcommands

| Command | Description |
|---------|-------------|
| add \<email> \<lbl> | Add an auto-labeling rule by sender. |
| add\_by\_title \<title> \<lbl> | Add an auto-labeling rule by subject prefix. |
| add\_domain \<msg\_id> \[lbl\] | Add an auto-labeling rule for all emails from a sender's domain. |
| del \<email|title> | Remove an auto-labeling rule. |
| list \[-a, --all\] | List custom routing rules. |
| [update](rule-update.md) | Sync rules from blacklisted senders. |
| export \[force\] | Export local rules to mail server filters. |
| export --sieve \<f> | Export rules as a Sieve script file. |

## Flags

| Flag | Description |
|------|-------------|
| `-export <file>` | Export all existing rules to a JSON file. |
| `-import <file>` | Import rules from a JSON file, ignoring duplicates. |

## Examples

- `mail_cli rule list`
- `mail_cli rule list --all`
- `mail_cli rule export`
- `mail_cli rule export force`
- `mail_cli rule -export rules.json`
- `mail_cli rule -import rules.json`
- `mail_cli rule add billing@netflix.com "Sort/Services/Netflix"`
- `mail_cli rule add_by_title "GitHub" "Sort/GitHub"`
- `mail_cli rule add_domain 12345 "Sort/Newsletters"`

---

[↑ mail\_cli](index.md) — [nav](nav.md)
