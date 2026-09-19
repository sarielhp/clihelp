# Deep review: command resolution (`resolve.go`, 2026-09-18)

> **Reading this later.** Items below struck through have since been settled; the note
> after each says how. The current state of the public API is
> `review/api-surface-2026-09-18.md`, and `docs/how-clihelp-decides.md` is the
> mechanism this library actually implements today.

**Scope:** `resolve.go` in full — 506 lines with zero citations across the three previous
deep reviews — plus `leading_flags_test.go`, `resolution_purity_test.go` and the resolution
parts of `execute_test.go`. Prompt: `prompts/review-2026-09-18-resolve.md`. Three
reviewers. Every high and medium finding below was reproduced by the author of this
document with an independent proof.

---

## One sentence

Resolution's hard parts — the pflag handoff, argument ordering, purity — are correct and
well defended; what fails is the soft edges, and it fails the same way six times: the
command silently does not run and the program exits 0.

## Executive summary

**Six separate inputs make a program print its help and exit successfully instead of doing
what was asked.** A calling script sees success every time. They are independent defects
with one shape:

| input | what happens |
|---|---|
| `app --no-cache build` | `--no-cache` is bound by pflag but unknown to the arity probe |
| `app remote --level 2 add` | a grouping command's own flag is not skipped |
| `app typo` (verbs in `Shortcuts`) | the unknown-command check never runs |
| `app -- junk` | the check is only on the path resolution did not take |
| `app echo help` | a leaf command cannot be handed the word `help` |
| `app d` (ambiguous) | a shortcut silently breaks the tie and runs something else |

The last of those is the one to fix first: with `AbbrevCommands` on, commands `deploy` and
`destroy`, and a shortcut `dance`, typing `app d` **runs `dance`**. The ambiguity error is
computed and then discarded.

The other theme is that **two traversals disagree**. `lookupCommandPath` and
`resolveCommandPath` order exact-match, abbreviation and shortcut differently, so `app dep`
runs `deploy` while `app help dep` documents a different command — the help you read is not
for the thing that runs.

Against that, the parts that were most at risk are sound, and the evidence is strong:
20,000 randomised argument vectors show `remaining` is exactly the input with the matched
command positions deleted — nothing dropped, duplicated or reordered — and a differential
against a real `pflag.FlagSet` across every option constructor found **zero phantom entries
and zero arity disagreements**. Resolution mutates nothing.

**The test suite is the weakest part.** 45 mutations, **17 survived — a 38% escape rate**.
`min3` has 100% statement coverage and can be made to return a non-minimum with the suite
still green. I reproduced that myself after first misreading my own run.

---

## High

### H1 — A toggle's negative spelling before the command name silently runs nothing  **[verified]**

`resolve.go:249-265`. `addFlagArity` registers only the long names `parseFlagSpec` finds in
the spec string. `bindToggle` (`options.go:451-472`) registers more: it derives `no-<base>`
for the toggle base *and* `no-<name>` for every positive long name. So pflag binds
`--no-cache`, `--no-colour`, `--no-verbose` and the arity probe knows none of them.

Resolution stops at the unknown flag, the command name after it survives into `remaining`
as a positional, `targetCmd` is nil, and the global help renders with exit 0.

Measured, four toggle shapes, `BoolToggle` in `PersistentOptions`:

```
--cache            → --no-cache      FAIL   command never runs, exit 0
--[no-]color, --colour → --no-colour FAIL
--verbose          → --no-verbose    FAIL
--[no-]color       → --no-color      PASS   ← the only shape any test covers
```

`--no-colour` is not obscure: `CHANGES.md:191` advertises it and `flag_spec_test.go:80`
asserts it — but only ever *after* the command name, where resolution never has to skip it.
`Audit`/`ValidateExamples` cannot catch it either: the scratch FlagSet accepts the flag and
treats the command name as a harmless positional.

**Fix.** `addFlagArity` must name everything `bindToggle`/`bindHelper` register:
`Option.toggle` is the discriminator and `aliasFlagName` already exists. With the fix the
pflag differential goes empty in all four configurations.

---

## Medium

### M1 — A shortcut abbreviation silently breaks an ambiguity among real commands  **[verified]**

`resolve.go:359-370`. `matchCommandOrShortcut` returns early only when `matched != nil`, so
an ambiguity *error* from `matchAbbrevCommand` falls through and a shortcut prefix-match
returns with a nil error. Commands `deploy` and `destroy`, shortcut `dance`,
`AbbrevCommands` on: **`app d` runs `dance`.** `lookupCommandPath` does the same at
`resolve.go:67`, so `help d` agrees — consistently wrong.

### M2 — `help X` and `X` resolve to different commands  **[verified]**

`resolve.go:58-73` versus `:345-370`. The two traversals order their strategies
differently: `lookupCommandPath` tries exact command → **exact shortcut** → abbreviation;
`resolveCommandPath` tries exact command → **abbreviation** → shortcut. With a command
`deploy` and a shortcut `dep`, `app dep` runs `deploy` while `app help dep` renders the
shortcut's page. The documentation the user reads describes something other than what runs.

**Fix.** One resolver with an explicit exact-before-abbrev order, used by both. It closes
M1 at the same time.

### M3 — An app whose verbs are all `Shortcuts` accepts any typo and exits 0  **[verified]**

`resolve.go:207`. `checkUnknownCommand`'s gate is `len(currentCommands) > 0`, and at the
root `currentCommands` is `a.Commands` only — `App.Shortcuts` are invisible to it. So an
app that puts its verbs in `Shortcuts` never runs the check at all: `app typo` prints the
global help, `err == nil`. A mistyped shortcut also loses its suggestion.

### M4 — A grouping command's own options swallow its subcommand  **[verified]**

`resolve.go:270-271, 277-288`. A command's non-persistent `Options` are deliberately kept
out of the arity probe on the grounds that they cannot precede their own command — true,
but the loop keeps using that probe for flags written *after* the command name, where they
can. Worse than reported: the flag is unusable in **both** positions.

```
app remote --level 2 add   → 127 bytes of help, exit 0, subcommand never runs
app remote add --level 2   → unknown flag: --level
```

### M5 — A leaf command can never be handed the word `help`  **[verified]**

`resolve.go:32-48`, run at every depth by `:391`. The abbreviation guard
`len(filterCommandsByPrefix(cmds, arg)) == 0` is *trivially satisfied* when the node has no
subcommands — which is exactly a leaf command taking positional arguments. And
`resolve.go:40` reserves the bare `h` before the guard is reached at all.

```
abbrev on:  app echo h    app echo hel     → echo's help page, ctx.Args == []
abbrev off: app echo help                  → same
```

There is no escape, because `isHelpToken` runs before the `-`/`--` branch. Separately, with
`AbbrevCommands` on and a command `hello`, `app h` renders help while `app hel` runs the
command — the shorter abbreviation resolves to something different from the longer one.

### M6 — Hidden commands and their full paths are disclosed in suggestions  **[verified]**

`resolve.go:165-184`. The library has an explicit, tested policy that a hidden command is
never suggested — `suggestCommand` says so in a comment and `TestSuggestionsSkipHiddenCommands`
enforces it. `findSubcommandPaths` checks `Hidden` on the visited node only, so a visible
subcommand under a hidden parent passes:

```
unknown command "wipe" for "testcli". Did you mean "internal wipe"?
```

A hidden *leaf* is correctly withheld, which is what makes this an inconsistency rather
than a policy. Hidden is what authors use for deprecated, internal and destructive
commands, and this hands the user the exact invocation.

### M7 — A flag after `help` turns the whole path into "unknown help topic"  **[reported]**

`resolve.go:394` appends `args[idx+1:]` raw. Flags *before* `help` work; flags *after* it do
not, and the order is the user's arbitrary choice:

```
app --verbose help deploy   → deploy help
app help --verbose deploy   → unknown help topic "--verbose deploy"
app help -- deploy          → unknown help topic "-- deploy"
```

**Fix.** Reuse the arity probe `resolveCommandPath` already holds, so a flag *and its value*
are dropped. A lone token at the root must be kept as written, because `-v` and `--version`
are themselves help topics.

### M8 — `--` turns a typo into a successful help screen  **[verified]**

`resolve.go:413`. The unknown-command check lives only on the `matched == nil` path. When
the loop exits through the flag branch instead — `--` is never "known", nor is any
unrecognised flag — the following word is never examined. `app junk` errors; `app -- junk`
exits 0.

### M9 — The suggester walks a full edit-distance matrix against every command and alias  **[measured]**

`resolve.go:440-493`. `bestDist` starts at 3 and only strictly smaller distances win, so a
candidate whose length differs by 3 or more can never be chosen — but the matrix is built
anyway, and `[]rune(strings.ToLower(typed))` is re-allocated once per candidate.

My measurement, 50 commands with aliases: 1KB → 1.4ms, 64KB → 87ms, **1MB → 1.38s**. The
reviewer's, at Linux's 128KB single-argument limit: 228ms and **75MB allocated** — and that
path runs on **every TAB press** through `__complete`, not just on a typo.

**Fix.** `levenshtein(a,b) >= |runes(a)-runes(b)|`, so a rune-length pre-check is exact and
cannot change any result. Measured ~3,800× faster at 128KB, allocation to ~zero. It must be
rune counts, not bytes, or `日本語`/`日本` would be wrongly skipped.

### M10 — 17 of 45 mutations survive the test suite  **[verified in part]**

A 38% escape rate. The sharpest, each independently proved non-equivalent:

| mutation | what it would break |
|---|---|
| `min3` returns a non-minimum | `buildd` stops suggesting `build` — **100% statement coverage** |
| drop the insertion cost from `levenshtein` | `app rn` confidently suggests `build` |
| index bytes instead of runes | multi-byte typos lose their suggestion |
| drop `strings.ToLower` | `app BUILD` stops suggesting `build` |
| `filterCommandsByPrefix` stops skipping `Hidden` | abbreviations resolve to hidden commands |
| `findSubcommandPaths` stops skipping `Hidden` | M6, undetected |
| `matchCommandOrShortcut` consults shortcuts at every depth | a root shortcut resolves under a subcommand |

I reproduced the `min3` one myself. **I first misread it as killed**, because the run also
showed an unrelated failure; the baseline showed the same failure, and with that removed
both baseline and mutant pass. Every typo in the suite is a same-length edit — no test uses
a typo with a missing or extra character, which is why four different ways of breaking the
distance function go unnoticed.

---

## Low

| ID | Finding |
|---|---|
| L1 | `app help ""` renders the **global flags reference** when `AbbrevCommands` is on, because `strings.HasPrefix("flags", "")` is true (`resolve.go:86`). With abbreviation off the same input correctly errors, so the behaviour flips on an unrelated switch. |
| L2 | `checkUnknownCommand` reads `a.Name` directly (`resolve.go:208`) where the rest of the file goes through `appName(a)`, so an app with no `Name` reports `unknown command "nope" for ""`. |
| L3 | `suggestCommand("")` recommends any command of two runes or fewer, because `levenshtein("", name) == len(name)` and the threshold is 3. `app "$UNSET"` → `Did you mean "up"?`, manufactured from nothing. |
| L4 | A shortcut's subcommand executes correctly with the shortcut's persistent flags, but `ancestorsForPath` (`clihelp.go:260-275`) has no shortcut fallback, so `app help sc sub` renders a page omitting flags that command accepts. |
| L5 | `handleComplete` propagates the resolution error by design, but only the bash template redirects stderr (`completion_templates.go:29`); zsh and fish do not, so an ambiguous half-typed line prints a multi-line error over the user's prompt on TAB. |
| L6 | `TestExecuteSubcommandLocationSuggestion/"sibling typo takes priority over deep command"` names the opposite of what the code does — `findSubcommandPaths` is tried first and returns from inside its branch — on an input that cannot exercise the contest. True but vacuous. |
| L7 | `matchAbbrevCommand` lists `cmd.Name` for every ambiguous candidate, so a prefix matching an *alias* prints a name the user's prefix does not match. |
| L8 | `resolution_purity_test.go` contains no purity test. Its three tests are about re-entrancy and silence; the real purity test is `TestResolutionDoesNotBindOptions` in `leading_flags_test.go`, and it covers three entry points, not `ValidateExamples`, `RenderGlobal`, `RenderMan` or `__explain`. |

---

## Ordered roadmap

**Now — the silent exit-0 class**

1. **H1 — `addFlagArity` names every spelling `bindToggle` registers.** The one high finding,
   and the fix is contained.
2. **M1 + M2 together — one resolver with an explicit exact-before-abbrev order**, used by
   both traversals. They are the same code; fixing either alone leaves the other disagreeing.
3. **M3 + M8 — close the two holes in the unknown-command check**: `Shortcuts` at the root
   (copying before appending, so the caller's backing array is not written), and a
   post-parse check for a leftover positional the command cannot use.
4. **M4 — a grouping command's leftover positional is an error, not a help page.** This is
   also the general safety net: it converts the residue of H1 and M8 from a silent exit-0
   into a visible error.

**Next — the ones that are wrong rather than silent**

5. **M5 — `help` is universal only where nothing else can consume it.** Keep the exact word
   `help` working at every depth, which is the git-like convention worth keeping; stop the
   ambiguous prefixes being greedy, and stop reserving `h` at a node with no subcommands.
6. **M6 — track hidden ancestry in `findSubcommandPaths`**, and extend
   `TestSuggestionsSkipHiddenCommands` to cover it.
7. **M7 — drop flags and their values from `helpPath`**, keeping a lone root token as
   written so `app help -v` still works.
8. **M9 — the rune-length pre-check**, which also removes L3 for free.

**Later — the test surface**

9. **A table test on `levenshtein` and `suggestCommand` directly**, with typos of unequal
   length. It kills six of the seventeen survivors on its own.
10. **A table test on `isHelpToken`** over `{h, he, hel, help, ""} × {abbrev on, off} ×
    {with and without a command starting in h}` — the behaviour is currently pinned in
    neither direction.
11. **Rename `resolution_purity_test.go` to what it is**, and extend the real purity test to
    the entry points it misses, with a structural snapshot of the command tree.
12. **L1, L2, L4, L5, L6, L7** — each one to five lines.

---

## Disproved — recorded because a disproved hypothesis is a result

- **Seed 6, the `--flag-<short>` phantom.** Disproved twice, independently. `bindHelper`
  really registers it, it is not hidden, and a user can type it: `app --flag-S x go` runs
  `go` with the value bound. The pflag differential found **no** entry in the probe that
  pflag does not bind, and **no** arity disagreement, across every constructor, every spec
  shape, both `ExtendedHelpFlag` settings, at the root and after a command.
- **Seed 10, the `remaining` reordering.** Disproved as a defect over 20,000 randomised
  argument vectors: `remaining` is always exactly `args` with the positions in `res.indices`
  deleted, in order. The structure guarantees it — the loop can only advance past flags,
  which land in `leading` in order, or command tokens, which are recorded. `--` never enters
  `leading`; `--` as a flag's *value* is consumed as one, matching pflag.
- **Seed 11, purity.** Sound. Nothing in `resolve.go` assigns to any `App` or `Command`
  field, the probe map is freshly allocated per call, and `res.remaining`, `helpPath` and
  `res.path` are fresh allocations. Verified by mutating `remaining[0]` in all 20,000 trials
  and re-comparing the caller's `args`. The residual exposure — `*Command` pointers into the
  caller's slice reaching user code as `Context.Command` — is inherent to the exported API,
  not something resolution introduces.
- **Seed 5, the arity probe rebuilt only after a match.** Sound, and the disagreement is
  loud rather than silent: a command's persistent flag written before its name is rejected
  by pflag with `unknown flag`, which matches cobra and is enforced consistently in both
  directions. Only the diagnosis is thin — it never hints that the flag must follow the
  command name.
- **Seed 7, double counting and name-vs-alias ambiguity.** Sound. A command matching by both
  its name and an alias is appended once; a prefix matching one command by name and another
  by alias is reported as an ambiguity, not silently picked.
- **Seed 8, positional arguments versus unknown subcommands.** Sound. The rule is "this node
  has subcommands **and** has no `Run`", which is exactly right, not something weaker.
- **Seed 4, shortcut depth.** Sound. `res.cmd == nil` is true exactly while no command has
  matched; skipped leading flags do not set it. No argument shape matches a shortcut below
  the root or misses one at it. A shortcut sharing a name with a command loses to the
  command in both traversals; a hidden shortcut behaves exactly like a hidden command.
- **`levenshtein` itself is correct** — rune-indexed, case-folded, empty strings handled,
  and its rolling arrays sized off the *command name*, not the typed word. The allocation
  blow-up in M9 is the per-candidate `[]rune(ToLower(typed))`, not the matrix.
- **`min3` is correct as written.** Only its test coverage is absent.
- **`res.path` fidelity.** Both resolvers record `cmd.Name`, never the alias or abbreviation
  typed, so a path handed to `RenderCommand` always re-resolves.
- **`leading_flags_test.go` is strong work.** `res.ancestors`, `res.indices`, the probe
  rebuild, `scanLeadingFlag`'s 1-versus-2, the shorthand-cluster arity and
  `matchAbbrevCommand`'s ambiguity error were each mutated and each killed.

---

## A note on method

Two reviewers independently hit a failing `TestAutoInstallCompletionOnExecute` and both
correctly diagnosed it as an environment artifact rather than their own mutation. It was
neither: it was a fix I had reported as landed in a previous session and which never did —
a scripted edit asserted a unique match, the pattern appeared twice, the assertion threw
before the write, and I described it as fixed. It is corrected in `51b2c92`. An edit is
confirmed by reading the result, not by the absence of a complaint.

One reviewer also flagged that a sibling was writing Go files into a shared scratchpad
directory it had created. It did not corrupt the results — the survivors were re-run in a
private tree and all 17 reproduced — but concurrent reviewers sharing one path is a real
hazard for mutation work, and future fan-outs should give each agent its own directory.

## Status

**Every finding in this document is fixed**, on branch
`fix/resolve-review-2026-09-18`, one commit per block of the roadmap, each with regression
tests whose teeth were checked against the unfixed behaviour. `make check` and `make audit`
are green.

**The mutation escape rate is the measure that matters here, and it moved.** Ten of the
seventeen survivors — chosen as the ones with a user-visible consequence — were re-run
against the new tests: **ten of ten are now killed**, including `min3`, which has 100%
statement coverage and could previously be made to return a non-minimum with the suite
green. Two required tests I had not written on the first pass (`filterCommandsByPrefix`'s
two policy decisions, and the rule that shortcuts are matched only at the root); an
eleventh apparent survivor turned out to be a mistake in my own harness — the mutation was
a no-op — which is its own reminder that a surviving mutant must be proved non-equivalent
before it is believed.

One deviation from the roadmap, recorded rather than quietly taken. The roadmap said to
**keep the exact word `help` universal at every depth**. The implementation does not: at a
node with no subcommands the word belongs to the command. The reason is that the finding's
own complaint was that an `echo`- or `grep`-shaped command can never be handed the word,
and "universal at every depth" does not fix that — it only narrows it. A leaf has nothing
below it to document, and `app echo --help` and `app help echo` both still reach the same
page, so the third route was costing a word and buying nothing. `help` stays universal
wherever there *is* something to explain: the application's own page at the root, whether
or not it has any commands, and a group's page below it.

Two things the fix pass found that the review did not:

- **The root is not a leaf.** Making a node with no subcommands own the word broke
  `app help version` for an application with no `Commands` at all, which the existing tests
  were right to expect. The rule needed the root to be special, and the first two attempts
  at expressing that re-broke the abbreviation case in the other direction.
- **`isHelpToken` needed to know where it was.** It took the command list and a flag, and
  inferred "root" from neither. The caller knows; it passes it now.
