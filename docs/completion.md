# Shell Autocompletion

`clihelp` provides built-in shell autocompletion for **Bash**, **Zsh**, and **Fish** through its native `__complete` protocol.

---

## Table of Contents

- [Overview & Architecture](#overview--architecture)
- [Adding a `completion` Command](#adding-a-completion-command)
- [Shell-Specific Script Generators](#shell-specific-script-generators)
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

## Adding a `completion` Command

To allow users to generate completion scripts for their shell, register a `completion` subcommand on your application:

```go
Commands: []clihelp.Command{
    {
        Name:        "completion",
        Description: "Generate shell autocompletion script",
        UsageLine:   "podctl completion <bash|zsh|fish>",
        Args:        clihelp.ExactArgs(1),
        Run: func(ctx *clihelp.Context) error {
            switch ctx.Args[0] {
            case "bash":
                return clihelp.GenBashCompletion(ctx.App, ctx.Stdout)
            case "zsh":
                return clihelp.GenZshCompletion(ctx.App, ctx.Stdout)
            case "fish":
                return clihelp.GenFishCompletion(ctx.App, ctx.Stdout)
            default:
                return fmt.Errorf("unsupported shell %q (use bash, zsh, or fish)", ctx.Args[0])
            }
        },
    },
}
```

---

## Shell-Specific Script Generators

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
