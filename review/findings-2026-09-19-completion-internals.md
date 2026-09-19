# Findings: why `Param.Complete` took three revisions

*2026-09-19, against 0.3.39 (`171c36b`). A companion to
`plan-2026-09-19-param-complete.md`. The plan says how to build the feature; this says
why building it needed three tries and two external reviews, and what in the repository
would have prevented that.*

---

## 1. The record

Ten defects were found across revisions 1 and 2 of the plan, by two reviewers reading
the same source. Classified by what the author got wrong:

| # | Defect | Class |
|---|---|---|
| 1 | Guard did not return after a valueless value-flag; `scanLeadingFlag` reports it as known | internals |
| 2 | Counting map built from `leadingFlagArity`, which omits the command's own `Options` | internals |
| 3 | Counting map built from `CollectOptions`, which drops `Hidden` | internals |
| 4 | `Variadic` scraped from `Param.Name`, contradicting "never parse Name" two sentences earlier | design |
| 5 | Candidate description fell back to `p.Description`, repeating one line under twenty candidates | taste |
| 6 | `--move` cited as a value-taking flag; `Optional` sets `arity = arityFlag` | **usage** |
| 7 | Typed nil appended into `leadingFlagArity`; panics on every root completion | internals |
| 8 | Flag-value guard ran above the `--` check, so it fired on a positional past the terminator | internals |
| 9 | Claimed `completePrevFlagValue` handles a shorthand cluster; it compares the whole word | internals |
| 10 | No `Audit` rule for `Variadic` on a non-final parameter, which is silently inert | design |

**Nine of ten are not about how to use the library.** They are about its internals: four
near-identical collectors of one fact, and the exact consumption rules of
`scanLeadingFlag`. Exactly one — #6 — is a usage error.

That ratio is the finding. The natural conclusion from an episode like this is "the
library must be clearer about how to use it." The record does not support it, and §2
shows the documentation already covers the one case that would have.

## 2. The documentation already said it. The chain to it is broken at one link.

`docs/how-clihelp-decides.md` covers three of these defects directly:

- **§8a** (line 127), *"Whether a flag's value may be omitted"*, uses `"--move [From]"`
  as its literal worked example and states the rule defect #6 violated: *"An equals sign
  is required for the value. With `--move x` there is no way to tell the value from the
  next positional argument."*
- **Invariant 7** (line 193): *"Everything after `--` is a positional argument."* That is
  the rule defect #8 violated, and revision 1 violated it a second way by offering flags
  there.
- **§9** (line 152) states that completion excludes hidden options — adjacent to defect
  #3, though not identical to it.

So the content exists, and is better than what a review would have produced from
scratch. The problem is that nobody reached it. The intended path is deliberate and
well built:

```
llms.txt  ──"Start here"──>  docs/how-clihelp-decides.md
   ▲
   └── "The index an LLM tool reads first, in the llms.txt convention"
```

`docs/index.md` says of that same page: *"Read this before writing code that produces
help, usage, suggestions or argument errors."*

**`AGENTS.md` mentions neither `llms.txt` nor `docs/`.** Not once, in the whole file. It
covers build commands, the `tools/` scripts, sizing limits, API stability and version
management — everything about how to *change* the repository, and nothing about where
the explanation of its mechanism lives. An agent working in this repository reads
`AGENTS.md`, finds no pointer, and goes to the source, which is precisely how nine
internals defects get made.

The chain is one link short. Adding that link is the cheapest fix in this document by a
wide margin.

> **Action.** A "Start here" block at the top of `AGENTS.md`: read `llms.txt` first; for
> anything touching resolution, flags, completion or help, read
> `docs/how-clihelp-decides.md` before the source. Name the invariants list explicitly —
> it is the part that reads as reference and behaves as specification.

## 3. Four statements of one fact

One question — *given argv and a resolved command, which words will pflag consume?* — is
answered in four places, with deliberately different inclusion rules:

| | includes own `Options` | includes `Hidden` | source |
|---|---|---|---|
| `leadingFlagArity` | **no**, by design | yes | `resolve.go:427` |
| `CollectOptions` | yes | **no**, by design | `clihelp.go:401` |
| `scanLeadingFlag` / `scanShorthandCluster` | n/a — consumption rules | n/a | `resolve.go:447`, `:473` |
| `flagTakesValue` / `opt.arity` | n/a — per-option truth | n/a | `resolve.go:363` |

Every one of these has a correct, well-argued doc comment explaining why it excludes what
it excludes. **None of them references any other.** There is no page, comment or test
that puts the table above in one place, so an author needing "the arity map" picks
whichever name they met first. Defects #2, #3 and #7 are three different wrong picks, and
each compiled and passed the suite.

The difference being *deliberate* is what makes this invisible: it reads as design rather
than duplication, so no audit flags it.

This repository has already named this exact failure mode, twice:

> *"The alternative was a second field on Command saying the same thing in other words,
> and two statements of one fact drift: helpFlagNames and bindHelpFlags were exactly
> that, and the drift made Audit reject examples that ran."* — `args.go:16`

> *"An escape hatch reached for by different people for the same standard idiom is a
> missing constructor, and `Binder` is the evidence trail that shows which."*
> — `CHANGES.md`, 0.3.38

The arity collectors are the third instance, and the one still outstanding.

> **Action.** Collapse them behind one named entry point whose parameter is the choice —
> `flagArity(scope)` with `scopeResolution` and `scopePositional`, or a small `argScanner`
> carrying the map — so selecting the wrong rule set becomes a visible decision rather
> than a lucky import. Put the table above in its doc comment.
>
> Failing that, at minimum: cross-reference the doc comments, and add an agreement test.
> `resolve_agreement_test.go` is the precedent — it exists because two things that had to
> agree drifted — and the assertion here is that the map and pflag's real parse segment
> the same corpus of argv lines identically.

## 4. The invariants are prose, not tests

Invariant 7 — *"Everything after `--` is a positional argument"* — is true of `Execute`.
It was **false of `handleComplete`** before this change: `completeFlags` ran on any
`toComplete` beginning with `-`, terminator or not, and `completeSubcommands` ran
unconditionally. The documentation asserted a property one entry point did not hold, for
as long as both have existed, and nothing noticed.

That is not a documentation defect. It is an untested specification. The list under
*"Invariants you can rely on"* (line 177) is eight crisp, checkable properties — already
written, already agreed, and never executed. It is a test table that has been sitting in
a Markdown file.

> **Action.** A table-driven test that drives each invariant through *every* entry point
> that must honour it: `Execute`, `resolveCommand`, `handleComplete`, `Audit`, the man
> page and the Markdown generator. Invariant 7 alone catches defect #8, catches revision
> 1's flags-after-`--`, and catches the pre-existing bug in `handleComplete` that neither
> reviewer would have found if this feature had not happened to pass through it.
>
> The general form is worth naming: **a documented invariant that no test drives is a
> claim, and the entry point added last is where it stops being true.**

## 5. A plan about internals is a low-fidelity medium

Separate from the library, and the reason three revisions were needed rather than one.

The plan was written by reading source and reasoning about it in prose. Every one of the
ten defects dies on `go test` in a single run — #7 with a panic on the first invocation,
#2, #3 and #8 as wrong slot indices in a table, #9 as an empty buffer where a candidate
was expected. Two rounds of careful human-scale review found them one at a time, in prose,
over hours.

This mirrors what the `mail_cli` work established twice in one week: a nil reporter and a
literal `\n` in eight files both shipped past a green suite and were caught by running
the binary. Reviewing an artifact is weaker than executing it, and the gap widens the
closer the work sits to the machine. Completion internals are about as close as this
library gets.

> **Action, for the implementing session.** Land §4 of the plan — the test table — first,
> against current `main`, before any design change. Most of it will be red, and the
> pattern of red is worth more than the plan's prose: it says which of these behaviours
> are already wrong today, independent of the feature.

## 6. What this episode does *not* show

Recorded because it is the conclusion the shape of the story invites, and it is wrong.

**It does not show that the library is hard to use.** Nine of ten defects were made while
modifying it, not while using it. The single usage error had its exact answer, with its
exact example, written down already (§2). Three applications are built on this library;
none of these defects could occur in any of them, because none of them can reach
`leadingFlagArity`.

**It does not show that `Param.Complete` is the wrong feature.** It sharpens the case for
it. §2 of the plan argues the feature belongs upstream because a downstream
reimplementation must re-derive the flag-arity rules. Nine internals defects, made by two
careful readers *with the source open*, is that argument's evidence. A downstream author
gets them wrong with nobody reviewing, and ships it.

**It does not call for more documentation.** The documentation that would have prevented
the one preventable defect exists and is good. What is missing is a pointer to it from
where an agent starts (§2), and an executable form of the part that reads as
specification (§4).

## 7. Ranked

| | Action | Cost | Prevents |
|---|---|---|---|
| 1 | "Start here" pointer in `AGENTS.md` → `llms.txt` → `how-clihelp-decides.md` | minutes | #6, and the general failure of reading source before mechanism |
| 2 | Invariants as an executed table across every entry point | an afternoon | #8, revision 1's flags-after-`--`, the pre-existing `handleComplete` bug |
| 3 | One arity entry point with an explicit scope, doc-comment table, agreement test | a day | #2, #3, #7 |
| 4 | Test table before design, in the implementing session | none — it is work already planned, reordered | all ten, at the point of writing rather than review |
