# Deep review — shell integration surface (2026-09-17, v0.3.14)

> **Reading this later.** Items below struck through have since been settled; the note
> after each says how. The current state of the public API is
> `review/api-surface-2026-09-18.md`, and `docs/how-clihelp-decides.md` is the
> mechanism this library actually implements today.

Evidence-based review of the code added after v0.3.9 and released as 0.3.12–0.3.14:
`explain.go`, `install.go`, `man.go`, `protocol.go`, and the generated-script templates in
`completion.go`. That surface had never been independently reviewed. Everything else in the
library was audited at v0.3.9 (`review/findings-2026-09-17.md`); all of those findings are
closed and none is re-litigated here.

Six parallel dimension reviews — correctness and data loss, injection into generated
artifacts, atomicity and concurrency, the generated shell code judged as shell, architecture
and API surface, tests and documentation — followed by independent verification of every
critical and high finding.

**Method.** Every finding marked *verified* below was reproduced by the reviewer personally,
in a throwaway copy of the tree, with `HOME` and `XDG_*` pointed at a temporary directory.
Injection proofs use inert payloads (`touch <marker>`). The working tree was not modified
during the review and the project binary was never run: `example/main.go` sets
`AutoInstallCompletion: true`, so running it writes into the real home directory. `make
check` was green before and after.

**What makes this surface different from the last review's.** It writes into the user's home
directory and edits their shell startup files, and one of its entry points
(`AutoInstallCompletion`) runs inside every invocation of every program built with the
library. A defect here is not wrong output; it is a file the user cannot get back.

---

## The one-sentence summary

Every *write* path in this surface decides a file belongs to clihelp because of its **name**;
exactly one path — a *delete* — actually checks for clihelp's marker; and nothing anywhere
validates or quotes `App.Name` before interpolating it into shell source.

---

## Critical

### S1 — Arbitrary code execution on Tab-press, in zsh  **[verified]**

`completion.go:319-322`. The zsh template passes the typed words to zsh's `_call_program`,
whose implementation ends in `eval $clocale $prefix "$argv[2,-1]"` — it re-parses its
arguments as shell code. `$words` holds the command line verbatim.

```
type:  pd $(touch /tmp/…/MARKER) <Tab>
→ marker created; nothing shown on screen
```

Reproduced under `zpty` with a trivial control script that does *not* exhibit it, so this is
clihelp's script and not zsh. **This is the previous review's E1 re-opened**: that fix
replaced bash's `compgen -W`, and the zsh path was never examined. The realistic trigger
needs no hostile author — the user pastes `myapp deploy --tag=$(date +%s)` from a README and
presses Tab.

The `else` branch at `:322` is correct and is the branch every zsh test takes, because
`completion_zsh_test.go` never defines `_call_program`. That is why nothing caught it.

**Fix.** Quote with zsh's own flag: `_call_program … ${(q)binary_cmd} __complete ${(q)words_to_pass[@]}`,
or delete the `_call_program` branch and keep the direct call. Bump `completionScriptVersion`
so installed scripts are replaced.

### S2/S3 — `App.Name` becomes executable content inside `~/.bashrc`, permanently  **[verified]**

`install.go:191-193` builds the startup-file line with `fmt.Sprintf("[ -r %[1]q ] && . %[1]q")`.
Go's `%q` is `strconv.Quote` — a *Go* literal, not a shell word. It wraps the path in **double**
quotes, where `$(…)` and backticks are still live, and escapes with Go's rules, which the
shell reads as literal backslashes. It is wrong in both directions.

`App.Name` reaches that path unvalidated (and reaches ten further interpolation points across
the three completion templates, the three key-binding templates, and the wrapper). Verified
end to end with a backtick in the name:

```
~/.bashrc:  [ -r "…/.config/pd`touch …/PWNED-NAME`/shell/bash" ] && . "…"
sourcing it → *** payload executed ***
```

The payload is **permanent**: it survives deleting the program, and `uninstall` cannot be
relied on to remove it. Independently of any hostile name, the same defect silently breaks
the line for any user whose `$HOME` contains a `$`, a backtick or a tab.

**Fix.** Use the POSIX quoter the library already owns (`escapeShellArg`, `interactive.go:104`)
and a fish sibling for fish's quoting rules; add one `safeAppName` validator (letters, digits,
`-`, `_`, `.`, `+`; not `.`/`..`; no leading `-`) called by every generator and every `*Path`
function. All of them already return `error`, so this is source-compatible.

### C1 — The fish drop-in is written and deleted with no ownership check  **[verified]**

`install.go:176` declares `~/.config/fish/conf.d/<app>.fish` "owned" from its path alone;
`install.go:298-309` writes the whole file; `install.go:339-346` calls bare `os.Remove`.

```
before: # my own fish config for myapp / set -gx MYAPP_TOKEN hunter2
after install:   destroyed
uninstall with nothing of ours ever installed: file deleted
```

That path is exactly where a fish user keeps their own settings for that tool.
`completion.go:691` tells them the opposite: *"On fish nothing shared is touched at all."*

**Fix.** The block already carries markers for fish; refuse to overwrite or remove a file that
does not contain them.

### C2 — The rc-block markers are matched as a substring anywhere  **[verified]**

`install.go:206-243`. `upsertBlock` and `removeBlock` use `strings.Index` on the whole file,
not anchored to a line. Any occurrence of the marker text — in a comment, in a quoted string —
is treated as the start of the block, and everything from there to the next end marker is
replaced or deleted.

A `.bashrc` whose first line merely *mentions* the marker lost `export EDITOR=vi`,
`alias ll='ls -l'` and `source ~/work/secrets.sh` after two installs. A half-block (end marker
deleted by hand) loses everything after it. Duplicate blocks are never cleaned up.

**Fix.** Match each marker as a whole line, treat an unterminated block as an error rather
than appending, and remove *all* copies on uninstall.

### C3 — `AutoInstallCompletion` overwrites a hand-written completion script  **[verified]**

`completion.go:477-479`. `completionIsCurrent` returns false for any file lacking clihelp's
marker — which is precisely what a *foreign* script looks like — so "not ours" is read as
"stale". One ordinary command replaced a hand-written
`~/.local/share/bash-completion/completions/myapp`. The delete path two files away
(`install.go:382`) gets this right and says so in its comment.

### E1/E2 — The test suite writes into the developer's real home directory  **[verified, and it already happened]**

Six test sites exec the example binary with the inherited environment;
`example/main.go` sets `AutoInstallCompletion: true`. The evidence is on this machine:
`~/.bashrc` lines 565-570 contain a `bare shell integration (clihelp)` block pointing at a
deleted `/tmp` test directory, written during the session that built this feature. Also
`~/.local/share/bash-completion/completions/podctl` and an empty `~/.local/share/man/man1/`.

The block is inert (the `[ -r … ]` guard fails), but it is permanent clutter in a dotfile.

**Fix.** Give every `exec.Command` in the shell tests an explicit sandboxed env, and add a
`TestMain` that fingerprints the real home before and after the suite and fails if anything
changed. The per-test `sandboxHome` discipline has already failed once; the guard cannot.

---

## High

| ID | Finding | Status |
|---|---|---|
| **D3** | `uninstall` is undone by the next ordinary run: the auto path finds no integration file, falls through to the legacy installer, and creates a *different* artifact in a *different* directory — which `install` then deletes as "superseded". | verified |
| **H1** | Auto-refresh edits startup files, contradicting the documented promise: it re-adds a block the user deleted by hand, and with `ZDOTDIR` adopted after install it *creates* a `~/.config/zsh/.zshrc` that never existed. | verified, 2 routes |
| **H2** | `os.Rename` replaces a symlinked `~/.zshrc` with a regular file, orphaning a dotfiles repo invisibly — `git status` stays clean. Reachable with no user action at all. | verified |
| **IO-1** | No lock around the startup-file read-modify-write. 4 of 8 concurrent installs lost their block while **all 8 reported success**. Never corrupt — rename is atomic — always lossy. | verified (counted) |
| **H3/D6** | `appName()`'s display fallback `"app"` is used as a filesystem identity: two nameless programs share one integration file, one `.bashrc` block and one man page, and either one's uninstall deletes the other's. `App.Name: "../../.ssh"` escapes the config directory. | verified |
| **S4/S5** | The wrapper generator interpolates `{{target}}` inside double quotes (where `escapeShellArg`'s single quotes are inert) and `{{name}}` into comments with no newline check. Both give code execution from `__clihelp wrapper` input. | relayed, high confidence |
| **S6** | The Alt-H dispatcher matches the registry on basename, then executes the path the user typed: registry entry `pd` authorizes `./pd`. Alt-H is sold as an inspection key. | relayed, high confidence |
| **D1/E5** | `__clihelp install bash --no-keys` installs the key binding anyway — the flag is only read in first position. The visible twin honours it. The silent failure always goes in the unsafe direction. | verified |
| **E3/E4** | Two tests name a promise they do not test, and one asserts the fish clobber is intended (`owned bool // clihelp owns the startup file outright`). They would block a correct fix. | relayed, high confidence |
| **SH-1** | Alt-H executes an arbitrary file from the current directory: the dispatcher matches the registry on *basename* and then runs the word as typed, so `./podctl` in a downloaded tarball runs on a keystroke the user believes is "explain, don't run". Found independently by two dimensions. | verified |
| **SH-2** | A leading space kills Alt-H in all three shells — `${LINE%% *}` yields the empty string — and in bash and zsh the empty word *matches* the space-padded registry, so the key is swallowed and `run-help` never gets it back. A leading space is deliberate for anyone using `HISTCONTROL=ignorespace`. | verified |
| **SH-3** | The generated wrapper's `__explain` branch is unreachable: nothing anywhere adds a wrapper name to `_clihelp_apps` (`protocol.go` never mentions it). `docs/completion.md:216` describes behaviour that cannot occur, and `TestLiveBashWrapperScript` misses it by calling the wrapper directly instead of through the dispatcher. | verified |
| **SH-4** | Wrapper argument fidelity: `escapeShellArg`'s deny-list omits `[`, `]`, `{`, `}` and `\`, and its single-quoted output is then interpolated into a *double-quoted* string. `a[1]` globbed to an existing file `a1`; a backslash was eaten; `$HOME` expanded in one emitted line and not the other. | relayed, high confidence |

---

## Medium and low

| ID | Finding |
|---|---|
| D2 | No stdout/stderr contract: `__clihelp uninstall` and `manpage --uninstall` put prose on stdout where their siblings put a bare path. **verified** |
| D4 | The block written into `~/.bashrc` tells the user to run `<app> completion install` to remove it — which does not exist unless the author added `CompletionCommand()`, the exact case `__clihelp` exists for. **verified** **[fixed: 51bee84]** |
| D7 | `manpage --install --uninstall` is accepted and silently prefers uninstall; `--force` without `--install` is ignored. **verified** |
| S7/E8 | `.TH`'s arguments go through Go's `%q`, not roff escaping: `\"` starts a roff comment, so a version containing a quote truncates the footer and silently drops the `"User Commands"` argument. `man` emits no warning, so the one live test (which asserts empty stderr) passes on a corrupted page. |
| IO-3 | The atomic replace installs a new inode: mode is preserved (**verified**), ownership, ACLs and hard links are not (**verified** for hard links). `sudo -E myapp anything` rewrites `~/.bashrc` as root. |
| IO-4 | No `fsync` before the rename, and no directory sync after: a power loss can leave `~/.bashrc` zero-length. The doc comment claims more than the code delivers. |
| IO-6 | The auto path discards every error, and writes the integration file *before* editing the startup file — so a failed bootstrap is never retried, because the staleness gate is already satisfied. |
| IO-7 | `man -w` runs with no timeout inside the user's program. (Disproved: it is *not* on the per-invocation path, and its stderr does not leak.) |
| IO-8/9 | Four marker checks read a fixed-size head with the error discarded; an interrupted write leaves a `.tmp-*` sibling in `$HOME` forever. |
| S9 | The wrapper's `rest=${2#* }` reconstruction leaks the wrapper's own name when the line has leading blanks, and drops every argument when the separator is a tab. **[fixed: 6eb1015]** |
| A1 | The five files form a complete dependency cycle; there is no layering. Shell resolution is written out six times with two different error messages. **[fixed: cce12a1]** — four layers with no upward edge, one `resolveShell`, and the rule written into `AGENTS.md`. The count was seven sites, not six, and the seventh disagreed about the default. |
| SH-5 | The Alt-H protocol has a version on the *dispatcher* but none on the *wire*: `_clihelp_apps` carries bare names, so a newer dispatcher cannot tell which release a registered program speaks. Nothing is broken today; the first bump of `keyDispatcherVersion` is what breaks, silently. Fix before the next bump, not after. **[fixed: 995d880]** |
| SH-6 | The key binds into whatever keymap is current at source time: with `set -o vi` after the block, Alt-H is dead in bash and zsh. It also silently clobbers a user's existing `\eh` binding. **[fixed: cf9ab99]** — the keymaps by binding all of them; the clobber by `CLIHELP_NO_KEY_BINDINGS`, an opt-out rather than detection, because readline gives no portable way to ask what `\eh` is currently bound to. |
| SH-7 | zsh completion is silently not registered when `compdef` does not exist yet — deferred/turbo loaders and late `compinit` all land here, with no diagnostic and no retry. **[fixed: 578a860]** |
| SH-8 | Three different file-completion behaviours for one program: bash falls back to filenames, zsh offers nothing, fish disables them outright with `-f`. **[fixed: de792a6]** |
| SH-9 | bash inserts a completion candidate containing a space unquoted, so the buffer is re-parsed as two arguments. zsh and fish quote correctly. **[fixed: 6eb1015]** |
| SH-11/12 | The bootstrap line is the last command in the rc, so a missing integration file leaves `$?=1` at every prompt (visible in prompts that render exit status); and the path is emitted with Go's `%q` inside shell double quotes — same defect as S3, second site. |
| SH-14 | fish alone does not check the child's exit status and passes the buffer unquoted as a list, so a multi-line buffer loses everything after the first line. **[fixed: 6eb1015]** |
| E10-E13 | `docs/completion.md` contains five false statements; `README.md` and `llms.txt` do not mention that `completion install` edits a shell startup file; `AGENTS.md`'s file table omits seven test files; the gate never runs `-race` and `make check` rewrites source. |

---

## Disproved — recorded because a disproved hypothesis is a result

- **Bash and fish completion do not expand candidates.** The previous review's E1 fix holds
  for both; verified live with `$(…)` in a candidate.
- **The Alt-H registry check is not glob-injectable** in any of the three shells — the
  expansion is inside double quotes in bash and zsh, and fish uses `contains`.
- **roff requests inside descriptions and inside `Note{Raw: true}` are correctly neutralised.**
  `manEscape`'s per-line `\&` guard runs after the backslash replacement, which is the right
  order. Only the `.TH` line bypasses it.
- **Concurrent installs never corrupt a startup file** — over 16 writers the user's content
  appeared exactly once and markers were always balanced. The defect is lost updates only.
- **File mode is preserved** across the atomic replace.
- **No file-handle leaks.** Every `os.Open` in the surface is closed on every path.
- **Install/uninstall is byte-exact round-trip** when C2 does not fire.
- **The dispatcher's version arbitration is correct** — the highest version wins in either
  sourcing order, confirmed both ways.
- **The registry's `case` pattern is not glob-injectable**, in bash or zsh: the expansion sits
  inside double quotes, so quoted portions match literally.
- **The zsh `funcstack[1]` guard works**, and `zle -I` + `reset-prompt` redraws exactly once
  with no duplication.
- **`bind \eh` is correct on fish 4**, and the `conf.d` binding survives fish's interactive
  key-binding setup.
- **`READLINE_POINT=${#READLINE_LINE}` is character-indexed**, verified with multi-byte text.
- **The generated wrapper is POSIX-clean** for `#!/bin/sh`.
- **`rest=${2#* }` is correct** for quoted arguments containing spaces, an empty `$2`, and
  invocation by path — only leading whitespace is wrong.
- **`man`'s stderr does not reach the terminal**, and `man -w` is not on the per-invocation
  path.

---

## Ordered roadmap

Ordered so that stopping after any step leaves the tree safer than it started. The ordering
matters more than usual here: fixing the write paths before the auto path would leave the
unattended writes running against half-changed code.

**Now — stop the unattended and the executable**

1. **H1/D3 — take the auto path out of the startup-file business.** Split a
   refresh-only entry point out of `InstallShellIntegration` and point
   `maybeAutoInstallCompletion` at it; stop it creating the legacy artifact after an
   uninstall. ~15 lines. This one change removes every unattended write to a startup file and
   to the fish drop-in, which is the unattended half of C1, C3, H2 and IO-1 — so every later
   fix only has to be right for explicit, user-initiated commands.
2. **S1 — quote the zsh `_call_program` arguments**, bump `completionScriptVersion`. Two
   lines plus the version bump; closes code execution on a keystroke.
3. **S3 — replace `%q` with `escapeShellArg`** in `bootstrapBlock`. Closes the only path that
   writes executable content permanently into `~/.bashrc`.
4. **S2 — add `safeAppName` and wire it into the ten generators and the path functions.**
   Defence in depth behind 3, and it closes H3/D6's traversal at the same gate.
5. **C3 — gate the auto completion-script write on clihelp's own marker.**
6. **C1 — marker-check the fish drop-in before writing or removing it.**
7. **C2 — line-anchored block matching, an error on an unterminated block, remove all copies.**

**Next**

8. **E1 — sandbox the six test exec sites and add the `TestMain` home fingerprint.**
9. **H2 — resolve symlinks in `writeFileAtomically`** (4 lines, fixes all five call sites).
10. **IO-4 — `fsync` the temp file and its directory.**
11. **D1/D7/D10 — one argument parser for every `__clihelp` verb.**
12. **SH-1 + SH-2 + S6 — the dispatcher's first-word extraction.** One edit per shell closes
    the local-file execution path and the leading-space dead zone together, and restores the
    `run-help` / `__fish_man_page` handback the docs already promise. Bump
    `keyDispatcherVersion`.
13. **S4/SH-4/S5 — the wrapper's quoting and name validation**, including inverting
    `escapeShellArg` to an allow-list.
14. **SH-3 — register the wrapper name**, so its `__explain` branch is reachable at all.
15. **S7 — a roff quoter for the `.TH` arguments.** Bump `manPageVersion`.
16. **IO-1 — an advisory lock around the read-modify-write**, and derive `StartupEdit` from a
    post-write verification rather than from intent.

**Later**

15. Documentation: the startup-file sentence in `README.md`/`llms.txt` first, then
    `docs/completion.md`'s five false statements, then `AGENTS.md`.
16. D2's stream contract; IO-3's ownership guard; IO-6's `CLIHELP_DEBUG`; IO-8's shared
    marker reader; the `-race` gate.
17. A1's file reorganization and the unified surface — only after 11 has settled what the
    unified behaviour *is*.

---

## Status

**Correction, added after the fact:** this section originally read as though every
finding was closed. It was not. The roadmap blocks were all applied, but four rows of
the medium/low table — D4, S9, SH-9 and SH-14 — were never *in* a roadmap block, so the
fix pass never saw them. They stayed open through two releases and are closed now, each
with its own regression test. The lesson is about the document rather than the code: a
roadmap is not a checklist of the findings, and the status has to be written against the
findings table, not against the roadmap.

**The Now and Next blocks are applied** on branch `fix/shell-integration-review-2026-09`,
one commit per fix, each with a regression test whose teeth were checked against the unfixed
behaviour with `teeth.sh`. Every proof-of-concept in this document was re-run against the
final tree and is closed. `make check` — which now includes the race detector — and
`make audit` are green.

The Later block is applied too, except for one item, deliberately at the time: **A1's file
reorganization**. It was a pure refactor, it touches every file in the surface, and a safety
pass was not the place for it. It has since been done on its own, after the defects were
closed — which is the right order, because the layering is only checkable once the behaviour
is settled.

Two things the fix pass proved this document wrong about, corrected in place:

- **IO-3's mode claim.** File mode *is* preserved across the atomic replace; only ownership,
  ACLs and hard links are not. Measured during verification, before the fix.
- **The wrapper's `__explain` branch.** The reviewer's own earlier report described it as
  working, on the strength of a test that called the wrapper directly. SH-3 showed the
  dispatcher never reaches it, because nothing registered a wrapper's name. Both the code and
  `docs/completion.md` are fixed.

Behaviour changes a user will notice, all of which fail visibly rather than silently:

| Change | Why |
|---|---|
| An `App.Name` with a space or a shell metacharacter, or no name at all, errors from the generators and the install paths | it becomes a file name and a shell symbol |
| `__clihelp install bash extra`, `manpage --install --uninstall`, `--force` without `--install` all error | they were silently ignored, always in the unsafe direction |
| Alt-H no longer answers for a program invoked by path (`./myapp`) | the registry holds names, and matching on the basename executed files out of the current directory |
| `completion install` and `completion uninstall` print their report on stderr | stdout is what a script captures |
| `AutoInstallCompletion` no longer creates anything after an explicit uninstall | uninstall did not stay uninstalled |

## What I would look at with more time

- ~~**SH-5's wire versioning, before the next `keyDispatcherVersion` bump.**~~ Done in
  `995d880`: entries are `name:protocol`, both forms are written for one release, and a
  dispatcher that meets an unknown protocol declines instead of calling.
- ~~**Live end-to-end install tests for zsh and fish.**~~ Done in `install_live_test.go`:
  every live test used to drive bash, so the two shells whose install paths actually differ
  were started by nothing. The deferred-`compinit` order is covered there too.
- ~~**The `zpty` harness from S1's verification, as shared test infrastructure.**~~ Done in
  `cf9ab99`, as the cheaper variant: `shell_harness_test.go` stubs `zle`, `bindkey`, `bind`
  and `commandline` with shell functions. It found nothing on its own, but SH-6, SH-7 and
  SH-8 were all provable the moment it existed — which is the point, since all three had
  reached a release through shells that were only ever syntax-checked.
- ~~**Whether `AutoInstallCompletion` should exist at all.** Every finding here is more severe
  because it runs unattended, and its kill switches are opt-out rather than opt-in. It is
  also why the safety rules for this review had to forbid running the example binary.~~ **Settled 2026-09-18:** kept, and renamed `AutoRefreshIntegration` because the old name described something it cannot do. Without it a user who upgrades keeps the script and man page from the previous version indefinitely, with no symptom. Measured cost: 6.4 µs per run.
- **`example/main.go` setting it to `true`** — a demonstration app that installs into the
  reader's home directory the first time they try it is a questionable thing to be modelling.
  *Partly answered 2026-09-18:* the flag is `AutoRefreshIntegration` now and can only refresh
  a file that already exists, so the example can no longer create anything. What remains is
  whether a demonstration should be doing anything unasked on every run at all; nobody has
  revisited that since the rule changed.
- **A fault-injection seam for the filesystem.** There is no way to test the error paths in
  this surface today, because `os` is called directly. *Still open*, and `atomicwrite_faults_test.go`
  covers only the writer's own seam.
