package clihelp

// The generated completion scripts.
//
// They live apart from the code that writes them because they are shell
// programs, not Go: they are read, reviewed and debugged as shell, and keeping
// them here leaves completion.go small enough to read in one sitting.

// bashCompletionTemplate is the generated bash completion script; see GenBashCompletion.
const bashCompletionTemplate = `# bash completion for %[1]s
# clihelp-completion-version: %[3]d
_%[2]s_complete() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        # -n := keeps "--flag=value" and "host:port" as single words, which is
        # what the completion protocol is given below.
        _init_completion -n := || return
    else
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
        cur="${words[cword]}"
        prev="${words[cword-1]}"
    fi

    # Send the words bash-completion computed, not the raw COMP_WORDS: those are
    # split at every character of COMP_WORDBREAKS, so "--unit=" arrived as three
    # words and a colon-bearing argument as three more.
    local out
    out=$( "${words[0]}" __complete "${words[@]:1:cword-1}" "$cur" 2>/dev/null ) || return

    # Candidates are data: add them literally. compgen -W would expand them,
    # running any command substitution a candidate happens to contain.
    COMPREPLY=()
    local line cand
    while IFS= read -r line; do
        [[ -z $line ]] && continue
        cand="${line%%%%	*}"
        [[ $cand == "$cur"* ]] && COMPREPLY+=("$cand")
    done <<< "$out"

    if declare -F __ltrim_colon_completions >/dev/null 2>&1; then
        __ltrim_colon_completions "$cur"
    fi
}
complete -o default -F _%[2]s_complete %[1]s
`

// zshCompletionTemplate is the generated zsh completion script; see GenZshCompletion.
const zshCompletionTemplate = `#compdef %[1]s
# clihelp-completion-version: %[3]d

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
        # ${(q)...} is essential: _call_program ends in "eval ... $argv[2,-1]",
        # so anything spliced in unquoted is re-parsed as shell code, and $words
        # holds the command line the user has typed verbatim. Without the quoting
        # flag, a line containing $(...) executes when Tab is pressed.
        output=(${(f)"$(_call_program %[1]s ${(q)binary_cmd} __complete ${(q)words_to_pass[@]})"})
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

    # Count the elements. "[ -n $completions ]" tested the array joined into one
    # string, so a candidate list whose entries are all empty looked non-empty
    # and was added, and the files fallback below never got its turn.
    if (( ${#completions_with_descriptions} )); then
        _describe -t commands '%[1]s' completions_with_descriptions
    fi
    if (( ${#completions} )); then
        compadd -a completions
    fi
    # The program had nothing to offer, so complete filenames — which is what the
    # bash script's "complete -o default" does. Without this an argument that is
    # a path could not be completed at all under zsh.
    (( ${#completions} + ${#completions_with_descriptions} )) || _files
}

# Autoloaded from $fpath this file *is* the completion function and has to call
# it; sourced from a startup file it must only register itself, because calling
# the function outside a completion context prints "can only be called from
# completion function" at every shell start. $funcstack[1] tells the two apart:
# the function's name when autoloaded, this file's path when sourced.
if [ "$funcstack[1]" = "_%[2]s" ]; then
    _%[2]s "$@"
elif type compdef >/dev/null 2>&1; then
    compdef _%[2]s %[1]s
else
    # compinit has not run yet. Sourcing this file before it — a plugin manager
    # that defers compinit, or an rc file that sources clihelp's bootstrap near
    # the top — used to leave completion unregistered with nothing printed to
    # say so. Retry from the first prompt, then take the hook back out.
    _%[2]s_deferred_compdef() {
        type compdef >/dev/null 2>&1 || return
        compdef _%[2]s %[1]s
        add-zsh-hook -d precmd _%[2]s_deferred_compdef
        unfunction _%[2]s_deferred_compdef
    }
    autoload -Uz add-zsh-hook 2>/dev/null &&
        add-zsh-hook precmd _%[2]s_deferred_compdef
fi
`

// fishCompletionTemplate is the generated fish completion script; see GenFishCompletion.
const fishCompletionTemplate = `# fish completion for %[1]s
# clihelp-completion-version: %[3]d
function __fish_%[2]s_complete
    set -l cmd (commandline -opc) (commandline -ct)
    test (count $cmd) -gt 1; and set -e cmd[1]
    %[1]s __complete $cmd
end

# __fish_%[2]s_needs_files reruns the completer only to ask whether it had
# anything to say. fish has no "files if nothing else matched" mode, so the two
# rules below reconstruct one: -f keeps filenames out of the program's own
# candidates, and -F offers them when there are none. Before this, -f alone meant
# an argument that is a path could not be completed at all.
function __fish_%[2]s_needs_files
    test (count (__fish_%[2]s_complete)) -eq 0
end

complete -c %[1]s -f -a '(__fish_%[2]s_complete)'
complete -c %[1]s -n __fish_%[2]s_needs_files -F
`
