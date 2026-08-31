package clihelp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SupportedShells lists available shell autocompletion formats.
var SupportedShells = []string{"bash", "zsh", "fish"}

func (a *App) handleComplete(ctx context.Context, args []string) error {
	w := a.stdout()
	if len(args) == 0 {
		// Output all root commands, de-duplicating shortcut/command names.
		seen := map[string]bool{}
		emit := func(name, desc string) {
			if seen[name] {
				return
			}
			seen[name] = true
			fmt.Fprintf(w, "%s\t%s\n", name, desc)
		}
		for _, cmd := range a.Commands {
			if !cmd.Hidden {
				emit(cmd.Name, cmd.Description)
			}
		}
		for _, s := range a.Shortcuts {
			if !s.Hidden {
				emit(s.Name, s.Description)
			}
		}
		return nil
	}

	toComplete := args[len(args)-1]
	prevWord := ""
	if len(args) > 1 {
		prevWord = args[len(args)-2]
	}

	// Resolve the active command path from the args up to len(args)-1
	currentCmd, _, path, _, _, err := a.resolveCommand(args[:len(args)-1])
	if err != nil {
		return err
	}

	// Collect active options
	activeOptions := a.collectOptions(path, currentCmd)

	// 1. If previous word was a flag, check if that option has a dynamic completion callback
	if strings.HasPrefix(prevWord, "-") {
		for _, opt := range activeOptions {
			if opt.Complete != nil {
				spec := parseFlagSpec(opt.Flags)
				matched := false
				for _, long := range spec.longNames {
					if "--"+long == prevWord {
						matched = true
						break
					}
				}
				for _, short := range spec.shortNames {
					if "-"+short == prevWord {
						matched = true
						break
					}
				}
				if matched {
					results := opt.Complete(toComplete)
					for _, res := range results {
						fmt.Fprintln(w, res)
					}
					return nil
				}
			}
		}
	}

	// 2. If toComplete starts with '-', suggest flag names. When "=" is already
	// typed the flag name is fixed; emit dynamic values or a completed "=".
	if strings.HasPrefix(toComplete, "-") {
		if eq := strings.Index(toComplete, "="); eq >= 0 {
			namePart := toComplete[:eq]
			for _, opt := range activeOptions {
				if opt.Hidden {
					continue
				}
				spec := parseFlagSpec(opt.Flags)
				nameOk := false
				for _, long := range spec.longNames {
					if "--"+long == namePart {
						nameOk = true
						break
					}
				}
				if !nameOk {
					for _, short := range spec.shortNames {
						if "-"+short == namePart {
							nameOk = true
							break
						}
					}
				}
				if !nameOk {
					continue
				}
				if opt.Complete != nil {
					for _, res := range opt.Complete(toComplete[eq+1:]) {
						fmt.Fprintf(w, "%s=%s\t%s\n", namePart, res, opt.Description)
					}
				} else {
					fmt.Fprintf(w, "%s=\t%s\n", namePart, opt.Description)
				}
			}
			return nil
		}
		for _, opt := range activeOptions {
			if opt.Hidden {
				continue
			}
			spec := parseFlagSpec(opt.Flags)
			for _, long := range spec.longNames {
				flagName := "--" + long
				if strings.HasPrefix(flagName, toComplete) {
					fmt.Fprintf(w, "%s\t%s\n", flagName, opt.Description)
				}
			}
			for _, short := range spec.shortNames {
				flagName := "-" + short
				if strings.HasPrefix(flagName, toComplete) {
					fmt.Fprintf(w, "%s\t%s\n", flagName, opt.Description)
				}
			}
		}
		return nil
	}

	// 3. Otherwise suggest subcommands
	subcommands := a.Commands
	if currentCmd != nil {
		subcommands = currentCmd.Subcommands
	}

	// Use the same prefix filtering logic as resolveCommand
	matches := filterCommandsByPrefix(subcommands, toComplete)
	for _, cmd := range matches {
		fmt.Fprintf(w, "%s\t%s\n", cmd.Name, cmd.Description)
		// Also include aliases as separate completions
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, toComplete) {
				fmt.Fprintf(w, "%s\t%s\n", alias, cmd.Description)
			}
		}
	}

	// Include shortcut commands at root level, de-duplicating command names.
	if currentCmd == nil {
		seen := map[string]bool{}
		for _, sub := range a.Commands {
			if !sub.Hidden {
				seen[sub.Name] = true
			}
		}
		for _, s := range a.Shortcuts {
			if s.Hidden || seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			if strings.HasPrefix(s.Name, toComplete) {
				fmt.Fprintf(w, "%s\t%s\n", s.Name, s.Description)
			}
		}
	}

	return nil
}

// GenBashCompletion writes a Bash tab-completion script to w.
func GenBashCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`# bash completion for %[1]s
_%[2]s_complete() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n : || return
    else
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
        cur="${words[cword]}"
        prev="${words[cword-1]}"
    fi

    local out
    out=$( "${COMP_WORDS[0]}" __complete "${COMP_WORDS[@]:1}" 2>/dev/null )
    if [[ $? -ne 0 ]]; then
        return
    fi

    local IFS=$'\n'
    local comps=()
    for line in $out; do
        comps+=("${line%%%%	*}")
    done
    COMPREPLY=( $(compgen -W "${comps[*]}" -- "$cur") )
}
complete -o default -F _%[2]s_complete %[1]s
`, name, cleanName)
	_, err := io.WriteString(w, tmpl)
	return err
}

// GenZshCompletion writes a Zsh tab-completion script to w.
func GenZshCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`#compdef %[1]s

_%[2]s() {
    local -a completions
    local -a completions_with_descriptions
    local line

    local -a words_to_pass
    if (( CURRENT > 1 )); then
        words_to_pass=("${(@)words[2,CURRENT]}")
    elif (( ${#words[@]} > 1 )); then
        words_to_pass=("${(@)words[2,-1]}")
    elif (( ${#@} > 0 )); then
        words_to_pass=("$@")
    fi

    local binary_cmd="${words[1]:-%[1]s}"
    local output
    if declare -f _call_program >/dev/null 2>&1; then
        output=(${(f)"$(_call_program %[1]s ${binary_cmd} __complete "${words_to_pass[@]}")"})
    else
        output=(${(f)"$(${binary_cmd} __complete "${words_to_pass[@]}")"})
    fi

    for line in "${output[@]}"; do
        if [[ -z "$line" ]]; then
            continue
        fi
        if [[ "$line" == *$'\t'* ]]; then
            local cand="${line%%%%	*}"
            local desc="${line#*	}"
            cand="${cand//:/\\:}"
            desc="${desc//:/\\:}"
            completions_with_descriptions+=("${cand}:${desc}")
        else
            completions+=("${line//:/\\:}")
        fi
    done

    if [ -n "$completions_with_descriptions" ]; then
        _describe -t commands '%[1]s' completions_with_descriptions
    fi
    if [ -n "$completions" ]; then
        compadd -a completions
    fi
}

if type compdef >/dev/null 2>&1; then
    compdef _%[2]s %[1]s
fi

_%[2]s "$@"
`, name, cleanName)
	_, err := io.WriteString(w, tmpl)
	return err
}

// GenFishCompletion writes a Fish tab-completion script to w.
func GenFishCompletion(app *App, w io.Writer) error {
	name := app.Name
	if name == "" {
		name = "app"
	}
	cleanName := strings.ReplaceAll(name, "-", "_")
	tmpl := fmt.Sprintf(`# fish completion for %[1]s
function __fish_%[2]s_complete
    set -l cmd (commandline -opc) (commandline -ct)
    test (count $cmd) -gt 1; and set -e cmd[1]
    %[1]s __complete $cmd
end

complete -c %[1]s -f -a '(__fish_%[2]s_complete)'
`, name, cleanName)
	_, err := io.WriteString(w, tmpl)
	return err
}

// CompletionPath returns the target installation path for the completion script.
func CompletionPath(app *App, shell string) (string, error) {
	if shell == "" {
		shell = detectShell()
	}
	shell = strings.ToLower(strings.TrimSpace(shell))

	appName := appName(app)

	var targetDir, fileName string
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user home directory: %w", err)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}

	switch shell {
	case "bash":
		targetDir = filepath.Join(dataHome, "bash-completion", "completions")
		fileName = appName
	case "zsh":
		targetDir = filepath.Join(dataHome, "zsh", "site-functions")
		fileName = "_" + appName
	case "fish":
		targetDir = filepath.Join(configHome, "fish", "completions")
		fileName = appName + ".fish"
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(SupportedShells, ", "))
	}

	return filepath.Join(targetDir, fileName), nil
}

// IsCompletionInstalled checks if the shell completion script is already installed
// in the user's standard XDG directory for the given shell (or detected active shell).
func IsCompletionInstalled(app *App, shell string) bool {
	if app == nil {
		return false
	}
	path, err := CompletionPath(app, shell)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// maybeAutoInstallCompletion checks and installs completions silently if enabled.
func (a *App) maybeAutoInstallCompletion(args []string) {
	if a == nil || !a.AutoInstallCompletion {
		return
	}
	// Never run during internal completion calls, or in CI/non-interactive test runs
	if len(args) > 0 && args[0] == "__complete" {
		return
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("TERM") == "dumb" {
		return
	}
	sh := detectShell()
	if !IsCompletionInstalled(a, sh) {
		_, _ = InstallCompletion(a, sh)
	}
}

// InstallCompletion installs the shell completion script for the given app and shell.
// If shell is empty, it detects the active shell via the SHELL environment variable.
// Returns the absolute file path where the completion script was written.
func InstallCompletion(app *App, shell string) (string, error) {
	targetPath, err := CompletionPath(app, shell)
	if err != nil {
		return "", err
	}
	if shell == "" {
		shell = detectShell()
	}
	shell = strings.ToLower(strings.TrimSpace(shell))

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %q: %w", targetDir, err)
	}

	f, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create completion file %q: %w", targetPath, err)
	}
	defer f.Close()

	switch shell {
	case "bash":
		err = GenBashCompletion(app, f)
	case "zsh":
		err = GenZshCompletion(app, f)
	case "fish":
		err = GenFishCompletion(app, f)
	}
	if err != nil {
		return "", fmt.Errorf("failed to generate %s completion: %w", shell, err)
	}

	return targetPath, nil
}

func detectShell() string {
	sh := filepath.Base(os.Getenv("SHELL"))
	for _, s := range SupportedShells {
		if sh == s {
			return s
		}
	}
	return "bash"
}

// CompletionCommand returns a standard clihelp.Command providing 'bash', 'zsh', 'fish', and 'install' subcommands.
func CompletionCommand() Command {
	return Command{
		Name:        "completion",
		Description: "Generate or install shell tab-completion scripts",
		Examples: []Example{
			{Line: "completion zsh", Description: "Generate Zsh tab-completion script"},
			{Line: "completion install", Description: "Install tab-completions for the active shell"},
		},
		Notes: []Note{
			{
				Heading: "Shell Tip",
				Text:    "Tip: <Tab> to complete, Ctrl-D to list choices, Alt-H for instant command help.",
			},
		},
		Subcommands: []Command{
			{
				Name:        "bash",
				Description: "Generate Bash tab-completion script",
				Args:        NoArgs,
				Run: func(ctx *Context) error {
					return GenBashCompletion(ctx.App, ctx.Stdout)
				},
			},
			{
				Name:        "zsh",
				Description: "Generate Zsh tab-completion script",
				Args:        NoArgs,
				Run: func(ctx *Context) error {
					return GenZshCompletion(ctx.App, ctx.Stdout)
				},
			},
			{
				Name:        "fish",
				Description: "Generate Fish tab-completion script",
				Args:        NoArgs,
				Run: func(ctx *Context) error {
					return GenFishCompletion(ctx.App, ctx.Stdout)
				},
			},
			{
				Name:        "install",
				Description: "Install tab-completion script to standard user directory",
				Examples: []Example{
					{Line: "completion install zsh", Description: "Install completions to ~/.local/share/zsh/site-functions"},
				},
				Parameters: []Param{
					{Name: "[<shell>]", Description: "Shell type ('bash', 'zsh', or 'fish'; defaults to current shell)"},
				},
				Args: MaximumNArgs(1),
				Run: func(ctx *Context) error {
					shell := ""
					if len(ctx.Args) > 0 {
						shell = ctx.Args[0]
					}
					path, err := InstallCompletion(ctx.App, shell)
					if err != nil {
						return err
					}
					fmt.Fprintf(ctx.Stdout, "✓ Autocompletion installed to: %s\n", path)
					if shell == "zsh" || (shell == "" && detectShell() == "zsh") {
						fmt.Fprintln(ctx.Stdout, "Note: If not already configured, ensure the directory is in your Zsh $fpath in ~/.zshrc:")
						fmt.Fprintln(ctx.Stdout, "    fpath=(~/.local/share/zsh/site-functions $fpath)")
					}
					fmt.Fprintln(ctx.Stdout, "Tip: <Tab> to complete, Ctrl-D to list choices, Alt-H for instant command help.")
					fmt.Fprintln(ctx.Stdout, "Restart your shell or open a new terminal session to activate.")
					return nil
				},
			},
		},
	}
}
