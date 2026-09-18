# Deep review: command resolution (`resolve.go`)

## Why this scope

`resolve.go` is 506 lines and has **zero citations across all three deep reviews** of this
library. The first went deep on flag binding and example validation, the second on shell
integration, the third on rendering; each left it out of scope. It is the last large file
nobody has read adversarially.

It is also load-bearing. Every invocation goes through `resolveCommand`, and so does
completion (`__complete`), the Alt-H expansion (`__explain`), example colourisation and
example validation. A defect here is a defect in all of them at once.

One thing is already known sound: `scanLeadingFlag` was differentially tested against pflag
over every flag shape during the first review, and that result held. Do not re-derive it —
but do look at everything around it.

## Scope

**In scope:** `resolve.go` in full, and `leading_flags_test.go`, `resolution_purity_test.go`
and the resolution parts of `execute_test.go`.

**Out of scope:** rendering (`render.go`, `format.go`, `topics.go`, `inline.go`,
`pager.go`), the shell-integration surface, and flag binding in `options.go` — all three
were reviewed and every finding is closed. Touch them only where resolution reaches in.

## Guardrails — obey these literally

1. **Read-only review. Do not modify, create or delete any file in the repository.**
2. **Never run the example binary** without both `CLIHELP_NO_AUTO_COMPLETION=1` and a
   sandboxed `HOME`, `XDG_CONFIG_HOME` and `XDG_DATA_HOME`. Prefer not running it.
3. **Never write to the real home directory.** No network. No `make bump`, `git push`,
   `git tag` or `git commit`.
4. `go test ./...`, `go vet`, `staticcheck` and throwaway programs under `$TMPDIR` are
   encouraged.

`AGENTS.md` outranks your taste: 80-line functions, table-driven tests, backward
compatibility (this is a published module with a `retract` directive).

## Seeded pressure points

Each is falsifiable by reading the code and, where possible, by a runnable proof.

1. **`isHelpToken` short-circuits on the literal `"h"` before it checks for a command
   prefix.** `resolve.go:38-47`: `arg == "help" || arg == "h"` returns true immediately,
   while the abbreviation branch below carefully requires
   `len(filterCommandsByPrefix(cmds, arg)) == 0`. So with `AbbrevCommands` enabled and a
   command named `hello`, does `app h` render help rather than resolving — or reporting an
   ambiguity — while `app he` resolves the command? Is `h` reserved even when the author
   never asked for it?
2. **`helpPath` takes `args[idx+1:]` raw.** `resolve.go:395-398`. What does
   `app help --verbose deploy` produce, and what does the help renderer do with a leading
   `--verbose` in a command path? Same question for `app help -- deploy` and
   `app help ""`.
3. **An empty argument.** `filterCommandsByPrefix(cmds, "")` matches every command.
   `isHelpToken` guards `arg == ""`, but `matchCommand` does not. What does
   `app ""` do under `AbbrevCommands` — an ambiguity error naming every command, a
   silent match, or something worse? A shell can produce an empty argument easily
   (`app "$UNSET"`).
4. **Shortcut resolution only happens at the root, and "root" is `res.cmd == nil`.**
   `resolve.go:359-370`, `resolve.go:409`. Is there any argument shape where a shortcut is
   matched at a depth it should not be, or missed at the root? What happens when a
   shortcut and a top-level command share a name, or when a shortcut is `Hidden`?
5. **The arity probe is rebuilt only after a command matches.** `resolve.go:380,428`. A
   command's `PersistentOptions` therefore do not count for flags written *before* that
   command's name. Is that the intended rule, and does it match what pflag later accepts?
   Consider `app --cmd-persistent-flag cmd` versus `app cmd --cmd-persistent-flag`.
6. **`addFlagArity` invents `--flag-<short>` for a short-only option.** `resolve.go:263`.
   Does `bindHelper` really register that name, and can a user type it? If not, it is a
   phantom flag in the arity map that could make resolution skip an argument pflag will
   reject.
7. **`matchAbbrevCommand` and aliases.** `resolve.go:142-163` plus
   `filterCommandsByPrefix`. When a prefix matches one command by name and a *different*
   command by alias, is that an ambiguity or a silent pick? Is the same command counted
   twice when both its name and an alias match?
8. **`checkUnknownCommand` versus a positional argument.** `resolve.go:206-223`. A command
   that takes positional arguments must not have its first argument mistaken for an unknown
   subcommand. Which commands get the error, and is the rule "has subcommands" or something
   weaker?
9. **`suggestCommand` and `levenshtein`.** `resolve.go:440-505`. Is the distance bounded
   before the matrix is allocated? What is the cost for a very long typo against many
   commands — and can a user supply one? Does `min3`/the matrix handle empty strings and
   multi-byte runes, or does it index bytes?
10. **`resolution.indices` and `remaining`.** `resolve.go:225-233`, `resolve.go:405,432`.
    `remaining` is `leading` (skipped flags) followed by the rest. Does that reordering ever
    change pflag's interpretation — for example when a skipped flag's value looks like a
    flag, or when `--` appears later?
11. **Purity.** `resolution_purity_test.go` exists because an earlier probe reset a running
    program's own flag variables. Does anything in resolution still mutate `App`, a
    `Command`, or a caller's memory? `filterCommandsByPrefix` returns `*Command` pointers
    into the caller's slice — can a caller mutate through them?
12. **`lookupCommandPath` versus `resolveCommandPath`.** Two different traversals
    (`resolve.go:51-82` and `:378`). Do they agree about abbreviations, shortcuts, hidden
    commands and aliases? A disagreement means `help <path>` and running `<path>` resolve
    differently.

## Required output

Per finding: ID and one-line title; **severity and confidence rated separately**;
`file:line` citations you actually opened; the failure path end to end with concrete
inputs; a runnable proof where the claim is behavioural; and a concrete fix. A finding
without a fix is an opinion — drop it.

Also report **what you checked and found sound**. Disproved hypotheses are results.
