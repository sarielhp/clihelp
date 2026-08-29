# Comparative Analysis: clihelp Limitations & Missing Features

This document outlines the features currently missing from `clihelp` as of version `v0.2.x` when compared to modern CLI tools (such as Git, Docker, and Cargo/Rust CLI) and cloud-native CLI libraries (like `spf13/cobra`).

---

## 1. Command-Line Parsing & Validation

### 🚫 Declarative Positional Argument Binding
* **Modern Standard:** Frameworks like Rust's `clap` or Python's `argparse` allow you to define positional parameters (e.g. `source_file`, `destination`) as structured fields. This enables automatic validation of optional vs. required arguments, type casting (e.g., automatically parsing integers, file paths, URLs), and name-based field bindings.
* **`clihelp` Limitation:** Only validates the count of positional arguments (using `ArgsValidator` functions like `ExactArgs(1)`). Developers must manually parse, type-cast, and extract values by slice index directly from `ctx.Args[0]`.

### 🚫 Complex Flag Inter-dependencies (Conflicts & Requirements)
* **Modern Standard:** Declarative configuration of mutually exclusive flags (e.g., `--json` conflicts with `--yaml`) or required co-requisites (e.g., `--cert-file` requires `--key-file`).
* **`clihelp` Limitation:** All flag dependency validation logic must be custom-coded by developers inside the command's `PreRun` or `Run` hooks.

### 🚫 Cascading Configuration Sources
* **Modern Standard:** Auto-binding environment variables (e.g., checking `$MY_TOKEN` if `--token` is omitted) and merging file-based configurations (`.yaml`, `.json`, `.toml`) in a predefined priority hierarchy:
  $$\text{Flags} > \text{Environment Variables} > \text{Configuration Files} > \text{Defaults}$$
* **`clihelp` Limitation:** While a `ConfigPath` is displayable in the help output, there is no built-in mechanism to parse configuration files or automatically map environment variables to options.

### 🚫 Context-Aware Positional Autocompletion
* **Modern Standard:** Shell tab-completion that dynamically resolves arguments based on context (e.g., completing active Git branches via `git checkout <tab>` or running Docker container names via `docker stop <tab>`).
* **`clihelp` Limitation:** Completion callbacks are only wired on individual flags/options (`Option.Complete`). There is no command-level hook to dynamically suggest and autocomplete positional arguments.

---

## 2. Help & Usage Display Messages

### 🚫 Flag & Option Deprecation Highlights
* **Modern Standard:** Automatic warnings displayed in the console when using deprecated commands/flags, accompanied by automated styling (such as strikethrough or dim colors) on the help screen.
* **`clihelp` Limitation:** Although `Option` defines a `Deprecated string` field, the help renderer does not format it, nor does the runtime execution pipeline warn the user when using a deprecated option.

### 🚫 Grid/Compact Layout for Subcommands
* **Modern Standard:** Large CLI suites with dozens of subcommands (like `git` or `docker`) display commands in a compact multi-column grid layout to conserve terminal height.
* **`clihelp` Limitation:** Commands and subcommands are always formatted as a vertical, single-column list. Although the built-in `$PAGER` integration mitigates this, it can still result in long scrolls on standard outputs.

### 🚫 Interactive Mode & Component Wizards
* **Modern Standard:** Built-in interactive prompts or confirmation screens (e.g., `aws configure` or `git add -i`) when requested via a flag or when run interactively without parameters.
* **`clihelp` Limitation:** Focuses exclusively on static argument-driven execution; it lacks a native interactive wizard or prompting engine.

### 🚫 Localization & i18n
* **Modern Standard:** Automatic translation of standard help sections ("Usage", "Flags", "Global Flags") and system errors to match the terminal's active language environment variable (`$LANG`).
* **`clihelp` Limitation:** Help presentation headers and system-level error messages are hardcoded in English.
