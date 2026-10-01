---
title: 'mail_cli'
has_children: true
---

# mail\_cli

## Commands

| Command | Description |
|---------|-------------|
| [account](account.md) | Manage and list configured mail accounts |
| [archive](archive.md) | Archive messages by ID, or everything in a label |
| [blacklist](blacklist.md) | Manage the personal sender blacklist |
| [cache](cache.md) | Manage the local email download cache |
| [caladd](caladd.md) | Add calendar events from .ics attachments in the inbox |
| [calendar](calendar.md) | Manage calendar events extracted from email attachments |
| [color](color.md) | Test terminal 24-bit true-color and 256-color support |
| [config](config.md) | Show or manage configuration options |
| [download](download.md) | Download a label's messages to a local mbox file |
| [filter](filter.md) | Manage remote filters on Gmail |
| [labels](labels.md) | Manage and organize folders/labels |
| [learn-ham](learn-ham.md) | Train Bogofilter on ham (non-spam) emails in a folder |
| [migrate](migrate.md) | Copy configuration and credentials to another machine |
| [rule](rule.md) | Manage auto-labeling rules for senders and subjects |
| [scan](scan.md) | Scan folders under a label prefix for spam |
| [show](show.md) | Show emails in matching folders, or a single email |
| [spam](spam.md) | Manage the Spam folder and train filters |
| [splice](splice.md) | Move messages into keep/YYYY/MM/<folder> |
| [split](split.md) | Scan messages in the source label |
| [test](test.md) | Run self-tests of credentials and mail flow |
| [tui](tui.md) | Open the interactive terminal email browser |
| [unspam](unspam.md) | Mark a message as not spam and restore it |
| [upload](upload.md) | Upload an mbox file's messages to a label |
| [whitelist](whitelist.md) | Manage the personal sender whitelist to bypass spam checks |

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
| `-1, -2, -3` | Shorthand flags to select configured accounts |

## Version

0.5.4

## Config file location

`/home/sariel/.config/mail_cli/config.json`

