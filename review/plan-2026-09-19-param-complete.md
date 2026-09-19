# Plan: `Param.Complete` — dynamic completion for positional arguments

*Written 2026-09-19 against `clihelp` 0.3.39 (`171c36b`, clean tree). Not implemented;
this is the brief for a session in this repository.*

*Revision 3. Revision 2 answered the critique in
`response_plan-2026-09-19-param-complete.md` — five of its seven findings confirmed
against the source, one rejected on the facts (§3.6), plus an error of revision 1's own
that it did not reach (§3.5). Revision 3 answers `response_2.md`: a nil dereference that
would have panicked on every root completion (§3.5), a guard that fired past `--`
(§3.3), a false claim about shorthand clusters (§3.4), and a missing `Audit` rule
(§3.7). All four confirmed in the source.*

---

## 1. The gap

`Option` carries a completion hook. `Param` does not.

```go
// clihelp.go
type Option struct {
    Flags       string
    Description string
    Complete    func(toComplete string) []string   // ← the hook
    ...
}

type Param struct {
    Name        string
    Description string
    // nothing
}
```

So an application can complete `--label <TAB>` and cannot complete `<label>`. The
asymmetry is visible in `handleComplete`, which has a branch for a flag's value and
none for a positional's:

```go
// completion.go
if completePrevFlagValue(w, activeOptions, prevWord, toComplete) { return nil }
if strings.HasPrefix(toComplete, "-") { completeFlags(w, activeOptions, toComplete); return nil }
a.completeSubcommands(w, currentCmd, toComplete)   // ← the only thing left
```

When the current command has no subcommands — which is true of every leaf, and leaves
are where positionals live — that last line emits nothing at all. Tab on a leaf command
is dead.

## 2. Why it belongs here and not in the application

`mail_cli` has just finished a CLI redesign in which labels are written with a `%`
sigil (`mail_cli scan %inbox`), and wants `scan %IN<TAB>` to complete against the live
label list. It already has the callback — `cli.CompleteFolderNames(toComplete string)
[]string`, the same signature as `Option.Complete`, verified against real fish 4.7.1 —
and nowhere to attach it.

It could be done downstream. It should not be, for two reasons.

**It would rebuild the resolver that was just deleted.** `pod/pkg/cli/parse.go` carries
this comment about the examples view:

> This used to be about a hundred and fifty lines here: the collection walk, a
> brute-force command resolver, a theme, and the rendering. clihelp grew the view in
> v0.3.35 and learned the per-command form in v0.3.36, so all of that is gone and the
> two cannot drift apart.

A downstream `__complete` interceptor needs exactly that again: parse argv, skip flags
and their values knowing each flag's arity, find which command was matched, work out
which positional slot the cursor is in. That is `resolveCommandPath` plus
`scanLeadingFlag` — several hundred lines of this library, reimplemented approximately,
in every application that wants the feature. §3.4 and §3.5 below are the evidence: two
of the five defects in revision 1 of this plan were mistakes about this library's own
flag-arity rules, made by someone reading its source. A downstream author gets those
wrong silently.

The `Option.Binder` escape hatch is the precedent this repository already recognizes
(CHANGES 0.3.38: *"an escape hatch reached for by different people for the same standard
idiom is a missing constructor, and `Binder` is the evidence trail that shows which"*).
This is the same shape, one step earlier: nobody has written the hack yet, and the fix is
cheaper before they do.

**A downstream implementation cannot do the interesting half.** Intercepting argv gets
you `scan %IN<TAB>` — sigil already typed, prefix in hand. It does not get you
`scan <TAB>` offering `%inbox`, `%receipts`, `%spam` before any sigil is typed, because
that requires knowing *this command's first positional is a label* — slot knowledge that
lives in `Command.Parameters` and nowhere else. Only the library can answer it.

`pod` benefits too: it has positionals of its own (`pod queue latest 10`, podcast IDs)
that today complete against nothing.

## 3. The change

### 3.1 The fields

```go
// Param describes a positional argument (or a listable subcommand/flag) with
// a display name and a free-form description.
type Param struct {
    Name        string
    Description string

    // Complete supplies candidates for this positional slot, in the same form
    // as Option.Complete: one candidate per string, optionally "value\tdescription".
    // It is called with the word under the cursor, which may be empty.
    //
    // Setting it declares that this Param occupies a positional slot (see
    // Variadic). It is ignored on Command.SubcommandEntries, which are display
    // entries rather than argument slots.
    Complete func(toComplete string) []string

    // Variadic marks the last slot as unbounded: it keeps completing for every
    // argument from its own index onward. Declared rather than read out of Name,
    // because Name is display text.
    Variadic bool
}
```

**`Complete` has the same signature as `Option.Complete`, deliberately.** A callback
written for a flag plugs into a positional unchanged and vice versa —
`CompleteFolderNames` is already attached to option-shaped completions and, after this,
to `<label>` positionals, with no adapter. The `"value\tdescription"` convention is
already honoured by `completePrevFlagValue`; reuse the same `strings.Cut` on the first
tab. §5 records the one open question about this signature.

### 3.2 Matching is by position, and `Parameters` is opt-in

`Parameters[0]` completes the first positional, `Parameters[1]` the second. `Name` is
display text (`"<lbl_prefix|message-id>"`) and is never parsed — which is why `Variadic`
is a field and not a suffix test on `Name`. Revision 1 proposed scraping `...` off the
trimmed name and contradicted itself in the same paragraph; drop it.

`Arity()` is **not** a usable fallback for variadicity, although `ArgsValidator` does
report `max < 0` for unbounded. `Parameters` is a documentation list, not a formal slot
list, and this repository's own example tree proves it: `archive` declares

```go
Parameters: []clihelp.Param{
    p("all [label]", "Archive all emails in the Inbox or specified label prefix."),
    p("<message-id...>", "One or more message IDs to archive (short 8-char or full)."),
},
```

— two *alternative* forms, not slots 0 and 1. A command-level arity of `max < 0` says
nothing about which `Param` is the unbounded one.

That mismatch is harmless precisely because the feature is opt-in: nothing is completed
unless the author attached a `Complete`, so an author whose `Parameters` are alternatives
simply leaves them nil and loses nothing. Say this in the doc comment — setting
`Complete` is the author asserting "this entry is slot *i*".

Past the end of `Parameters`, fall back to the last entry if and only if it is
`Variadic`; otherwise complete nothing.

### 3.3 Where it goes in `handleComplete`

```go
res, err := a.resolveCommand(args[:len(args)-1])
if err != nil { return err }
currentCmd := res.cmd
activeOptions := a.collectOptions(res.path, currentCmd)
arity := a.positionalArity(res, currentCmd)
n, afterTerminator := positionalIndex(arity, res.remaining)

if !afterTerminator {
    // A value-taking flag owns the next word. Whether or not the flag has a
    // callback, nothing else may answer for it -- see 3.4.
    if count, known := scanLeadingFlag(arity, []string{prevWord, toComplete}); known && count == 2 {
        completePrevFlagValue(w, activeOptions, effectiveFlagToken(prevWord), toComplete)
        return nil
    }
    if strings.HasPrefix(toComplete, "-") {
        completeFlags(w, activeOptions, toComplete)
        return nil
    }
    if n == 0 {
        a.completeSubcommands(w, currentCmd, toComplete)   // a subcommand only ever sits at slot 0
    }
}
completePositional(w, currentCmd, n, toComplete)
return nil
```

Four things to note about that ordering.

*Subcommands and positionals coexist at slot 0.* The real `mail_cli unspam`
(`cli/misc.go:132-181`) has a genuine `Subcommands` entry `folder` **and** `Args:
MinimumNArgs(1)` with `Parameters: <message_id...>`; both are legal first words, so both
are emitted and the shell merges them.

*Subcommands are suppressed past slot 0.* Today `completeSubcommands` runs at every
depth, so `myapp cmd alreadyTyped <TAB>` offers `cmd`'s subcommands — which cannot
legally appear there. This is a small behaviour fix riding along; it earns its own line
in CHANGES under **Fixed**, separate from the **Added** entry, rather than being
smuggled in.

*Past `--`, neither flags nor subcommands may be offered.* Revision 1 ran the
`strings.HasPrefix(toComplete, "-")` branch before computing the index at all, so
`app scan -- -<TAB>` offered flags after the terminator; and it tested `n == 0` as a
proxy for "a command name may stand here", so `app scan -- <TAB>` — index genuinely 0 —
offered subcommands. The index was right; conflating it with "slot 0 admits a
subcommand" was the error. Hence the separate `afterTerminator`.

*The flag-value guard sits inside the terminator check, not above it.* Revision 2 ran it
first, so `mycli scan -- --output <TAB>` — where `--output` is a positional by
construction, because it follows `--` — had its `prevWord` read as a live flag, fired the
guard and returned, suppressing slot 1. Past `--` nothing is a flag, so the guard must
not be reachable there. Within the guard it returns unconditionally; see next.

### 3.4 The flag-value guard — why it returns even when nothing was emitted

`scanLeadingFlag` reports a value-taking flag with no value after it as occupying one
argument, *known*, leaving pflag to produce the error:

```go
if !takesValue || len(args) == 1 { return 1, true }
```

So in `mycli scan --output <TAB>`, `res.remaining` ends with `--output`, the walk
consumes it as one word, the index comes out 0, and revision 1 fell through and offered
positional candidates for slot 0 while the user was completing a filename.

That is not merely wrong, it *suppresses* the shell's own fallback. All three generated
scripts fall back to files only when the binary emits nothing:

- bash: `complete -o default -F _%[2]s_complete %[1]s`
- zsh: `(( ${#completions} + ${#completions_with_descriptions} )) || _files`
- fish: `__fish_..._needs_files` reruns the completer to ask, then `complete -c ... -F`

Emitting `%inbox` there replaces filename completion with nonsense. So the guard returns
`nil` whether or not `completePrevFlagValue` found a callback — the emptiness is the
signal.

Implement the guard with `scanLeadingFlag` itself, on the two-word window
`{prevWord, toComplete}`: a result of `(2, true)` means exactly "the flag in `prevWord`
consumes `toComplete` as its value". That reuses the parser rather than restating its
rules, and correctly does *not* fire for an `Optional` flag — see §3.5.

**Shorthand clusters need one more line, contrary to revision 2.** `scanLeadingFlag`
does walk to the last short in a run, so the *guard* fires correctly for `-vo <TAB>`.
But `completePrevFlagValue` matches on the whole word:

```go
for _, short := range spec.shortNames {
    if "-"+short == prevWord { matched = true; break }
}
```

`"-o" == "-vo"` is false, so `-o`'s callback is never reached and the guard returns
having emitted nothing. This is a pre-existing gap — today `-vo <TAB>` falls through to
`completeSubcommands` and the callback is skipped just the same — so the change does not
regress it, but it is one line from being fixed and we are standing on it:

```go
// effectiveFlagToken reduces a shorthand cluster to the flag that actually takes
// the value: in "-vo <val>" that is "-o". Only meaningful once scanLeadingFlag has
// reported that the word consumes the next argument, which guarantees every short
// before the last takes no value.
func effectiveFlagToken(prevWord string) string {
    if strings.HasPrefix(prevWord, "--") || !strings.HasPrefix(prevWord, "-") || len(prevWord) <= 2 {
        return prevWord
    }
    return "-" + prevWord[len(prevWord)-1:]
}
```

The `count == 2` precondition is what makes taking the last byte safe: `-fo` where `-f`
takes a value is `-f` with the inline value `o`, and `scanShorthandCluster` returns
`(1, true)` for it, so the guard never fires and this helper is never asked. Fixing it
here earns a **Fixed** entry in CHANGES of its own.

### 3.5 Computing the index — the arity map is the whole difficulty

`res.remaining` is documented as *"arguments left for pflag: skipped flags, then the
rest"*. It holds both flags and positionals, so the index is the count of words in it
that are not flags and not flag values. That needs an arity map, and getting that map
right is where revision 1 went wrong twice.

**Do not reuse `leadingFlagArity`.** Its doc comment says why:

> A command's own options are deliberately absent, since they cannot precede the command
> they belong to.

A command's own options are precisely the flags that appear *before its positionals*.
Under that map, `mail_cli scan --inbox-move someone@example.com %<TAB>` counts
`someone@example.com` as a positional and completes slot 1 instead of slot 0.

*(Revision 1 illustrated this with `--move` instead, which is wrong and worth recording:
`--move` is declared `clihelp.Optional(...)`, and `Optional` sets `arity = arityFlag`
precisely so that resolution never skips the following argument. In `scan --move x %<TAB>`
the word `x` genuinely **is** positional 0. `--inbox-move` is a plain `clihelp.String` on
the same command and is the real case. The same fact is why the §3.4 guard must be
written with `scanLeadingFlag` rather than a hand-rolled "does this flag take a value"
test: only the arity machinery knows that an optional-value flag does not.)*

**Do not build it from `collectOptions` either.** `CollectOptions` documents that
*"Hidden options are skipped"*, and `collectLocalOptions`/`collectGlobalOptions` filter
on `!o.Hidden`. A hidden value-taking flag then goes unrecognized:
`mycli scan --token 12345 <TAB>` counts `--token` and `12345` as two positionals and
completes slot 2. `leadingFlagArity` reads the raw option slices and has no such hole;
the counting map must do the same.

So build it from the raw slices, adding the current command's own `Options` — which is
the one thing `leadingFlagArity` omits:

```go
// positionalArity maps every flag legal at this command to whether it consumes
// the following argument, hidden flags included. It is leadingFlagArity plus the
// command's own Options: those cannot precede the command, but they are exactly
// the flags that precede its positionals.
func (a *App) positionalArity(res resolution, cmd *Command) map[string]bool {
    resolved := append([]*Command{}, res.ancestors...)
    if cmd != nil {
        resolved = append(resolved, cmd)
    }
    arity := a.leadingFlagArity(resolved)
    if cmd != nil {
        addFlagArity(arity, cmd.Options)
    }
    return arity
}
```

**Both `cmd != nil` checks are load-bearing, and the first one is not the obvious one.**
Revision 2 wrote `append(append([]*Command{}, res.ancestors...), cmd)` unconditionally,
putting a typed nil into the slice. `leadingFlagArity` then does
`addFlagArity(arity, cmd.PersistentOptions)` over it and dereferences nil. `res.cmd` is
nil on every completion at the root — `mycli --<TAB>`, `mycli --global <TAB>` — so this
would have panicked on the commonest invocation there is. It is exactly the class of
mistake §2 predicts a downstream reimplementation making silently.

Then walk, reusing `scanLeadingFlag` so the counting agrees with the parsing:

```go
func positionalIndex(arity map[string]bool, remaining []string) (n int, afterTerminator bool) {
    for i := 0; i < len(remaining); {
        arg := remaining[i]
        if arg == "--" {
            return n + len(remaining) - i - 1, true
        }
        if strings.HasPrefix(arg, "-") && len(arg) > 1 {
            if count, known := scanLeadingFlag(arity, remaining[i:]); known {
                i += count
                continue
            }
        }
        n++
        i++
    }
    return n, false
}
```

Cases the walk must get right, each a table row:

| input tail | index | terminator | why |
|---|---|---|---|
| *(empty)* | 0 | no | first slot |
| `--verbose` | 0 | no | switch consumes nothing |
| `--out file` | 0 | no | value flag consumes two |
| `--out=file` | 0 | no | inline value is one word |
| `-o file` | 0 | no | shorthand with value |
| `-vo file` | 0 | no | cluster, value on the last short |
| `--opt x` | 1 | no | `Optional` flag never consumes the next word |
| `--hidden-val x` | 0 | no | hidden flags are in the map |
| `alpha` | 1 | no | one positional typed |
| `alpha --verbose` | 1 | no | flag after a positional |
| `-` | 1 | no | a bare `-` is a positional (`scanLeadingFlag` says so) |
| `--` | 0 | **yes** | slot 0, but no command name may stand here |
| `-- --verbose` | 1 | **yes** | past the terminator, a flag-looking word is a positional |
| `--unknown alpha` | 2 | no | unknown flag counted as a word — see below |

The last row is the honest limit. An unrecognized `-x` cannot be classified — the
library does not know whether it takes a value — so it is counted as one positional and
the index is off by one for the rest of the line. `resolveCommandPath` makes the same
choice (an unknown flag ends resolution), the user is mid-typo, and the damage is a
wrong completion menu rather than a wrong command. Say so in the doc comment; do not
try to be clever.

### 3.6 The emitter

```go
func completePositional(w io.Writer, cmd *Command, n int, toComplete string) {
    if cmd == nil {
        return
    }
    p := paramAt(cmd.Parameters, n)   // nil past the end unless the last is Variadic
    if p == nil || p.Complete == nil {
        return
    }
    for _, res := range p.Complete(toComplete) {
        cand, desc, _ := strings.Cut(res, "\t")
        emitCandidate(w, cand, desc)
    }
}
```

**No fallback to `p.Description`.** Revision 1 filled an empty description from the
parameter's own, citing `completeFlagInlineValue`. The nearer precedent is
`completePrevFlagValue`, which completes a separate value exactly as this does and
deliberately passes the description through untouched. The reason is visible in a menu:
twenty folders under one parameter description render as twenty identical `-- The folder
name containing emails` trailers in zsh and fish. An empty description is the correct
output when the callback gave none.

`emitCandidate` already sanitizes and `completionDescription` already truncates, so a
callback returning something long or containing a newline cannot forge a record — no
extra guarding needed.

**`Param` is also used for display.** `Command.SubcommandEntries` is `[]Param`, and
`Command.Parameters` entries may be alternatives rather than slots (§3.2). `Complete`
and `Variadic` are meaningless on a display entry; the doc comment says so, and
`completePositional` never reads `SubcommandEntries`.

*(A note on the example tree: the critique read `example/mail_cli_fake/tree.go`, where
`unspam`'s `folder` is a `SubcommandEntries` entry, and concluded the coexistence case
in §3.3 was fictional. It is not — it is in the real program, `mail_cli/cli/misc.go`,
as a genuine `Subcommands` entry beside `Parameters: <message_id...>`. The fake tree is
a rendering fixture and does not mirror the real command tree. But the reading did
surface the `SubcommandEntries` overlap above, which is worth documenting.)*

**Root-level positionals.** When `currentCmd == nil` the application itself is taking
the argument (`mail_cli %inbox`, which `App.Options` in 0.3.39 made expressible). `App`
has no `Parameters` field today. **Recommendation: out of scope, stated in CHANGES.**
`App.Parameters` is a second decision — it affects help rendering, `Audit`, the markdown
generator and the man page, none of which this change otherwise touches. `mail_cli`
reaches its root positional through a shortcut, and shortcut completion is a separate
question already.

### 3.7 `paramAt`, and the `Audit` rule that makes it safe

```go
// paramAt returns the Param occupying slot n, or nil. Past the end only a
// Variadic final parameter answers; Audit enforces that Variadic appears
// nowhere else, so the last entry is the only one that can be unbounded.
func paramAt(params []Param, n int) *Param {
    if n < len(params) {
        return &params[n]
    }
    if len(params) == 0 {
        return nil
    }
    if last := &params[len(params)-1]; last.Variadic {
        return last
    }
    return nil
}
```

Only the final entry is consulted past the end, which means a `Variadic` marker anywhere
else is silently inert: an author writing

```go
Parameters: []clihelp.Param{
    {Name: "<files...>", Variadic: true},
    {Name: "<dest>"},
},
```

gets nothing at slot 2 and no indication why. An unbounded slot followed by another slot
is not a shape any command can have, so this is not a runtime question to answer
gracefully — it is a static error. Add a rule to `Audit`, called from
`auditCommandTree` beside `auditCommandOptions`:

> `Variadic: true` may appear only on the last entry of `Command.Parameters`.

`Audit` is what the README tells people to run in CI, and 0.3.37 established the
principle that a declaration which cannot work is a build failure rather than a puzzle
at the terminal. Worth adding in the same rule, since the walk is already there:
`Complete` or `Variadic` set on a `SubcommandEntries` entry, which §3.6 documents as
ignored — an author who sets it has misunderstood the field, and silence is the worst
answer.

## 4. Tests

Unit, in `completion_test.go`:

- The table in §3.5 driven straight into `positionalIndex`, both return values asserted.
- **The flag-value guard**: a value-taking flag with `Complete: nil` as `prevWord` emits
  *nothing at all* — asserted on an empty buffer, since that emptiness is what the shell
  reads as "fall back to files". Also with a callback, and with a shorthand cluster.
- **The two arity traps, asserted directly**, so a refactor that swaps either map back
  fails loudly: a value-taking `Command.Options` flag before the positional does not
  shift the slot; a `Hidden: true` value-taking flag does not shift it either.
- **`Optional`**: `--opt x <TAB>` completes slot 1, not slot 0.
- `Parameters[0]`/`Parameters[1]` each fire at their own slot and not the other's.
- `Variadic` keeps firing at slots 1, 2, 3; a non-variadic last parameter stops.
- A `Param` with a nil `Complete`, a `Command` with nil `Parameters`, and a nil `cmd`
  each emit nothing and do not panic.
- **Root completion does not panic**: `positionalArity` with `res.cmd == nil` and with
  non-empty `res.ancestors`, plus an end-to-end `mycli --<TAB>` and `mycli --global <TAB>`.
  This is a one-line regression away at all times (§3.5) and deserves a named test.
- **The guard is terminator-gated**: `scan -- --output <TAB>` completes positional slot 1
  rather than returning as if `--output` were a live flag.
- **Shorthand cluster**: `-vo <TAB>` reaches `-o`'s callback (§3.4); `-fo <TAB>` where
  `-f` takes a value does *not* fire the guard, since the value is already inline.
- `"value\tdescription"` splits on the first tab only; a candidate containing a newline
  is sanitized to one record; a candidate with no description emits an empty one.
- Subcommands and slot-0 positionals together; subcommands absent at slot 1; neither
  flags nor subcommands after `--`.

End-to-end, in the live-shell harness (`completion_bash_test.go`,
`completion_zsh_test.go`, `completion_fish_test.go`): one case per shell where the
candidate contains a character the shell would otherwise act on. `%` is the motivating
one — it is not special in bash, zsh, dash or fish (fish reserves only `%self`, and
removed `%jobname` in 3.0) — but a candidate with a space, and one with a `$`, are the
cases worth pinning. **Add one per shell for the fallback**: `--output <TAB>` on a flag
with no callback must still offer filenames, which is the only way to prove §3.4 end to
end. This is the part that cannot be faked: `mail_cli`'s sigil work was validated against
real fish 4.7.1 precisely because reasoning about quoting is unreliable.

`Audit`, in the audit tests: a command with `Variadic: true` on a non-final parameter is
rejected; on the final one it passes; `Complete` on a `SubcommandEntries` entry is
rejected.

Drift: `docs_drift_test.go` and `tree/drift_test.go` will want a look, since
`docs/completion.md` gains a section.

## 5. The one open design question: callback context

`Param.Complete` as specified sees only the word under the cursor. Positionals form
dependent chains more often than flags do:

- `mail_cli show <lbl_prefix> [message-id]` — the candidate IDs depend on which label is
  in slot 0.
- `mail_cli -A work scan %<TAB>` — the candidate labels depend on a flag on the same
  line.

Two ways to go, and the repository's own policy forces the choice to be made now rather
than drifted into. `AGENTS.md`: *"a name, field, or function that is wrong should be
changed or removed outright. Prefer that to adding a correct alternative beside it — a
deprecated alias is a permanent cost paid to avoid a break that is currently free."*

**(a) Narrow now (recommended).** Ship `func(toComplete string) []string`. Symmetry with
`Option.Complete` is the property that lets one callback serve both, no downstream call
site changes, and the dependent cases degrade rather than fail: message-ID completion
falls back to every cached ID instead of the ones in that folder, and `-A work` is
already read out of band by `CompleteFolderNames` from config. If a dependent case is
later wired up for real, the pre-1.0 answer is to widen **both** hooks to a context value
in one change — never to add a second field beside `Complete`.

**(b) Context now.** `func(ctx CompleteContext) []string` carrying at minimum the word,
the preceding positionals, and the parsed flags — applied to `Option.Complete` too, so
the two do not diverge. Costs a break in `pod` and `mail_cli` (both small, both the same
author), buys never breaking again.

Settle this before writing code; everything else in this plan is unaffected by the
answer. Review concurs with (a). Whichever is chosen, `docs/completion.md` must state
plainly that a positional callback cannot inspect earlier positionals or flags on the
same line — the limitation is invisible from the signature, and an author will otherwise
discover it by writing a callback that silently completes the wrong set.

## 6. Documentation

- **`docs/completion.md`** — rename *"Dynamic Completion Callbacks"* to cover both, and
  add the positional example beside the existing `Option` one. State the position rule,
  the `Variadic` rule, the opt-in nature of `Parameters` as slots, and the unknown-flag
  limit.
- **`CHANGES.md`** — one **Added** entry for `Param.Complete`/`Param.Variadic`, and
  **Fixed** entries for subcommands past slot 0, for anything offered after `--`, and for
  a flag's callback being skipped when the flag is written inside a shorthand cluster
  (§3.4) — the last is an independent bug this change happens to stand on, and it is
  worth naming as such rather than folding into the feature. In this repository's
  register: say which application wanted it and what it was going to have to write
  instead.
- **godoc** — the field comments above carry the "same signature as `Option.Complete`"
  sentence and the `SubcommandEntries` exclusion.
- **`llms.txt` / `README.md`** — check whether either enumerates `Param`'s fields.

## 7. Sizing and scope

Per `AGENTS.md`: production functions are capped at 80 lines. The four new functions
(`positionalArity`, `positionalIndex`, `paramAt`, `completePositional`), plus
`effectiveFlagToken` and the `Audit` rule, are each well under. `handleComplete` grows by
about ten lines and stays short; if it crowds, the guard and the index are the natural
extraction.

Pre-1.0 the surface is not frozen and these two fields are additive: `Param` is a keyed
struct literal everywhere it is used, including the `p(name, desc)` helper in
`example/mail_cli_fake/tree.go`, so nothing breaks.

**Not in scope:** `App.Parameters` (§3.6), shortcut positionals, and — pending §5 —
context-aware callbacks.

## 8. Downstream, after it lands

`mail_cli` needs one line per label positional:

```go
Parameters: []clihelp.Param{
    {Name: "<lbl_prefix>", Description: "...", Complete: cli.CompleteFolderNames},
},
```

`CompleteFolderNames` already strips the `%` sigil before matching and re-attaches it to
every candidate, because a shell replaces the whole token. Nothing else downstream
changes.
