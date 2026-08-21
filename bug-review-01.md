# Bug Review — Iteration 1

## Bug 1: Double render when using `help` subcommand

**File**: `execute.go:214-221`
**Severity**: Medium

When `help <command>` is used and the app has no custom `help` command, `resolveCommand` renders help directly and returns `nil, nil, nil, nil, nil`. Then `ExecuteContext` sees `targetCmd == nil && len(path) == 0 && len(remaining) == 0 && a.Run == nil` and renders global help again, producing duplicate output.

**Fix**: Add an early return in `ExecuteContext` after `resolveCommand` when the original args started with `"help"`.

## Bug 2: Ancestor persistent options not shown in command help

**File**: `render.go:410-420`
**Severity**: Medium

`RenderCommand` only collects `PersistentOptions` and `Options` from the target command itself. Ancestor persistent options (from parent/grandparent commands) are not displayed in the help output, even though they are bound and parsed during execution.

**Fix**: Traverse ancestors and include their `PersistentOptions` in the rendered flag list.

## Bug 3: Misleading "Detailed Help" suggestion

**File**: `render.go:340-342`
**Severity**: Low

The global help text says `podctl <command>` but the correct invocation is `podctl <command> --help`.

**Fix**: Update the text to say `podctl <command> --help`.