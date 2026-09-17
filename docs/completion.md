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

## `__clihelp`: Setup Without the Author's Opt-In

`clihelp` reserves three argument names. The first two are the protocol the generated scripts call; the third is for setup:

| Name | Called by | Purpose |
|---|---|---|
| `__complete` | the completion scripts | completion candidates |
| `__explain` | the Alt-H key bindings | expanded command line, then its help |
| `__clihelp` | a human or a setup script | the verbs below |

They are hidden, not secret: absent from help and completion output because nobody needs them in the way, documented here, and none of them acts unless invoked.

```console
$ myapp __clihelp
myapp __clihelp — shell integration for this program, built with clihelp 0.3.11

  __clihelp version                     report the clihelp version this program was built with
  __clihelp install [<shell>]           install the completion script, printing its path
  __clihelp keys [<shell>]              print the key bindings to source from a shell startup file
  __clihelp wrapper <name> [<args>...]  print a wrapper script for this program, with arguments
```

**Why this exists alongside the `completion` command.** `ExecuteContext` serves `__complete` before it ever looks at the command tree, so *every* clihelp program can complete — but only a program whose author added `clihelp.CompletionCommand()` could be *asked* to install that completion. `__clihelp` closes the gap, which matters most for the people who are not the author: a dotfiles script, or a packager, can set up any clihelp program uniformly:

```bash
for bin in ~/.local/bin/*; do
    "$bin" __clihelp version >/dev/null 2>&1 && "$bin" __clihelp install
done
```

`install` prints the installed path on stdout and nothing else, so it can be captured; notes for humans go to stderr.

---

## Wrapper Scripts and Aliases

A wrapper script — `pd` running `myapp deploy "$@"` — is opaque to every shell, so completion for it has to be arranged. Shell *aliases* mostly do not: fish turns `alias pd='myapp deploy'` into a `--wraps` function and zsh expands aliases before completing, so both already work. Bash is the exception.

`__clihelp wrapper` writes a wrapper that answers clihelp's protocol on behalf of the program it wraps, so completion and Alt-H keep working through it:

```console
$ myapp __clihelp wrapper pd deploy > ~/.local/bin/pd && chmod +x ~/.local/bin/pd

# Put pd somewhere on your PATH, then register it with your shell,
# after the completion script for myapp has been loaded:
#   complete -F _myapp_complete pd
```

The script carries a `# clihelp-wraps: myapp deploy` marker in its second line — the convention pyenv and asdf use for their shims, so that a wrapper can be recognised and followed without being executed.

Two things worth knowing:

- **The registration line is unavoidable.** No shell calls a completion function for a name it was never told about; git ships `__git_complete` for exactly this reason. In fish, `complete -c pd --wraps 'myapp deploy'` does the whole job on its own.
- **Alt-H does not expand a wrapper.** The wrapper answers `__explain` with the line as typed, then the help for the wrapped command: rewriting `pd prod` into `myapp deploy prod` would replace something deliberately typed short.

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
