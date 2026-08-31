---
title: 'mail_cli — Navigation'
---

# mail\_cli — Navigation

## Commands

- [account](account.md) — Manage and list configured mail accounts.
- [archive (arc)](archive.md) — Move message(s) by ID from their current folder to the Archive or Received folder. Or archive all messages in Inbox (default) or the specified label (by prefix).
- [blacklist (blist)](blacklist.md) — Manage the personal sender blacklist to instantly classify messages as spam.
  - [add](blacklist-add.md) — Add a sender email address to your personal blacklist. Senders on the blacklist are immediately marked as spam without querying Bogofilter.
  - [del](blacklist-del.md) — Remove a sender email address from your personal blacklist.
- [cache](cache.md) — Manage the local email download cache.
  - [prune](cache-prune.md) — Prune cached emails and scores older than a certain number of days.
  - [reset](cache-reset.md) — Reset the per-account cache directory, removing all cached data and recreating it empty.
- [caladd (calendar add-all)](caladd.md) — Scan the inbox for messages containing .ics attachments, and add them to the calendar if they are not already present.
- [calendar](calendar.md) — Manage calendar events extracted from email attachments.
- [color](color.md) — Test terminal 24-bit true-color and 256-color support
- [config](config.md) — Show or manage configuration options.
- [download](download.md) — Download all messages in the specified label (which must match a unique label) to a local mbox file.
- [filter](filter.md) — Manage remote filters on Gmail.
- [labels](labels.md) — Manage and organize folders/labels.
  - [create](labels-create.md) — Create a new label on the server.
  - [print](labels-print.md) — Print all labels/folders, one per line, with their full paths and no decorative layout or statistics.
  - [rename](labels-rename.md) — Rename an existing label and move all corresponding emails.
  - [del](labels-del.md) — Delete an existing label by its name.
  - [search](labels-search.md) — Search labels whose full path contains the given substring (case-insensitive). Uses the cached labels list; refreshes asynchronously if the cache is older than 24 hours.
  - [cache](labels-cache.md) — Manage the labels cache used by the search subcommand.
    - [update](labels-cache-update.md) — Force an immediate update of the labels cache from the server.
- [learn-ham (learn\_ham)](learn-ham.md) — Train Bogofilter on ham (non-spam) emails in a folder. The folder must be an exact match and cannot have subfolders.
- [migrate](migrate.md) — Copy configuration and credentials to a remote machine via SSH/SCP
- [rule](rule.md) — Manage auto-labeling rules for matching senders or subject prefixes.
  - [add](rule-add.md) — Add an auto-labeling rule by sender. Emails from the specified sender address will automatically be labeled with the target label and archived (the "received" label will be removed).
  - [add\_domain](rule-add-domain.md) — Add an auto-labeling rule for all emails from the sender's domain. Extracts the domain of the sender of the specified cached email and creates a rule to auto-label all emails from that domain.
  - [add\_by\_title](rule-add-by-title.md) — Add an auto-labeling rule by subject prefix. Emails with subjects starting with the specified title prefix will automatically be labeled with the target label and archived (the "received" label will be removed).
  - [del](rule-del.md) — Remove an auto-labeling rule for a sender email address or subject prefix.
  - [export](rule-export.md) — Export local auto-labeling rules from config.json to mail server filters. If the 'force' keyword is supplied, conflicting remote filters are overwritten. For JMAP accounts (e.g. FastMail), server-side filters are not supported; use '--sieve <file>' to export as a Sieve script.
  - [list](rule-list.md) — List custom routing and auto-labeling rules for the selected account.
  - [delete\_all](rule-delete-all.md) — Delete all custom routing rules for the selected account.
  - [update](rule-update.md) — Ensure all blacklisted senders have a corresponding local auto-labeling rule pointing to the SpamLearn folder.
- [scan](scan.md) — Scan all folders starting with the given label prefix (case-insensitive) for spam.
- [show](show.md) — Show the contents of emails in folders matching a label prefix, or show a specific email's details and body without running spam checks.
- [spam](spam.md) — Manage Spam folder, train filters, and unsubscribe from political mail.
- [splice](splice.md) — Move messages from a folder into the keep/YYYY/MM/<folder> structure. The root "keep" is fixed. Use -f to change the target folder name, or -F to change the target folder name and automatically suffix it with the year and month.
- [split](split.md) — Scan messages in the source label. If their subject matches the pattern (which may contain wildcards * and ?), move them to the target label. Runs in dry-run mode by default; use --do to perform actual operations.
- [test](test.md) — Run system and integration self-tests to verify API credentials and mail flow.
- [tui](tui.md) — Open the interactive terminal email browser. With an optional label_prefix argument, open the TUI with the matching label as the initial folder. The prefix is matched case-insensitively as a substring against the full label path. If exactly one label matches, the TUI opens on that label. If multiple match, all matching labels are printed and the program exits.
- [unspam](unspam.md) — Mark a message as not being spam: train bogofilter as ham and move it from Spam back to Inbox on the server.
- [upload](upload.md) — Upload all email messages from a local mbox file to the specified target label/folder on the server.
- [whitelist (wlist)](whitelist.md) — Manage the personal sender whitelist to bypass spam checks.
  - [add](whitelist-add.md) — Add a sender email address to your personal whitelist. Senders on the whitelist bypass all language, script, and spam filters.
  - [del](whitelist-del.md) — Remove a sender email address from your personal whitelist.

## Shortcut Commands

- [ss](ss.md) — Shortcut alias for: scan spam
- [sb](sb.md) — Shortcut alias for: spam bye
