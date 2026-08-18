# Flag & Option Specification Reference

`clihelp` integrates with `pflag` to provide declarative, typed, and expressive command-line option bindings.

---

## Table of Contents

- [Flag Specification Syntax](#flag-specification-syntax)
- [Option Constructors Reference](#option-constructors-reference)
- [Boolean Toggle Flags (`--[no-]flag`)](#boolean-toggle-flags---no-flag)
- [Enumerations & Custom Value Types](#enumerations--custom-value-types)
- [Advanced Option Fields](#advanced-option-fields)
- [Automatic Help Flag Collision Trap](#automatic-help-flag-collision-trap)

---

## Flag Specification Syntax

Option constructors parse a flag specification string that defines short names, long names, aliases, and value placeholders:

| Syntax Pattern | Example Spec | Parsed Result |
|---|---|---|
| Single Short Flag | `"-v"` | Binds short flag `-v`. |
| Single Long Flag | `"--verbose"` | Binds long flag `--verbose`. |
| Combined Short & Long | `"-o, --output PATH"` | Binds `-o` and `--output` with placeholder `PATH`. |
| Multiple Aliases | `"-p, -P, --port, --listen-port <num>"` | Binds `-p`, `-P`, `--port`, `--listen-port`. |
| Boolean Toggle | `"--[no-]cache"` | Registers `--cache` (`true`) and `--no-cache` (`false`). |
| Value Placeholders | `<file>`, `PATH`, `[value]`, `NAME` | Uppercase or bracketed tokens set help placeholder text. |

---

## Option Constructors Reference

All option constructors initialize the target variable with the default value and return a `clihelp.Option`:

| Constructor | Spec Example | Target Type | Description & Default Handling |
|---|---|---|---|
| `clihelp.String(target, spec, default, usage)` | `"-o, --output PATH"` | `*string` | Sets `*target = default`. Parses string value. |
| `clihelp.Int(target, spec, default, usage)` | `"-p, --port N"` | `*int` | Sets `*target = default`. Parses integer. |
| `clihelp.Bool(target, spec, default, usage)` | `"-v, --verbose"` | `*bool` | Sets `*target = default`. Sets `true` when passed. |
| `clihelp.BoolToggle(target, spec, default, usage)` | `"--[no-]normalize"` | `*bool` | Registers positive and negative toggle flags. Sets `*target = default`. |
| `clihelp.Duration(target, spec, default, usage)` | `"-t, --timeout SEC"` | `*time.Duration` | Parses durations like `"30s"`, `"5m"`. Sets `*target = default`. |
| `clihelp.StringSlice(target, spec, default, usage)` | `"-i, --include ITEM"` | `*[]string` | Repeatable or comma-separated string list. |
| `clihelp.Enum(target, spec, allowed, default, usage)` | `"-s, --stage STAGE"` | `*string` | Restricts value to `allowed` slice; provides completion candidates. |
| `clihelp.Var(targetValue, spec, usage)` | `"--custom VAL"` | `pflag.Value` | Binds custom type implementing `pflag.Value`. |

---

## Boolean Toggle Flags (`--[no-]flag`)

Toggle flags allow command users to explicitly enable or disable a boolean feature:

```go
var normalize bool
clihelp.BoolToggle(&normalize, "--[no-]normalize", true, "Apply LUFS audio normalization")
```

- Users can pass `--normalize` to set `true`
- Users can pass `--no-normalize` to set `false`
- If neither is passed, `normalize` retains its default (`true`).

---

## Enumerations & Custom Value Types

### `clihelp.Enum`

Use `clihelp.Enum` to constrain flag input to a known set of allowed values and automatically generate shell tab-completion suggestions:

```go
var stage string
clihelp.Enum(&stage, "-s, --stage STAGE", []string{"staging", "production"}, "staging", "Target deployment environment")
```

If an invalid value is supplied, `clihelp` returns an error:
```
Error: invalid value "testing": must be one of [staging, production]
```

### Custom Types via `clihelp.Var`

Bind any type implementing `pflag.Value` (`String() string`, `Set(string) error`, `Type() string`):

```go
type IPFilter struct {
    IPs []net.IP
}

func (f *IPFilter) String() string { /* ... */ }
func (f *IPFilter) Set(val string) error { /* ... */ }
func (f *IPFilter) Type() string { return "ip" }

var filter IPFilter
clihelp.Var(&filter, "--ip <address>", "Allowed IP address")
```

---

## Advanced Option Fields

The returned `clihelp.Option` struct exposes configuration hooks:

```go
opt := clihelp.String(&host, "-H, --host <host>", "127.0.0.1", "Bind IP host address")
opt.DefaultText = "localhost"                     // Custom display string for default value in help
opt.Hidden = true                                 // Hide from terminal help output and completions
opt.Deprecated = "Use --listen-addr instead"       // Render deprecation notice
opt.Complete = func(toComplete string) []string { // Dynamic tab completion callback
    return []string{"127.0.0.1", "0.0.0.0"}
}
```

---

## Automatic Help Flag Collision Trap

> [!CAUTION]
> **Never manually declare `-h` or `--help` in Options**: `clihelp` automatically binds `-h` and `--help` to every command during execution. Explicitly registering a help flag will cause a `pflag` panic at runtime:
>
> ```
> panic: help flag redefined: help
> ```
>
> Always let `clihelp` manage help flags and help text generation automatically.
