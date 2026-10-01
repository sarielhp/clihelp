// Package main regenerates the complete mail_cli usage/help tree using
// clihelp's data model, reproducing every detailed usage page and the global
// command overview produced by mail_cli.
package main

import (
	"os"
	"path/filepath"

	"github.com/sarielhp/clihelp"
)

func p(name, desc string) clihelp.Param      { return clihelp.Param{Name: name, Description: desc} }
func opt(flags, desc string) clihelp.Option  { return clihelp.Option{Flags: flags, Description: desc} }
func ex(line string) clihelp.Example         { return clihelp.Example{Line: line} }
func note(heading, text string) clihelp.Note { return clihelp.Note{Heading: heading, Text: text} }

func buildScan() clihelp.Command {
	return clihelp.Command{
		Name:        "scan",
		Title:       "scan",
		Description: "Scan all folders starting with the given label prefix (case-insensitive) for spam.",
		UsageLine:   "mail_cli scan <lbl_prefix> [flags]",
		Parameters: []clihelp.Param{
			p("<lbl_prefix>", "The prefix of the label/folder to scan (e.g. 'inbox' or 'receipts')."),
		},
		Options: []clihelp.Option{
			opt("-m, --move <From>", "Move identified spam emails to Spam folder. Optional: specify From address to move a single unique message."),
			opt("--inbox-move <From>", "Move identified emails from a specific From address back to the Inbox folder."),
			opt("-p, --pattern <pattern>", "Only process messages whose subject contains this pattern."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli scan inbox"),
			ex("mail_cli scan inbox -m"),
			ex("mail_cli scan receipts -m spammer@example.com"),
		},
	}
}

func buildTest() clihelp.Command {
	return clihelp.Command{
		Name:        "test",
		Title:       "test",
		Description: "Run system and integration self-tests to verify API credentials and mail flow.",
		UsageLine:   "mail_cli test run",
		SubcommandEntries: []clihelp.Param{
			p("run", "Execute connection and integration tests."),
		},
		Examples: []clihelp.Example{ex("mail_cli test run")},
	}
}

func buildUnspam() clihelp.Command {
	return clihelp.Command{
		Name:        "unspam",
		Title:       "unspam",
		Description: "Mark a message as not being spam: train bogofilter as ham and move it from Spam back to Inbox on the server.",
		UsageLine:   "mail_cli unspam <message_id...>\n  mail_cli unspam folder <folder_name>",
		SubcommandEntries: []clihelp.Param{
			p("folder <folder_name>", "Mark all messages in the specified folder as ham and move them back to Inbox."),
		},
		Parameters: []clihelp.Param{
			p("<message_id...>", "One or more message IDs to unspam (short 8-char or full)."),
		},
		Examples: []clihelp.Example{ex("mail_cli unspam abc123de"), ex("mail_cli unspam folder Spam")},
	}
}

func buildArchive() clihelp.Command {
	return clihelp.Command{
		Name:        "archive",
		Aliases:     []string{"arc"},
		Title:       "archive (alias: arc)",
		Description: "Move message(s) by ID from their current folder to the Archive or Received folder. Or archive all messages in Inbox (default) or the specified label (by prefix).",
		UsageLine:   "mail_cli archive <all [label] | message-id...>",
		Parameters: []clihelp.Param{
			p("all [label]", "Archive all emails in the Inbox or specified label prefix."),
			p("<message-id...>", "One or more message IDs to archive (short 8-char or full)."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli archive abc123de"),
			ex("mail_cli archive all"),
			ex("mail_cli archive all receipts"),
		},
	}
}

func buildLearnHam() clihelp.Command {
	return clihelp.Command{
		Name:        "learn-ham",
		Aliases:     []string{"learn_ham"},
		Title:       "learn-ham (alias: learn_ham)",
		Description: "Train Bogofilter on ham (non-spam) emails in a folder. The folder must be an exact match and cannot have subfolders.",
		UsageLine:   "mail_cli learn-ham <label> [flags]",
		Parameters:  []clihelp.Param{p("<label>", "The folder name containing ham emails to train on.")},
		Options:     []clihelp.Option{opt("--force", "Bypass trained message database and re-train all emails.")},
		Examples:    []clihelp.Example{ex("mail_cli learn-ham receipts --force")},
	}
}

func buildCache() clihelp.Command {
	return clihelp.Command{
		Name:        "cache",
		Title:       "cache",
		Description: "Manage the local email download cache.",
		UsageLine:   "mail_cli cache <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("prune [days]", "Prune cached emails and scores older than [days] (default: 30)."),
			p("reset", "Reset per-account cache — removes all cached emails, scores, labels, and indexes for the current account."),
		},
		Options: []clihelp.Option{
			opt("--wipe", "Wipe the entire cache (equivalent to prune with 0 days)."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli cache prune"),
			ex("mail_cli cache prune 15"),
			ex("mail_cli cache prune --wipe"),
			ex("mail_cli cache reset"),
		},
		Subcommands: []clihelp.Command{{
			Name:        "prune",
			Title:       "cache prune [days]",
			Description: "Prune cached emails and scores older than a certain number of days.",
			UsageLine:   "mail_cli cache prune [days] [--wipe]",
			Parameters:  []clihelp.Param{p("[days]", "Number of days (default: 30).")},
			Options:     []clihelp.Option{opt("--wipe", "Wipe the entire cache immediately.")},
			Examples:    []clihelp.Example{ex("mail_cli cache prune 7")},
		}, {
			Name:        "reset",
			Title:       "cache reset",
			Description: "Reset the per-account cache directory, removing all cached data and recreating it empty.",
			UsageLine:   "mail_cli cache reset",
			Examples:    []clihelp.Example{ex("mail_cli cache reset")},
		}},
	}
}

func buildWhitelist() clihelp.Command {
	return clihelp.Command{
		Name:        "whitelist",
		Aliases:     []string{"wlist"},
		Title:       "whitelist (alias: wlist)",
		Description: "Manage the personal sender whitelist to bypass spam checks.",
		UsageLine:   "mail_cli whitelist <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("add <email>", "Add an email address to the whitelist."),
			p("del <email>", "Remove an email address from the whitelist."),
			p("list", "List all whitelisted email addresses."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli whitelist list"),
			ex("mail_cli whitelist add friend@example.com"),
		},
		Subcommands: []clihelp.Command{{
			Name:        "add",
			Title:       "whitelist add <email>",
			Description: "Add a sender email address to your personal whitelist. Senders on the whitelist bypass all language, script, and spam filters.",
			UsageLine:   "mail_cli whitelist add <email>",
			Parameters:  []clihelp.Param{p("<email>", "The sender email address to whitelist (e.g. mom@gmail.com).")},
			Examples:    []clihelp.Example{ex("mail_cli whitelist add mom@gmail.com")},
		}, {
			Name:        "del",
			Title:       "whitelist del <email>",
			Description: "Remove a sender email address from your personal whitelist.",
			UsageLine:   "mail_cli whitelist del <email>",
			Parameters:  []clihelp.Param{p("<email>", "The whitelisted email address to remove.")},
			Examples:    []clihelp.Example{ex("mail_cli whitelist del mom@gmail.com")},
		}},
	}
}

func buildBlacklist() clihelp.Command {
	return clihelp.Command{
		Name:        "blacklist",
		Aliases:     []string{"blist"},
		Title:       "blacklist (alias: blist)",
		Description: "Manage the personal sender blacklist to instantly classify messages as spam.",
		UsageLine:   "mail_cli blacklist <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("add <email>", "Add an email address to the blacklist."),
			p("del <email>", "Remove an email address from the blacklist."),
			p("list", "List all blacklisted email addresses."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli blacklist list"),
			ex("mail_cli blacklist add spammer@example.com"),
		},
		Subcommands: []clihelp.Command{{
			Name:        "add",
			Title:       "blacklist add <email>",
			Description: "Add a sender email address to your personal blacklist. Senders on the blacklist are immediately marked as spam without querying Bogofilter.",
			UsageLine:   "mail_cli blacklist add <email>",
			Parameters:  []clihelp.Param{p("<email>", "The sender email address to blacklist (e.g. spammer@gmail.com).")},
			Examples:    []clihelp.Example{ex("mail_cli blacklist add spammer@gmail.com")},
		}, {
			Name:        "del",
			Title:       "blacklist del <email>",
			Description: "Remove a sender email address from your personal blacklist.",
			UsageLine:   "mail_cli blacklist del <email>",
			Parameters:  []clihelp.Param{p("<email>", "The blacklisted email address to remove.")},
			Examples:    []clihelp.Example{ex("mail_cli blacklist del spammer@gmail.com")},
		}},
	}
}

func buildRule() clihelp.Command {
	return clihelp.Command{
		Name:        "rule",
		Title:       "rule",
		Description: "Manage auto-labeling rules for matching senders or subject prefixes.",
		UsageLine:   "mail_cli rule <subcommand> [args...]\nmail_cli rule -export <file>\nmail_cli rule -import <file>",
		SubcommandEntries: []clihelp.Param{
			p("add <email> <lbl>", "Add an auto-labeling rule by sender."),
			p("add_by_title <title> <lbl>", "Add an auto-labeling rule by subject prefix."),
			p("add_domain <msg_id> [lbl]", "Add an auto-labeling rule for all emails from a sender's domain."),
			p("del <email|title>", "Remove an auto-labeling rule."),
			p("list [-a, --all]", "List custom routing rules."),
			p("update", "Sync rules from blacklisted senders."),
			p("export [force]", "Export local rules to mail server filters."),
			p("export --sieve <f>", "Export rules as a Sieve script file."),
		},
		Options: []clihelp.Option{
			opt("-export <file>", "Export all existing rules to a JSON file."),
			opt("-import <file>", "Import rules from a JSON file, ignoring duplicates."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli rule list"),
			ex("mail_cli rule list --all"),
			ex("mail_cli rule export"),
			ex("mail_cli rule export force"),
			ex("mail_cli rule -export rules.json"),
			ex("mail_cli rule -import rules.json"),
			ex(`mail_cli rule add billing@netflix.com "Sort/Services/Netflix"`),
			ex(`mail_cli rule add_by_title "GitHub" "Sort/GitHub"`),
			ex(`mail_cli rule add_domain 12345 "Sort/Newsletters"`),
		},
		Subcommands: []clihelp.Command{
			{
				Name:        "add",
				Title:       "rule add <email> <lbl>",
				Description: "Add an auto-labeling rule by sender. Emails from the specified sender address will automatically be labeled with the target label and archived (the \"received\" label will be removed).",
				UsageLine:   "mail_cli rule add <email> <lbl>",
				Parameters: []clihelp.Param{
					p("<email>", "The sender email address (e.g. newsletter@example.com)."),
					p("<lbl>", "The target label hierarchy (e.g. \"Sort/Newsletters\")."),
				},
				Examples: []clihelp.Example{ex(`mail_cli rule add newsletter@example.com "Sort/Newsletters"`)},
			},
			{
				Name:        "add_domain",
				Title:       "rule add_domain <message_id> [lbl]",
				Description: "Add an auto-labeling rule for all emails from the sender's domain. Extracts the domain of the sender of the specified cached email and creates a rule to auto-label all emails from that domain.",
				UsageLine:   "mail_cli rule add_domain <message_id> [lbl]",
				Parameters: []clihelp.Param{
					p("<message_id>", "The message ID or short ID of the email."),
					p("[lbl]", "The target label hierarchy (optional; defaults to message folder or SpamLearn folder)."),
				},
				Examples: []clihelp.Example{ex(`mail_cli rule add_domain 12345 "Sort/Newsletters"`), ex(`mail_cli rule add_domain 12345`)},
			},
			{
				Name:        "add_by_title",
				Title:       "rule add_by_title <title> <lbl>",
				Description: "Add an auto-labeling rule by subject prefix. Emails with subjects starting with the specified title prefix will automatically be labeled with the target label and archived (the \"received\" label will be removed).",
				UsageLine:   "mail_cli rule add_by_title <title> <lbl>",
				Parameters: []clihelp.Param{
					p("<title>", "The subject prefix to match (e.g. \"[Alert]\")."),
					p("<lbl>", "The target label hierarchy (e.g. \"Sort/Alerts\")."),
				},
				Examples: []clihelp.Example{ex(`mail_cli rule add_by_title "[Alert]" "Sort/Alerts"`)},
			},
			{
				Name:        "del",
				Title:       "rule del <email|title>",
				Description: "Remove an auto-labeling rule for a sender email address or subject prefix.",
				UsageLine:   "mail_cli rule del <email|title>",
				Parameters:  []clihelp.Param{p("<email|title>", "The sender email address or subject prefix of the rule to remove.")},
				Examples:    []clihelp.Example{ex(`mail_cli rule del newsletter@example.com`), ex(`mail_cli rule del "[Alert]"`)},
			},
			{
				Name:        "export",
				Title:       "rule export [force]",
				Description: "Export local auto-labeling rules from config.json to mail server filters. If the 'force' keyword is supplied, conflicting remote filters are overwritten. For JMAP accounts (e.g. FastMail), server-side filters are not supported; use '--sieve <file>' to export as a Sieve script.",
				UsageLine:   "mail_cli rule export [force]\nmail_cli rule export --sieve <path>",
				Examples: []clihelp.Example{
					ex("mail_cli rule export"),
					ex("mail_cli rule export force"),
					ex("mail_cli rule export --sieve rules.sieve"),
				},
			},
			{
				Name:        "list",
				Title:       "rule list [-a, --all]",
				Description: "List custom routing and auto-labeling rules for the selected account.",
				UsageLine:   "mail_cli rule list [-a, --all]",
				Options:     []clihelp.Option{opt("-a, --all", "List all custom routing rules, including those already exported to server filters.")},
				Examples:    []clihelp.Example{ex("mail_cli rule list"), ex("mail_cli rule list --all")},
			},
			{
				Name:        "delete_all",
				Title:       "rule delete_all",
				Description: "Delete all custom routing rules for the selected account.",
				UsageLine:   "mail_cli rule delete_all",
				Examples:    []clihelp.Example{ex("mail_cli rule delete_all")},
			},
			{
				Name:        "update",
				Title:       "rule update",
				Description: "Ensure all blacklisted senders have a corresponding local auto-labeling rule pointing to the SpamLearn folder.",
				UsageLine:   "mail_cli rule update",
				Examples:    []clihelp.Example{ex("mail_cli rule update")},
			},
		},
	}
}

func buildLabels() clihelp.Command {
	return clihelp.Command{
		Name:        "labels",
		Title:       "labels",
		Description: "Manage and organize folders/labels.",
		UsageLine:   "mail_cli labels <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("list [-a, --all]", "List labels/folders."),
			p("create <lbl>", "Create a new label."),
			p("print", "Print all labels/folders, one per line (full path only)."),
			p("rename <old> <new>", "Rename a label and move all its emails."),
			p("fix", "Fix nested folder parent hierarchies."),
			p("del <lbl>", "Delete a label."),
			p("search <str>", "Search labels by substring (matches full path)."),
			p("cache", "Manage the labels cache."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli labels list"),
			ex("mail_cli labels list --all"),
			ex(`mail_cli labels create "Work/ProjectA"`),
			ex("mail_cli labels print"),
			ex(`mail_cli labels rename "sort-coop" "Sort/Services/Coop"`),
			ex("mail_cli labels search work"),
			ex("mail_cli labels cache update"),
		},
		Subcommands: []clihelp.Command{
			{
				Name:        "create",
				Title:       "labels create <lbl_name>",
				Description: "Create a new label on the server.",
				UsageLine:   "mail_cli labels create <lbl_name>",
				Parameters:  []clihelp.Param{p("<lbl_name>", "The fully specified name of the new label to create (e.g. \"Work/ProjectA\").")},
				Examples:    []clihelp.Example{ex(`mail_cli labels create "Work/ProjectA"`)},
			},
			{
				Name:        "print",
				Title:       "labels print",
				Description: "Print all labels/folders, one per line, with their full paths and no decorative layout or statistics.",
				UsageLine:   "mail_cli labels print",
				Examples:    []clihelp.Example{ex("mail_cli labels print")},
			},
			{
				Name:        "rename",
				Title:       "labels rename <old_name> <new_name>",
				Description: "Rename an existing label and move all corresponding emails.",
				UsageLine:   "mail_cli labels rename <old_name> <new_name>",
				Parameters: []clihelp.Param{
					p("<old_name>", "The current label name (e.g. \"sort-coop\")."),
					p("<new_name>", "The new label name (e.g. \"Sort/Services/Coop\")."),
				},
				Examples: []clihelp.Example{ex(`mail_cli labels rename "sort-coop" "Sort/Services/Coop"`)},
			},
			{
				Name:        "del",
				Title:       "labels del <lbl_name>",
				Description: "Delete an existing label by its name.",
				UsageLine:   "mail_cli labels del <lbl_name>",
				Parameters:  []clihelp.Param{p("<lbl_name>", "The name of the label to delete (e.g. \"temp-label\").")},
				Examples:    []clihelp.Example{ex(`mail_cli labels del "temp-label"`)},
			},
			{
				Name:        "search",
				Title:       "labels search <substring>",
				Description: "Search labels whose full path contains the given substring (case-insensitive). Uses the cached labels list; refreshes asynchronously if the cache is older than 24 hours.",
				UsageLine:   "mail_cli labels search <substring>",
				Parameters:  []clihelp.Param{p("<substring>", "Substring to search for in label paths.")},
				Examples:    []clihelp.Example{ex("mail_cli labels search work"), ex("mail_cli labels search sort")},
			},
			{
				Name:        "cache",
				Title:       "labels cache",
				Description: "Manage the labels cache used by the search subcommand.",
				UsageLine:   "mail_cli labels cache <subcommand>",
				SubcommandEntries: []clihelp.Param{
					p("update", "Update the labels cache from the server."),
				},
				Subcommands: []clihelp.Command{{
					Name:        "update",
					Title:       "labels cache update",
					Description: "Force an immediate update of the labels cache from the server.",
					UsageLine:   "mail_cli labels cache update",
					Examples:    []clihelp.Example{ex("mail_cli labels cache update")},
				}},
			},
		},
	}
}

func buildFilter() clihelp.Command {
	return clihelp.Command{
		Name:        "filter",
		Title:       "filter",
		Description: "Manage remote filters on Gmail.",
		UsageLine:   "mail_cli filter <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("list", "List all remote filters on Gmail with detailed action descriptions."),
		},
		Examples: []clihelp.Example{ex("mail_cli filter list")},
	}
}

func buildSpam() clihelp.Command {
	return clihelp.Command{
		Name:        "spam",
		Title:       "spam",
		Description: "Manage Spam folder, train filters, and unsubscribe from political mail.",
		UsageLine:   "mail_cli spam <subcommand> [args...]\nmail_cli spam <message_id...>           Mark one or more messages as spam by ID.",
		SubcommandEntries: []clihelp.Param{
			p("del", "Permanently purge all emails in the Spam folder."),
			p("pol audit", "Scan Spam folder for political fundraising emails and print heuristic scoring details."),
			p("pol unsub", "Scan the Spam folder for political messages, execute unsubscription opt-outs, and delete matching emails. NOTE: Unsubscribing from political mail is safe because PACs/campaigns are registered entities that respect opt-out requests. For regular spam, unsubscribing is unsafe as it confirms your email is active to malicious actors."),
			p("bye", "Execute a complete sweep: unsubscribe political spam, train the spam classifier on the remaining spam folder, and then permanently purge the spam folder."),
			p("learn [force]", "Spam Learning Mode: Connect to Spam folder and train local Bogofilter. If 'force' is specified, bypasses trained message database."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli spam del"),
			ex("mail_cli spam pol audit"),
			ex("mail_cli spam pol unsub"),
			ex("mail_cli spam bye"),
			ex("mail_cli spam learn"),
			ex("mail_cli spam learn force"),
			ex("mail_cli spam abc123de"),
		},
	}
}

func buildAccount() clihelp.Command {
	return clihelp.Command{
		Name:        "account",
		Title:       "account",
		Description: "Manage and list configured mail accounts.",
		UsageLine:   "mail_cli account <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("list", "List all configured mail accounts with status."),
			p("new <jmap|gmail|outlook> [name]", "Add a new JMAP, Gmail, or Outlook account template to config.json."),
			p("associate [account_name] <prog>", "Associate a program/symlink name with an account."),
			p("rename [old_name] [new_name]", "Rename an existing account and update cache/tokens."),
			p("delete <account_name>", "Delete an existing account and its credentials."),
			p("test [account_name]", "Test validation and server connection for an account."),
			p("calendar [account_name]", "Designate or show the calendar manager account."),
			p("login [account_name]", "Perform interactive OAuth login for a Gmail or Outlook account."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli account list"),
			ex("mail_cli account new outlook outlook-personal"),
			ex("mail_cli account login outlook-personal"),
			ex("mail_cli account test outlook-personal"),
			ex("mail_cli account delete outlook-personal"),
			ex("mail_cli account associate outlook-personal personal-mail"),
		},
	}
}

func buildShow() clihelp.Command {
	return clihelp.Command{
		Name:        "show",
		Title:       "show",
		Description: "Show the contents of emails in folders matching a label prefix, or show a specific email's details and body without running spam checks.",
		UsageLine:   "mail_cli show <lbl_prefix> [message_id] [flags]",
		Parameters: []clihelp.Param{
			p("<lbl_prefix>", "The prefix of the label/folder to view (e.g. 'inbox' or 'receipts')."),
			p("[message_id]", "Optional message ID (short 8-char or full) of a specific email to show."),
		},
		Options:  []clihelp.Option{opt("-w, --web", "Open the HTML body of the email in your configured browser.")},
		Examples: []clihelp.Example{ex("mail_cli show inbox"), ex("mail_cli show inbox abc123de"), ex("mail_cli show inbox abc123de -w")},
	}
}

func buildConfig() clihelp.Command {
	return clihelp.Command{
		Name:        "config",
		Title:       "config",
		Description: "Show or manage configuration options.",
		UsageLine:   "mail_cli config <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("show", "Show the current configuration (download directory, limits, accounts, and browser)."),
			p("set <key> <value>", "Set configuration parameters (spam_learn, unspam_learn, browser)."),
			p("reset <key>", "Reset configuration parameters to system default (browser)."),
			p("validate", "Validate configurations, account parameters, DNS reachability, and Bogofilter service."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli config show"),
			ex("mail_cli config set spam_learn [Gmail]/Spam"),
			ex("mail_cli config set browser brave-browser"),
			ex("mail_cli config reset browser"),
			ex("mail_cli config validate"),
		},
	}
}

func buildCalendar() clihelp.Command {
	return clihelp.Command{
		Name:        "calendar",
		Title:       "calendar",
		Description: "Manage calendar events extracted from email attachments.",
		UsageLine:   "mail_cli calendar <subcommand> [args...]",
		SubcommandEntries: []clihelp.Param{
			p("add [label_prefix] <message_id>", "Add a calendar event from an .ics attachment. Default prefix is 'inbox'."),
			p("week", "Show all events in the upcoming week in the default calendar."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli calendar add abc123de"),
			ex("mail_cli calendar add receipts xyz789gh"),
			ex("mail_cli calendar week"),
		},
	}
}

func buildCalAdd() clihelp.Command {
	return clihelp.Command{
		Name:        "caladd",
		Aliases:     []string{"calendar add-all"},
		Title:       "caladd (alias: calendar add-all)",
		Description: "Scan the inbox for messages containing .ics attachments, and add them to the calendar if they are not already present.",
		UsageLine:   "mail_cli caladd",
		Examples:    []clihelp.Example{ex("mail_cli caladd"), ex("mail_cli calendar add-all")},
	}
}

func buildSplice() clihelp.Command {
	return clihelp.Command{
		Name:        "splice",
		Title:       "splice",
		Description: "Move messages from a folder into the keep/YYYY/MM/<folder> structure. The root \"keep\" is fixed. Use -f to change the target folder name, or -F to change the target folder name and automatically suffix it with the year and month.",
		UsageLine:   "mail_cli splice <folder> [flags]",
		Options: []clihelp.Option{
			opt("-n, --n <int>", "Number of messages to process (default 10)."),
			opt("--folder, -f", "Folder name to use for the destination path without suffix."),
			opt("--folder-suffix, -F", "Folder name to use for the destination path with year/month suffix attached."),
			opt("--move", "Actually move the messages instead of dry run."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli splice research/cfps"),
			ex("mail_cli splice research/cfps -f archive (keep/YYYY/MM/archive)"),
			ex("mail_cli splice research/cfps -F wuna (keep/YYYY/MM/wuna-YYYY-MM)"),
			ex("mail_cli splice research/cfps -n 20 --move"),
		},
		Notes: []clihelp.Note{
			note("", "The destination folder/label is created on the server automatically if it does not exist."),
			note("When dry run only (no --move)", "messages are not moved - this shows where they would go."),
		},
	}
}

func buildTUI() clihelp.Command {
	return clihelp.Command{
		Name:        "tui",
		Title:       "tui [label_prefix]",
		Description: "Open the interactive terminal email browser. With an optional label_prefix argument, open the TUI with the matching label as the initial folder. The prefix is matched case-insensitively as a substring against the full label path. If exactly one label matches, the TUI opens on that label. If multiple match, all matching labels are printed and the program exits.",
		UsageLine:   "mail_cli tui [label_prefix]",
		Parameters:  []clihelp.Param{p("[label_prefix]", "Substring to match against label full paths. If exactly one label matches, the TUI opens on that label. If multiple match, all matches are printed and the program exits. If omitted, the TUI opens on INBOX.")},
		Examples:    []clihelp.Example{ex("mail_cli tui"), ex("mail_cli tui wuna"), ex("mail_cli tui work")},
	}
}

func buildSplit() clihelp.Command {
	return clihelp.Command{
		Name:        "split",
		Title:       "split <source_label> <pattern> <target_label>",
		Description: "Scan messages in the source label. If their subject matches the pattern (which may contain wildcards * and ?), move them to the target label. Runs in dry-run mode by default; use --do to perform actual operations.",
		UsageLine:   "mail_cli split <source_label> <pattern> <target_label> [flags]",
		Parameters: []clihelp.Param{
			p("<source_label>", "The name or prefix of the label containing messages to scan (must match a unique label)."),
			p("<pattern>", "A subject match pattern supporting * (matches any characters) and ? (matches any single character)."),
			p("<target_label>", "The name or prefix of the target label (must match a unique label and already exist)."),
		},
		Options:  []clihelp.Option{opt("--do", "Perform the actual move operations on the server instead of dry-run.")},
		Examples: []clihelp.Example{ex(`mail_cli split inbox "*invoice*" Work/Billing`), ex(`mail_cli split inbox "*urgent*" Urgent --do`)},
	}
}

func buildDownload() clihelp.Command {
	return clihelp.Command{
		Name:        "download",
		Title:       "download <label> <file_name>",
		Description: "Download all messages in the specified label (which must match a unique label) to a local mbox file.",
		UsageLine:   "mail_cli download <label> <file_name>",
		Parameters: []clihelp.Param{
			p("<label>", "The name or prefix of the label containing messages to download (must match a unique label)."),
			p("<file_name>", "Path to the destination local mbox file (e.g. archive.mbox)."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli download inbox my_inbox.mbox"),
			ex("mail_cli download Work/ProjectA project_a.mbox"),
		},
	}
}

func buildUpload() clihelp.Command {
	return clihelp.Command{
		Name:        "upload",
		Title:       "upload <label> <file_name>",
		Description: "Upload all email messages from a local mbox file to the specified target label/folder on the server.",
		UsageLine:   "mail_cli upload <label> <file_name>",
		Parameters: []clihelp.Param{
			p("<label>", "The name or prefix of the target label/folder to upload emails to (must match a unique label)."),
			p("<file_name>", "Path to the local mbox file containing emails to upload."),
		},
		Examples: []clihelp.Example{
			ex("mail_cli upload archive archive.mbox"),
			ex("mail_cli upload Work/ProjectA project_a.mbox"),
		},
	}
}

// buildApp reconstructs the mail_cli usage tree and global overview as a clihelp App.
func buildApp() *clihelp.App {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "mail_cli")

	colorCmd := clihelp.Command{Name: "color", Description: "Test terminal 24-bit true-color and 256-color support"}
	migrateCmd := clihelp.Command{Name: "migrate", Description: "Copy configuration and credentials to a remote machine via SSH/SCP"}

	return &clihelp.App{
		Name:       "mail_cli",
		Version:    "0.5.4",
		ConfigPath: filepath.Join(configDir, "config.json"),
		Shortcuts: []clihelp.Command{
			{Name: "ss", Description: "Shortcut alias for: scan spam"},
			{Name: "sb", Description: "Shortcut alias for: spam bye"},
		},
		GlobalFlags: []clihelp.Option{
			opt("-v, --verbose", "Enable verbose diagnostic log output"),
			opt("-A, --account", "Specify target account name from config.json"),
			opt("-1, -2, -3...", "Shorthand flag to select configured accounts"),
		},
		Commands: []clihelp.Command{
			buildAccount(),
			buildArchive(),
			buildBlacklist(),
			buildCache(),
			buildCalAdd(),
			buildCalendar(),
			colorCmd,
			buildConfig(),
			buildDownload(),
			buildFilter(),
			buildLabels(),
			buildLearnHam(),
			migrateCmd,
			buildRule(),
			buildScan(),
			buildShow(),
			buildSpam(),
			buildSplice(),
			buildSplit(),
			buildTest(),
			buildTUI(),
			buildUnspam(),
			buildUpload(),
			buildWhitelist(),
		},
	}
}

// detailedPaths are the command paths (relative to the app) that have a
// dedicated detailed usage page. The first element is the command and the
// optional second element the subcommand.
func detailedPaths(app *clihelp.App) [][]string {
	var paths [][]string
	var walk func(cmds []clihelp.Command, prefix []string)
	walk = func(cmds []clihelp.Command, prefix []string) {
		for i := range cmds {
			c := &cmds[i]
			if c.Title != "" || len(c.Subcommands) > 0 {
				paths = append(paths, append(append([]string{}, prefix...), c.Name))
			}
			walk(c.Subcommands, append(append([]string{}, prefix...), c.Name))
		}
	}
	walk(app.Commands, nil)
	return paths
}
