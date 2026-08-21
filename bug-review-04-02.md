# Bug Review — Task 4, Iteration 2

## Bug 10: Built-in help overrides custom "help" subcommand on nested commands

**File**: `execute.go:219`
**Severity**: Medium

The built-in `help` subcommand handler checks `!a.hasCommandNamed("help")` which only looks at top-level commands. If a nested command (e.g. `config`) has its own `help` subcommand, typing `podctl config help` triggers the built-in help instead of the custom subcommand.

**Fix**: Also check `!hasSubcommandNamed(currentCommands, "help")` to respect custom help subcommands at any level.

## Bug 11: "Detailed Help" text doesn't mention subcommands

**File**: `render.go:340-342`
**Severity**: Low

The "Detailed Help" section says `podctl <command> --help` but doesn't mention that subcommands can also have help. Updated to `podctl <command> [<subcommand>...] --help`.

## Bug 12: "Detailed Help" text not using body theme color

**File**: `render.go:340-342`
**Severity**: Low

The "Detailed Help" section text was printed with `fmt.Fprintf` instead of `th.Body.Fprintf`, so it didn't use the configured body color.