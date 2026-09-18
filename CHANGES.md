# Changelog

All notable changes to `clihelp` will be documented in this file.

## [0.3.30] - 2026-09-18

### Internal
- **Thirteen Tests That Could Not Fail.** Six mutation surveys all landed between 29% and 32%, and the recurring cause was not missing tests but tests present and inert. A scan for them found four that contained nothing capable of failing: `TestExecuteGlobalFlagsBound` **logged** the two values it existed to check and asserted only that `Execute` returned no error, so the global flags could have stopped binding to their targets entirely; `FuzzBindFlagSpec`'s comment said "a second bind of the same spec must be refused, not panic" and then discarded both results, so every name-collision check in `options.go` could have been deleted with a million executions still green — that was one of the four mutations the `options.go` survey could not score; `TestPrintError` called it twice and never looked at the output, and, having no `Stderr` of its own, printed `Error: sample error` into the middle of every run of the suite; and `TestHelpFlagsInExamplesActuallyRun` checked only the error, so a help flag that quietly printed nothing passed as working.
- **Nine more asserted properties of output without asserting there was output.** Every one is of the form "no line does X", and every one holds vacuously of no output at all. This was measured, not guessed: `reflowMargin` — the function that writes every wrapped line in this library — was made to return without writing, and the width, hyperlink-balance and whitespace properties all still passed; the same was true of `RenderCommand` made to render nothing. `requireRendered` now sits in the two shared assertion helpers and in the tests that loop over lines themselves, and with it all nine fail as they should.
- **`TestEveryTestCanFail` makes the class self-policing.** It parses every `_test.go` file in the library and its subpackages and fails on a test function containing no `t.Error`/`t.Fatal`, no assertion helper, and no call to another test helper that can itself fail — the transitive rule, so that a test written as three named steps is not called inert. The exception list is empty and adding to it requires a written reason, like `sharedHelperExceptions` beside it. The precedent is `TestSharedHelperExceptionsAreSorted`, which used `t.Logf` where it meant `t.Errorf`, could not fail, and on being fixed immediately caught that the list it guards was unsorted.
- The new assertion in `FuzzBindFlagSpec` was checked over 1.22 million executions before being kept, and it holds.

## [0.3.29] - 2026-09-18

### Internal
- **The Rendering Path Judged by Mutation: 27 of 38, Then 38 of 38.** `render.go`, `format.go` and `inline.go` are the largest surface never put under mutation, and they were the one case where the result was genuinely uncertain: they already have a deep review behind them (v0.3.19–0.3.22) and are covered by golden-output assertions from `example/` and `example/mail_cli_fake`, whose oracle is a second renderer. If goldens were worth more than targeted tests, this is where it would show. **It does not.** The escape rate was 29%, indistinguishable from `install.go`/`man.go` at 29% and `doc/`/`tree/` at 32%. A golden test pins what the sample output happens to exercise; it says nothing about the branch beside it.
- What could be broken with the whole suite green: a tab could advance one column instead of to the next eight-column stop, putting a description one column out for every tab before it; `clampIndent` could narrow a description column below the six-column floor, turning a narrow terminal's flag list into one character per row; the width a redirected `--help` is laid out at — 70 columns, the value every piped help page in every program built on this library uses — was asserted nowhere; `App.NoColor` could stop reaching the renderer, because `fatih/color` already suppresses colour in a test process and so the assertion was vacuous; a default the author had already written into a description could be printed a second time; and a bullet could be recognised without the space after it, so that `*emphasis* text` opened with a list marker and took a hanging indent. In `inline.go`, both rules that stop an emphasis marker binding across a space could be removed, which turns a bare `**` or a multiplication sign in prose into the start of bold.
- Four things were restructured rather than merely asserted. `conciseBudget` now takes the terminal height as an argument (`conciseBudgetFor`), because `Options.height()` returns zero for every writer a test can hand it, so the short-terminal branch was unreachable from the suite — the same seam `doc/`'s `markdownHashOf` uses for its format version. The wrapping test asserts a property rather than an output: **a line may exceed its width only if it holds a single word**, which is true of every wrapper and false of the one that judges emptiness by width. The concise-budget test covers output that ends in a blank line, which is where passing through and truncating differ, since the truncating path trims trailing newlines and that final blank line is deliberate spacing. And `inline`'s doc comment claimed that colour-free output spells a link as `text (url)`; it has not since v0.3.22, when that was changed to the label alone — only `man.go` asks for the URL form, and it asks explicitly.
- **38 of 38 mutations are killed now**, the first survey to reach a clean sweep. Five of the thirty-eight could not be scored on the first run — four did not compile once mutated, and one matched in three places — and fixing the harness rather than dropping them is what turned up the unreachable short-terminal branch.

### Changed
- **`AutoInstallCompletion` Is Now `AutoRefreshIntegration`.** The old name described something the field cannot do. It has not installed anything since the unattended path was narrowed to refreshing what the user already put there, and it refreshes the man page as well as the completion script — so an author reading `AutoInstallCompletion: true` would reasonably conclude that setting it installs completions for their users, which is exactly the misreading that produced the original overreach. Both fields are honoured and either enables the behaviour; the old one is deprecated, not removed, and a test pairs them so they cannot drift. `autoinstall.go` is `autorefresh.go`.
- The decision to keep the behaviour on by default was re-examined and confirmed. Without it, a user who upgrades the program keeps the completion script and man page they installed against the previous version — forever, with no prompt to re-run setup and no symptom to notice, because the integration file is read at shell startup and nobody re-reads it. The refresh is what makes a one-time setup stay correct across upgrades. Measured cost on a warm cache: 6.4 µs per run, five stats and two partial reads, against a Go process start of roughly a millisecond.

### Internal
- **The Home-Directory Guard Failed a Release With Nothing Wrong.** It fingerprinted each watched file by size and modification time, so any process on the machine touching `~/.bashrc` inside the twenty seconds the suite runs failed the build — and it did, reporting the file as modified while its content was byte for byte what it had been, with none of clihelp's markers in it. What the guard protects is the user's content, so content is what it compares now: the bytes are hashed, a changed hash still fails the suite, and a modification time that moves with the bytes unchanged is printed as a note. An audit of every test that reaches a write path confirmed all of them sandbox `HOME`; the earlier `~/.bashrc` report during the v0.3.26 release had the same signature — identical content, no markers — and may have had the same cause.

## [0.3.28] - 2026-09-18

### Internal
- **The File That Names Every Flag Was the Worst-Tested in the Library.** A mutation survey of `options.go` — the 728 lines that turn a spec string like `--tag <v>, -t, -T` into pflag registrations — killed **11 of 22**, a 50% escape rate against 29% for `install.go`/`man.go` and 32% for `doc/`/`tree/`. No defect: every survivor was a missing assertion, which is the point of doing this on code whose correctness the rest of the library assumes. Among them, `parseFlagSpec` could stop recognising the short `-[no-]` toggle marker, keep an empty long name, or stop treating an all-uppercase word as a value placeholder — the last of which feeds the arity probe that decides where the command name is. `aliasFlagName` could collide for two short-only options, or drop the option's name entirely, and `--tag`'s synthetic `--tag-alias-T` could be registered visibly instead of hidden, putting an internal spelling in the help. The group annotation that makes every spelling of an option one option — what `Option.Group` and "was this flag set" both read — could be dropped from the primary flag. And four mutations could not even be scored, because the harness only mutated one of the two binder paths.
- The survey is **25 of 26** now, over a harness that mutates both binder paths. The remaining survivor is equivalent and is left surviving: `addFlagArity` negates the toggle's base name as well as every long name the spec declared, and `parseFlagSpec` always puts the base among the long names, so the first is redundant today. The two were folded into one loop, and a test records the invariant — if the base ever stops being a long name, that test says so rather than the redundancy being quietly tidied away.
- Two survivors were redundancy rather than gaps, and are asserted against their contract rather than through behaviour. `optionGroup` falls back to the flag's own name, which is a safety net for flags clihelp did not bind, and that fallback made tagging the primary flag unobservable through any parse; the annotation is now asserted directly. `toggleVal.IsBoolFlag` never runs during parsing — pflag decides whether `--color` consumes the next argument from `NoOptDefVal` — but it is part of the `Value` contract that cobra and completion generators read, so it is asserted against the interface.

## [0.3.27] - 2026-09-18

### Fixed
- **Two of the Three Programs This Library Runs Could Hang It.** A mutation survey of `install.go` and `man.go` turned up a real defect: `manPageElsewhere`'s deadline did not bound it. The context kills `man`, but `Output` waits for its standard output to close, and a `man` that has forked leaves a child holding that pipe — measured at a full 30 seconds against a 3-second deadline. This is the same defect the pager had, fixed there without looking for the other places a program is run. There are three, and the third — `doc/`'s `git check-ignore`, which keeps the hash sidecar out of version control — had neither a deadline nor a `WaitDelay`, so a repository on an unreachable mount or a `git` that stops for credentials would hang documentation generation indefinitely. All three are bounded now, and the two new ones have a test that drives them with a stub that never returns.

### Internal
- The same survey found five assertion gaps, two with a user-visible consequence. zsh's `$ZDOTDIR` was unasserted, so the sourcing line could have gone to a file the user's shell never reads while the install reported success; and `integrationHasKeys` could answer "yes" always, which would add the Alt-H key binding to an integration the user had installed with `--no-keys`, during an automatic refresh they did not ask for. The other three: `upsertBlock`'s changed-or-not answer, which is what the install report tells the user it did; the uninstalled marker, which is what keeps a deliberate uninstall uninstalled; and `manPageIsCurrent`'s version check, without which an upgrade would never refresh a page.
- One mutation survives and is left surviving: replacing the uninstalled marker's filename. Both the writer and the reader go through the same function, so a consistent change to it alters no behaviour — only the name of a file in the user's configuration directory. Pinning that name would be a golden assertion on an internal detail, and a surviving mutant has to be shown to change something observable before it earns a test.

## [0.3.26] - 2026-09-18

### Internal
- **The Test Suite Wrote to the Developer's Own `~/.bashrc`, and the Guard Caught It Mid-Release.** `shell_test.go` calls `InstallShellIntegration` and `UninstallShellIntegration` — asking each entry point to reject a shell it cannot write for — and was the only one of the seven test files reaching a write path that did not sandbox `HOME`. It normally returns before writing, which is why nine attempts to reproduce it failed, and why it survived until a `make bump` happened to catch it. The file was not damaged: no block was added and it still parses. It sandboxes now, and the package's `TestMain` sandboxes `HOME` for every test rather than trusting each one to remember — the fingerprint is the alarm, this is the lock. The alarm only reports afterwards, and afterwards turned out to be from inside a release.

## [0.3.25] - 2026-09-18

### Internal
- **`doc/` and `tree/` Judged by Mutation, Not Coverage.** 22 mutations, each proved to change an observable value; **7 survived the suite**, all of them in two places. `tree`'s continuation columns — the box-drawing that makes a tree a tree, and the only decision about whether a parent's vertical bar continues past a child — could be broken in all three of their branches with nothing failing, because the existing tests assert the *child* prefixes and those are built elsewhere. `markdownHash` is the cache key `RenderMarkdown` compares against the value stored beside the pages, and its determinism was asserted while its sensitivity was not: dropping the format version from it, so that raising `markdownFormatVersion` would regenerate nothing, and dropping the page names, so that renaming a page or two pages swapping content would go unnoticed, both passed. `tree` also rendered hidden commands and untruncated descriptions with no test objecting.
- The hash takes its version as an argument now (`markdownHashOf`), so the version's contribution can be asserted without editing the constant — the same seam the atomic writer uses for its failure paths. New tests cover the tree's structure at every branch of the continuation logic, its filtering and truncation, and the hash against all five ways its input can change. **22 of 22 mutations are now killed**, stable across three consecutive runs.
- One intermediate run reported 21 of 22 and I could not reproduce it; the three runs since are identical. Recorded rather than rounded away.

## [0.3.24] - 2026-09-18

### Fixed
- **`Audit` Rejected Examples That Run** - the example validator bound its own idea of the help flags, `--help` with shorthand `-h`, while the real flag set binds `--help-concise` with `-h` and `--help` with `-H` when `App.ExtendedHelpFlag` is set. So an example line using `--help-concise`, or `-H` on an application that offers it, was reported as `invalid flag in example` — by `Audit`, which the README recommends running in CI, for a line that works at run time. The validator calls the same binder the real path uses, so the two cannot drift again, and a test pins `helpFlagNames` against what `bindHelpFlags` actually binds: they are two lists of the same thing, one for what resolution skips and one for what pflag parses, and that pair is what the validator drifted from.
- The package documentation named only `-h` and `--help` as the flags an application must not register itself. It names all four now, under both `ExtendedHelpFlag` settings, and `docs/flags-and-options.md` explains what `--help-concise` is: `pflag` cannot bind a shorthand without a long name, so `-h` needs one, and it is hidden because `-h` is the spelling to type. It is a real flag all the same — accepted on a command line and in an `Example` — and one of the four an option must not collide with, so a collision is now something an author can look up rather than discover from a parse error.

## [0.3.23] - 2026-09-18

Fixes from the deep review of command resolution
(`review/findings-2026-09-18-resolve.md`) — 506 lines that the three previous reviews had
all left out of scope. Every finding is closed.

### Fixed
- **Six Ways a Command Silently Did Not Run** - each of these printed the help page and exited 0, so a calling script saw success. A toggle's negative spelling written before the command name (`app --no-cache build`) was bound by pflag and unknown to the arity probe, so the command name became a positional — three of the four toggle spellings failed, and the one that worked is the only one any test covered, because `--no-colour` is only ever asserted *after* the command name where resolution never has to skip it. A grouping command's own option consumed its subcommand's name (`app remote --level 2 add`), and the flag was unusable in the other position too. An application whose verbs all live in `App.Shortcuts` never ran the unknown-command check at all, so any typo was accepted. `app -- junk` was accepted where `app junk` errors, because the check sat only on the path resolution did not take. A leaf command could not be handed the word `help`. And an ambiguity between real commands was silently broken by a shortcut that shared the prefix — with commands `deploy` and `destroy` and a shortcut `dance`, typing `app d` ran `dance`.
- **`help X` and `X` Resolved to Different Commands** - the two traversals ordered exact-match, abbreviation and shortcut differently, so `app dep` ran the command `deploy` while `app help dep` documented the shortcut `dep`. There is one resolver now, exact before abbreviated, used by both.
- **Hidden Commands Were Named in Suggestions** - the "did you mean" search checked `Hidden` on the visited command only, so a visible subcommand under a hidden parent was offered with the hidden parent's name spelled out — the exact invocation, for the commands authors hide because they are deprecated, internal or destructive. The library has an explicit, tested policy against this; only one of its two suggesters honoured it.
- **A Flag After `help` Broke the Help Path** - `app help --verbose deploy` reported "unknown help topic" while `app --verbose help deploy` worked, and the order is the user's arbitrary choice. Flags and their values are dropped from the path now, with a lone root token kept as written because `-v` and `--version` are themselves topics.
- **A Mistyped Argument Was Walked in Full, Once Per Command and Alias** - the edit-distance matrix was built with no bound on the typed word, and that word comes from argv. Only a distance below three can win, so a rune-length difference of three or more decides the answer before any work: 128 KB — the most a single argument can be — went from 228ms and 75MB of garbage to microseconds, on a path that also runs on every press of Tab through `__complete`.
- **Smaller Corrections** - an empty help topic is an error rather than a prefix match on `flags`, which made the behaviour depend on whether `AbbrevCommands` was set; an empty argument no longer manufactures a suggestion; an unknown command in an app with no `Name` no longer reports `for ""`; a shortcut's subcommand help shows the shortcut's persistent flags; an ambiguity names the spelling the user's prefix matched rather than one it does not; and the zsh and fish completion scripts redirect the program's stderr as bash already did, so a resolution error no longer prints over the prompt mid-edit.

### Internal
- The resolution test suite had a 38% mutation escape rate: 17 of 45 mutations survived, including one against `min3`, which has 100% statement coverage and could be made to return a non-minimum with the suite still green. Every typo in the suite was a same-length edit, which is why four separate ways of breaking the distance function went unnoticed. New table tests cover `levenshtein`, `min3`, `suggestCommand` and `isHelpToken` directly, with typos of unequal length, multi-byte runes and case folding.
- `resolution_purity_test.go` contained no purity test — its three tests are about re-entrancy and silence, which it is now named for. The real one covers the entry points the existing check missed and snapshots the command tree, because resolution hands out `*Command` pointers into the caller's own slice.
- A subtest named "sibling typo takes priority over deep command" asserted the opposite of what the code does, on an input that could not exercise the contest. Renamed, with the real contest asserted beside it.
- Two disproofs worth keeping: 20,000 randomised argument vectors show the arguments handed to pflag are exactly the input with the matched command positions deleted — nothing dropped, duplicated or reordered — and a differential against a real `pflag.FlagSet` across every option constructor found no flag the probe knows that pflag does not bind, and no arity disagreement.

## [0.3.22] - 2026-09-18

Fixes from the deep review of the terminal rendering path
(`review/findings-2026-09-18-rendering.md`) — about 2,000 lines that neither previous
review had opened. Every finding is closed. Each fix ships with a regression test whose
teeth were checked against the unfixed behaviour.

### Security
- **An Author String Could Steer the User's Terminal** - every string an application supplies — descriptions, usage lines, notes, headings, parameter names — was copied to the terminal byte for byte, ESC included, and the author's URL was interpolated straight into clihelp's own OSC 8 payload. A description containing `\x1b[2J` cleared the screen mid-help, `\x1b[?25l` hid the cursor for good, `\x1b[?1049h` switched to the alternate buffer, and an ST or BEL inside a URL closed clihelp's own escape so that everything after it executed. The `?`-marker forms were worse than passthrough: the stripper could not match them, so they reached the terminal *and* corrupted the column arithmetic. These are not always compile-time literals — descriptions come from config files, embedded JSON and translation catalogues, `__clihelp` exists so a packager can set up a program whose author mounted nothing, and the completion generator pushes the same strings into a generated shell script. Control bytes other than newline and tab are now replaced, and a URL is percent-encoded before it enters an escape.

### Fixed
- **The Pager Has Never Paged** - `os/exec` hands a child a descriptor only when `Stdout` is an `*os.File`, and the destination was wrapped in a counting writer, so the pager always got a pipe, saw a non-terminal stdout and degraded to `cat`. `App.Pager`, `Options.Pager`, the `-R` plumbing and the help text promising a paged manual amounted to a fork and an extra copy. The counting writer was added deliberately, with a comment and a regression test, to tell "the pager never ran" from "it ran and quit" — a correct fix for a real bug that silently disabled the feature it was protecting, and nothing noticed because no test starts a terminal.
- **A Pager That Printed Nothing Swallowed the Help** - `PAGER=true` exits 0 having written nothing, which was reported as success, so the fallback was skipped and the user saw no help at all. `PAGER=""` — the conventional way to disable paging — launched `less` anyway, as did `PAGER=cat`.
- **Paging Counted Newlines, Not Screen Rows** - a page of twenty logical lines that each wrap to three occupies sixty rows and counted as twenty, so help that overflowed the screen printed unpaged. A `$PAGER` that backgrounds anything held the output pipe open and hung the program with no bound. Ctrl-C killed the parent and orphaned a pager that traps SIGINT, which kept the terminal and wrote over the shell prompt.
- **A Wrapped Hyperlink Was Left Open** - the wrapper splits the rendered string, and the per-line reset that `fatih/color` appends is an SGR reset, which does not close an OSC 8 hyperlink. Anything that truncates by line then dropped the terminator — `Alt-H` does exactly that — and the terminal went on hyperlinking every cell it drew, including the shell prompt, after the program had exited.
- **`-h` Ignored Its Own 24-Line Promise** - stated in `README.md`, `llms.txt`, `docs/flags-and-options.md` and twice in the cobra comparison, where it is the library's claimed differentiator. A command with thirty flags rendered 92 lines, byte-identical to `--help`, and `RenderGlobal` ignored the concise tier entirely. The bound is enforced now, with `Options.ConciseMaxLines` to change or remove it.
- **Output Said Something False About Grouped Commands** - a command declaring no `Group` was printed under the previous group's heading, and the return to that group printed no heading. The same for flags in the manual page. Ungrouped entries now get a heading of their own.
- **Colour Died at the First Nested Sequence** - every nested colour closes with a full SGR reset, so wrapping a whole line in the body colour meant the prefix's closer killed it for the rest of that row while the wrapped continuations kept it: every flags table had its first description line uncoloured. Colour is applied to the text, not the line.
- **Escapes Survived `NO_COLOR`** - the theme's colours vanish when output is not a terminal, but the inline renderer's own escapes did not, so a redirected help page was not plain text. `Options.NoColor` and `App.NoColor` make it a per-render and per-application choice; the example app's own `--no-color` flag was bound and never read.
- **Narrow Terminals Were Unreadable** - the usage line was never wrapped at all, and the description column never looked at the terminal, so a 13-column flag name at width 20 left about three columns for the text. Both are fixed, and the manual page's nested lists no longer sit outdented from their own headings.
- **Smaller Corrections** - `Options.Theme` layers onto `App.Theme` rather than replacing it; `Usage:` promises `<subcommand>` when subcommands are documented declaratively; a link's blank-line decision is made on what is drawn; `FirstSentence` no longer cuts inside markdown; a backslash escapes only punctuation, so `C:\temp\x` survives; `[](url)` falls back to the URL as its label; a manual-page heading renders its markdown; a tab measures the columns a terminal advances; a line of spaces is still a paragraph break; a heading is not printed with no rows under it; the link parser is no longer quadratic.

### Internal
- `acarl005/stripansi` is gone. The test suite used it as its default stripper — the one `StripANSI`'s doc comment exists to warn about, which does not merely miss an OSC 8 sequence but eats seven bytes out of the middle of it.
- The rendering path gains its first property tests: that a rendered line fits the width it was given, measured in display columns rather than runes, at five widths including 20 and 40, which no test had ever rendered at. Emoji, ZWJ sequences and combining marks appear in a test for the first time.
- `tools/check.sh` was red when the review started, from a guard test committed an hour earlier: `go/parser.ParseDir` is deprecated and staticcheck is fatal there. The review gate treats staticcheck as advisory, which is the gap it lived in.

## [0.3.21] - 2026-09-18

### Fixed
- **The Startup-File Lock Was Barely a Lock** - `flock` is held on an inode rather than a name, and the release function unlinked the lock file, so a waiter already blocked on the old inode could acquire it at the moment a newcomer's `O_CREATE` produced a fresh inode and locked that one instead — two writers, each certain it held the lock. A deterministic test (twelve workers, forty read-modify-writes each) reports **478 of 480 updates lost** against the old code; the install test's "2 of 8 blocks lost" was the mild case, and it only ever appeared under a full `-race` run, never on its own. The lock file is no longer removed.
- **Tree Rendering Mis-measured Hyperlinks and Multi-line Descriptions** - when `doc/` and `tree/` were split into subpackages, four text helpers were copied rather than shared, and three had drifted. `tree`'s width measurement used a stripper that does not understand OSC sequences, so an OSC 8 hyperlink — which this library emits — measured 22 columns instead of 4. Its `firstSentence` checked for `". "` before truncating at a line break, so a description whose first full stop fell on a later line came back with the newline still in it, and the reflow that receives it assumes one line. `VisualWidth`, `StripANSI` and `SubcommandList` are exported, both subpackages defer to them, and each has a test asserting it still asks rather than answering.
- **The Manual-Page Lookup Could Hang, and Collided With Itself** - `man -w` ran with no deadline, so an unreachable `MANPATH` entry would hang an install silently; it now has three seconds. It also compared paths as strings, and `man` prints the path it resolved — so clihelp's own page, reached through a symlinked `$XDG_DATA_HOME`, was reported as a foreign collision and the user was told to pass `--force` to overwrite their own file. `os.SameFile` settles it.
- **Manual Pages Were Not Reproducible** - the `.TH` line carried today's date, so two builds of the same program produced different bytes. `SOURCE_DATE_EPOCH` is honoured, per the reproducible-builds convention. `manPageVersion` is raised so installed pages are replaced.

### Internal
- **The Atomic Write's Contract Is Now Checked Rather Than Claimed.** Every syscall in `writeFileAtomically` can fail, and each failure owes the caller three things: the error returned, the original file untouched, and no `.tmp-*` sibling left in the user's home. None of that was reachable from a test while `os` was called directly. A small `fileOps` seam passes the operations in — with no package-level variable to swap, since a mutable global would be visible to every other test — and all six failure points are driven, with all three obligations asserted at each.
- `tools/bump-version.sh` regenerates both example documentation trees as part of a release. The pages embed `App.Version`, so they were one version stale after every bump, which is how they drifted far enough to hide two generator defects.
- `docs/completion.md` describes the lock file, since it is a fourth artifact appearing in the user's home.

## [0.3.20] - 2026-09-17

### Fixed
- **The Generated Docs Disagreed With `--help` About Subcommands** - `clihelp` prefers an explicit `SubcommandEntries` over walking the `Subcommands` tree, which is the point of that field: it documents subcommands the tree does not carry. When `doc` became a subpackage the helper was copied without that preference, so a documented-only entry — `mail_cli whitelist list` — appeared in the terminal help and was missing from the generated page. The two lists agree again, and entries that do have a page are still linked, matched on the command word so that `add <email>` finds `whitelist-add.md`.
- **A Generated Table Header Read `clihelp.Command`** - a `Command` → `clihelp.Command` rename, applied when `doc` became a subpackage, reached two Markdown table headers inside string literals. It also reached the golden string in the test that would have caught it, so the test agreed with the mangled output. Both are corrected.

### Internal
- `docs/clihelp/` and `docs/mail_cli_fake/` are regenerated. They had drifted from the example apps, which is how the two defects above surfaced: the fix for stale generated documentation is to regenerate it, and regenerating showed the generator was wrong.
- **A Failed Release Left the Tree Half-Bumped, and Said Nothing.** `tools/bump-version.sh` writes the new version into `VERSION`, `example/main.go` and `clihelp.go` before running the check, because the check verifies they agree — but it discarded the check's output to `/dev/null` and, on failure, left all three modified with nothing committed. The next run then bumped again from there and skipped a version outright, with no message anywhere to suggest looking. The version files are restored on any failure up to the commit, the check's output is kept and its last 25 lines printed with the path to the rest, and each step after the commit reports for itself with the command to finish by hand — a tag or a push that fails no longer looks like a tag or a push that worked. Both paths were exercised in a throwaway clone.
- The race-detector timeout goes from 120s to 300s in `tools/check.sh` and the `test` target. 120s is comfortable against a warm build cache and not obviously so against a cold one, which is the state a release runs in most often.

## [0.3.19] - 2026-09-17

### Internal
- **The Shell-Integration Surface Is Layered Now, and Acyclic.** This is A1 from the review, the one finding deliberately left open through the fix pass. Five files formed a complete dependency cycle: `completion.go` reached into `install.go` nine ways, `install.go` reached back six, and `protocol.go` sat at the top while `install.go` reached up into it for one string constant. There are four layers now — leaves (`shell.go`, `names.go`, `versions.go`, `atomicwrite.go`, `completion_templates.go`), generators that produce text and touch no files (`completion.go`, `explain.go`, `man.go`), installers that write them (`install.go`, `autoinstall.go`), and the command layer over those (`protocol.go`, `completion_command.go`) — with no edge pointing up. `completion.go` is 372 lines rather than 724, and `AGENTS.md` records the rule so the cycle does not grow back.
- **`autoinstall.go` Is Its Own File.** The unattended path is the only code here that writes to a home directory unasked, and it is held to a much narrower rule than the explicit installers, so it no longer sits in the middle of a file about something else.

### Fixed
- **Entry Points Disagreed About What a Shell Argument Means** - "which shell, and can we write for it?" was answered in seven places. Six read an empty argument as "the shell the user is running" and `GenShellIntegration` alone rejected it. The error came in two wordings, one of which never listed the supported shells, and five sites reported an undetectable shell as `unsupported shell ""` — which names neither the problem nor the fix. One resolver answers for all of them now, and an unset `$SHELL` says so and asks for a name.

## [0.3.18] - 2026-09-17

### Internal
- **The One-Step Install Is Now Proved in zsh and fish, Not Only bash.** Every live test in this repository drove bash — the shell whose install path is the simplest, with one rc file and no completion system to initialise — while zsh and fish, where the mechanism actually differs (`ZDOTDIR`, a `conf.d` drop-in, `compinit` ordering), were covered only at the Go level. `install_live_test.go` starts a real zsh and a real fish against a sandboxed home and asks what the install left them: the completion registered with `compdef`, the ZLE widget defined, the registry carrying both entry forms, and — in the order plugin managers actually produce — the deferred-`compinit` retry firing and then removing itself. It checks uninstall too, including the manual page.

### Fixed
- **Fish Lost Everything After the First Line of a Multi-Line Command** - the Alt-H dispatcher read the buffer with `(commandline)`, which splits a multi-line command line into one element per line and then handed those to `__explain` as separate arguments, of which only the first was read. The continuation lines of exactly the long command someone would press Alt-H on were dropped. It now reads the buffer with `string collect` and passes it as one argument, which is what bash and zsh have always done.
- **Bash Inserted a Completion Candidate Containing a Space Unquoted** - completing `prod east` put it on the command line verbatim, where the shell re-parsed it as two arguments. Candidates are quoted for insertion now; zsh and fish already quoted theirs.
- **The Wrapper Dropped Every Argument After a Tab** - the generated wrapper reconstructs the wrapped command line by cutting its own name off the front, and it cut at the first space. A tab between the name and the arguments matched nothing, so the remainder was emptied and the bare command was explained as though nothing had been typed after it. A run of several spaces was mangled the same way. It splits on any run of blanks now.

These are the review's SH-14, SH-9 and S9. Along with D4, fixed in 0.3.17, they were findings the roadmap never scheduled, so the fix pass never reached them; the review document now says so.

## [0.3.17] - 2026-09-17

### Changed
- **`install` Now Sets the Program Up Completely** - it installed the shell integration and left the manual page to a second command, `__clihelp manpage --install`, which the user had to know existed. Since an installed man page is also what makes Alt-H answer natively in zsh and fish, the half most people would never run was the half that finished the feature. `install` now writes the page too, `--no-man` declines it the way `--no-keys` declines the key binding, and `uninstall` removes it — its report promises to undo everything, and a man page left behind is one the user is never told about again. A page already installed system-wide is still refused rather than overwritten, but it is now reported as a warning instead of failing a shell setup that otherwise succeeded.

### Fixed
- **The Removal Instructions Named a Command That May Not Exist** - every install printed, and wrote permanently into the user's startup file, "run `<app> completion uninstall`". That subcommand exists only when the application author mounted `CompletionCommand()` — which is the exact case `__clihelp` was added to cover, so a user who set the program up through `__clihelp install` was handed instructions that do not work, with one copy of them in `~/.bashrc` forever. The hint now names whichever entry point that program actually has.

- **`AutoInstallCompletion` Created a File Nobody Asked For** - with no shell integration present, an ordinary program run fell through to the older completion-script location and *created* a script there, so running a program for the first time installed something into the user's home unprompted. The automatic path now only ever rewrites a file that already exists, and only one carrying clihelp's marker. The flag is the application author's choice while the file lands in the user's home directory; creating is what `__clihelp install` is for. Two tests asserted the old behaviour and now assert the rule.

## [0.3.16] - 2026-09-17

Follow-ups to the shell-integration review: three defects in the surface
0.3.15 shipped, and the versioning that has to be in place before the
Alt-H protocol changes again.

### Fixed
- **`set -o vi` Killed Alt-H** - a key binding belongs to one keymap, and all three snippets bound only whichever keymap happened to be current when they were sourced. bash now binds `emacs-standard`, `vi-insert` and `vi-command`, zsh binds `emacs`, `viins` and `vicmd`, and fish binds `default` and `insert`. Setting `CLIHELP_NO_KEY_BINDINGS` before the snippet is sourced declines the key entirely — zsh's `run-help` and fish's man-page binding stay yours — without giving up completion.
- **Zsh Completion Was Silently Not Registered Under a Deferred `compinit`** - the script registered with `compdef` or, when autoloaded, called itself; if neither held it did nothing and said nothing. A plugin manager that defers `compinit`, or an rc file that sources the bootstrap before it, landed here. A self-removing `precmd` hook now retries at the first prompt.
- **Filename Completion Differed in Every Shell** - bash fell back to filenames when the program offered no candidates, zsh offered nothing, and fish's `-f` forbade them outright, so an argument that is a path could not be completed at all in two shells out of three. zsh calls `_files` and fish gets a conditional rule; candidates and filenames are still never mixed. zsh's emptiness test was separately wrong — it measured the candidate array joined into one string.

### Changed
- **The Alt-H Registry Carries a Protocol Version** - `_clihelp_apps` entries are now `name:protocol`. The dispatcher is shared by every clihelp program in the shell, including ones built against other library versions, and nothing in the registry said what the program on the other end speaks; the first change to the `__explain` protocol would have been a silent misread. A bare name still means protocol 1, both forms are registered for one release, and a dispatcher that meets a protocol it does not know declines rather than calling.

### Internal
- The generated completion scripts move to `completion_templates.go`; they are shell programs, and they had pushed `GenZshCompletion` past the 80-line limit.
- `shell_harness_test.go` drives the generated zsh and fish code headlessly by stubbing `zle`, `bindkey`, `bind` and `commandline` — in all three shells a function shadows a builtin. Until now those two shells were only syntax-checked, which is why every defect above reached a release.
- The home-directory guard is extended: the `example` package gets its own sandbox (its app sets `AutoInstallCompletion`, so rendering a help page in a test installed into the real home), and every remaining subprocess in the root tests goes through `sandboxedCommand`.

## [0.3.15] - 2026-09-17

Fixes from the deep review of the shell-integration surface
(`review/findings-2026-09-17-shell-integration.md`). Every finding below was
reproduced before it was fixed, and each fix ships with a regression test whose
teeth were checked against the unfixed behaviour.

### Security
- **Every Release Before This One Is Retracted** - `go.mod` now carries `retract [v0.2.2, v0.3.14]`. The zsh Tab-press execution below has been present since v0.2.2, when the zsh generator was added — it is not a regression from the recent work, and every published version carries it. Deleting the tags would not have removed the code: the module proxy and checksum database keep what they have fetched, and `tools/bump-version.sh` has published each tag to them on release. Retraction is the mechanism that actually reaches a user: `go get` skips retracted versions and `go list -m -u` warns about one already pinned.
- **Arbitrary Code Execution on Tab, in Zsh** - the generated zsh script spliced the typed words into zsh's `_call_program`, whose body ends in `eval … "$argv[2,-1]"`. Typing `myapp $(command) <Tab>` executed the substitution, with nothing shown on screen. This is the 2026-09-17 audit's E1 in the shell that fix did not cover. The arguments now go through `${(q)}`, and `completionScriptVersion` is raised so installed scripts are replaced.
- **Executable Content Written Permanently Into `~/.bashrc`** - the startup-file line was built with Go's `%q`, which emits a *Go* literal in double quotes, where the shell still expands `$( )` and backticks. Combined with an unvalidated `App.Name` it put live code into the user's startup file, outliving the program. The line is now a single-quoted shell word, and one `safeAppName` gate validates every name that becomes a path, a shell symbol or an rc-file marker.
- **Alt-H Ran a File From the Current Directory** - the dispatcher matched its registry on the *basename* of the first word and then executed the word as typed, so `./myapp` in a cloned repo ran on a keystroke the user pressed because they had *not* decided to run it. Path-bearing words are now refused in all three shells.
- **The Generated Wrapper Mangled and Executed Its Arguments** - `escapeShellArg` tested a deny-list that missed the glob characters and the backslash, and its output was interpolated into a double-quoted context where the quoting was inert. `a[1]` matched a file on disk and a command substitution ran on Alt-H. The quoter is now an allow-list, and the wrapper quotes once into shell variables.

### Fixed
- **An Ordinary Program Run Edited Shell Startup Files** - `AutoInstallCompletion` called the full installer, so an upgrade re-added a block the user had deleted by hand and, with `ZDOTDIR` adopted after installing, created a `.zshrc` that had never existed. The refresh path now writes only the generated file; editing a startup file requires an explicit install. Uninstall also leaves a marker, because the auto path used to fall through and create a *different* artifact in a *different* directory — so uninstall did not stay uninstalled.
- **Files Clihelp Did Not Write Were Overwritten and Deleted** - the fish `conf.d` drop-in was written whole and removed unconditionally, and `AutoInstallCompletion` replaced hand-written completion scripts because "no marker" was read as "stale" rather than "someone else's". Both now require clihelp's marker.
- **The Rc-Block Markers Matched Anywhere in the File** - a `~/.bashrc` that merely *mentioned* the marker in a comment lost every line between the mention and the next end marker. Matching is now anchored to whole lines, an unterminated block is an error rather than a guess, and uninstall removes every copy.
- **A Symlinked Dotfile Was Replaced by a Regular File** - `rename(2)` replaces the link, so installing detached a dotfiles repo silently. Writes now resolve the path first, and the data and directory are flushed before returning.
- **Concurrent Installs Lost Blocks** - measured at 7 of 8 lost with every caller reporting success. The read-modify-write is now serialized with an advisory lock keyed on the startup file.
- **`__clihelp` Verbs Disagreed About Their Own Grammar** - `install bash --no-keys` installed the key binding the user had just declined, `manpage --install --uninstall` silently preferred one, and surplus arguments were discarded. One parser now serves every verb.
- **The Wrapper's Alt-H Branch Was Unreachable** - nothing ever registered a wrapper's name with the dispatcher, so documented behaviour could not occur. The printed registration now covers both the completion table and the registry.
- **A Hostile `App.Version` Corrupted the Man Page Header** - `.TH`'s arguments were Go-quoted, and `\"` starts a roff comment, so a version containing a quote truncated the header and dropped the section title. `man` warns about none of it.
- **A Replaced File Lost Its Owner** - the atomic replace installs a new inode, so a run under a different effective uid — `sudo -E myapp anything` is enough, since the auto path runs before command dispatch — took the user's own startup file away from them. Ownership is now carried across, and writing a file belonging to another user while running as root is refused. `CLIHELP_DEBUG` surfaces the errors the unattended path otherwise swallows.
- **The Test Suite Wrote Into the Developer's Home Directory** - six sites ran the example binary with the inherited environment. All are sandboxed, and `TestMain` now fingerprints every path this library can write to and fails the suite if anything changed.

### Changed
- **One Output-Stream Rule** - stdout carries what a script captures — a path, a generated script, a version, one per line — and stderr carries everything written for a person to read. The rule was asserted in three doc comments and broken by half the surface: `completion install` printed its report on stdout where its `__clihelp` twin printed a bare path, and `manpage --uninstall` printed an English sentence where a script expected a path. A visible command now behaves exactly like its `__clihelp` twin. Interactive users see no difference, since both streams reach the same terminal.
- **The Quality Gate Runs the Race Detector** - `AGENTS.md` has always required it and nothing ran it; `make test` and `make check` now do.
- **Names Are Validated** - an `App.Name` containing a space or a shell metacharacter, or no name at all, now produces an error from the generators and the install paths instead of a broken or dangerous script. Help rendering is unaffected.

## [0.3.14] - 2026-09-17

### Added
- **Manual Page Generation** - `__clihelp manpage` writes a roff manual page for the program — one page with a subsection per command, options, examples and notes — and `--install` puts it under `$XDG_DATA_HOME/man`, which is on man's default search path. This is more than documentation: zsh binds Alt-H to `run-help` and fish binds it to `__fish_man_page`, both of which call `man`, so a generated page makes Alt-H answer natively in those shells for a program that ships no manual. It does not replace clihelp's own binding, which explains the command line *as typed* rather than showing the whole manual. Installation refuses when a manual page for the program already exists elsewhere — which of two pages `man` shows is not predictable, and the one that loses is invisible — unless `--force` is given; `--uninstall` removes only a page clihelp generated. `clihelp.ManPageCommand()` offers the same thing as a visible command for authors who want one, and `AutoInstallCompletion` refreshes an installed page without ever creating one. New: `GenManPage`, `ManPagePath`, `InstallManPage`, `UninstallManPage`, `ManPageCommand`.

## [0.3.13] - 2026-09-17

### Fixed
- **`__clihelp -H`** - `-H` is clihelp's extended-help flag everywhere else, so it asks `__clihelp` for more as well: the reserved argument names, and the exact paths `install` would write on this machine. A hidden entry point cannot answer those through the ordinary help system. It is accepted whatever `App.ExtendedHelpFlag` says, since that field governs the application's flags and this is the library's own surface.
- **`__clihelp` Took `--help` for an Argument** - `__clihelp --help`, `-h` and `help` were unknown verbs, and a verb's own `--help` was read as its first argument: `__clihelp wrapper --help` generated a wrapper script named `--help`, and `__clihelp install --help` tried to install for a shell of that name. All four spellings (`-h`, `--help`, `help`, `-H`) now print the verb list, and a verb given one prints its own usage.

## [0.3.12] - 2026-09-17

Completes the 2026-09-17 deep review — every finding in `review/findings-2026-09-17.md` is fixed — and adds the shell-integration work that came out of it.

### Added
- **One-Command Shell Setup** - `completion install` now writes one generated file under the application's own configuration directory (`~/.config/<app>/shell/<shell>`), holding both the tab completion and the Alt-H key binding, and one permanent line in the shell's startup file that sources it — a marked, idempotent block in `~/.bashrc` or `~/.zshrc`, and a `conf.d` drop-in on fish, where nothing shared is touched. The line is a file test and a `source`, so nothing runs at shell startup; it names a fixed path and never changes again, because upgrades rewrite the file it points at. That also retires the `fpath=(...)` reminder zsh users needed, since a sourced script registers itself with `compdef`. `--no-keys` installs tab completion alone, `completion uninstall` removes the file and the block and leaves the rest of the startup file byte for byte as it was, and a completion script installed the old way into the shell's own directory is removed so that no shell loads two copies. `AutoInstallCompletion` refreshes an installed integration when a clihelp upgrade changes the templates, and never creates one or edits a startup file on its own. New: `InstallShellIntegration`, `UninstallShellIntegration`, `GenShellIntegration`, `IntegrationPath`.
- **`__clihelp`: Setup in Every Program** - `ExecuteContext` serves `__complete` before it looks at the command tree, so every clihelp program could always complete — but only a program whose author added `clihelp.CompletionCommand()` could be asked to install that completion. The reserved `__clihelp` argument closes the gap with five verbs — `version`, `install [--no-keys] [<shell>]`, `uninstall [<shell>]`, `keys [<shell>]` and `wrapper <name> [<args>...]` — available in every clihelp program. Hidden from help and completion output, documented, and inert unless invoked; a dotfiles script or a packager can now set up any clihelp program uniformly. `install` prints the installed path alone on stdout so it can be captured.
- **Wrapper Script Generation** - `__clihelp wrapper pd deploy` writes a wrapper that answers clihelp's protocol on behalf of the program it wraps, so shell completion and Alt-H keep working through it, and carries a `# clihelp-wraps:` marker in its second line — the convention pyenv and asdf use for their shims, readable without executing the script. The one line that registers the wrapper with the shell is printed to stderr, so the script itself can be redirected straight into a file. A wrapper is usually written by someone who is not the application's author, after it was built, which is why it describes itself rather than the application declaring wrappers it cannot know about. New: `GenWrapperScript`, and `clihelp.Version` reporting the library's own version.
- **Alt-H: Expand and Explain** - `completion keys [<shell>]` prints a shell snippet that binds Alt-H to two actions: it rewrites the abbreviated command names on the line to their full names (`podctl b d` becomes `podctl build deploy`, leaving flags, arguments, quoting and redirections exactly as typed), and it prints the help for the command that line names, capped at two thirds of the terminal height with a note saying how much was cut and where to read it. The binding acts only on command lines that begin with the application's own name; in zsh, where Alt-H is `run-help` by default, everything else is handed back to `run-help`. It is a separate snippet rather than part of the completion script because every shell loads completion scripts lazily, on the first `<Tab>` for that command. One dispatcher serves every clihelp program on the machine — a key binding is global to the shell, so a per-program binding would be replaced by the next program installed, leaving Alt-H silently dead for all but the last one; the dispatcher is versioned, so two programs shipping different clihelp releases cannot fight over the key. Alt-H is already zsh's `run-help` and fish's man-page key, so a command line belonging to no clihelp program is handed straight back to the shell's own handler. New: `App.Explain`, `GenKeyBindings`, and the `__explain` protocol call.

### Fixed
- **The Zsh Completion Script Could Not Be Sourced** - it ended with the bare call to its own completion function that an `$fpath` autoload needs, so `source <(myapp completion zsh)` — or a startup file reading the installed script — ran that function outside any completion context and printed `can only be called from completion function` four times at every shell start. The call is now guarded by `$funcstack[1]`, which holds the function's name when the file is autoloaded and the file's path when it is sourced; sourcing registers the completion with `compdef` instead. All three scripts are now tested for being silent when sourced.
- **Zsh and Fish Completion Scripts Were Rewritten on Every Run** - the staleness check looks for a `clihelp-completion-version` marker in the installed script, but only the Bash template carried one. For a zsh or fish user of an app with `AutoInstallCompletion`, the script was therefore always "stale" and reinstalled on *every single invocation* of the program. All three templates now carry the marker, so a current script is left alone.
- **Shell Menus Showed Whole Paragraphs of Markdown** - the completion protocol sent `Description` fields verbatim, so zsh's and fish's menus printed entire descriptions, markup and all: `**deep** — This is the [deep command](https://example.com/deep) at the root…`. Descriptions now go out as one plain line — markdown rendered away, whitespace collapsed, first sentence only, capped at 72 display columns.
- **`completion install` Promised a Key Nobody Bound** - the printed tip advertised "Alt-H for instant command help", which did nothing: bash has no default Alt-H binding, and zsh's `run-help` found no man page. It now points at `completion keys`, which makes the promise true.
- **A Failing Pager Threw the Help Away** - `os/exec` drains an `io.Reader` stdin as soon as the child starts, so the buffer that was both the pager's input and the fallback copy was empty by the time a failing pager returned: `PAGER=false` printed a blank help screen, and for output past the pipe buffer the fallback printed only the tail. The pager reads from its own reader now, and the fallback fires only when nothing reached the user. `less`'s `-R` is also no longer detected by finding the letter "r" anywhere in the arguments, which found one in `--clear-screen`.
- **Example Lines Were Rewritten Before Being Shown** - the colorizer returned its result through the inline-markdown renderer, which swallowed the backslashes of `--path C:\temp\x`, turned `'*.go'` into emphasis and ate the escape in `echo a\ b`; the renderer then word-wrapped long commands into two unpastable lines. Example lines are now colored and emitted exactly as written.
- **Validating Examples Overwrote the Program's Variables** - `ValidateExample` (and `Audit`, which the README recommends for CI) bound every option for real and parsed the example through it, and pflag writes both the declared default and the parsed value through the caller's pointer. Validation now binds to storage of its own, while still parsing and still checking values. It also applies `Option.Required`, and reports a binding error as itself instead of resurfacing it as "unknown flag" on an innocent example.
- **Shortcut Commands Could Not Be Run** - `App.Shortcuts` were offered by completion and listed in help, but command resolution never consulted them: `app <shortcut>` was "unknown command" and `help <shortcut>` printed nothing.
- **An Empty Argument Matched Everything** - an empty string is a prefix of every command name, so under `AbbrevCommands` `app ""` — what `app --filter $UNSET` expands to — ran the only command or was taken for a help request. It now names nothing and reaches the program as the argument it is.
- **A Category Command Skipped Half Its Lifecycle** - a command that only groups subcommands returned right after printing its help, so `PostRun` and `AfterRun` never ran although `BeforeRun` and `PreRun` had.
- **Help Layout Measured in Runes and Bytes** - two-column listings padded the first column by rune count while the column is measured in display width, so a wide (CJK) name pushed its description out of line; the command tree measured its indentation in bytes, and every box-drawing glyph it draws with is three bytes and one column, so descriptions were indented and wrapped as if the tree were two and a half times as wide as it is.
- **Rows With No Description Disappeared** - a command whose description was empty or had no visible width (a `[](url)` link, a stray `**`, a zero-width space) vanished from `--help` entirely, name and all.
- **`help flags` Suppressed Its Own Rows** - whether to synthesize `--help` and `--version` was decided by testing the raw flag spec for a substring, so a global `-v, --verbose` deleted the `--version` row and a `--host` deleted the `--help` row. A taken shorthand now narrows the synthesized row to its long name instead of removing it.
- **Inline Emphasis Ate Asterisks** - an unterminated `**` was consumed as an empty italic span, and `2 * 3 * 4` rendered as `2  3  4`, because emphasis was allowed to open and close on whitespace.
- **Example Scanners Disagreed With the Shell** - `#` or `//` anywhere inside a token started a comment in the colorizer, greying out the rest of a URL line; whitespace was tested byte-wise, so the 0xA0 continuation byte of `à` was read as a non-breaking space and split a token — and its color run — mid-character, emitting invalid UTF-8; and the validator cut the command at `" | "` and friends, ignoring redirections and unspaced operators, so `app logs > out.txt` was validated with `> out.txt` as two positional arguments.
- **Generated Markdown Could Not Be Restored, and Deleted Files It Did Not Write** - the generation gate compared only hashes, so a page deleted by hand was never written again; pruning removed every `.md` the pass had not just written, destroying a directory that already held documentation on the first run. The sidecar now records the generated page names, only those are pruned, and a missing page forces regeneration. Table cells escape `|` everywhere, including inside code spans, and `mdCode` fences a backtick instead of trying to escape it with a backslash.
- **Completion Installed for Shells That Have No Script** - `detectShell` turned any unknown or unset `$SHELL` into "bash", so a dash, ksh or nushell user got a Bash script written into their home. Installation is also atomic now, and `Close`'s error — where a full disk shows up — is no longer dropped. `CompletionPath` and `InstallCompletion` no longer panic on a nil `App`.
- **Bash Completion Shredded `--flag=` and Colons** - the generated script discarded the `words`/`cword` that `_init_completion` computed and sent the raw `COMP_WORDS`, which bash splits at every character of `COMP_WORDBREAKS`.
- **Smaller Corrections** - a resolution that failed part-way no longer discards the commands it did match, so a failing example keeps its command highlighted; `help db nonesuch` names the whole path rather than the word that resolved; hidden commands are no longer named in "Did you mean"; and `.gitignore` detection no longer reads the negated `!.clihelp-hash` as the rule it was about to add.

### Changed
- **`Audit` Is Stricter** - it now validates flag specs, inspects `App.PersistentOptions` and `App.GlobalFlags`, and reports collisions across the scopes that share one flag set. An app whose examples omit a required flag, or whose declarations collide, fails an audit that used to pass — and used to fail at runtime instead.
- **Command Resolution Moved to `resolve.go`** - `execute.go` had grown past the project's file-length warning threshold; resolution is a self-contained half of it. No API change.

## [0.3.11] - 2026-09-17

### Fixed
- **Every Spelling of an Option Is Now One Flag** - each alias in a flag spec was registered as an independent pflag flag that happened to write to the same variable, so the aliases did not share the state pflag keeps per flag. A `Required` option set through an alias was reported missing (`--title x` → `required flag(s) "name" not set`, with the target already set), a `StringSlice` kept only the values given under the last-used name (`--tag a --tag b -T c` → `[c]`), a relational validator written with shorthands (`MutuallyExclusive("-a", "-b")`) resolved nothing and enforced nothing, and a deprecation notice was missed when the user wrote an alias. Aliases now share the primary flag's value and are tagged with the option they belong to, so accumulation, required-flag detection, the relation validators and the deprecation warnings all see one option however it was spelled.
- **Relation Validators Accept Shorthands and Report Unknown Names** - `MutuallyExclusive`, `RequiredTogether`, `RequiredWith` and `RequiredIf` resolved names through `pflag.Lookup`, which indexes long names only. A constraint written with shorthands silently enforced nothing. Names are now resolved as shorthands too, and a name that no option on the command declares is reported as an error instead of quietly disabling the constraint.
- **Malformed Flag Specs Are Errors, Not Panics** - a shorthand of more than one ASCII character (`"-out <F>"`, a single missing dash) panicked inside `pflag.ShorthandLookup`, and a spec written without dashes (`"out <F>"`) bound nothing at all, leaving an option that silently did not exist. Both are now reported as errors when the option is bound, and `Audit` reports them statically.
- **Deprecation Warning Crash on a Short-Only Option** - `checkDeprecatedFlags` built an alias name from the first long name without checking there was one, so an `Option` with `Flags: "-a, -b"` and a `Deprecated` notice panicked with `index out of range [0]` as soon as the user set any flag. The check now works from the tag each flag carries and warns once per option, naming its primary spelling.
- **A Name Repeated Inside One Spec** - `"-a, -a"` registered the shorthand twice and panicked inside pflag's `AddFlag`; a repeated long or short name in a single spec is now an error. A fuzz target (`FuzzBindFlagSpec`) drives arbitrary specs through every typed constructor's binder to keep malformed input from reaching pflag unchecked.
- **Toggle Aliases and Short-Only Toggles** - `BoolToggle` bound only the base name and its `--no-` form, so extra long names in the spec were silently dropped; `"--[no-]color, --colour"` now binds `--colour` and `--no-colour` as well, and extra shorthands are bound too. A toggle spec with no long name to derive the negative spelling from is reported as an error instead of registering a nameless flag.
- **Audit Covers Every Binding Scope** - `Audit` checked flag collisions only within a single command and never looked at `App.PersistentOptions` or `App.GlobalFlags`, while `setupFlagSet` binds the app's options, the ancestors' persistent options and the command's own options into one flag set. A collision between those scopes made every invocation fail while `Audit` reported success. Collisions are now detected across the scopes that share a flag set, sibling commands may still reuse a name, and the reported command path no longer aliases its parent's backing array.

## [0.3.10] - 2026-09-17

### Fixed
- **Shell Completion Executed Its Own Candidates** - `handleComplete` wrote candidates and descriptions into the line-oriented completion protocol without escaping, so a newline in a `Description` or in an `Option.Complete` result forged extra records, and the generated bash script passed the candidate list to `compgen -W`, which performs command substitution on its words. A candidate containing `$(...)` — the realistic source being a `Complete` callback that lists files or remote names — executed when the user pressed Tab, with nothing shown on screen. Records now go through a sanitizer and the bash template appends each candidate literally after a prefix test. The template carries a version marker so the auto-install path replaces scripts generated before this fix.
- **Help Rendering Could Recurse Until the Process Died** - `resolveCommandPath` rendered help as a side effect of resolution, so an app with an example line such as `app help` re-entered the renderer through example colorization until it died with a stack overflow. Resolution now reports that help was requested and `ExecuteContext` performs the single dispatch, which also stops help text being emitted into the shell-completion stream and out of `ValidateExamples`/`Audit`.
- **Resolution Reset the Application's Own Flag Variables** - the arity probe added in 0.3.9 ran every `Option.Binder`, and pflag writes each declared default through the caller's pointer, so resolving arguments — including the resolution done while rendering a command's examples — overwrote a running program's parsed flag values with defaults. Arity is now recorded on `Option` by the typed constructors and read back from the flag spec, leaving consumer memory untouched.

## [0.3.9] - 2026-09-17

### Fixed
- **Global Flags Before a Subcommand** - Command resolution no longer stops at the first token beginning with `-`, so `app --global cmd args` resolves `cmd` and binds the command's own flags. Previously any global or persistent flag written before the subcommand left the command unresolved, which surfaced either as `unknown command "cmd"` or as the command's own flags being rejected (`unknown shorthand flag: 'n' in -n`) — while `cmd --help` still listed those flags, because help rendering never went through flag binding. Flag arity is read from pflag itself, so a flag's value is never mistaken for a command name: in `app -A last search`, `last` is the value of `-A` and `search` is the command. Unrecognized flags and `--` still end resolution, and a command's own non-persistent options still cannot precede it.

### Changed
- **Help Flags Before a Command** - `app --help cmd` and `app -h cmd` now render the help for `cmd` rather than the global help, matching `app cmd --help`.
- **Completion and Examples After Global Flags** - Shell completion resolves the command when global flags precede it, and example colorization marks command tokens by their actual position instead of assuming they lead the line.

## [0.3.8] - 2026-09-15

### Fixed
- **Race Condition in Paged Output** - Removed package-level `color.NoColor` mutation in `pageOutput`, eliminating data races when rendering help concurrently.
- **Tree Command Description Color** - Wired `bodyColor` in `tree.reflowTree` so tree descriptions correctly honor `Theme.Body`.
- **Code Hygiene & Shadowing** - Replaced custom `min` with Go builtin, renamed `min`/`max` parameters in `RangeArgs`, and resolved variable shadowing.

### Changed
- **Cognitive Complexity Flattening** - Decomposed `renderInline` in `inline.go`, `collectRenderFlags` in `topics.go`, and `promptForMissing` in `interactive.go` into focused helpers conforming to cognitive complexity thresholds (depth $\le 4$, branches $\le 15$).
- **Test Suite Partitioning** - Extracted help rendering unit tests from `clihelp_test.go` into `render_test.go` to keep all test files comfortably within sizing limits.

## [0.3.5] - 2026-09-08

### Added
- **Extended Command Documentation & Tiered Help** - Added `Command.LongDescription` for in-depth command documentation, `Note.Raw` to preserve preformatted text and verbatim spacing, and `App.ExtendedHelpFlag` for opt-in `-H` single-letter shortcut for extended help on commands and root.
- **Hanging Indentation for Lists** - `reflowSegment` detects bullet lists (`- `, `* `, `• `) and numbered lists (`1. `, `2. `, etc., including any leading whitespace) and wraps continuation lines with hanging indents aligned to the start of the list item text.
- **Raw & Preformatted Block Support** - `renderCommandNotes` and `RenderMan` support `Note.Raw` and fenced code blocks (```), bypassing reflow and space collapsing to preserve internal spacing and verbatim indentation.
- **Tiered Progressive Help (`-h` vs `--help` / `-H`)** - Differentiated concise help (`-h`) from extended help (`--help`, `help <cmd>`, or `-H`). Concise help suppresses `Notes`, uses `Description`, and outputs a clean footer hint pointing to extended documentation. Extended help renders full `LongDescription`, all parameters, flags, examples, and notes, with automatic `$PAGER` support.
- **Markdown Documentation Site Generation** - `doc.RenderMarkdown` renders `LongDescription` (preferring it over `Description`) and formats raw notes inside Markdown fenced code blocks.
- **UV-Style Command Listings** - Two-column command index tables in `RenderGlobal` and `RenderCommand` display clean, bare command names and aliases without argument or flag clutter, guaranteeing single-line scannability.
- **Command Tree Traversal (`App.Walk`)** - Programmatic depth-first traversal of all commands and nested subcommands with path slice isolation and early error-exit for testing, interface coverage, and static analysis.
- **Example App Testing Demonstration** - Added `example/main_test.go` demonstrating how consumer applications can test command coverage, leaf usage lines, example validity via `ValidateAllExamples`, and smoke-render all command help pages.
- **Sizing Audit Automation** - Added `tools/audit_lines.rb` and `make audit` target enforcing the 80-line function hard limit (with declarative builder exceptions and relaxed test limits) and file sizing comfort metrics.

### Changed
- **Modular Function Decompositions** - In-place decomposition of oversized functions in `execute.go`, `render.go`, `completion.go`, `examples.go`, `testing.go`, `format.go`, and `topics.go` to strictly adhere to the 80-line function limit.
- **Modularized Test Suites** - Separated monolithic completion tests into shell-specific suites (`completion_bash_test.go`, `completion_zsh_test.go`, `completion_fish_test.go`, `completion_test.go`) and extracted formatting tests to `format_test.go`.

## [0.3.4] - 2026-09-07

### Fixed
- **Duplicate Flags Section** - Eliminated duplicate `Flags:` header and repeated option listings in `RenderCommand` when a command defined local options but the application defined no global options.

## [0.3.3] - 2026-08-31

### Added
- **Autocompletion Kill-Switches** - Added `NO_AUTO_COMPLETION` and `CLIHELP_NO_AUTO_COMPLETION` environment variable support to immediately bypass autocompletion handling.

## [0.3.2] - 2026-08-31

### Changed
- **Modular Subpackages** - Extracted Markdown documentation site generator into `doc` subpackage (`github.com/sarielhp/clihelp/doc`) and command hierarchy visualization into `tree` subpackage (`github.com/sarielhp/clihelp/tree`).

## [0.3.1] - 2026-08-29

### Added
- **Example Syntax Colorization & Theming** - Examples in command help, global overview, and manual pages are now rendered with ANSI syntax highlighting for commands, flags, arguments, values, comments, and prompts (`Theme.ExampleCmd`, `Theme.ExampleFlag`, `Theme.ExampleArg`, `Theme.ExampleComment`, `Theme.ExampleDesc`).
- **Static Example Validation & CLI Parsing** - Added `clihelp.ValidateExample`, `(*App).ValidateExamples()`, `(*App).ValidateAllExamples()`, and POSIX shell tokenizer `SplitExampleCommandLine` to statically parse and verify example strings against real flag specs, option validators, and positional argument constraints.
- **Example Tree Auditing** - `clihelp.Audit(app)` and `clihelp.AuditWithOptions` now automatically validate all examples declared on the application and across the entire command hierarchy.
- **Top-Level Application Examples** - Added `App.Examples` field rendered under an `Examples:` section in `RenderGlobal`, `RenderMan`, and `RenderMarkdown`.
- **Global Flag De-Cluttering & Topic Routing** - Added `Option.Group` and `clihelp.Group()` to categorize options under section headings, and `App.OmitGlobalFlagsInCommands` to suppress verbose global flags in subcommand help.
- **Dedicated Flags Directory (`help flags`)** - Added `App.RenderFlags()` (accessible via `help flags` or `help options`) to display an exhaustive, categorized reference for all global and persistent options.
- **Comprehensive Paged Manual (`help man`)** - Added `App.RenderMan()` (accessible via `help man` or `help all`) to render a full Unix manual with all commands, subcommands, arguments, flags, and examples through `$PAGER`.
- **Help Topic Index (`help topics`)** - Added `App.RenderHelpTopics()` (accessible via `help topics` or `help help`) to list available help topics.
- **Fish Shell Autocompletion** - Added `clihelp.GenFishCompletion(app, writer)` generator with native tab-separated description formatting.
- **Zero-Touch Auto-Installation** - Added `App.AutoInstallCompletion` field, `clihelp.CompletionPath()`, and `clihelp.IsCompletionInstalled()` to silently drop shell autocompletion scripts into standard user XDG directories on the first interactive execution.
- **Auto-Installation Support** - Added `clihelp.InstallCompletion(app, shell)` for installing Bash, Zsh, and Fish completions directly to standard XDG user directories without root permissions.
- **Pre-Built `CompletionCommand`** - Added `clihelp.CompletionCommand()` factory function returning ready-to-mount subcommands for `bash`, `zsh`, `fish`, and `install`.
- **Zsh Autocompletion Robustness** - Handled dynamic `compdef` registration, colon escaping in descriptions for `_describe`, and cursor-aware word slicing.

## [0.3.0] - 2026-08-28

### Added
- **Declarative Options Validation** - Added support for attaching an `OptionsValidator` callback to `Command` using built-in rule helpers (`MutuallyExclusive`, `RequiredTogether`, `RequiredWith`, and `RequiredIf`).
- **Required Option Constraints** - Added support for marking individual flags as required using `clihelp.Required()` which automatically appends `(required)` to help descriptions and returns a validation error if omitted.
- **Interactive Fallback** - When `App.InteractiveFallback` is true and execution runs in a terminal (TTY), `clihelp` automatically prompts the user interactively to collect missing required options.
- **CLI Constructor Tip** - After collecting missing required flags interactively, `clihelp` prints an educational shortcut tip showing how to bypass prompts in future executions.
- **Option Deprecations** - Added option deprecation message formatting in help pages, and warning prints during CLI execution if a deprecated flag is supplied.
- **Unit Testing Harness** - Added `clihelp.TestExecute` and `clihelp.TestExecuteWithStdin` to run mock executions and assert output/error content cleanly.
- **Command Tree Audit** - Added `clihelp.Audit` and `clihelp.AuditWithOptions` to statically verify command trees (valid descriptions, no collisions, and consistent subcommand path ordering).
- **Custom Flag Coloring** - Added a configurable `Flag` color field to `Theme` (defaults to Cyan).

## [0.2.19] - 2026-08-25

### Added
- **Pager support** - When `App.Pager` or `Options.Pager` is true, help output is automatically paged through `$PAGER` when it exceeds terminal height.
- **Command tree view** - Added `App.RenderTree()` method to render the full command hierarchy as a tree with box-drawing characters.
- **Prefix command matching** - Added `App.AbbrevCommands` field to enable abbreviated command names (e.g. `podctl b` instead of `podctl build`).
- `Options.MaxContentWidth` (default 80) makes the content wrap cap configurable (`min(termWidth, indent+MaxContentWidth)`).
- `Command.Group` group headings in global command lists and structural subcommand lists.
- `parseFlagSpec` now accepts `--flag=VALUE` specs.
- `tools/check.sh` verifies `example/main.go`'s `Version:` literal matches the `VERSION` file.
- CJK-aware column measurement via `go-runewidth` (wide East-Asian chars count as two columns).

### Fixed
- `Enum` rejects a default value outside the allowed set at bind time.
- `StringSlice` no longer aliases the caller's slice backing array.
- Root `Run` handlers now receive positional args even when subcommands exist.
- `--version` returns an error when `App.Version` is empty instead of silently exiting 0.
- `Options.width()` probes the render Writer when it is a terminal before falling back to stdout.
- `splitLines` strips trailing CR for CRLF input.
- Markdown pages/nav/index skip `Hidden` commands; slug collisions error instead of silently overwriting; front-matter title/parent are YAML-quoted.
- Completion supports `--flag=` prefixes, de-duplicates root command/shortcut names, and propagates `resolveCommand` errors.
- `Command` tree traversal unified through `findCommand`; option collection unified through `App.collectOptions`.
- Example Run handlers write to `ctx.Stdout` instead of `fmt.Printf`.

## [0.2.17] - 

### Added
- `Options.MaxContentWidth` (default 80) makes the content wrap cap configurable (`min(termWidth, indent+MaxContentWidth)`).
- `Command.Group` group headings in global command lists and structural subcommand lists.
- `parseFlagSpec` now accepts `--flag=VALUE` specs.
- `tools/check.sh` verifies `example/main.go`'s `Version:` literal matches the `VERSION` file.
- CJK-aware column measurement via `go-runewidth` (wide East-Asian chars count as two columns).

### Fixed
- `Enum` rejects a default value outside the allowed set at bind time.
- `StringSlice` no longer aliases the caller's slice backing array.
- Root `Run` handlers now receive positional args even when subcommands exist.
- `--version` returns an error when `App.Version` is empty instead of silently exiting 0.
- `Options.width()` probes the render Writer when it is a terminal before falling back to stdout.
- `splitLines` strips trailing CR for CRLF input.
- Markdown pages/nav/index skip `Hidden` commands; slug collisions error instead of silently overwriting; front-matter title/parent are YAML-quoted.
- Completion supports `--flag=` prefixes, de-duplicates root command/shortcut names, and propagates `resolveCommand` errors.
- `Command` tree traversal unified through `findCommand`; option collection unified through `App.collectOptions`.
- Example Run handlers write to `ctx.Stdout` instead of `fmt.Printf`.

## [0.2.15] - 

### Added
- Green subcommand names in command and global help via `Theme.Subcommand` (default green).
- Per-section wrap width: `wrapWidth(termWidth, indent) = min(termWidth, indent+80)` so indented lists can use more horizontal space.
- Exported `Inline` helper for rendering inline markdown to ANSI/OSC8 strings.
- `RenderCommand` now includes app-level and ancestor persistent options in the Flags section.

### Fixed
- `UsageLine` and `Examples` now pass through inline markdown rendering (no raw `**` or visible URLs).
- Oracle test (`example/mail_cli_fake`) updated to match green subcommands, per-section wrap width, and inline rendering.

## [0.2.13] - 
n### Fixed
- Fixed nested help double-render for `podctl config help` and similar nested commands
- Fixed `-v` version flag hijacking `-v, --verbose` persistent flags
- Fixed `App.GlobalFlags` not being parsed
- Fixed `Theme.Separator` dead code
- Fixed `BoolToggle` duplicate registration panicking instead of returning friendly error
- Fixed markdown subcommand alias links broken
- Fixed markdown tables unescaped `|` in descriptions
- Fixed silent `help <unknown>` behavior
- Fixed `PrintError` bypassing `App.Stderr`
- Fixed execute-path help ignoring custom theme
- Fixed markdown command pages omitting app/ancestor persistent flags

### Changed
- Updated example to version 0.2.13

## [0.2.11] - 2026-08-18

### Added
- Expanded Cobra comparison with detailed breakdown of terminal styling, plain-text defaults, and ANSI-aware width wrapping.
- Enriched `example/main.go` demonstrating all inline Markdown formatting features (bold, italic, code, strikethrough, links, notes).
- Added pre-checks in `bindHelper` for duplicate flag and shorthand declarations across parent and child flagsets.

## [0.2.10] - 2026-08-18

### Added
- Demonstrated clickable OSC 8 hyperlinks and inline formatting in `example/main.go` and `example_test.go` (`ExampleApp_Render`).

## [0.2.9] - 2026-08-18

### Changed
- Clarified and refined package description at the top of `README.md` and `llms.txt`.

## [0.2.7] - 2026-08-18

### Added
- Prominently integrated `llms.txt` and AI optimization documentation across `README.md`, `docs/index.md`, and `docs/ai-guidelines.md`.

## [0.2.6] - 2026-08-18

### Added
- In-depth architectural comparison guide with `spf13/cobra` in `docs/comparison-with-cobra.md` covering global state tradeoffs, declarative vs imperative patterns, and terminal aesthetics.

## [0.2.5] - 2026-08-18

### Added
- Standardized `llms.txt` AI specification at repository root for single-fetch LLM consumption.
- Testable Go examples in `example_test.go` (`ExampleApp_Execute`, `ExampleBoolToggle`, `ExampleExactArgs`) for `pkg.go.dev`.
- Automatic help flag collision protection: Intercepts accidental `-h`/`--help` declarations in `Option` constructors and returns actionable error messages.

## [0.2.0] - 2026-08-15

### Changed (breaking rendering API)
- Unified the two renderers (the original classic layout and the mail_cli-style detailed layout) into a single theme-driven engine. The old printing methods (`PrintUsage`, `PrintGlobalUsage`, `PrintCommandUsage`, `PrintSection`, `Print*To`) and helpers (`wrapText`, `indentLines`, `describeLabel`, `describeFlags`) were removed in favor of:
  - `(*App).Render(o Options, path ...string) bool` — dispatch global vs command help.
  - `(*App).RenderGlobal(o Options)` — application overview.
  - `(*App).RenderCommand(o Options, path ...string) bool` — detailed command page.
- Rendering is now `io.Writer`-first: `Options.Writer` (defaults to stdout) replaces hardcoded stdout output.
- Render width is deterministic and injectable via `Options.Width` (0 = auto-detect with a 70-column fallback), replacing the two previously inconsistent width functions and the separate 80/70 defaults.
- One ANSI-aware word reflow (`reflow`) replaces the diverging `wrapText` and reflow implementations, so all sections wrap identically.
- `Theme` (colors, `Separator`, `TitlePrefix`) now drives all styling via `App.Theme` or `Options.Theme`; nil color fields fall back to the default mail_cli palette.

### Remaining API
- `App`, `Command`, `Option`, `Example`, `Param`, `Note`, `Theme`, `Options`, and `(*App).LookupCommand` are unchanged.

### Fixed
- `reflow` and `colIndent` now measure **visible width** (ANSI-stripped runes) instead of raw byte length, so multi-byte runes and colored labels wrap and align correctly.
- `App.Description` and `App.GlobalNote` are now rendered on the global overview (when set).
- `Example.Description` is now rendered beneath its example line (when set).
- Corrected the `Theme` and `reflow` doc comments (`Theme`'s zero value is safe; nil colors fall back to defaults).

### Notes
- The `example/mailcli` reconstruction still reproduces mail_cli's usage pages **byte-for-byte** (verified by the oracle test), now via the unified `Render` API.

## [0.1.1] - 2026-08-13

### Added
- Added receiver methods on `*App` (`a.PrintGlobalUsage()`, `a.PrintCommandUsage(path...)`, `a.PrintUsage(path...)`, `a.LookupCommand(path...)`) for a clean, object-oriented Go API surface.
- Added comprehensive Go doc comments across all exported structs, fields, and functions in `clihelp.go`.
- Extensively documented `example/main.go` step-by-step to demonstrate how external applications should integrate `clihelp`.
- Rewrote `README.md` into a complete package guide featuring API reference tables, multi-level subcommand examples, and quick start instructions.
- Added recursive subcommand support (`Subcommands []Command`) to `clihelp` with variadic command path resolution in `PrintCommandUsage(a *App, path ...string)`.
- Added nested `config set` subcommands in `example/main.go` supporting `time`, `space`, and `location` attributes with recursive help dispatching and flag options.
- Added unit tests in `clihelp_test.go` verifying multi-level nested subcommand help generation.
- Expanded demonstration CLI application in `example/main.go` (`podctl`) with multiple subcommands (`build`, `serve`, `config`, `deploy`, `status`), extensive options, usage examples, and command-line help dispatching.
- Improved `describeLabel` multi-line description alignment and tuned command column width.
- Replaced custom ANSI stripping loop and direct escape code literals (`\033`, `\x1b`) with external package `github.com/acarl005/stripansi`.
- Added rule in `AGENTS.md` prohibiting direct ANSI escape codes in favor of external packages (`fatih/color`, `stripansi`).
- Added green bold ANSI color styling (`color.FgGreen, color.Bold`) for subcommand names under `COMMANDS`.
- Added ANSI-aware padding calculation (`formatPadded`) to ensure aligned description columns when labels include ANSI escape codes.
- Added `AGENTS.md` guidelines for AI-assisted development adapted for `clihelp`.
- Added automation scripts in `scripts/` (`check.sh`, `format.sh`, `lint.sh`, `map.sh`, `version.sh`, `bump-version.sh`, `commit.sh`, `checkpoint.sh`, `run_example.sh`).
- Added `Makefile` for standard development tasks (`make check`, `make lint`, `make test`, `make map`, `make run`, etc.).
- Added `VERSION` file (`0.1.0`) as single source of truth for versioning.
- Added unit tests in `clihelp_test.go`.
- Added `CHANGES.md` project changelog.

## [0.1.0] - 2026-08-10

### Added
- Initial release of `clihelp` Go package.
- Colored section headers (`USAGE`, `COMMANDS`, `OPTIONS`, `EXAMPLES`) powered by `fatih/color`.
- Terminal width auto-detection and ANSI-aware text wrapping via `golang.org/x/term`.
- Core data structures: `App`, `Command`, `Option`, and `Example`.
- `PrintGlobalUsage` for displaying global application overview with registered commands.
- `PrintCommandUsage` for detailed command help text, flags, and usage examples.
- Sample command-line application in `example/main.go`.
