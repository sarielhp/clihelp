---
title: 'podctl completion keys'
parent: 'podctl completion'
---

# podctl completion keys

Print shell key bindings (Alt-H expands the command line and explains it)

## Usage

```
completion keys [<shell>]
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

## Examples

- `completion keys bash` — Print the Bash key bindings

## Why This Is Separate

Every shell loads a completion script lazily, on the first completion of the command, so a key binding written there would not exist until <Tab> had already been pressed once. Source this from your shell's rc file instead; the first line of the output says how.

---

[↑ podctl](index.md) — [nav](nav.md)
