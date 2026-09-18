---
title: 'podctl completion'
has_children: true
---

# podctl completion

Generate or install shell tab-completion scripts

## Usage

```
completion <subcommand>
```

## Subcommands

| Command | Description |
|---------|-------------|
| [bash](completion-bash.md) | Generate Bash tab-completion script |
| [zsh](completion-zsh.md) | Generate Zsh tab-completion script |
| [fish](completion-fish.md) | Generate Fish tab-completion script |
| [keys](completion-keys.md) | Print shell key bindings (Alt-H expands the command line and explains it) |
| [install](completion-install.md) | Set this program up: tab completion, the Alt-H key binding and the manual page |
| [uninstall](completion-uninstall.md) | Remove the installed tab completion and key binding |

## Flags

| Flag | Description |
|------|-------------|
| `--token TOKEN` | Bearer token for cluster authentication |
| `--api-key KEY` | API key for cloud provider access |
| `-c, --config PATH` | Path to configuration file (default: ~/.config/podctl.yaml) |
| `--endpoint URL` | API service endpoint URL (default: https://api.podctl.example.com) |
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `--no-color` | Disable ANSI color output |

## Examples

- `completion zsh` — Generate Zsh tab-completion script
- `completion install` — Install tab-completions for the active shell

## Shell Tip

Tip: <Tab> to complete, Ctrl-D to list choices. Run 'completion keys' and source the result from your shell's rc file to bind Alt-H, which expands the command line and shows the help for the command it names.

---

[↑ podctl](index.md) — [nav](nav.md)
