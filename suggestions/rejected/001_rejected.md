# Rejection Notice: 001 External CLI Usage Metadata & JSON Overlay

- **Proposal:** [`001_external_help.md`](file:///home/sariel/prog/26/go/clihelp/suggestions/rejected/001_external_help.md)
- **Status:** Rejected
- **Date:** 2026-09-04
- **Target Component:** [`clihelp`](file:///home/sariel/prog/26/go/clihelp)

---

## 1. Proposal Summary

The proposal suggested decoupling CLI routing logic (commands, flags, argument validators, execution callbacks) from documentation prose (descriptions, usage lines, notes, examples, option descriptions) by loading an embedded JSON catalog (`//go:embed assets/help.json`) at runtime via an overlay decorator pattern (`app.LoadMetadataJSON()`).

---

## 2. Reasons for Rejection

### A. Data Evasion vs. Idiomatic Function Decomposition
* **Claim:** Command initializers (such as `configCmd` in `bws`) exceed the 80-line function limit due to inlined strings, notes, and examples.
* **Finding:** Moving strings into an external JSON file to satisfy a function length limit is "data evasion"—it hides code in another language/format rather than fixing the underlying structural problem.
* **Resolution:** In Go, the idiomatic solution to large command initializers is **decomposition in place**: breaking subcommands into dedicated helper functions (`configShowCmd()`, `configSetCmd()`) or separate files (`cmd_config_set.go`). This achieves compliance with line limits in minutes without adding library infrastructure or runtime overhead.

### B. Phantom Requirements & YAGNI
1. **Internationalization (i18n):**
   * Developer-focused system CLI tools (such as `bws`) target technical users and operate exclusively in English.
   * Designing and maintaining an i18n overlay catalog for a CLI with no localization requirement is unnecessary complexity (YAGNI).
2. **Documentation Drift:**
   * The proposal argued that Go definitions drift from Markdown reference documentation.
   * `clihelp` already provides [`github.com/sarielhp/clihelp/doc`](file:///home/sariel/prog/26/go/clihelp/doc) (`doc.RenderMarkdown`), which automatically generates Markdown documentation directly from the compiled Go command tree. The compiled Go code is already the single source of truth.
3. **External Copy-Editing:**
   * Non-developer copywriters do not maintain these command strings; developers do. Co-locating documentation with implementation provides superior developer experience.

### C. Developer Experience (DX) & Type-Safety Regressions
* **Split-Brain Maintenance:** Developers must maintain two separate files in parallel (Go for flags/routing and JSON for descriptions).
* **Loss of Compile-Time Verification:** Renaming a flag or command in Go will not update JSON keys. Typos fail at runtime or require custom test assertions (`ValidateMetadata()`) rather than being caught instantly by the Go compiler and LSP (`gopls`).
* **Broken IDE Navigation:** Code symbols cannot be resolved across the JSON boundary. "Jump to definition" and automated refactoring tools cease to function for documentation text.

### D. Runtime Performance & Fragility
* **Startup Penalty:** Ingesting and unmarshaling a multi-kilobyte JSON tree with reflection and map allocations on every CLI invocation degrades cold-start latency on performance-sensitive commands (e.g. `bws run -- ...` and shell tab completion).
* **Fragile Option Matching:** Matching flag signatures (such as `"-p, --podcast <name>"` or `"-N, --no-net, --offline"`) against arbitrary JSON string keys requires runtime heuristic parsing, introducing subtle collision and mismatch bugs.

### E. Historical Precedent: The String Table Anti-Pattern
* The proposal closely resembles legacy Win32 resource scripts (`.rc` / `STRINGTABLE` / `LoadString(IDS_...)`). 
* Experience demonstrated that decoupled string tables introduce an indirection tax, accumulate orphaned "ghost" strings as code evolves, and decouple format specifiers from their call sites. Modern CLI design intentionally avoids this pattern.

---

## 3. Decision & Guidelines

1. **Do not implement runtime JSON overlays or external catalog loaders in `clihelp`.**
2. **Maintain Go definitions as the single source of truth** for all commands, options, notes, and examples.
3. **Resolve function sizing issues via decomposition:**
   - Extract subcommands into individual functions or dedicated files.
   - For commands with extensive notes or examples, declare them as package-level `var` blocks within the same package.
4. **Generate external documentation from Go:**
   - Continue using [`clihelp/doc`](file:///home/sariel/prog/26/go/clihelp/doc) (`doc.RenderMarkdown`) to produce external Markdown references when needed.
