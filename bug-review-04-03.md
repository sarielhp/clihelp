# Bug Review — Task 4, Iteration 3

## Bug 13: URL parsing breaks on parentheses in URLs

**File**: `inline.go:49-61`
**Severity**: Low

The `[text](url)` link parser uses `strings.IndexByte` to find the closing `)`, which breaks on URLs containing parentheses (e.g. Wikipedia URLs like `https://en.wikipedia.org/wiki/Go_(programming_language)`).

**Fix**: Use a parenthesis depth counter to find the matching closing parenthesis, correctly handling nested parentheses in URLs.