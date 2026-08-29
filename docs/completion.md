# Shell Autocompletion

`clihelp` provides built-in shell autocompletion for **Bash**, **Zsh**, and **Fish** through its native `__complete` protocol, ready-to-mount [`CompletionCommand`](#zero-boilerplate-completioncommand), and XDG-compliant [`InstallCompletion`](#automatic-self-installation-installcompletion).

---

## Table of Contents

- [Overview & Architecture](#overview--architecture)
- [Zero-Boilerplate `CompletionCommand`](#zero-boilerplate-completioncommand)
- [Automatic Self-Installation (`InstallCompletion`)](#automatic-self-installation-installcompletion)
- [Manual Shell Script Generation](#manual-shell-script-generation)
- [Dynamic Completion Callbacks](#dynamic-completion-callbacks)
- [Testing Shell Completions](#testing-shell-completions)

---

## Overview & Architecture

When shell completion functions execute, they invoke the target application with `__complete` and the current command-line tokens:

```bash
$ podctl __complete build --
--output	Write output to PATH
--bitrate	Target audio bitrate in kbps
--normalize	Apply LUFS loudness normalization
```

`clihelp` inspects the active command path, resolves available subcommands and flags, and returns tab-delimited suggestions.

---

## Zero-Boilerplate `CompletionCommand`

The fastest way to expose completion in your CLI is using `clihelp.CompletionCommand()`. It creates a standard `Command` with `bash`, `zsh`, `fish`, and `install` subcommands:

```go
app := &clihelp.App{
    Name:        "podctl",
    Description: "Podcast distribution & audio processing tool",
    Commands: []clihelp.Command{
        // Application commands...
        clihelp.CompletionCommand(),
    },
}
```

This immediately equips your CLI with:
- `podctl completion bash` — outputs Bash completion script to stdout
- `podctl completion zsh` — outputs Zsh completion script to stdout
- `podctl completion fish` — outputs Fish completion script to stdout
- `podctl completion install [<shell>]` — installs completion directly to standard user directories

---

## Automatic Self-Installation (`InstallCompletion`)

`clihelp.InstallCompletion(app, shell)` installs completion scripts into standard non-root XDG user directories:

| Shell | Target User Directory | Target Filename |
| :--- | :--- | :--- |
| **Bash** | `${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions` | `<app-name>` |
| **Zsh** | `${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions` | `_<app-name>` |
| **Fish** | `${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions` | `<app-name>.fish` |

### CLI Usage:

```bash
# 1. Automatic detection (detects active shell from $SHELL):
podctl completion install

# 2. Explicit shell target:
podctl completion install bash
podctl completion install zsh
podctl completion install fish
```

### Go API:

```go
// Install for active shell (detected via $SHELL)
installedPath, err := clihelp.InstallCompletion(app, "")

// Install for specific shell
installedPath, err := clihelp.InstallCompletion(app, "zsh")
```

---

## Manual Shell Script Generation

### Bash (`clihelp.GenBashCompletion`)

Writes a bash programmable completion script. Includes a fallback mechanism so it operates properly even if the optional `bash-completion` package (`_init_completion`) is not pre-installed on the host.

```bash
# Load in current session:
source <(podctl completion bash)

# Install permanently (Linux):
podctl completion bash | sudo tee /etc/bash_completion.d/podctl
```

### Zsh (`clihelp.GenZshCompletion`)

Writes a `#compdef` script compatible with Zsh's `compinit` completion system:

```bash
# Load in current session:
source <(podctl completion zsh)

# Install permanently:
podctl completion zsh > "${fpath[1]}/_podctl"
```

### Fish (`clihelp.GenFishCompletion`)

Writes a Fish completion script utilizing `complete -c`:

```bash
# Load in current session:
podctl completion fish | source

# Install permanently:
podctl completion fish > ~/.config/fish/completions/podctl.fish
```

---

## Dynamic Completion Callbacks

Attach a `Complete` callback to any `Option` to provide dynamic, contextual suggestions:

```go
clihelp.Option{
    Flags:       "-p, --podcast <id>",
    Description: "Target podcast ID",
    Complete: func(toComplete string) []string {
        podcasts := []string{"pod1\tTech News", "pod2\tHistory"}
        var results []string
        for _, p := range podcasts {
            if strings.HasPrefix(p, toComplete) {
                results = append(results, p)
            }
        }
        return results
    },
}
```

---

## Testing Shell Completions

`clihelp` includes live integration tests that compile binaries and execute them inside live `bash`, `zsh`, and `fish` subshells. See `completion_test.go` for examples of asserting completion behavior in end-to-end test suites.
