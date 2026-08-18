# GitHub Markdown Documentation Generation

`clihelp` includes a built-in markdown documentation generator that produces an interlinked, navigable GitHub Markdown documentation tree for your CLI application.

---

## Table of Contents

- [Overview](#overview)
- [How to Integrate](#how-to-integrate)
- [SHA-256 Change Detection](#sha-256-change-detection)
- [CI / Pre-Commit Workflow](#ci--pre-commit-workflow)

---

## Overview

Calling `clihelp.RenderMarkdown(app, MarkdownOptions{})` generates:
- An `index.md` listing commands, global flags, and version information.
- Dedicated `.md` pages for every command and subcommand.
- Navigable relative links between parents, children, and peer commands.

---

## How to Integrate

Add a gated bootstrap check to your application's `main()`:

```go
func main() {
    app := &clihelp.App{ /* ... */ }

    // Run: CLIHELP_GEN=1 go run .
    if os.Getenv("CLIHELP_GEN") != "" {
        changed, err := clihelp.RenderMarkdown(app, clihelp.MarkdownOptions{
            Dir: "docs/clihelp",
        })
        if err != nil {
            fmt.Fprintf(os.Stderr, "error generating docs: %v\n", err)
            os.Exit(1)
        }
        if changed {
            fmt.Println("Documentation updated in docs/clihelp/")
        }
        return
    }

    if err := app.Execute(os.Args[1:]); err != nil {
        clihelp.PrintError(err)
        os.Exit(1)
    }
}
```

---

## SHA-256 Change Detection

The generator computes a SHA-256 hash of your command hierarchy and writes a sidecar `.clihelp-hash` file in the output directory.

- If the command tree has not changed, regeneration is skipped.
- The `.clihelp-hash` file is automatically added to `.gitignore` inside the output directory so it does not clutter git status.

---

## CI / Pre-Commit Workflow

Add documentation generation to your build script or CI workflow:

```bash
# Generate markdown documentation
CLIHELP_GEN=1 go run ./example

# Commit and push
git add -A docs/clihelp
git commit -m "docs: update CLI help pages"
git push
```
