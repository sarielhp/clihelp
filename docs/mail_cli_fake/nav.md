---
title: 'mail_cli — Navigation'
---

# mail\_cli — Navigation

## Commands

- [account](account.md) — Manage and list configured mail accounts
- [archive](archive.md) — Archive messages by ID, or everything in a label
- [blacklist](blacklist.md) — Manage the personal sender blacklist
  - [add](blacklist-add.md) — Add a sender email address to your personal blacklist
  - [list](blacklist-list.md) — List all blacklisted email addresses
  - [del](blacklist-del.md) — Remove a sender email address from your personal blacklist
- [cache](cache.md) — Manage the local email download cache
  - [prune](cache-prune.md) — Prune cached emails and scores older than a certain number of days
  - [reset](cache-reset.md) — Reset the per-account cache directory, removing all cached data and recreating it empty
- [caladd](caladd.md) — Add calendar events from .ics attachments in the inbox
- [calendar](calendar.md) — Manage calendar events extracted from email attachments
- [color](color.md) — Test terminal 24-bit true-color and 256-color support
- [config](config.md) — Show or manage configuration options
- [download](download.md) — Download a label's messages to a local mbox file
- [filter](filter.md) — Manage remote filters on Gmail
- [labels](labels.md) — Manage and organize folders/labels
  - [list](labels-list.md) — List labels/folders
  - [create](labels-create.md) — Create a new label on the server
  - [print](labels-print.md) — Print all labels/folders, one per line, with their full paths and no decorative layout or statistics
  - [rename](labels-rename.md) — Rename an existing label and move all corresponding emails
  - [del](labels-del.md) — Delete an existing label by its name
  - [search](labels-search.md) — Search labels whose full path contains the given substring (case-insensitive)
  - [cache](labels-cache.md) — Manage the labels cache used by the search subcommand
    - [update](labels-cache-update.md) — Force an immediate update of the labels cache from the server
- [learn-ham](learn-ham.md) — Train Bogofilter on ham (non-spam) emails in a folder
- [migrate](migrate.md) — Copy configuration and credentials to another machine
- [rule](rule.md) — Manage auto-labeling rules for senders and subjects
  - [add](rule-add.md) — Add an auto-labeling rule by sender
  - [add\_domain](rule-add-domain.md) — Add an auto-labeling rule for all emails from the sender's domain
  - [add\_by\_title](rule-add-by-title.md) — Add an auto-labeling rule by subject prefix
  - [del](rule-del.md) — Remove an auto-labeling rule for a sender email address or subject prefix
  - [export](rule-export.md) — Export local auto-labeling rules from config.json to mail server filters
  - [list](rule-list.md) — List custom routing and auto-labeling rules for the selected account
  - [delete\_all](rule-delete-all.md) — Delete all custom routing rules for the selected account
  - [update](rule-update.md) — Ensure all blacklisted senders have a corresponding local auto-labeling rule pointing to the SpamLearn folder
- [scan](scan.md) — Scan folders under a label prefix for spam
- [show](show.md) — Show emails in matching folders, or a single email
- [spam](spam.md) — Manage the Spam folder and train filters
- [splice](splice.md) — Move messages into keep/YYYY/MM/<folder>
- [split](split.md) — Scan messages in the source label
- [test](test.md) — Run self-tests of credentials and mail flow
- [tui](tui.md) — Open the interactive terminal email browser
- [unspam](unspam.md) — Mark a message as not spam and restore it
- [upload](upload.md) — Upload an mbox file's messages to a label
- [whitelist](whitelist.md) — Manage the personal sender whitelist to bypass spam checks
  - [add](whitelist-add.md) — Add a sender email address to your personal whitelist
  - [list](whitelist-list.md) — List all whitelisted email addresses
  - [del](whitelist-del.md) — Remove a sender email address from your personal whitelist

## Shortcut Commands

- [ss](ss.md) — Shortcut alias for: scan spam
- [sb](sb.md) — Shortcut alias for: spam bye
