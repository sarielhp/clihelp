**Policy**: After every step of any rewrite/implementation, perform `make commit` (or `git add` + `git commit` + `git push`) to save progress.

# TODO: Extend example with deep subcommand hierarchy

Modify `example/main.go` to generate a command tree with subcommands up to depth 5.

## Requirements

- Each command at depth < 5 must have exactly **2 subcommands** (binary tree structure).
- Each command at depth < 5 must have a **distinct usage message** containing:
  - Long lines that trigger word-wrapping
  - **Bold text** (markdown `**...**`)
  - An **embedded link** (markdown `[label](url)`)
- Commands at **full depth 5** should still generate help when invoked with `--help`.
- Keep existing commands (`build`, `serve`, `config`, `deploy`, `status`, `completion`) intact — add the deep tree as a new command (e.g. `deep` or `demo`).

## Structure

```
root
├── build (existing)
├── serve (existing)
├── config (existing)
├── deploy (existing)
├── status (existing)
├── completion (existing)
└── deep (new)
    ├── alpha
    │   ├── alpha-one
    │   │   ├── alpha-one-a
    │   │   │   ├── alpha-one-a-i (depth 5 — leaf)
    │   │   │   └── alpha-one-a-ii (depth 5 — leaf)
    │   │   └── alpha-one-b
    │   │       ├── alpha-one-b-i (depth 5 — leaf)
    │   │       └── alpha-one-b-ii (depth 5 — leaf)
    │   └── alpha-two
    │       ├── alpha-two-a
    │       │   ├── alpha-two-a-i (depth 5 — leaf)
    │       │   └── alpha-two-a-ii (depth 5 — leaf)
    │       └── alpha-two-b
    │           ├── alpha-two-b-i (depth 5 — leaf)
    │           └── alpha-two-b-ii (depth 5 — leaf)
    └── beta
        ├── beta-one
        │   ├── beta-one-a
        │   │   ├── beta-one-a-i (depth 5 — leaf)
        │   │   └── beta-one-a-ii (depth 5 — leaf)
        │   └── beta-one-b
        │       ├── beta-one-b-i (depth 5 — leaf)
        │       └── beta-one-b-ii (depth 5 — leaf)
        └── beta-two
            ├── beta-two-a
            │   ├── beta-two-a-i (depth 5 — leaf)
            │   └── beta-two-a-ii (depth 5 — leaf)
            └── beta-two-b
                ├── beta-two-b-i (depth 5 — leaf)
                └── beta-two-b-ii (depth 5 — leaf)
```

## Verification

- `go run ./example deep --help` — shows global help with `deep` listed
- `go run ./example deep alpha --help` — shows alpha help with wrapping, bold, link
- `go run ./example deep alpha alpha-one --help` — shows alpha-one help
- `go run ./example deep alpha alpha-one alpha-one-a --help` — shows alpha-one-a help
- `go run ./example deep alpha alpha-one alpha-one-a alpha-one-a-i --help` — shows leaf help
- `go run ./example deep alpha alpha-one alpha-one-a alpha-one-a-i` — runs the leaf command
- `make check` passes

---



# TODO: Bug audit & fix (Task 3)

Carefully and thoroughly analyze the entire library, write a detailed review, and fix **all** bugs found. Repeat this process **3 iterations** — in each iteration, fix every bug discovered before moving to the next. For every bug, design a new test to capture it (if possible).

## Process per iteration

1. Read all source files and test files
2. Write a detailed code review documenting every issue found
3. For each bug, write a failing test first
4. Fix the bug
5. Run `make check` to verify
6. Commit the iteration

## Deliverables

- Bug review document per iteration (e.g. `bug-review-01.md`, `bug-review-02.md`, `bug-review-03.md`)
- New tests for each bug
- All bugs fixed
- `make check` passes after each iteration

---

# TODO: Bug audit & fix (Task 4)

Repeat Task 3: carefully and thoroughly analyze the entire library, write a detailed review, and fix **all** bugs found. Repeat this process **3 iterations** — in each iteration, fix every bug discovered before moving to the next. For every bug, design a new test to capture it (if possible).

## Process per iteration

1. Read all source files and test files
2. Write a detailed code review documenting every issue found
3. For each bug, write a failing test first
4. Fix the bug
5. Run `make check` to verify
6. Commit the iteration

## Deliverables

- Bug review document per iteration (e.g. `bug-review-04-01.md`, `bug-review-04-02.md`, `bug-review-04-03.md`)
- New tests for each bug
- All bugs fixed
- `make check` passes after each iteration