---
title: 'podctl completion install'
parent: 'podctl completion'
---

# podctl completion install

Set this program up: tab completion, the Alt-H key binding and the manual page

## Usage

```
completion install [--no-keys] [--no-man] [<shell>]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `[<shell>]` | Shell type ('bash', 'zsh', or 'fish'; defaults to current shell) |

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
| `--no-keys` | Install tab completion only, leaving Alt-H alone |
| `--no-man` | Skip the manual page |

## Examples

- `completion install` — Set up the active shell
- `completion install --no-keys zsh` — Set up Zsh completion without the Alt-H binding

## What It Writes

One generated file under this application's configuration directory, one permanent line in the shell's startup file that sources it, and a manual page under the user's data directory. The line never changes; the generated file is rewritten whenever the application is upgraded. On fish nothing shared is touched at all, because conf.d is a drop-in directory. Run 'completion uninstall' to remove all of it.

---

[↑ podctl](index.md) — [nav](nav.md)
