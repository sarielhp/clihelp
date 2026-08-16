---
title: mail_cli
has_children: true
---

# mail\_cli

## Commands

| Command | Description |
|---------|-------------|
| [account](account.md) | Manage and list configured mail accounts. |
| [archive (arc)](archive.md) | Move message(s) by ID from their current folder to the Archive or Received folder. Or archive all messages in Inbox (default) or the specified label (by prefix). |
| [blacklist (blist)](blacklist.md) | Manage the personal sender blacklist to instantly classify messages as spam. |
| [cache](cache.md) | Manage the local email download cache. |
| [caladd (calendar add-all)](caladd.md) | Scan the inbox for messages containing .ics attachments, and add them to the calendar if they are not already present. |
| [calendar](calendar.md) | Manage calendar events extracted from email attachments. |
| [color](color.md) | Test terminal 24-bit true-color and 256-color support |
| [config](config.md) | Show or manage configuration options. |
| [download](download.md) | Download all messages in the specified label (which must match a unique label) to a local mbox file. |
| [filter](filter.md) | Manage remote filters on Gmail. |
| [labels](labels.md) | Manage and organize folders/labels. |
| [learn-ham (learn\_ham)](learn-ham.md) | Train Bogofilter on ham (non-spam) emails in a folder. The folder must be an exact match and cannot have subfolders. |
| [migrate](migrate.md) | Copy configuration and credentials to a remote machine via SSH/SCP |
| [rule](rule.md) | Manage auto-labeling rules for matching senders or subject prefixes. |
| [scan](scan.md) | Scan all folders starting with the given label prefix (case-insensitive) for spam. |
| [show](show.md) | Show the contents of emails in folders matching a label prefix, or show a specific email's details and body without running spam checks. |
| [spam](spam.md) | Manage Spam folder, train filters, and unsubscribe from political mail. |
| [splice](splice.md) | Move messages from a folder into the keep/YYYY/MM/<folder> structure. The root "keep" is fixed. Use -f to change the target folder name, or -F to change the target folder name and automatically suffix it with the year and month. |
| [split](split.md) | Scan messages in the source label. If their subject matches the pattern (which may contain wildcards * and ?), move them to the target label. Runs in dry-run mode by default; use --do to perform actual operations. |
| [test](test.md) | Run system and integration self-tests to verify API credentials and mail flow. |
| [tui](tui.md) | Open the interactive terminal email browser. With an optional label_prefix argument, open the TUI with the matching label as the initial folder. The prefix is matched case-insensitively as a substring against the full label path. If exactly one label matches, the TUI opens on that label. If multiple match, all matching labels are printed and the program exits. |
| [unspam](unspam.md) | Mark a message as not being spam: train bogofilter as ham and move it from Spam back to Inbox on the server. |
| [upload](upload.md) | Upload all email messages from a local mbox file to the specified target label/folder on the server. |
| [whitelist (wlist)](whitelist.md) | Manage the personal sender whitelist to bypass spam checks. |

## Shortcut Commands

| Command | Description |
|---------|-------------|
| [ss](ss.md) | Shortcut alias for: scan spam |
| [sb](sb.md) | Shortcut alias for: spam bye |

## Global Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose diagnostic log output |
| `-A, --account` | Specify target account name from config.json |
| `-1, -2, -3...` | Shorthand flag to select configured accounts |

## Version

0.5.4

## Config file location

`/home/sariel/.config/mail_cli/config.json`

