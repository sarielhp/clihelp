# Bug Review — Iteration 2

## Bug 4: Shortcut commands missing from shell completion

**File**: `completion.go:94-98`
**Severity**: Low

`handleComplete` only lists `a.Commands` at the root level. Shortcut commands (`a.Shortcuts`) are not included in completion results.

**Fix**: After listing regular commands at root level, also iterate `a.Shortcuts` and include matching shortcuts.

## Bug 5: App-level persistent options missing from command help

**File**: `render.go:410-428`
**Severity**: Medium

`RenderCommand` collects ancestor persistent options and the command's own options, but does not include the app-level `PersistentOptions`. These flags are bound and parsed during execution for every command, so they should appear in command help output.

**Fix**: Add app-level `PersistentOptions` to the collected flag list before ancestor and command options.