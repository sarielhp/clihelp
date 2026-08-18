# Comparing `clihelp` and `spf13/cobra`

When designing command-line applications in Go, [`spf13/cobra`](https://github.com/spf13/cobra) is the venerable industry standard. Created by Steve Francia and maintained by a fantastic team of open-source contributors, Cobra powers legendary cloud-native tools including Kubernetes (`kubectl`), GitHub CLI (`gh`), Hugo, Docker CLI, and Helm. The Go ecosystem owes an enormous debt of gratitude to Cobra for pioneering structured CLI design in Go.

`clihelp` was built with deep respect for Cobra's foundation—in fact, `clihelp` uses Cobra's sister library `github.com/spf13/pflag` under the hood for POSIX-compliant flag handling. 

However, `clihelp` takes a different philosophical approach tailored for modern Go development, clean state management, rich terminal aesthetics, and AI-assisted workflows.

---

## High-Level Comparison Matrix

| Feature | `spf13/cobra` | `clihelp` |
|---|---|---|
| **Ecosystem & Community** | Industry standard (Kubernetes, GitHub CLI, Docker) | Focused, modern alternative |
| **Architecture Style** | Imperative tree construction via `init()` & `AddCommand()` | Declarative, pure struct tree instantiation |
| **Global State** | Common pattern relies on global `var`s & `init()` functions | Zero global state; fully encapsulated in structs |
| **Terminal Formatting** | Monochromatic, plain-text default layout | Theme-driven ANSI colors, width detection, OSC 8 links |
| **Markdown Help Sites** | Generates raw markdown files (requires `cobra/doc`) | Built-in GitHub Markdown tree generator with SHA-256 caching |
| **Flag Specification** | Separate method calls (`Flags().StringVarP(...)`) | Expressive single-string spec (`"-o, --output PATH"`) |
| **Shell Completion** | Bash, Zsh, Fish, PowerShell | Bash, Zsh, Fish |
| **AI Agent Friendliness** | Prone to `init()` mutation hallucinations | Single declarative literal, token-efficient, `llms.txt` |

---

## Architectural Differences

### 1. Declarative Struct Trees vs Imperative `init()` Mutation

In traditional Cobra applications, commands are defined as package-level global variables, and subcommand trees are wired imperatively inside `init()` functions:

#### Cobra Pattern (Imperative & Global)

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "app",
    Short: "Application root",
}

// cmd/build.go
var (
    output  string
    verbose bool
)

var buildCmd = &cobra.Command{
    Use:   "build [target]",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ...
        return nil
    },
}

func init() {
    buildCmd.Flags().StringVarP(&output, "output", "o", "dist", "Output file")
    buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
    rootCmd.AddCommand(buildCmd) // Mutates global command tree
}
```

#### `clihelp` Pattern (Pure Declarative Struct Literal)

`clihelp` eliminates package-level mutations and `init()` side-effects. The entire application, hierarchy, options, and hooks can be declared in a single, transparent Go struct literal:

```go
func NewApp(cfg *Config) *clihelp.App {
    return &clihelp.App{
        Name:        "app",
        Description: "Application root",
        Commands: []clihelp.Command{
            {
                Name:      "build",
                UsageLine: "app build [options] <target>",
                Args:      clihelp.ExactArgs(1),
                Options: []clihelp.Option{
                    clihelp.String(&cfg.Output, "-o, --output PATH", "dist", "Output file"),
                    clihelp.Bool(&cfg.Verbose, "-v, --verbose", false, "Verbose logging"),
                },
                Run: func(ctx *clihelp.Context) error {
                    target := ctx.Args[0]
                    fmt.Fprintf(ctx.Stdout, "Building %s -> %s\n", target, cfg.Output)
                    return nil
                },
            },
        },
    }
}
```

---

## Key Advantages of `clihelp`

### 1. Zero Global State & Superior Testability
Because `clihelp` applications do not rely on `init()` functions or global command pointers:
- You can instantiate multiple independent `*clihelp.App` instances concurrently in unit tests.
- Every render and execution accepts custom `io.Writer` destinations (`Stdout`, `Stderr`) and `context.Context` without monkey-patching global variables.

### 2. Built-in Aesthetic Terminal Experience
- **ANSI Color Themes**: Headers, flags, aliases, and descriptions are rendered with polished, theme-driven palettes out-of-the-box.
- **Dynamic Width Wrapping**: Word-wrapping is ANSI-aware (preventing broken escape sequences) and automatically matches the user's terminal width.
- **OSC 8 Hyperlinks & Inline Markdown**: Terminal descriptions support bold, italic, code, and clickable URLs (`[Docs](https://example.com)`).

### 3. Integrated GitHub Documentation Site Generator
While Cobra offers the external `cobra/doc` package, `clihelp` integrates GitHub-friendly Markdown generation directly:
- Generates fully linked, navigable Markdown documentation trees.
- Features SHA-256 hash change detection so documentation is only updated when the command hierarchy changes.

### 4. Expressive Flag Specification Syntax
Instead of remembering multiple separate method calls for long names, shorthand letters, default values, and usage strings, `clihelp` uses an intuitive spec string:

```go
// Short + long + placeholder + default + description
clihelp.String(&out, "-o, --output <path>", "dist", "Destination path")

// Automatic positive and negative boolean toggle flags
clihelp.BoolToggle(&cache, "--[no-]cache", true, "Enable build cache")
```

### 5. AI Agent & LLM-Assisted Coding Efficiency
AI coding assistants (Claude, GPT, Gemini, Cursor) thrive on declarative, self-contained syntax. Cobra's scattered `init()` functions and `Flags().StringVarP()` mutations frequently lead LLMs into hallucinated setup sequences. `clihelp`'s single-pass struct definitions and bundled [`llms.txt`](../llms.txt) provide a token-efficient, unambiguous target for AI code generation.

---

## When Cobra is Still the Right Choice

We enthusiastically recommend `spf13/cobra` if your project:
- Relies on plugins or extensions from the vast Cobra ecosystem (such as `kubectl` plugin frameworks).
- Requires PowerShell or Windows Command Prompt completion scripts.
- Integrates deeply with `spf13/viper` configuration bindings across large legacy codebases.
- Is maintained by large teams already deeply accustomed to Cobra's imperative conventions.

---

## Summary

- Use **Cobra** for massive enterprise ecosystems with existing tooling tied to Cobra's architecture.
- Use **`clihelp`** when you want a modern, declarative, zero-global-state CLI framework with gorgeous terminal output, integrated Markdown documentation generation, and a frictionless experience for both human engineers and AI assistants.
