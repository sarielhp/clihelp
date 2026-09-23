# Design Plan: Automatic Shell Completion for Wrapper Scripts (`clihelp`)

> [!NOTE]
> **Status:** Superseded by [`plans/clihelp_wrapper_revised.md`](file:///home/sariel/prog/26/go/clihelp/plans/clihelp_wrapper_revised.md) following critical systems review. In-place script mutation and `$PPID` inspection were rejected in favor of safe read-only `--from` extraction, `App.DisableSetup`, and a complete `__clihelp` UI/theming overhaul.

**Target Repository:** [`github.com/sarielhp/clihelp`](file:///home/sariel/prog/26/go/clihelp)  
**Proposed UX:** `<wrapper> __clihelp install [wrapper_name]`  
**Complementary Command:** `<binary> completion wrap <wrapper_name> [-- <preset_args...>]`  
**Date:** 2026-09-23

---


## 1. Summary

Users frequently write lightweight wrapper scripts or aliases around CLI tools to fix flags or target specific subcommands/accounts. For example, a wrapper `mt` for `mail_cli`:

```bash
#!/bin/bash
mail_cli -2 tui "$@"
```

Currently, shell autocompletion breaks when invoked through `mt` because the shell has no completion specification registered for `mt`. Manually crafting custom completion scripts for Fish, Bash, and Zsh requires deep shell scripting expertise and understanding of internal completion protocols.

This proposal introduces first-class wrapper script completion installation directly into `clihelp`. By running:

```console
$ mt __clihelp install
```

`clihelp` automatically detects that it is being executed via a wrapper, extracts the preset arguments (`-2 tui`), determines the wrapper script's name (`mt`), and installs native completion handlers for the user's active shell.

---

## 2. Invocation & Execution Flow

When a user executes:
```console
$ mt __clihelp install
```

### Step 1: Argument Layout
The wrapper script forwards its arguments via `"$@"`, so the underlying CLI receives:
```
os.Args = ["/path/to/mail_cli", "-2", "tui", "__clihelp", "install"]
```

### Step 2: Early Interception in `clihelp.Execute`
Before standard subcommand and flag resolution in [`execute.go`](file:///home/sariel/prog/26/go/clihelp/execute.go):
1. `clihelp` scans `os.Args[1:]` for the reserved token `__clihelp`.
2. All tokens appearing **before** `__clihelp` are parsed as **preset arguments**:
   ```go
   presetArgs = ["-2", "tui"]
   ```
3. All tokens appearing **after** `__clihelp` form the clihelp sub-operation:
   ```
   "install" [wrapper_name]
   "uninstall" [wrapper_name]
   ```

### Step 3: Wrapper Name Discovery
To register completion with the shell, `clihelp` must know the wrapper command's name (`mt`):

1. **Positional Override**: If provided as `mt __clihelp install my-alias`, use `my-alias`.
2. **Process Inspection (Linux / Unix)**:
   - Read parent process PID via `os.Getppid()`.
   - Read `/proc/$PPID/cmdline`.
   - In standard shell wrappers (`bash`, `sh`, `dash`, `zsh`), the parent process invocation contains the script path (e.g. `/bin/bash /home/sariel/bin/mt __clihelp install`).
   - Extract the script filename: `filepath.Base(...)` &rarr; `mt`.
3. **Fallback**: If `$PPID` inspection reveals an interactive shell (e.g., if the wrapper used `exec mail_cli ...` or was an interactive shell alias), emit an actionable error:
   ```
   Error: could not determine wrapper name from parent process (did the script use 'exec'?).
   Please supply the wrapper name explicitly:
       mt __clihelp install <wrapper_name>
   ```

### Step 4: Shell Detection & Script Generation
`clihelp` inspects `$SHELL` (or an explicit `--shell <name>` flag) and generates a completion file tailored to the wrapper and preset arguments.

---

## 3. Generated Completion Scripts

### Fish Shell
Installed to: `~/.config/fish/completions/<wrapper>.fish`

Fish automatically discovers and reloads completions placed in `~/.config/fish/completions/` without restarting the shell.

```fish
# fish completion for mt wrapping mail_cli
# clihelp-wrapper-completion: 1
function __fish_mt_complete
    set -l cmd (commandline -opc) (commandline -ct)
    test (count $cmd) -gt 1; and set -e cmd[1]
    mail_cli __complete -2 tui $cmd 2>/dev/null
end

function __fish_mt_needs_files
    test (count (__fish_mt_complete)) -eq 0
end

complete -c mt -f -a '(__fish_mt_complete)'
complete -c mt -n __fish_mt_needs_files -F
```

### Bash Shell
Installed to: `~/.local/share/bash-completion/completions/<wrapper>` (or appended to app's integration file).

```bash
# bash completion for mt wrapping mail_cli
# clihelp-wrapper-completion: 1
_mt_complete() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n := || return
    else
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
        cur="${words[cword]}"
        prev="${words[cword-1]}"
    fi

    local out
    out=$( mail_cli __complete -2 tui "${words[@]:1:cword-1}" "$cur" 2>/dev/null ) || return

    COMPREPLY=()
    local line cand
    while IFS= read -r line; do
        [[ -z $line ]] && continue
        cand="${line%%	*}"
        [[ $cand == "$cur"* ]] && COMPREPLY+=("$(printf '%q' "$cand")")
    done <<< "$out"

    if declare -F __ltrim_colon_completions >/dev/null 2>&1; then
        __ltrim_colon_completions "$cur"
    fi
}
complete -o default -F _mt_complete mt
```

### Zsh Shell
Installed to: `~/.zfunc/_mt` (or registered via integration file).

```zsh
#compdef mt
# clihelp-wrapper-completion: 1

_mt() {
    local -a completions
    local -a completions_with_descriptions
    local line

    local -a words_to_pass
    if (( CURRENT > 1 )); then
        words_to_pass=("${(@)words[2,CURRENT]}")
    elif (( ${#words[@]} > 1 )); then
        words_to_pass=("${(@)words[2,-1]}")
    elif (( ${#@} > 0 )); then
        words_to_pass=("$@")
    fi

    local output
    output=(${(f)"$(mail_cli __complete -2 tui ${(q)words_to_pass[@]} 2>/dev/null)"})

    for line in "${output[@]}"; do
        if [[ -z "$line" ]]; then
            continue
        fi
        if [[ "$line" == *$'\t'* ]]; then
            local cand="${line%%	*}"
            local desc="${line#*	}"
            cand="${cand//:/\\:}"
            desc="${desc//:/\\:}"
            completions_with_descriptions+=("${cand}:${desc}")
        else
            completions+=("${line//:/\\:}")
        fi
    done

    if (( ${#completions_with_descriptions} )); then
        _describe -t commands 'mt' completions_with_descriptions
    fi
    if (( ${#completions} )); then
        compadd -a completions
    fi
    (( ${#completions} + ${#completions_with_descriptions} )) || _files
}

if [ "$funcstack[1]" = "_mt" ]; then
    _mt "$@"
elif type compdef >/dev/null 2>&1; then
    compdef _mt mt
fi
```

---

## 4. Edge Cases & Mitigations

| Edge Case | Problem | Mitigation |
|---|---|---|
| **Wrapper uses `exec`** | `exec mail_cli -2 tui "$@"` replaces the wrapper process; `$PPID` becomes the interactive terminal. | Inspect process name. If parent process is a shell without a script path in cmdline, report error prompting for explicit name: `mt __clihelp install mt`. |
| **Shell Aliases / Functions** | `alias mt="mail_cli -2 tui"` executes without a subshell process. | Auto-detection fails gracefully; user specifies wrapper name: `mt __clihelp install mt`. |
| **Non-Linux Platforms (macOS / BSD)** | `/proc/$PPID/cmdline` does not exist. | Use platform-specific process query (`ps -p $PPID -o args=`), falling back to requiring the positional name argument if unavailable. |
| **Wrapper Uninstallation** | User deletes the wrapper script and wants stale completions removed. | Support `<wrapper> __clihelp uninstall [name]`. |
| **Binary Re-naming / Symlinks** | User creates `fastmail -> mail_cli` without arguments. | If `presetArgs` is empty, generate wrapper forwarding directly to `fastmail __complete` or aliasing completion. |

---

## 5. Complementary CLI Command

In addition to `<wrapper> __clihelp install`, `clihelp` should provide an explicit, non-magic command on the base binary:

```console
$ mail_cli completion wrap mt -- -2 tui
$ mail_cli completion wrap-install mt -- -2 tui
```

And optional auto-inspection:
```console
$ mail_cli completion wrap mt
```
where `clihelp` locates `mt` via `exec.LookPath("mt")`, parses the shell script line calling `mail_cli`, extracts the preset args, and installs the completion automatically.

---

## 6. Implementation Scope in `clihelp`

Files to touch in [`github.com/sarielhp/clihelp`](file:///home/sariel/prog/26/go/clihelp):

1. **[`execute.go`](file:///home/sariel/prog/26/go/clihelp/execute.go)**:
   Add early check for `__clihelp` token in `Execute()` before command dispatch.
2. **`wrapper_install.go`** (New):
   - `detectWrapperFromPPID() (string, error)`
   - `extractPresetArgs(args []string) ([]string, []string)`
   - `InstallWrapperCompletion(app *App, shell, wrapperName string, presetArgs []string) error`
   - `UninstallWrapperCompletion(app *App, shell, wrapperName string) error`
3. **[`completion_templates.go`](file:///home/sariel/prog/26/go/clihelp/completion_templates.go)**:
   Add parametrized templates supporting `presetArgs` for Fish, Bash, and Zsh.
4. **[`completion_command.go`](file:///home/sariel/prog/26/go/clihelp/completion_command.go)**:
   Expose `completion wrap` and `completion wrap-install` subcommands.
