# Changelog

All notable changes to `clihelp` will be documented in this file.

## [0.3.17] - unreleased

### Fixed
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
