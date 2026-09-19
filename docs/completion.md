# Shell Autocompletion

`clihelp` provides built-in shell autocompletion for **Bash**, **Zsh**, and **Fish** through its native `__complete` protocol and a ready-to-mount [`CompletionCommand`](#zero-boilerplate-completioncommand). Installing is something the *user* does, by running one command; it is not an API a program calls on their behalf.

---

## Table of Contents

- [Overview & Architecture](#overview--architecture)
- [Zero-Boilerplate `CompletionCommand`](#zero-boilerplate-completioncommand)
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

The fastest way to expose completion in your CLI is using `clihelp.CompletionCommand()`. It creates a standard `Command` with `bash`, `zsh`, `fish`, `keys`, `install` and `uninstall` subcommands:

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

> **`App.Name` becomes a file name and a shell symbol.** Anything that installs or
> generates shell integration requires it to be letters, digits, `-`, `_`, `.` or `+`.
> A name with a space or a shell metacharacter is refused rather than turned into a
> broken — or dangerous — script, and a program with no `Name` cannot install at all.

## Installation: One Command

```console
$ podctl completion install
✓ bash shell integration installed
    generated:  ~/.config/podctl/shell/bash
    added the line that sources it to ~/.bashrc
Restart your shell to activate it: <Tab> completes, Alt-H explains.
Run 'podctl completion uninstall' to undo all of this.
```

That is the whole setup. It writes **one generated file** and **one permanent line**:

| shell | generated file | startup file |
| :--- | :--- | :--- |
| **bash** | `~/.config/<app>/shell/bash` | a marked block in `~/.bashrc` |
| **zsh** | `~/.config/<app>/shell/zsh` | a marked block in `$ZDOTDIR/.zshrc` or `~/.zshrc` |
| **fish** | `~/.config/<app>/shell/fish` | `~/.config/fish/conf.d/<app>.fish` — a drop-in directory, so no shared file is edited. The drop-in *file* is clihelp's; if something else already owns that name, installation refuses rather than overwriting it. |

The generated file holds both halves — the completion registration *and* the Alt-H key binding — because a key binding has to exist before the key is pressed, while every shell loads a completion script lazily, on the first `<Tab>` for that command. Once a startup file has to source something anyway, there is no reason left for two artifacts, and one directory owned by the application means one uninstall and no zsh `$fpath` juggling.

**Nothing runs at shell startup.** The startup line is a file test and a `source` — no command substitution, no `eval`, nothing that invokes the program. The line names a fixed path and never changes again; upgrades rewrite the file it points at, not your config.

**One more file.** Editing a startup file is a read-modify-write, and two programs
installing at once contend for it, so the edit is serialized with an advisory lock on a
companion file — `~/.bashrc.clihelp-lock`, empty and mode 0600. It is deliberately never
deleted: `flock` is held on an inode rather than a name, so removing the file lets a waiter
and a newcomer end up holding two different inodes and believing they hold the same lock.
Measured with the file being deleted, 478 of 480 serialized updates were lost. fish needs no
lock and gets none, because `conf.d` is a drop-in directory and nothing shared is edited.

**Keeping it current.** The generated file carries a `clihelp-integration-version` marker. When the application is upgraded and its clihelp templates change, the next run of the program rewrites the file. `App.AutoRefreshIntegration` refreshes what is already installed — the generated file, and a completion script at the older XDG location if clihelp wrote it — and *never* creates a file or edits a startup file on its own. The flag is the author's choice; the files are in the user's home, so nothing there appears because someone ran an unrelated command. Installing is what `install` is for.

**Options.**

```bash
podctl completion install --no-keys      # tab completion only, leave Alt-H alone
podctl completion install --no-man       # skip the manual page
podctl completion install zsh            # a shell other than the active one
podctl completion uninstall              # remove the file and the block
```

The block is marked with `# >>> <app> shell integration (clihelp) >>>`, so `uninstall` removes exactly it and leaves the rest of your startup file byte for byte as it was.

**Superseded installs.** A completion script this library installed into the shell's own directory (`~/.local/share/bash-completion/completions/<app>` and friends) is removed when the integration is installed, so no shell loads two copies. A file clihelp never wrote is left alone.

---

## Lower-Level Entry Points

Installing a completion script into the shell's own directory is still supported, and still happens when an older install is found there — but only through the setup command. It is no longer something a program can do to a user's home directory from inside an ordinary run, because nothing supervises that: the rule that the unattended path may only refresh what already exists lives in one place, and a second entry point would go around it.

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
- **Several clihelp programs share one dispatcher.** A key binding is global to the shell, so each program registers its name in a shared list and the first snippet loaded installs the binding. Without that, the last program installed would own Alt-H and refuse every other program's command line. The dispatcher is versioned, so two programs built against different clihelp releases settle on the newer one rather than fighting, and each registry entry is `name:protocol`, so the dispatcher knows what the program on the other end speaks before it calls it.
- **Every keymap is bound**, not just the one that happens to be current when the snippet is sourced: `emacs-standard`, `vi-insert` and `vi-command` in bash, `emacs`, `viins` and `vicmd` in zsh, `default` and `insert` in fish. Otherwise `set -o vi` would silently leave the key dead.
- **`CLIHELP_NO_KEY_BINDINGS` declines the key.** Set it before your shell sources the integration file and nothing is bound — zsh's `run-help` and fish's man-page binding stay exactly as they were — while tab completion is unaffected. There is no portable way to ask readline what `\eh` is already bound to, so this is an opt-out rather than a check.
- The shell passes its own `$LINES` and `$COLUMNS` through `CLIHELP_TERM_LINES` / `CLIHELP_TERM_COLUMNS`, because the binding captures the program's stdout and a pipe has no size to measure.
- The underlying protocol call is `<app> __explain "<command line>"`, whose first output line is always the expanded command line. Its first output line is always the expanded command line, so a shell widget can use it without parsing the rest.

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
$ myapp __clihelp                 # --help, -h and help do the same
myapp __clihelp — shell integration for this program, built with clihelp <version>

  __clihelp version                                report the clihelp version this program was built with
  __clihelp install [--no-keys] [<shell>]          set up the shell: tab completion and the Alt-H binding
  __clihelp uninstall [<shell>]                    remove what install wrote
  __clihelp keys [<shell>]                         print the key bindings, for inspection or manual setup
  __clihelp wrapper <name> [<args>...]             print a wrapper script for this program, with arguments
  __clihelp manpage [--install|--uninstall] [--force]  print a roff manual page, or install it for man(1)
```

A verb given `--help` or `-h` prints its own usage rather than treating the flag as an argument, and `-H` — clihelp's extended-help flag — adds the reserved argument names and the exact paths `install` would write on this machine:

```console
$ myapp __clihelp -H
...
What 'install' would write here, for bash:
  generated:  ~/.config/myapp/shell/bash
  sourced by: a marked block in ~/.bashrc
Nothing runs at shell startup: the line is a file test and a source, and an
upgrade rewrites the generated file rather than your configuration.
```

**Why this exists alongside the `completion` command.** `ExecuteContext` serves `__complete` before it ever looks at the command tree, so *every* clihelp program can complete — but only a program whose author added `clihelp.CompletionCommand()` could be *asked* to install that completion. `__clihelp` closes the gap, which matters most for the people who are not the author: a dotfiles script, or a packager, can set up any clihelp program uniformly:

```bash
for bin in ~/.local/bin/*; do
    "$bin" __clihelp version >/dev/null 2>&1 && "$bin" __clihelp install
done
```

`install` prints the installed path on stdout and nothing else, so it can be captured; notes for humans go to stderr. That rule holds for every verb and for the visible `completion` commands too: stdout carries paths, generated scripts and versions, one per line; stderr carries everything written for a person to read.

---

## Manual Pages

```console
$ myapp __clihelp manpage > debian/myapp.1     # for a package to install
$ myapp __clihelp manpage --install            # for this user
~/.local/share/man/man1/myapp.1
```

`$XDG_DATA_HOME/man` is on man's default search path, so an installed page is found with no configuration. That matters beyond documentation: **zsh binds Alt-H to `run-help` and fish binds it to `__fish_man_page`, and both go to `man`** — so a generated page makes Alt-H answer natively in those shells, for a program that ships no manual of its own.

It does not replace clihelp's own Alt-H binding, which answers a different question: `man myapp` is the whole manual, while the binding explains *the command line as typed*, expanding abbreviated command names and showing that subcommand's help within two thirds of the screen. And bash has no native help key at all.

**Installation refuses to create a second page.** If a manual page for the program already exists — packaged by a distribution, say — `--install` stops and says where it is, because which of two pages `man` shows is not predictable and the one that loses is invisible. `--force` installs anyway. `--uninstall` removes only a page clihelp generated; a hand-written one at the same path is left alone.

**The author decides how much of this is on offer.** `clihelp.ManPageCommand()` added to `App.Commands` gives a visible `myapp manpage`; without it, `__clihelp manpage` still works, because a packager is not the author. Nothing is ever installed unasked: `AutoRefreshIntegration` refreshes a generated page when a clihelp upgrade changes the template, and never creates one.

The generator is `clihelp.GenManPage(app, w)` if you want to drive it yourself.

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
- **Alt-H does not expand a wrapper.** The wrapper answers `__explain` with the line as typed, then the help for the wrapped command: rewriting `pd prod` into `myapp deploy prod` would replace something deliberately typed short. For Alt-H to reach the wrapper at all, its name has to be in the dispatcher's registry — `__clihelp wrapper` prints that line next to the completion one.

---

## Manual Shell Script Generation

All three scripts fall back to filename completion when the program returns no
candidates, and never mix filenames into candidates it did return. The three
shells reach that differently — bash's `complete -o default`, an explicit
`_files` in zsh, a second conditional `complete` rule in fish — but the
behaviour a user sees is the same, so an argument that is a path completes
whichever shell they are in.

### Bash (`clihelp.GenBashCompletion`)

Writes a bash programmable completion script. Includes a fallback mechanism so it operates properly even if the optional `bash-completion` package (`_init_completion`) is not pre-installed on the host.

```bash
# Load in current session:
source <(podctl completion bash)

# Install permanently (Linux):
podctl completion bash | sudo tee /etc/bash_completion.d/podctl
```

### Zsh (`clihelp.GenZshCompletion`)

Writes a `#compdef` script compatible with Zsh's `compinit` completion system. The script works both ways: autoloaded from `$fpath`, where it *is* the completion function, and sourced from a startup file, where it registers itself with `compdef` instead. (Before 0.3.12 the sourced form printed `can only be called from completion function` at every shell start.)

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

## Dynamic Completion Callbacks (Flags & Positionals)

Attach a `Complete` callback to any `Option` or `Param` to provide dynamic suggestions:

### Flags

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

### Positional Arguments

Attach `Complete` to entries in `Command.Parameters`:

```go
clihelp.Command{
    Name:        "scan",
    Description: "Scan a mailbox folder",
    Parameters: []clihelp.Param{
        {
            Name:        "<folder>",
            Description: "Folder name to scan",
            Complete: func(toComplete string) []string {
                folders := []string{"%inbox\tPrimary inbox", "%archive\tArchived mail", "%spam\tSpam"}
                var results []string
                for _, f := range folders {
                    if strings.HasPrefix(f, toComplete) {
                        results = append(results, f)
                    }
                }
                return results
            },
        },
        {
            Name:        "<message-id...>",
            Description: "Message IDs to inspect",
            Variadic:    true,
            Complete: func(toComplete string) []string {
                return fetchCachedMessageIDs(toComplete)
            },
        },
    },
    Run: runScan,
}
```

#### Rules & Constraints
- **Slot alignment:** `Parameters[0]` completes the first positional argument, `Parameters[1]` the second, and so on. `Parameters` is opt-in; slots without `Complete` return no candidates, allowing shells to fall back to default filename completion.
- **Variadic tail:** Mark the final parameter with `Variadic: true` to keep completing for every argument from its index onward. `Audit` enforces that `Variadic: true` appears only on the final parameter.
- **Context boundary:** The callback receives only `toComplete string` (the current token under the cursor). Positional callbacks cannot inspect earlier positionals or flags on the same command line.
- **Unknown flags:** Unrecognized flags before a positional argument cannot be classified for arity and are counted as single positional words.
- **SubcommandEntries:** `Complete` and `Variadic` are ignored on `Command.SubcommandEntries`, which are display entries rather than argument slots (enforced by `Audit`).

---

## Testing Shell Completions

`clihelp` includes live integration tests that compile binaries and execute them inside live `bash`, `zsh`, and `fish` subshells. See `completion_test.go` for examples of asserting completion behavior in end-to-end test suites.
