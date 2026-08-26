# Comparing `clihelp` and `spf13/cobra`

[`spf13/cobra`](https://github.com/spf13/cobra) is the most widely adopted CLI framework in the Go ecosystem, powering tools such as Kubernetes (`kubectl`), GitHub CLI (`gh`), Hugo, Docker CLI, and Helm. `clihelp` utilizes Cobra's underlying POSIX flag library, `github.com/spf13/pflag`, for flag parsing.

While Cobra excels at building large CLI ecosystems, `clihelp` offers a different design model focused on declarative struct composition, zero global state, integrated terminal theming, and built-in Markdown documentation generation.

---

## Feature Comparison Matrix

| Feature | `spf13/cobra` | `clihelp` |
|---|---|---|
| **Ecosystem** | De facto standard across cloud-native tooling | Lightweight, focused alternative |
| **Command Definition** | Imperative tree construction via `init()` and `AddCommand()` | Pure declarative Go struct tree instantiation |
| **State Management** | Typically relies on package-level `var`s and `init()` | Zero global state; fully encapsulated in struct instances |
| **Terminal Output** | Monochromatic, plain-text default layout | Theme-driven ANSI colors, width detection, OSC 8 links |
| **Markdown Documentation** | Separate package (`cobra/doc`) | Built-in GitHub Markdown tree generator with SHA-256 caching |
| **Flag Specification** | Method-based binding (`Flags().StringVarP(...)`) | Concise spec string (e.g. `"-o, --output PATH"`) |
| **Shell Completion** | Bash, Zsh, Fish, PowerShell | Bash, Zsh, Fish |
| **AI Agent Prompting** | Multi-step `init()` wiring patterns | Self-contained struct literal with `llms.txt` spec |

---

## Architectural Differences

### 1. Declarative Struct Trees vs. Imperative `init()` Wiring

In typical Cobra applications, commands are defined as package-level global variables and attached to parent commands inside `init()` functions:

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

#### `clihelp` Pattern (Declarative Struct Literal)

`clihelp` avoids package-level global variables and `init()` mutations. An entire application tree, its flags, and lifecycle hooks are defined in a single struct value:

```go
func NewApp(cfg *Config) *clihelp.App {
    return &clihelp.App{
        Name:        "app",
        Description: "Application root",
        Pager:       true,
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

## Key Differences & Tradeoffs

### 1. Zero Global State & Testability
Because `clihelp` applications do not rely on `init()` functions or global pointers:
- Multiple isolated `*clihelp.App` instances can be created and run concurrently during testing.
- Target `io.Writer` destinations (`Stdout`, `Stderr`) and `context.Context` are passed explicitly through `clihelp.Context` without modifying global state.

### 2. Built-in Terminal Formatting, Theming & Clickable Hyperlinks
- **Plain Text vs. Native Theming**: Cobra renders monochromatic plain-text help by default. Getting colors in Cobra requires overriding its internal template engine (`SetHelpTemplate` or `SetHelpFunc`). In contrast, `clihelp` provides a built-in, theme-driven ANSI engine with headers, accents, and separators configured via `Theme`.
- **Clickable OSC 8 Hyperlinks**: `clihelp` translates Markdown links `[title](https://...)` into terminal OSC 8 hyperlinks out-of-the-box. Clicking the link in modern terminals opens the URL directly.
- **Inline Markdown Reflow**: Descriptions and notes support `` `code` ``, `**bold**`, `*italic*`, and `~~strikethrough~~`.
- **ANSI-Aware Wrapping**: If you embed ANSI escape codes or hyperlink escape sequences in Cobra descriptions, Cobra's string wrapper measures raw byte lengths rather than visible character widths, which can break line wraps and table alignment. `clihelp` measures visual rune width (stripping ANSI escapes during layout calculation) so formatted text always wraps cleanly.

### 3. Integrated Markdown Documentation Site Generator
`clihelp` includes built-in generation of navigable GitHub Markdown documentation trees (`RenderMarkdown`) with SHA-256 caching, eliminating the need for external documentation tools.

### 4. Expressive Flag Specification Syntax
`clihelp` combines flag names, aliases, placeholders, and descriptions into concise declarations:

```go
// Short + long + placeholder + default + description
clihelp.String(&out, "-o, --output <path>", "dist", "Destination path")

// Boolean toggle pair (--cache / --no-cache)
clihelp.BoolToggle(&cache, "--[no-]cache", true, "Enable build cache")
```

### 5. AI Agent & LLM Code Generation
Self-contained struct declarations are straightforward for LLMs to generate in a single pass without hallucinating missing `init()` registrations or unimported flag methods.

---

## When to Choose Cobra

Cobra is well-suited for projects that:
- Rely on plugins or integrations in the Cobra ecosystem (e.g. `kubectl` plugin frameworks).
- Need PowerShell or Windows completion scripts out-of-the-box.
- Rely heavily on `spf13/viper` configuration bindings across large multi-package architectures.
- Are maintained by teams with established conventions around Cobra.

---

## Summary

- Choose **Cobra** for large enterprise ecosystems with existing tooling tied to Cobra and Viper.
- Choose **`clihelp`** when you prefer a declarative, zero-global-state architecture with built-in theming, integrated Markdown documentation generation, and straightforward unit testing.
