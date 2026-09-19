# Critique & Response: `plan-2026-09-19-param-complete.md` (Revision 2)

**Target:** [plan-2026-09-19-param-complete.md](file:///home/sariel/prog/26/go/clihelp/review/plan-2026-09-19-param-complete.md)  
**Date:** 2026-09-19  
**Status:** Substantial progress over Revision 1; blocked on fatal nil dereference and edge-case guard bugs.

---

## 1. Executive Assessment

- **Progress:** Revision 2 is a significant step forward. It resolved five core defects from Revision 1:
  1. Replaced ad-hoc string scraping of `Name` with an explicit `Param.Variadic` boolean field.
  2. Introduced a flag-value guard to prevent value-taking flags without callbacks from stealing slot 0 and breaking shell filename fallback.
  3. Decoupled `--` terminator detection via `afterTerminator` to suppress subcommands and flags past `--`.
  4. Derived `positionalArity` from [`leadingFlagArity`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L427), ensuring hidden flags are included in argument counting.
  5. Removed repetitive `p.Description` fallback noise from candidate emission.

- **Verdict:** **Not ready to implement as written.** Revision 2 introduces a **fatal nil-pointer dereference panic** on all root completions, contains a flag-value guard bypass after `--`, and makes an invalid assumption about shorthand clusters in [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79-L112).

---

## 2. Critical Bugs & Deficiencies in Revision 2

### 2.1 Fatal Nil-Pointer Panic on Root Completion

In §3.5:
```go
func (a *App) positionalArity(res resolution, cmd *Command) map[string]bool {
    arity := a.leadingFlagArity(append(append([]*Command{}, res.ancestors...), cmd))
    if cmd != nil {
        addFlagArity(arity, cmd.Options)
    }
    return arity
}
```

When tab completion is invoked at the application root (e.g. `mycli <TAB>`, `mycli --<TAB>`, or `mycli --global <TAB>`):
1. `res.cmd` is `nil`, so `currentCmd` is `nil`.
2. `append(..., cmd)` appends a typed `nil` pointer into the `resolved` slice.
3. [`leadingFlagArity`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L427-L438) iterates over `resolved`:
   ```go
   for _, cmd := range resolved {
       addFlagArity(arity, cmd.PersistentOptions) // cmd is nil!
   }
   ```
4. **Result:** An immediate runtime crash (`panic: runtime error: invalid memory address or nil pointer dereference`) on every root completion invocation.

**Required Fix:**
```go
resolved := append([]*Command{}, res.ancestors...)
if cmd != nil {
    resolved = append(resolved, cmd)
}
arity := a.leadingFlagArity(resolved)
if cmd != nil {
    addFlagArity(arity, cmd.Options)
}
return arity
```

---

### 2.2 Flag-Value Guard Misfires After `--` Terminator

In §3.3:
```go
// A value-taking flag owns the next word. Whether or not the flag has a
// callback, nothing else may answer for it -- see 3.4.
if count, known := scanLeadingFlag(arity, []string{prevWord, toComplete}); known && count == 2 {
    completePrevFlagValue(w, activeOptions, prevWord, toComplete)
    return nil
}

n, afterTerminator := positionalIndex(arity, res.remaining)
```

The flag-value guard runs **before** `positionalIndex` checks for `--`.

If the user passes a positional argument after `--` that happens to match a flag name:
```bash
mycli scan -- --output <TAB>
```
1. `--output` is explicitly a positional argument because it appears after `--`.
2. The user is now attempting to complete the argument following `--output` (slot 1).
3. The guard inspects `prevWord = "--output"` in isolation, finds `count == 2` in `arity`, and calls `completePrevFlagValue`, which returns `nil`.
4. **Result:** Positional completion is suppressed because the guard assumes `--output` is an active flag.

**Required Fix:** Check for terminator status or evaluate `positionalIndex` before or as part of the guard. If `--` appears in `res.remaining`, `prevWord` cannot be an active flag, and the flag-value guard must not fire.

---

### 2.3 Shorthand Cluster Fallacy in `completePrevFlagValue`

In §3.4, the plan asserts:
> *"...and gets the shorthand cluster (`-vo <TAB>`) right for free, since `scanShorthandCluster` already walks to the last short in the run."*

This is incorrect. While `scanLeadingFlag` detects that `-vo` takes a value (`count == 2`), the guard delegates to [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79-L112):
```go
for _, short := range spec.shortNames {
    if "-"+short == prevWord {
        matched = true
        break
    }
}
```
When `prevWord` is `"-vo"`, `"-"+short` is `"-o"`. `"-o" == "-vo"` evaluates to **false**.
- If `-o` has an `opt.Complete` callback, it will **never be called** when clustered as `-vo <TAB>`.
- The guard returns `nil`, silently discarding completion for clustered flags.

**Required Fix:** Either update [`completePrevFlagValue`](file:///home/sariel/prog/26/go/clihelp/completion.go#L79-L112) to match the trailing short in a cluster, or explicitly note that cluster completion delegation is an existing limitation of `completePrevFlagValue` that requires a helper to resolve the active short name.

---

### 2.4 Missing Static Audit for `Param.Variadic`

In §3.2 and §3.6:
`paramAt` only checks whether the **last** parameter has `Variadic: true`:
```go
last := &params[len(params)-1]
if last.Variadic { return last }
```
If an author configures:
```go
Parameters: []clihelp.Param{
    {Name: "<files...>", Variadic: true},
    {Name: "<dest>"},
}
```
Any argument past index 1 will evaluate `last.Variadic` (`<dest>`), find `false`, and return `nil`.
[`Audit`](file:///home/sariel/prog/26/go/clihelp/audit.go#L166) should verify that if any `Param` has `Variadic: true`, it is strictly the last entry in `cmd.Parameters`.

---

## 3. Analysis of Section 5: Callback Context

The plan's recommendation in §5 to choose **(a) Narrow now (`func(toComplete string) []string`)** is sound:
1. **Symmetry:** Preserves exact type parity with [`Option.Complete`](file:///home/sariel/prog/26/go/clihelp/clihelp.go#L17). Callbacks like `CompleteFolderNames` can attach directly to both options and positionals.
2. **Simplicity:** Over 90% of dynamic completions are simple prefix filters (enums, tags, names). Forcing a `CompleteContext` struct would add boilerplate to every closure.
3. **Pre-1.0 Flexibility:** In accordance with `clihelp`'s `AGENTS.md` guidelines, if dependent argument context is ever formally required, widening both hooks simultaneously before 1.0 remains an open option.

However, the documentation must explicitly state that dependent positionals (e.g. completing `<item-id>` based on the value in slot 0) cannot inspect earlier arguments under this hook.

---

## 4. Required Checklist Before Implementation

1. **Guard `cmd == nil` in `positionalArity`:** Ensure `nil` is never passed into [`leadingFlagArity`](file:///home/sariel/prog/26/go/clihelp/resolve.go#L427).
2. **Terminator-gate the flag-value guard:** Ensure `prevWord` is not treated as a flag if `--` has appeared.
3. **Fix cluster matching in `completePrevFlagValue`:** Enable `-vo <TAB>` to find `-o`'s callback.
4. **Add `Audit` rule for `Param.Variadic`:** Enforce `Variadic: true` only on the terminal parameter.
