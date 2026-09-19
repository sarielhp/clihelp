# How clihelp decides

This document is the mechanism, not the API. The API is in `go doc`; the recipes
are in [recipes-and-patterns.md](recipes-and-patterns.md). What is here is the
set of decisions clihelp makes on your behalf, what it reads to make each one,
and what you lose by leaving that input undeclared.

It is written this way because the rules are derivable from the mechanism but not
the other way round. An author — human or agent — who knows *"the library decides
whether an unrecognised word is a typo by reading the arity of `Args`"* will
declare `Args` correctly in situations no rule anticipated. An author given the
rule *"declare `Args` to prevent out-of-bounds index errors"* will declare it and
still be surprised, because the rule names one consequence out of four.

That is not hypothetical. A 65,000-line application built on this library
declared `Args` on all 71 of its commands, and then wrote 81 lines rewriting its
own command tree to get a behaviour that the same declaration already entitled it
to. Nothing had told it that `Args` was an input the library reads.

---

## The one idea

**clihelp acts on what you declare, not on what your handlers do.**

Every field on `App`, `Command` and `Option` is an input to a decision the
library makes before, and sometimes instead of, calling your code: which command
runs, what the help page says, what the shell offers at `<Tab>`, what the manual
page contains, what an error says. Anything you do imperatively inside `Run` is
invisible to all of it.

The corollary is the one mistake worth naming first: **if you find yourself
writing code to produce behaviour the help page already describes, you are
working against the library.** Check this document before writing it.

---

## The decisions

### 1. Which command runs

- **Reads:** the argument list; `App.Commands`, `Command.Subcommands`,
  `Command.Name`, `Command.Aliases`; `App.Shortcuts`; `App.AbbrevCommands`; and
  the *arity of every flag*, so that `--out file build` knows `file` is `--out`'s
  value and `build` is the command.
- **Undeclared:** exact names and aliases only; no abbreviation.
- **Cost of getting it wrong:** resolution stopping early, so the command name
  becomes a positional argument, help prints, and the program exits 0. Six
  distinct instances of that were fixed in v0.3.23; all six looked like success
  to a calling script.

### 2. Whether an unrecognised word is a typo or an argument

- **Reads:** `Command.Args` / `App.Args` arity, and whether a handler exists. A
  command that can take positional arguments owns the words after it —
  `myapp scan inbox` is `scan`'s argument, not a misspelled subcommand. A command
  that cannot is being handed something it has no use for, so the word is a typo.
- **Undeclared:** an application *with commands* is assumed to take no positional
  arguments of its own; an application with no commands is unaffected.
- **Cost:** typos accepted as arguments, exit 0, and no "did you mean". If you
  set `App.Run` to handle a bare invocation — a good reason, and a common one —
  declare `App.Args: clihelp.NoArgs` unless the root really does take arguments.

### 3. Whether a missing argument gets an error or a help page

- **Reads:** `Command.Args` arity and how many arguments were supplied.
- **Behaviour:** *none at all* supplied to a command that needs some prints that
  command's help to stderr, then the error, and exits non-zero. A *wrong number*
  keeps its precise message, because that user tried and a page of help is noise.
- **Cost:** `accepts 1 arg(s), received 0` is the whole answer the user gets, and
  it never names the parameter they are missing.

### 4. What `Parameters` are for

- **Reads:** `Command.Parameters` — a `Param` per positional argument.
- **Used by:** the help page's Parameters section, the generated usage line when
  `UsageLine` is absent, and the manual page.
- **Cost:** the user is told the command takes an argument but never what it is.
  `Parameters` and `Args` describe the same thing from two sides; keep them
  agreeing, and let `Audit` check it.

### 5. Which help the user gets

- **Reads:** which flag was given, plus `App.ExtendedHelpFlag`.
  - `-h` (bound to the hidden long name `--help-concise`) — concise: no notes,
    held to a line budget, with a footer pointing at the extended form.
  - `--help` — extended: `LongDescription`, every note, every example.
  - `-H` — the same as `--help`, only when `App.ExtendedHelpFlag` is true.
- **Never declare any of those four yourself.** They are bound for you and a
  collision is an error at startup.

### 6. What goes on a help page, and where

- `Command.Description` — one sentence. Appears in the parent's command table,
  and at the top of both help tiers. Keep it to a line.
- `Command.LongDescription` — paragraphs. Extended help only.
- `Command.Notes` — sections with headings. Extended help only. `Note.Raw` keeps
  the body verbatim, for tables and diagrams that must not be re-wrapped.
- `Command.Examples` — both tiers, the manual page, and `help examples`, which
  collects every example in the tree into one page; `Audit` checks that each one
  actually parses against the command tree.
- `Command.Group` — a heading in the parent's command list. Commands with no
  group render ungrouped; if *any* sibling has one, the rest fall under a default
  heading rather than floating.
- `Command.Hidden` — out of help and out of completion, still runnable.
- `App.GlobalNote` — the application's own note: where to read more, who
  maintains it. Under `Description` on the extended global help, in the manual
  page, and in the generated Markdown; left off the concise tier.

### 7. How wide, how tall, and whether to page

- **Reads:** `Options.Width`, else the terminal, else **70 columns**;
  `Options.MaxContentWidth`, else 80; `App.Pager` / `Options.Pager` with the
  terminal height.
- **Note:** 70 is what every redirected `--help` in every program built on this
  library is laid out at.

### 8. Whether to emit colour

- **Reads:** `App.NoColor` (application-wide), `Options.NoColor` (one render),
  and `fatih/color`'s process-wide switch, which is off already whenever stdout
  is not a terminal.
- **Consequence:** with colour off, a `[label](url)` link renders as its label
  alone — never as a bare URL. Only the manual page asks for the `text (url)`
  form, and it asks explicitly.

### 8a. Whether a flag's value may be omitted

- **Reads:** the placeholder's brackets in the spec string, and `Optional`.
  `"--move [From]"` wrapped in `clihelp.Optional(opt, "true")` accepts `--move`
  on its own — the target gets `"true"` — or `--move=x@y.z`.
- **An equals sign is required for the value.** With `--move x` there is no way
  to tell the value from the next positional argument, and guessing is how a
  command name gets eaten.
- **The two halves must agree.** Brackets without `Optional` is caught by
  `Audit`; `Optional` without brackets is refused at bind time. A help page that
  promises an omittable value the flag does not accept is found by typing the
  flag.

### 8b. Which flags a given command has

- **Reads:** `App.PersistentOptions` and `App.GlobalFlags` (every command),
  `App.Options` (the root alone), each ancestor's `PersistentOptions`, and the
  target's own `PersistentOptions` and `Options`.
- **The root distinction matters** when the root does real work and its
  subcommands reuse its flag names. `App.Options` is the counterpart to
  `Command.Options`: bound when the application itself runs, not inherited, so
  a subcommand may declare `--channel` without colliding with the root's.
- **Cost of using the inherited kind instead:** `Audit` refuses the application
  for a duplicate option, which is the right answer to the wrong declaration.

### 9. What the shell offers at `<Tab>`

- **Reads:** the command tree, the bound flags, and `Option.Complete` for dynamic
  values. Hidden commands and options are excluded.
- **Cost:** no `Option.Complete` means the shell can complete the flag's *name*
  but never its value.

### 10. Whether a required option may be prompted for

- **Reads:** `Option.Required`, `App.InteractiveFallback`, and whether output is
  a terminal.
- **Behaviour:** a missing required flag is an error — unless
  `InteractiveFallback` is set *and* the session is interactive, in which case it
  is prompted for. A script therefore still fails rather than hanging.

### 11. What is written into the user's home directory

- **Reads:** `App.AutoRefreshIntegration`, plus whether a file is already there
  carrying clihelp's marker.
- **Rule:** the unattended path **only ever refreshes a file that already
  exists**. It never creates one and never edits a shell startup file. Creating
  is what the setup command is for, and the user runs that once.

---

## Invariants you can rely on

These are always true, so code that re-checks them is dead weight:

1. By the time `Run` is called, flags are parsed and **removed**; `ctx.Args`
   holds positional arguments only.
2. `ctx.Args` has already passed `Command.Args`. Indexing what the validator
   guaranteed needs no second check.
3. `Command.OptionsValidator` has already run, and `Option.Required` has already
   been enforced.
4. The flag spec string — `"--tag <v>, -t, -T"` — is the only place a flag's
   names live. Every spelling binds to the same target, and "was it set" is true
   for the option however the user spelled it.
5. `pflag` is an implementation detail. The only place it is named in the public
   API on purpose is `Option.Binder`.
6. `-h`, `--help`, `--help-concise` and `-H` are bound for you on every command.
7. Everything after `--` is a positional argument.
8. **A binding target must outlive the declaration.** `clihelp.String(&cfg.Output, …)`
   stores the pointer and writes through it during `Execute`, long after the
   declaration returned. Point at a field of a struct that lives as long as the
   application, or at a package-level variable — never at a local that goes out
   of scope, which is the one memory-shaped mistake this API makes easy.

---

## Already done — do not build these

The single most expensive mistake is rebuilding something the library already
does, because nothing in an API listing tells you what *not* to write. All of
these exist:

| Do not write | It is already |
|---|---|
| "Did you mean …?" for a mistyped command | `resolve.go`, including nested paths (`prune` → `cache prune`), aliases, and hidden-command filtering |
| Levenshtein or prefix matching over command names | the same, plus `App.AbbrevCommands` for unique prefixes |
| Code to print help when a command is called bare | decision 3 |
| A wrapper that intercepts `Run` to show usage | decision 3 — and it will shadow the errors above |
| Your own `-h` handling | decision 5 |
| Width-aware wrapping, column alignment, ANSI-safe truncation | `format.go`, and exported as `VisualWidth` / `StripANSI` if you need to match it |
| A second renderer to check the first | mutation-tested at 38/38; a golden test of your own output is cheaper and catches more |
| Shell completion scripts, Alt-H key bindings, a man page | one mounted `CompletionCommand()`, one command for the user |
| Static checks that your examples still parse | `Audit(app)` — run it in CI |

---

## Escape hatches, and what they cost

Each of these buys you something the vocabulary above cannot express, and each
makes the library blind to something in exchange.

- **`Option.Binder`** — bind a flag yourself, reaching any pflag feature
  (`NoOptDefVal` for an option whose value is optional, for instance). The
  library can still render and complete it, because `Flags` and `Description`
  are still declared; what it cannot do is know the default or the value shape
  beyond what the spec string says.
- **`ArgsFunc(fn)`** — a positional rule the built-ins cannot express. Its arity
  is unknowable, so decisions 2 and 3 fall back to *asking* it, and the usage
  line cannot be generated from it.
- **`Optional(opt, whenBare)`** — a value that may be omitted. The cost is the
  sentinel: `whenBare` has to be a string the real values cannot be, because
  that is what tells "given bare" from "given a value".
- **`Var(target, …)`** — a custom value type. Example validation cannot write
  through it, so it binds a permissive stand-in instead.

Prefer the declared form. Reach for these when the declared form genuinely
cannot say what you mean, not to save a line.

---

## The short version

Before writing code in a `Run` handler that produces user-visible structure —
help, usage, suggestions, errors about arguments — ask which of the eleven
decisions above it belongs to, and whether declaring a field would get it for
free. It usually does, and the declared form is the one the help page, the
completion script, the manual page and `Audit` can all see.

---

## A complete application

Everything above, in the shape it is meant to be written: one struct literal, no
`init()` wiring, and every decision expressed as a declaration.

```go
package main

import (
	"fmt"
	"os"

	"github.com/sarielhp/clihelp"
)

// The targets outlive the declaration — invariant 8.
type config struct {
	Verbose bool
	Output  string
}

func main() {
	var cfg config

	app := &clihelp.App{
		Name:        "myapp",
		Description: "Sample CLI tool",
		Version:     "1.0.0",
		Pager:       true,
		// The root takes no positional arguments, so an unrecognised first word
		// is a typo and gets a suggestion — decision 2.
		Args: clihelp.NoArgs,
		PersistentOptions: []clihelp.Option{
			clihelp.Bool(&cfg.Verbose, "-v, --verbose", false, "Enable verbose output"),
		},
		Commands: []clihelp.Command{
			{
				Name:        "process",
				Description: "Process input file",
				UsageLine:   "myapp process [options] <input>",
				// Arity: one argument. Decisions 2 and 3 both read this, so
				// "myapp process" prints this command's help rather than
				// "accepts 1 arg(s), received 0".
				Args: clihelp.ExactArgs(1),
				// What that argument is — decision 4.
				Parameters: []clihelp.Param{
					{Name: "<input>", Description: "File to process."},
				},
				Options: []clihelp.Option{
					clihelp.String(&cfg.Output, "-o, --output PATH", "out.bin", "Output file path"),
				},
				Examples: []clihelp.Example{
					{Line: "myapp process notes.txt -o notes.bin"},
				},
				Run: func(ctx *clihelp.Context) error {
					// ctx.Args has already passed ExactArgs(1) — invariant 2.
					fmt.Fprintf(ctx.Stdout, "Processing %s -> %s (verbose=%v)\n",
						ctx.Args[0], cfg.Output, cfg.Verbose)
					return nil
				},
			},
			clihelp.CompletionCommand(), // one command, and the user runs it once
		},
	}

	if err := app.Execute(os.Args[1:]); err != nil {
		app.PrintError(err) // a method on App, not a package function
		os.Exit(1)
	}
}
```

Run `clihelp.Audit(app)` in CI: it walks the tree and checks that every example
above still parses against the commands and flags that exist.
