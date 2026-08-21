package clihelp

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func (a *App) handleComplete(ctx context.Context, args []string) error {
	w := a.stdout()
	if len(args) == 0 {
		// Output all root commands
		for _, cmd := range a.Commands {
			if !cmd.Hidden {
				fmt.Fprintf(w, "%s\t%s\n", cmd.Name, cmd.Description)
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
	currentCmd, ancestors, _, _, _ := a.resolveCommand(args[:len(args)-1])

	// Collect active options
	var activeOptions []Option
	activeOptions = append(activeOptions, a.PersistentOptions...)
	for _, anc := range ancestors {
		activeOptions = append(activeOptions, anc.PersistentOptions...)
	}
	if currentCmd != nil {
		activeOptions = append(activeOptions, currentCmd.PersistentOptions...)
		activeOptions = append(activeOptions, currentCmd.Options...)
	}

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

	// 2. If toComplete starts with '-', suggest flag names
	if strings.HasPrefix(toComplete, "-") {
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

	for _, sub := range subcommands {
		if sub.Hidden {
			continue
		}
		if strings.HasPrefix(sub.Name, toComplete) {
			fmt.Fprintf(w, "%s\t%s\n", sub.Name, sub.Description)
		}
		for _, alias := range sub.Aliases {
			if strings.HasPrefix(alias, toComplete) {
				fmt.Fprintf(w, "%s\t%s\n", alias, sub.Description)
			}
		}
	}

	// Include shortcut commands at root level
	if currentCmd == nil {
		for _, s := range a.Shortcuts {
			if s.Hidden {
				continue
			}
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
    if (( ${#words[@]} > 1 )); then
        words_to_pass=("${(@)words[2,-1]}")
    elif (( ${#@} > 0 )); then
        words_to_pass=("$@")
    fi
    
    local binary_cmd="${words[1]:-%[1]s}"
    local output
    if declare -f _call_program >/dev/null 2>&1; then
        output=(${(f)"$(_call_program %[1]s ${binary_cmd} __complete ${words_to_pass[@]})"})
    else
        output=(${(f)"$(${binary_cmd} __complete ${words_to_pass[@]})"})
    fi
    
    for line in "${output[@]}"; do
        if [[ "$line" == *$'\t'* ]]; then
            completions_with_descriptions+=("${line/$'\t'/:}")
        else
            completions+=("$line")
        fi
    done
    
    if [ -n "$completions_with_descriptions" ]; then
        _describe -t commands '%[1]s' completions_with_descriptions
    fi
    if [ -n "$completions" ]; then
        compadd -a completions
    fi
}

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
	tmpl := fmt.Sprintf(`function __%[2]s_complete
    set -l cmd (commandline -cop)
    test (count $cmd) -gt 1; and set -e cmd[1]
    eval (commandline -o)[1] __complete $cmd
end

complete -c %[1]s -f -a '(__%[2]s_complete)'
`, name, cleanName)
	_, err := io.WriteString(w, tmpl)
	return err
}
