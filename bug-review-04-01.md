# Bug Review — Task 4, Iteration 1

## Bug 8: Shortcut commands missing from empty-args completion

**File**: `completion.go:12-20`
**Severity**: Low

When `handleComplete` is called with no args (empty completion), it lists all root commands but does not include shortcut commands.

**Fix**: Also iterate `a.Shortcuts` and include non-hidden shortcuts in the output.

## Bug 9: Subcommand links broken when using SubcommandEntries with aliases

**File**: `md.go:357-363`
**Severity**: Low

`renderCommandPage` generates subcommand links by iterating `cmd.Subcommands` and matching by `Name`. If `SubcommandEntries` is used with entries whose names match subcommand aliases rather than primary names, the link is not generated.

**Fix**: Also check subcommand aliases when looking for the matching subcommand entry.