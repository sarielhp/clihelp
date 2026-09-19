# API surface audit, 2026-09-18

> **Status, 2026-09-19 (v0.3.40).** The recommendations in this audit have been settled:
> - **A1 (Test framework isolation):** Shipped in `clihelp/clihelptest`.
> - **A2 (pflag decoupling):** Shipped; `OptionsValidator` takes `Flags` interface and `Var` takes `Value`.
> - **A3 (Redundant aliases):** `RenderGlobalFlags`, `CheckExample`, `ValidateExamples`, `AuditWithOptions`, `ColorizeExampleLine` retired.
> - **A4 & B1–B3 (Export pruning):** Internal installers and renderers unexported; queries (`IntegrationPath`, `CompletionPath`, `ManPagePath`, `IsCompletionInstalled`) and `Option.Binder` preserved for downstream CLI consumers (Addendum 2); shared helpers preserved or tested.
> - **F1–F3 (Downstream feature additions):** All three shipped (F1 help on missing args, F2 unknown command handling with `App.Run`, F3 `Optional` flag values), plus positional completion (`Param.Complete` in v0.3.40).
> The public callable API is down to 16 standalone functions and 14 cohesive types.

**Scope.** The exported surface of `github.com/sarielhp/clihelp`, ahead of 1.0.
Nothing here is about behaviour; four deep reviews and six mutation surveys have
covered that. This is about what the package promises to keep.

**Constraint, corrected.** `AGENTS.md` says "preserve backward compatibility for
all exported types and methods" and "avoid breaking existing function
signatures". That is the rule *after* 1.0. Before it, removal is free, and the
author has confirmed it is on the table. This document therefore recommends
deleting things, not deprecating them. A deprecated alias is a permanent cost
paid to avoid a break that is currently free. `AGENTS.md` should say so.

## The measurement

| | count |
|---|---|
| Exported entries in `go doc` | 201 |
| Exported declarations including methods | 523 |
| Distinct exported names used by `example/` — a complete demo application with grouped commands, toggles, enums, validators, examples and notes | **19** |
| `Deprecated:` markers in the library | 1 (added 2026-09-18) |

Nineteen names carry a full application. The other ~180 are either advanced use,
support for the library's own subpackages, or accidents. The ratio is not itself
a defect — a rendering library reasonably exposes its renderer — but a surface
that has grown across seventeen releases and been pruned zero times is worth
reading once as a surface.

## Now — fix before 1.0

### A1 — The library links the test framework into every consumer's binary

`testing.go` is an ordinary (non-`_test.go`) file, and it imports `testing`.
Measured on this machine: `go list -deps .` reports **100 packages with it and 95
without**, and the example binary is **4,992,085 bytes with it against 4,972,624
without** — about 19 KB. The size is minor. The design is not: a library has no
business putting `testing` and `flag` into the dependency graph of a program that
merely wants a help page, and `flag` in particular is a package this library
deliberately avoids in favour of `pflag`.

`TestResult`, `TestExecute`, `TestExecuteWithStdin` and the four `Assert*`
methods belong in their own package — `clihelp/clihelptest` — which consumers
import only from their own tests. This is the same shape as `net/http/httptest`.

### A2 — `pflag` is welded into the public API

```go
type OptionsValidator func(fs *pflag.FlagSet) error
func Var(target pflag.Value, flags string, usage string) Option
```

Any consumer writing a custom validator must import `github.com/spf13/pflag` and
name its types. At 1.0 that makes clihelp's compatibility promise depend on
pflag's: a pflag v2 would break every consumer that ever wrote a validator, and
clihelp could not shield them. The whole point of the `Flags: "--tag <v>, -t"`
string is that the binding library is an implementation detail; these two
declarations undo that for exactly the users who go furthest with the library.

`OptionsValidator` should take an interface clihelp owns — enough to answer
"was this option set" and "what is its value", which is all the four built-in
validators use. `Var` should take a small clihelp interface that `pflag.Value`
happens to satisfy structurally.

### A3 — Same function, two names

| Keep | Delete | Evidence |
|---|---|---|
| `RenderFlags` | `RenderGlobalFlags` | its own doc comment reads "RenderGlobalFlags is an alias for RenderFlags" |
| `ValidateExample(app, ex, cmd)` | `App.CheckExample(ex, cmd)` | identical contract, one spelled as a method |
| `App.ValidateAllExamples() error` | `App.ValidateExamples() []error` | two shapes of one answer; the combined error is what every caller in this repo uses |
| `AuditWithOptions(app, opts)` | `Audit(app)` | keep one and let `AuditOptions{}` mean the default — or keep both, but then `Audit` must be documented as exactly `AuditWithOptions(app, AuditOptions{})` |
| `ColorizeExampleLineWithApp` | `ColorizeExampleLine` | the app-aware form is strictly better; the other cannot identify subcommands |
| `TestExecuteWithStdin` | `TestExecute` | one variadic or options form, in `clihelptest` |

`DisplayName` / `DisplayNameWithArgs` are a genuine pair (two different strings
for two different places) and should stay, but see A4.

### A4 — Exported for the build, not for users

These are exported because `doc/` and `tree/` are separate packages and needed
them, which is a build-system reason wearing an API's clothes:

`StripANSI`, `VisualWidth`, `FirstSentence`, `SubcommandList`, `Inline`,
`DisplayName`, `DisplayNameWithArgs`, `ColorizeExampleLine*`,
`SplitExampleCommandLine`, `DefaultMaxColIndent`.

Go's `internal/` rule is scoped to the subtree containing it, so
`github.com/sarielhp/clihelp/internal/text` is importable by `clihelp/doc` and
`clihelp/tree` and **not** by any consumer. Moving the measurement and text
helpers there removes about ten names from the promise while leaving both
subpackages working.

It also retires `crosspackage_test.go`'s reason for existing. That guard was
written because four helpers were *copied* into subpackages and three of them
drifted into real defects. A shared internal package is the fix that guard was
asking for.

Of the eight exports above that a real consumer never touches, **all eight are
held up by `example/mail_cli_fake`** — the 499-line oracle that reimplements the
renderer to check the renderer. The rendering mutation survey found that oracle
buys nothing measurable (29% escape rate, the same as files with no oracle at
all). Retiring it and moving these to `internal/` are one piece of work.

## Next — naming that no longer describes the thing

- **`App.AutoInstallCompletion`** is already a deprecated alias for
  `AutoRefreshIntegration`, kept on 2026-09-18 out of a compatibility promise
  that does not yet apply. Delete it.
- **`IsCompletionInstalled`** answers "a script exists at the path", not "the
  user has working completion" — the integration file, the startup-file line and
  the uninstalled marker are all invisible to it. Its own internals comment says
  so. Either rename it to what it does (`CompletionScriptExists`) or make it
  answer the question its name asks.
- **`App.Render(o, path...)`** dispatches to `RenderGlobal` or `RenderCommand`
  depending on whether `path` is empty. Three entry points for one decision, and
  the dispatching one is the least clear. Keep it only if something needs to
  decide at run time.
- **`NoColor` on both `App` and `Options`** is deliberate and documented
  (application-wide versus per-render) and should stay — but `Options.NoColor`
  and the global `color.NoColor` interact in a way that produced two separate
  defects, and the rule deserves one paragraph of package documentation rather
  than three comments in three files.

## Later

- `GenWrapperScript` is the only `Gen*` function not about the current
  application's own shell support. It may belong with the installer functions.
- `AuditOptions`, `ArgsValidator`, `OptionsValidator` and `Param` are the four
  types a consumer meets without asking for them. Each is fine; the group is
  worth one documentation page rather than four entries in `go doc`.

## What I did not do

I did not check whether the 19 names a real application uses are the *right* 19
— that is a design question about `Command` and `Option`, not a surface
question, and it wants a second opinion rather than an audit.

---

# Addendum — how much further it can go

The counts above use `go doc` lines, which include struct fields. The callable
surface is cleaner to reason about:

| | count |
|---|---|
| Exported functions and methods | 74 |
| Exported types | 14 |
| Exported consts and vars | 3 |
| **Total nameable, callable API** | **91** |

Two measurements narrow it much further than the first pass did.

**Forty-nine of the 91 are used only by the library's own tests** — not by
`doc/`, not by `tree/`, not by a consumer. That alone proves nothing: the demo
application is one application, and `MinimumNArgs` being unused there is not
evidence nobody wants it. What matters is the split inside those 49.

**Twenty-three of them the library calls itself**, which is the real signal: a
function the library invokes internally and no consumer invokes at all is an
implementation detail that happens to start with a capital letter.

## B1 — The installer family is the help command's internals, exposed

```
InstallCompletion   InstallShellIntegration   UninstallShellIntegration
InstallManPage      UninstallManPage          InstallResult
CompletionPath      IntegrationPath           ManPagePath
IsCompletionInstalled
```

Ten names. Every one reads or writes inside the **user's** home directory, and
every one is called by the library itself, from `__clihelp` and
`CompletionCommand()`. That is the supported route: the author mounts one
command, the user runs it once.

Exporting them offers a second way in that nothing supervises. The rule that the
unattended path may only refresh files that already exist is enforced in
`autorefresh.go`; a consumer calling `InstallShellIntegration` directly gets none
of it, and can edit a startup file from inside an ordinary program run. Making
these unexported turns a documented rule into one the type system keeps.

The packager's genuine need is served by the other half, which stays exported:
`GenBashCompletion`, `GenZshCompletion`, `GenFishCompletion`, `GenManPage`,
`GenShellIntegration`, `GenKeyBindings`, `GenWrapperScript` all write to an
`io.Writer`. Generating a script into `/usr/share` at build time is a packager's
job; writing into `$HOME` is not.

## B2 — Four of the seven render entry points are the help system's internals

`RenderMan`, `RenderFlags`, `RenderHelpTopics` are each called by the library to
serve `help man`, `help flags` and `help`. No consumer calls them; a consumer
reaches all three through the help command. `RenderGlobalFlags` is an alias of
one of them. Unexport the first three, delete the fourth, and the render surface
becomes `RenderGlobal`, `RenderCommand` and the `Render` dispatcher — three, from
seven.

`Explain` is the same shape: it exists for the `__explain` protocol call, the
library calls it, and no consumer does.

## B3 — Four names for "check the examples", two for "audit"

`ValidateExample`, `App.ValidateExamples`, `App.CheckExample` and
`App.ValidateAllExamples` are one job. `Audit` already validates examples as part
of its traversal, and `Audit` is what the README tells people to run. Keep
`Audit` and `ValidateAllExamples`; the other three go, along with
`AuditWithOptions` and `AuditOptions` folded into `Audit(app, opts ...AuditOption)`.

## The projected surface

| Cut | Names |
|---|---|
| Test helpers move to `clihelp/clihelptest` (A1) | 7 |
| Installer family unexported (B1) | 10 |
| Text and measurement helpers move to `internal/` (A4) | 11 |
| Help-system renderers unexported or deleted (B2) | 4 |
| Example-validation and audit collapse (B3) | 5 |
| `Explain`, `SupportedShells` unexported | 2 |
| **Total** | **39 of 91** |

That leaves about **52**: the five data types an author fills in, ten `Option`
constructors, twelve validators, seven `Gen*` writers for packagers, the two
mountable commands, and a dozen methods on `App`. It reads as a library rather
than as a library with an installer, a renderer and a test framework attached.

Three more are arguable and I would not do them without a reason:
`App.CollectOptions` (called twice internally, once by `doc/`, never by a
consumer — it is really part of the internal helper set), `App.PrintError` (a
consumer rarely needs it, since `Execute` already prints what it returns), and
the `Render` dispatcher itself, which exists to choose between two functions the
caller could choose between.

---

# Addendum 2 — checked against a real application

A 65,000-line program built on this library (`mail_cli`: 77 commands, 90
examples, 46 declared parameters) was read as evidence. Nothing in it was
modified. This section replaces guesswork with one data point — a strong one for
what is needed, a weak one for what is not, since one program's silence does not
make a feature dead.

## What it uses

Twenty-three distinct exported names, and thirteen `Command` fields:

```
Command  Example  Context  Param  Bool  NoArgs  ExactArgs  Option  Options  App
String   MaximumNArgs  RangeArgs  MinimumNArgs  Group  MutuallyExclusive  Int
Audit    ValidateOptions  ManPageCommand  IntegrationPath  Enum  CompletionCommand
```

`Name`, `Description`, `UsageLine`, `Examples`, `Run`, `Args`, `Parameters`,
`Subcommands`, `Options`, `Aliases`, `Hidden`, `OptionsValidator`,
`LongDescription`.

Never touched, by a program of that size: `Command.Notes`, `Command.Group`,
`Command.Deprecated`, `App.Shortcuts`, `Theme`, `Walk`, `BoolToggle`, `Duration`,
`StringSlice`, `Var`, `Required`, `StripANSI`, `VisualWidth`, `Inline`. That is
not an argument for removing them. It is an argument that the demo application
this audit's first numbers came from is a showcase rather than a sample.

## Two things this reduction got wrong

Both were found by building the application against this branch.

**`Option.Binder` had to stay exported.** The application sets it by hand:

```go
Binder: func(fs *pflag.FlagSet) error {
    fs.StringVarP(&app.FlagMoveSpamStr, "move", "m", "", "...")
    if f := fs.Lookup("move"); f != nil { f.NoOptDefVal = "true" }
    return nil
},
```

That is a flag whose value is optional — `-m` means true, `-m=addr` means an
address — which needs pflag's `NoOptDefVal`, and which clihelp has no constructor
for. Unexporting `Binder` closed the only door out of the library's vocabulary.
It is exported again, documented as the one place pflag is named on purpose.

**The four path queries had to stay exported.** `IntegrationPath`,
`CompletionPath`, `ManPagePath` and `IsCompletionInstalled` compute or stat a
path and change nothing. The line drawn in B1 — "reads or writes inside the
user's home" — was wrong; the right one is **actions go through the setup
command, queries stay open**. The application's own test asks `IntegrationPath`
where `completion install` put the file, in order to check that it worked. That
is exactly what a query is for.

With both restored, the application builds and its whole test suite passes
against this branch after **one line changed**: `AutoInstallCompletion` to
`AutoRefreshIntegration`.

## Three features it had to build itself

This is what an API audit cannot find by reading the surface, and it is worth
more than the rest of this document.

### F1 — Show the command's help when required arguments are missing

`cli/usage_wrapper.go`, 81 lines, recursively rewrites the entire command tree to
get one behaviour: a command invoked with no arguments, which requires some,
should print its own help rather than a generic argument error.

The workaround is expensive and fragile. It **disables `Args` validation**
(returns `nil` where the validator said no) and re-implements the check in three
more places; it calls `origArgs([]string{})` as an *oracle*, up to four times per
invocation, to ask "would zero arguments be acceptable"; it silently swallows
`PreRun` and `PostRun` on that path; and it replaces `Run` on **every command in
the tree**, so clihelp's own view of which commands have handlers is now always
"yes".

It should be a field — `Command.HelpOnMissingArgs`, or `App`-wide — or simply the
default, since a bare `error: accepts 1 arg(s), received 0` is worse than the
help page for every CLI I can think of.

### F2 — An unknown command should still be an unknown command when `App.Run` is set

`cli/suggest.go`, 112 lines, reimplements `suggestCommand`, `levenshtein`,
subcommand-path search, hidden filtering and alias matching — all of which
`resolve.go` already has, and which v0.3.23 spent a release getting right.

The cause is structural: an application that defines `App.Run` receives leftover
arguments itself, so clihelp's unknown-command check never fires and its
suggestions never appear. The author had no way to say "handle a bare invocation,
but let the library keep rejecting typos". Worse, F1's wrapper replaces `Run`
throughout the tree, so its `unknown subcommand %q` — with no suggestion at all —
shadows the library's good error for every subcommand too.

A real application therefore has **worse** command-not-found messages than the
library provides, because the only way to get the behaviour it wanted was to take
over the path that produces them.

### F3 — An option whose value is optional

The `Binder` escape hatch above exists only because there is no constructor for
it. `Bool` cannot carry a value, `String` requires one. This is also the only
reason a consumer of this library imports pflag at all — which makes it the last
mile of the pflag decoupling, not a separate feature.

## What this changes about the roadmap

The surface reduction was worth doing and is now verified against a real
consumer at the cost of one line. But the ratio is worth stating plainly: this
audit removed about thirty names nobody was using, while the same application
carries **193 lines written to work around three things the library does not
do** — two of which make its user-facing behaviour worse than clihelp's own.

Adding F1, F2 and F3 is now more valuable than removing anything further.
