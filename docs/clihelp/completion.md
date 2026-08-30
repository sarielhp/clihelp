---
title: 'podctl completion'
has_children: true
---

# podctl completion

Generate or install shell tab-completion scripts

## Usage

```
<command> completion <bash|zsh|fish|install> [options]
```

## Subcommands

| Command | Description |
|---------|-------------|
| [bash](completion-bash.md) | Generate Bash tab-completion script |
| [zsh](completion-zsh.md) | Generate Zsh tab-completion script |
| [fish](completion-fish.md) | Generate Fish tab-completion script |
| install \[\<shell>\] | Install tab-completion script to standard user directory |

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

---

[↑ podctl](index.md) — [nav](nav.md)
