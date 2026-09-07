# Specification: External CLI Usage Metadata & JSON Overlay for `clihelp`

**Document:** `clihelp_usage_msg.md`  
**Status:** Draft / Proposed  
**Target Component:** [`github.com/sarielhp/clihelp`](file:///home/sariel/prog/26/go/clihelp)  
**Primary Consumer:** [`bws`](file:///home/sariel/prog/26/bws) and `clihelp`-based applications  

---

## 1. Overview & Motivation

In CLI applications built with `clihelp`, command definitions currently couple **business routing logic** (flag pointers, argument validators, execution callbacks) with **long-form documentation prose** (descriptions, usage lines, multi-paragraph notes, and command examples).

### Problems in the Current Design
1. **Code Bloat & Sizing Violations**: In programs enforcing strict function size limits (e.g. 80-line hard limits), command initializers (such as `configCmd`, `profileCmd`, `gitWorkflowCmd` in `bws`) routinely exceed limits solely due to inlined strings and notes.
2. **Duplication & Documentation Drift**: Help messages are defined in Go structs and manually mirrored in Markdown reference documentation (`docs/commands.md`). Changes to copy require editing Go code and recompiling.
3. **No Internationalization (i18n)**: Supporting localized help requires branching Go code rather than swapping a data catalog.
4. **Code Review Noise**: Modifying a typo or adding a CLI example churns Go source files rather than documentation data.

### Solution
Allow `clihelp` to ingest a single, unified JSON document (embedded into the application binary at compile time via standard `//go:embed`) that hydrates all descriptive and presentational metadata across the entire command hierarchy via an **Overlay / Decorator Pattern**.

---

## 2. Architecture & Design Principles

```
  ┌────────────────────────────────────────────────────────┐
  │                   assets/help.json                     │
  │  (Single source of truth for descriptions, examples,   │
  │   notes, usage lines, and option text)                 │
  └───────────────────────────┬────────────────────────────┘
                              │
                    //go:embed assets/help.json
                              │
  ┌───────────────────────────▼────────────────────────────┐
  │                 Application Go Code                    │
  │  (Lean routing: Command names, flag pointers,          │
  │   argument validators, Run closures)                   │
  └───────────────────────────┬────────────────────────────┘
                              │
           app.LoadMetadataJSON(helpJSON)
                              │
  ┌───────────────────────────▼────────────────────────────┐
  │                Hydrated clihelp.App                    │
  │  (Rendered via clihelp's colorized terminal output)    │
  └────────────────────────────────────────────────────────┘
```

1. **Zero External Runtime Dependencies**: The JSON metadata file is baked into the Go binary via `//go:embed`. At runtime, the binary remains completely standalone with zero disk I/O and zero external file requirements.
2. **Clear Separation of Concerns**:
   - **Go Code**: Handles typed constructs — command names, flag variable pointers, argument counts (`clihelp.ExactArgs(2)`), and `Run: func(ctx *clihelp.Context) error`.
   - **JSON Metadata**: Handles presentation — `description`, `usage`, `group`, `notes`, `examples`, parameter explanations, and option descriptions.
3. **Zero New Third-Party Dependencies**: Parsed using Go standard library `encoding/json`.
4. **Fallback & Non-Destructive**: Metadata from JSON overlays onto the command tree. If a field is explicitly set in Go and omitted in the JSON, the Go value is preserved.
5. **Bidirectional Verification**:
   - `app.ValidateMetadata()` can be called in unit tests to ensure that every command and option has matching metadata and that no orphan keys exist in the JSON.
   - `app.ExportMetadataJSON()` can dump the current in-memory command tree into a canonical JSON template.

---

## 3. JSON Schema Specification

The metadata file (e.g. `assets/help.json`) is structured as a tree matching the application and command hierarchy.

### 3.1 Top-Level Schema (`HelpCatalog`)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "app": {
    "description": "string",
    "usage": "string",
    "global_note": "string",
    "examples": [
      {
        "line": "string",
        "description": "string"
      }
    ],
    "options": {
      "--flag": "string | OptionMetadata"
    }
  },
  "commands": {
    "<command_name>": {
      "description": "string",
      "usage": "string",
      "group": "string",
      "title": "string",
      "notes": [
        {
          "heading": "string",
          "text": "string"
        }
      ],
      "examples": [
        {
          "line": "string",
          "description": "string"
        }
      ],
      "parameters": [
        {
          "name": "string",
          "description": "string"
        }
      ],
      "options": {
        "<option_signature_or_name>": "string | OptionMetadata"
      },
      "subcommands": {
        "<subcommand_name>": {
          "...": "recursive CommandMetadata"
        }
      }
    }
  }
}
```

### 3.2 Option Metadata (`OptionMetadata`)

Option entries in `"options"` can be expressed either as a simple description string or a structured object:

```json
"options": {
  "-f, --force": "Bypass the file count safety check / force overwrite",
  "--proxy": {
    "description": "Tunnel outbound sandbox network traffic through an in-process host proxy",
    "group": "Network",
    "default_text": "disabled"
  }
}
```

### 3.3 Matching Rules
1. **Commands**: Looked up by primary `Name`. If not matched, lookup checks `Aliases`.
2. **Subcommands**: Traversed recursively using the `"subcommands"` map.
3. **Options**: Matched against registered `Option.Flags`. Matches on:
   - Exact signature: `"-f, --force"`
   - Long flag: `"--force"` or `"force"`
   - Short flag: `"-f"` or `"f"`

---

## 4. Go API for `clihelp`

### 4.1 Data Types in `clihelp`

```go
package clihelp

// OptionMetadata defines descriptive attributes for a flag.
type OptionMetadata struct {
	Description string `json:"description"`
	Group       string `json:"group,omitempty"`
	DefaultText string `json:"default_text,omitempty"`
}

// CommandMetadata defines descriptive attributes for a command node.
type CommandMetadata struct {
	Description string                     `json:"description,omitempty"`
	Usage       string                     `json:"usage,omitempty"`
	Group       string                     `json:"group,omitempty"`
	Title       string                     `json:"title,omitempty"`
	Notes       []Note                     `json:"notes,omitempty"`
	Examples    []Example                  `json:"examples,omitempty"`
	Parameters  []Param                    `json:"parameters,omitempty"`
	Options     map[string]json.RawMessage `json:"options,omitempty"`
	Subcommands map[string]CommandMetadata `json:"subcommands,omitempty"`
}

// AppMetadata defines application-level metadata and top-level commands.
type AppMetadata struct {
	Description string                     `json:"description,omitempty"`
	Usage       string                     `json:"usage,omitempty"`
	GlobalNote  string                     `json:"global_note,omitempty"`
	Examples    []Example                  `json:"examples,omitempty"`
	Options     map[string]json.RawMessage `json:"options,omitempty"`
}

// HelpCatalog represents the root JSON document.
type HelpCatalog struct {
	App      AppMetadata                `json:"app,omitempty"`
	Commands map[string]CommandMetadata `json:"commands,omitempty"`
}
```

### 4.2 Methods on `clihelp.App`

```go
// LoadMetadataJSON parses JSON bytes and overlays metadata onto the App and its command hierarchy.
func (a *App) LoadMetadataJSON(data []byte) error

// MustLoadMetadataJSON panics if JSON parsing or metadata overlay fails.
// Ideal for init() or main() setup.
func (a *App) MustLoadMetadataJSON(data []byte)

// ValidateMetadata inspects the loaded App against catalog JSON and returns
// errors for unmatched commands, unmatched flags, or missing required descriptions.
func (a *App) ValidateMetadata(data []byte) []error

// ExportMetadataJSON serializes the current in-memory App metadata to JSON.
func (a *App) ExportMetadataJSON() ([]byte, error)
```

---

## 5. Concrete Application Example: `bws`

### 5.1 JSON Asset: `assets/help.json`

```json
{
  "app": {
    "description": "Launch a declarative, unprivileged Bubblewrap sandbox with composable profiles, SSH forwarding, X11, and shell theming.",
    "usage": "bws [options] [command | -- [args...]]",
    "options": {
      "-f, --force": "Bypass the file count safety check / force overwrite",
      "-g, --global": "Target the global config file (~/.config/bws/config.jsonc)",
      "-l, --local": "Target the local config file (.bws/config.jsonc in current directory)",
      "--no-ssh": "Disable SSH agent forwarding and Git SSH commands",
      "-N, --no-net, --offline": "Completely block network access (air-gapped network namespace)",
      "--proxy": "Tunnel outbound sandbox network traffic through an in-process host proxy",
      "--no-proxy": "Disable the in-process host proxy",
      "--dbus": "Enable filtered session D-Bus access via xdg-dbus-proxy",
      "--no-dbus": "Disable session D-Bus access",
      "--no-init": "Skip auto-configuration of workspace when entering uninitialized directory",
      "-v, --verbose": "Print verbose debug information (config paths, bwrap args, etc.)"
    }
  },
  "commands": {
    "config": {
      "group": "Configuration & sync",
      "notes": [
        {
          "heading": "Configuration architecture",
          "text": "Configuration is stored in two JSONC files: global (~/.config/bws/config.jsonc) and local (.bws/config.jsonc). The local config overrides the global for the current workspace only."
        }
      ],
      "subcommands": {
        "show": {},
        "set": {
          "examples": [
            { "line": "bws config set enable_proxy true", "description": "Enable proxy in local workspace config" },
            { "line": "bws config set enable_ssh true -g", "description": "Enable SSH forwarding in global config" },
            { "line": "bws config set max_file_count 25000", "description": "Set file count safety limit" }
          ]
        },
        "get": {
          "examples": [
            { "line": "bws config get enable_proxy", "description": "Get proxy setting from local config" },
            { "line": "bws config get max_file_count -g", "description": "Get file limit from global config" }
          ]
        },
        "unset": {
          "examples": [
            { "line": "bws config unset enable_proxy", "description": "Remove proxy setting from local config" }
          ]
        },
        "edit": {},
        "where": {},
        "reset": {},
        "push": {
          "examples": [
            { "line": "bws config push user@laptop:", "description": "Push global config and theme to remote host" }
          ]
        },
        "completion": {
          "examples": [
            { "line": "source <(bws config completion bash)", "description": "Load bash completions immediately" },
            { "line": "bws config completion install", "description": "Install completion script into user profile" }
          ]
        }
      }
    }
  }
}
```

### 5.2 Go Code Before vs. After

#### Before: `commands_config.go` (129 lines, exceeds 80-line limit)
```go
// 129 lines inlining strings, notes, and examples for 8 subcommands...
func configCmd(f *appFlags, glValidator clihelp.OptionsValidator) clihelp.Command {
	return clihelp.Command{
		Name:             "config",
		Aliases:          []string{"conf"},
		Group:            "Configuration & sync",
		Description:      "Manage global and local sandbox configuration files, sync, and completions",
		UsageLine:        "bws config [subcommand] [-g | -l]",
		OptionsValidator: glValidator,
		Notes: []clihelp.Note{
			{Heading: "Configuration architecture", Text: "Configuration is stored in two JSONC files..."},
		},
		Subcommands: []clihelp.Command{
			{
				Name: "set",
				Description: "Set a configuration key value in local or global configuration",
				UsageLine: "bws config set <key> <value> [-g | -l]",
				Args: clihelp.ExactArgs(2),
				Examples: []clihelp.Example{ ... },
				Run: ...
			},
			// ... 7 more subcommands with repetitive prose ...
		},
	}
}
```

#### After: `commands_config.go` (~35 lines, well within 80-line limit)
```go
func configCmd(f *appFlags, glValidator clihelp.OptionsValidator) clihelp.Command {
	return clihelp.Command{
		Name:             "config",
		Aliases:          []string{"conf"},
		OptionsValidator: glValidator,
		Subcommands: []clihelp.Command{
			{Name: "show", Aliases: []string{"cat", "view"}, Args: clihelp.NoArgs, OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigShow(f.global, f.local); return nil }},
			{Name: "set", Args: clihelp.ExactArgs(2), OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigSet(ctx.Args[0], ctx.Args[1], f.global, f.local); return nil }},
			{Name: "get", Args: clihelp.ExactArgs(1), OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigGet(ctx.Args[0], f.global, f.local); return nil }},
			{Name: "unset", Args: clihelp.ExactArgs(1), OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigUnset(ctx.Args[0], f.global, f.local); return nil }},
			{Name: "edit", Args: clihelp.NoArgs, OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigEdit(f.global, f.local); return nil }},
			{Name: "where", Aliases: []string{"paths"}, Args: clihelp.NoArgs, Run: func(ctx *clihelp.Context) error { cli.HandleConfigWhere(); return nil }},
			{Name: "reset", Aliases: []string{"init"}, Args: clihelp.NoArgs, OptionsValidator: glValidator, Run: func(ctx *clihelp.Context) error { cli.HandleConfigReset(f.global, f.local); return nil }},
			{Name: "push", Aliases: []string{"scp", "sync"}, Args: clihelp.ExactArgs(1), Run: func(ctx *clihelp.Context) error { cli.HandleConfigPush(ctx.Args[0], f.verbose); return nil }},
			{Name: "completion", Args: clihelp.ExactArgs(1), Run: func(ctx *clihelp.Context) error { cli.HandleConfigCompletion(ctx.Args[0]); return nil }},
		},
	}
}
```

#### Application Entry Point: `main.go`
```go
package main

import (
	_ "embed"
	"github.com/sarielhp/clihelp"
)

//go:embed assets/help.json
var helpCatalogJSON []byte

func buildApp() *clihelp.App {
	app := &clihelp.App{
		Name: "bws",
		PersistentOptions: []clihelp.Option{ ... },
		Commands: []clihelp.Command{
			configCmd(flags, glValidator),
			profileCmd(flags, glValidator),
			mountCmd(flags, glValidator),
			// ...
		},
	}

	// Hydrate descriptions, usage lines, examples, and notes from JSON
	app.MustLoadMetadataJSON(helpCatalogJSON)
	return app
}
```

---

## 6. Testing & CI Verification

With this capability in `clihelp`, test suites can enforce complete documentation coverage with zero boilerplate:

```go
func TestCLIHelpMetadataCoverage(t *testing.T) {
	app := buildApp()
	errors := app.ValidateMetadata(helpCatalogJSON)
	for _, err := range errors {
		t.Errorf("CLI help metadata error: %v", err)
	}
}
```

This ensures:
1. Every registered command and subcommand has a non-empty `Description`.
2. Every declared flag has a descriptive help string.
3. No typos exist in JSON keys (e.g., a command renamed in Go but left under its old name in JSON is caught immediately during `go test`).

---

## 7. Migration Plan

1. **Step 1: Implement in `clihelp`**
   - Add `LoadMetadataJSON`, `MustLoadMetadataJSON`, and `ValidateMetadata` to [`clihelp.go`](file:///home/sariel/prog/26/go/clihelp/clihelp.go).
   - Add unit tests covering command hydration, alias resolution, nested subcommand traversal, and flag description overrides.
   - Tag/release updated `clihelp`.

2. **Step 2: Adopt in `bws`**
   - Create [`assets/help.json`](file:///home/sariel/prog/26/bws/assets/help.json) containing help messages for all commands.
   - Remove inlined descriptions, examples, and notes from `commands_*.go`.
   - Re-run `tools/audit_lines.rb` to verify all commands now satisfy the 80-line limit.
   - Add `TestCLIHelpMetadataCoverage` to `main_test.go`.
