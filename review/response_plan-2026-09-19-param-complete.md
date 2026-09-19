# Critique & Response: `plan-2026-09-19-param-complete.md`

**Target:** [plan-2026-09-19-param-complete.md](file:///home/sariel/prog/26/go/clihelp/review/plan-2026-09-19-param-complete.md)  
**Date:** 2026-09-19  
**Status:** Rejected in current form; requires revision before implementation.

---

## 1. Executive Verdict

- **Is the problem real?** **Yes.** Leaf commands currently lack dynamic tab completion for positional arguments. Downstream applications (such as `mail_cli` with `%inbox` or `podctl` with podcast identifiers) cannot complete positionals without reimplementing several hundred lines of flag arity scanning and command resolution.
- **Is the plan sound?** **No.** The design contains multiple fatal edge cases, incorrect assumptions about flag-value parsing, architectural contradictions, and regressions in standard shell completion fallbacks.
- **Should it be implemented?** **Not as written.** The core capability belongs in `clihelp`, but the implementation logic outlined in the plan must be revised to address the defects detailed below.

---

## 2. Critical Defects & Traps in the Plan

### 2.1 The Missing Value Trap (Breaks Shell Filename Completion)

In §3.3 and §3.4, the plan assumes every value-taking flag in `res.remaining` already has its value following it (all table rows in §3.4 test `--out file`, `-o file`, etc.).

When a user completes a flag value that has no custom completer:
```bash
mycli scan --output <TAB>
```
1. `args[:len(args)-1]` passed to [`resolveCommand`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L624) is `["scan", "--output"]`. `toComplete` is `""`.
2. [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79-L112) returns `false` because `--output` has `Complete: nil`.
3. In §3.3, execution falls through to `positionalIndex`.
4. In [`scanLeadingFlag`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L464), when `len(args) == 1` (since `--output` is the last item in `remaining`), it returns `1, true` ("flag with missing value").
5. `positionalIndex` consumes `--output` as 1 word and returns `n = 0`.
6. **`completePositional` fires for slot 0.**

**Consequence:** Typing `--output <TAB>` emits candidates for positional argument 0 (e.g. `%inbox`), completely suppressing the shell's fallback filename completion (`complete -o default` / `_files`). The user is offered positionals instead of filenames when trying to complete a flag value.

**Correction:** If `prevWord` is a known flag that takes a value (and was not written with an inline `--flag=val`), `toComplete` is that flag's value. If [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79-L112) does not emit anything, [`handleComplete`](file:///home/sariel/prog/26/go/clihelp/completion.go#L209) must return `nil` immediately. It must never fall through to `completePositional`.

---

### 2.2 The Double-Hyphen (`--`) Terminator Bugs

The plan mishandles `--` in two distinct places:

1. **Subcommands offered after `--`:**  
   In `positionalIndex` (§3.4):
   ```go
   if arg == "--" {
       return n + len(remaining) - i - 1
   }
   ```
   If the user types `app scan -- <TAB>`, `remaining` is `["--"]`, so `positionalIndex` returns `0`. Because `n == 0`, `if n == 0 { a.completeSubcommands(...) }` executes and offers subcommands after `--`. Universal CLI semantics dictate that everything after `--` is strictly a positional argument.
2. **Flags offered after `--`:**  
   If the user types `app scan -- -<TAB>`, `strings.HasPrefix(toComplete, "-")` runs **before** `positionalIndex`. It calls [`completeFlags`](file:///home/sariel/prog/26/go/clihelp/completion.go#L149) and offers `--flags` past the `--` terminator.

**Correction:** If `--` is encountered in `remaining`, bypass both [`completeFlags`](file:///home/sariel/prog/26/go/clihelp/completion.go#L149) and `completeSubcommands`. All tokens past `--` must route exclusively to `completePositional`.

---

### 2.3 Hidden Flags Desynchronize Positional Indexing

In §3.4:
```go
func (a *App) positionalArity(activeOptions []Option) map[string]bool
```
The plan builds `positionalArity` from `activeOptions`, which is populated by [`App.CollectOptions`](file:///home/sariel/prog/26/go/clihelp/clihelp.go#L398-L406).
[`CollectOptions`](file:///home/sariel/prog/26/go/clihelp/clihelp.go#L370-L373) explicitly discards options where `opt.Hidden == true`.

If a command has a hidden flag taking a value (`--token <val>`, `Hidden: true`), and the user runs:
```bash
mycli scan --token 12345 <TAB>
```
`--token` is absent from `positionalArity`. [`scanLeadingFlag`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L447) returns `0, false`. `positionalIndex` treats `--token` as positional 0 and `12345` as positional 1, computing `n = 2` instead of `0`.

**Correction:** `positionalArity` must include hidden options, matching the behavior of [`leadingFlagArity`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L427) and `setupFlagSet`.

---

### 2.4 Inconsistency & Fragility in Variadic Detection

In §3.2, the plan contains a direct contradiction within two sentences:
> *"The `Name` field is display text (`"<lbl_prefix|message-id>"`) and must never be parsed."*  
> *"A name ending in `...` before the closing bracket — `<message_id...>`, `[files...]` — keeps completing for every slot at or beyond its index. Detect it on the trimmed name: strip one leading `<` or `[` and one trailing `>` or `]`, then test `strings.HasSuffix(name, "...")`."*

This violates existing library conventions:
1. `clihelp` already provides [`ArgsValidator.Arity() (min, max int)`](file:///home/sariel/prog/26/go/clihelp/args.go#L25), where `max < 0` formally designates unbounded/variadic arguments.
2. If an author writes `Name: "<file>"` and specifies `Args: MinimumNArgs(1)`, the command is variadic, but the string scraper fails to detect it. At slot 1 (`app process file1 <TAB>`), completion stops completely.
3. Conversely, display names like `<file>...` (trailing dots outside brackets), `...files`, or `args...` create fragile edge cases when stripping delimiters.

**Correction:** Determine variadic status via an explicit `Variadic bool` field on [`Param`](file:///home/sariel/prog/26/go/clihelp/clihelp.go#L121) and/or by checking whether `cmd.Args.Arity()` reports an unbounded maximum (`max < 0`).

---

### 2.5 Description Repetition Noise in Shell Menus

In §3.5:
```go
if desc == "" {
    desc = p.Description
}
emitCandidate(w, cand, desc)
```
The plan claims this mirrors [`completeFlagInlineValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L128). However, [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L102-L107) (which completes separate flag values) intentionally does **not** fall back to `opt.Description`.

If `CompleteFolderNames` returns 20 folder names with no per-item description, every line in Zsh and Fish renders:
```text
%inbox     -- The folder name containing emails
%archive   -- The folder name containing emails
%sent      -- The folder name containing emails
...
```
Repeating the parameter header across dozens of candidate values pollutes the terminal menu. If the callback does not provide candidate-specific descriptions via `value\tdescription`, `desc` must remain empty.

---

## 3. Architectural Gaps

### 3.1 Parameter Context Inaccessibility

`Param.Complete` is typed as `func(toComplete string) []string`. Unlike flags, positional arguments frequently form dependent hierarchies:
- In `mail_cli show <lbl_prefix> [message_id]`: candidate message IDs depend on which label was typed in slot 0.
- In commands with flags: candidate positionals often depend on flags (e.g. `--account work` determines which folders exist).

With `func(toComplete string) []string`, `Complete` receives only the prefix under the cursor. It cannot inspect earlier positional arguments or active flags without parsing `os.Args` out-of-band. While pre-1.0, the signature should either accept context/preceding arguments or explicitly document this limitation as a deliberate trade-off for symmetry with `Option.Complete`.

### 3.2 Motivating Example Discrepancy

§3.3 cites `mail_cli unspam` as an example of a command having both a subcommand (`folder`) and a positional (`<message_id...>`). In the repository's own [`example/mail_cli_fake/tree.go`](file:///home/sariel/prog/26/go/clihelp/example/mail_cli_fake/tree.go#L58-L64), `folder` is documented in `SubcommandEntries`, not in `Subcommands`. [`completeSubcommands`](file:///home/sariel/prog/26/go/clihelp/completion.go#L174) only traverses real `Command.Subcommands`. The motivating example in the plan diverges from the actual codebase behavior.

---

## 4. Actionable Blueprint for Implementation

To implement dynamic positional completion cleanly, apply the following adjustments:

1. **Flag Value Guard in [`handleComplete`](file:///home/sariel/prog/26/go/clihelp/completion.go#L209):**  
   Inspect `prevWord` against `positionalArity`. If `prevWord` is a known flag that expects a value and lacks an inline `=` assignment:
   - Call [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79).
   - Immediately `return nil`, regardless of whether candidates were emitted, allowing shell filename fallback when `opt.Complete` is nil.
2. **Terminator Awareness:**  
   If `--` appears in `remaining`:
   - Do not invoke [`completeFlags`](file:///home/sariel/prog/26/go/clihelp/completion.go#L149), even if `toComplete` starts with `-`.
   - Do not invoke `completeSubcommands`.
   - Route directly to `completePositional`.
3. **Include Hidden Flags in Arity:**  
   Construct `positionalArity` using an option collector that retains hidden flags across ancestors and the current command.
4. **Reliable Variadic Identification:**  
   Support an explicit `Param.Variadic` boolean field, with a fallback checking `cmd.Args.Arity()` for `max < 0`.
5. **Clean Candidate Emission:**  
   Emit empty descriptions when `strings.Cut(res, "\t")` yields no description, matching [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L102-L107).
