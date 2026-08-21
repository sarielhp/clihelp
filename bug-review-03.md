# Bug Review — Iteration 3

## Bug 6: Empty "Commands:" section when all commands are hidden

**File**: `render.go:285-297`
**Severity**: Low

`RenderGlobal` always prints the "Commands:" header even when all commands are hidden or the command list is empty, resulting in a section with no entries.

**Fix**: Only print the "Commands:" section when there is at least one visible command.

## Bug 7: Example lines word-wrapped instead of printed verbatim

**File**: `render.go:446-453`
**Severity**: Medium

Example lines (typically code snippets like `podctl build ep.wav`) were passed through `reflow` which word-wraps them. Code examples should be printed verbatim to preserve their intended formatting.

**Fix**: Print example lines directly with `fmt.Fprintf` instead of passing through `reflow`.