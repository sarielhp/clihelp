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

### Shell Detection

Detection reads `$SHELL` and names what it finds. A shell `clihelp` has no script
for — dash, ksh, nushell — is reported as an error rather than treated as Bash,
and `App.AutoInstallCompletion` installs nothing for it: a Bash script in a ksh
user's home is one their shell cannot read and nothing ever removes. With `$SHELL`
unset, pass the shell name explicitly.

The script is generated in full before anything is written, and the file is
renamed into place, so an interrupted install cannot replace a working script
with half of one.

---

## Alt-H: Expand and Explain (`completion keys`)

`completion keys [<shell>]` prints a snippet that binds **Alt-H**. Pressing it does two things:

1. **Expands the command line.** With `App.AbbrevCommands`, `podctl b d` becomes `podctl build deploy` on the prompt — abbreviated command names are replaced by their full names, and everything else (flags, arguments, quoting, spacing, pipes, redirections) is left exactly as typed. An ambiguous abbreviation stops the expansion there rather than guessing.
2. **Prints the help for the command it names**, capped at **two thirds of the terminal height** so the command being explained stays on screen. What does not fit is replaced by one line saying how much was cut and which `help` command prints the rest.

```bash
# ~/.bashrc
eval "$(podctl completion keys bash)"

# ~/.zshrc
eval "$(podctl completion keys zsh)"

# ~/.config/fish/config.fish
podctl completion keys fish | source
```

This is deliberately **not** part of the completion script: every shell loads that lazily, on the first completion of the command, so a binding written there would not exist until `<Tab>` had already been pressed once.

### What the binding touches

- **Alt-H is the shell's own convention for this.** zsh binds it to `run-help` and fish binds it (and F1) to `__fish_man_page` — both meaning "explain the command I am typing". clihelp fills that in for programs that ship no man page, and hands the key straight back to `run-help` / `__fish_man_page` for any command line that is not a clihelp program's. In bash, Alt-H is unbound by default.
- **Several clihelp programs share one dispatcher.** A key binding is global to the shell, so each program registers its name in a shared list and the first snippet loaded installs the binding. Without that, the last program installed would own Alt-H and refuse every other program's command line. The dispatcher is versioned, so two programs built against different clihelp releases settle on the newer one rather than fighting.
- The shell passes its own `$LINES` and `$COLUMNS` through `CLIHELP_TERM_LINES` / `CLIHELP_TERM_COLUMNS`, because the binding captures the program's stdout and a pipe has no size to measure.
- The underlying protocol call is `<app> __explain "<command line>"`, whose first output line is always the expanded command line. `App.Explain` is exported if you want to drive it yourself.

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

Writes a `#compdef` script compatible with Zsh's `compinit` completion system. The script works both ways: autoloaded from `$fpath`, where it *is* the completion function, and sourced from a startup file, where it registers itself with `compdef` instead. (Before 0.3.13 the sourced form printed `can only be called from completion function` at every shell start.)

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
