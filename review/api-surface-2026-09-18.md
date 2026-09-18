# API surface audit, 2026-09-18

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
