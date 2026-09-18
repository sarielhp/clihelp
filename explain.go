package clihelp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Terminal geometry for __explain arrives in the environment, because the key
// binding that calls it captures stdout: the process cannot measure a pipe, and
// the shell already knows its own size.
const (
	envTermLines   = "CLIHELP_TERM_LINES"
	envTermColumns = "CLIHELP_TERM_COLUMNS"

	defaultTermLines   = 24
	defaultTermColumns = 80
)

// explainHeightFraction is how much of the screen an explanation may take.
// Anything more scrolls the command being explained off the top, which is the
// one line the reader needs to keep.
const explainHeightFraction = 2.0 / 3.0

// expandCommandLine rewrites every abbreviated command name in a typed command
// line to its full name and leaves the rest of the line — spacing, flags,
// arguments, redirections, quoting — exactly as it was written. It returns the
// rewritten line and the command path it resolved.
func (a *App) expandCommandLine(line string) (string, []string) {
	toks := extractSegmentTokens(line)
	if len(toks) < 2 {
		return line, nil // nothing beyond the program name
	}

	texts := make([]string, 0, len(toks)-1)
	for _, t := range toks[1:] {
		texts = append(texts, t.text)
	}

	var (
		path     []string
		resolved []*Command
		full     = make(map[int]string)
		cmds     = a.Commands
		arity    = a.leadingFlagArity(nil)
		i        = 0
	)
	for i < len(texts) {
		if strings.HasPrefix(texts[i], "-") {
			count, known := scanLeadingFlag(arity, texts[i:])
			if !known {
				break
			}
			i += count
			continue
		}
		cmd, err := a.matchCommandOrShortcut(cmds, texts[i], len(path) == 0)
		if err != nil || cmd == nil {
			break
		}
		full[i+1] = cmd.Name // +1: texts is offset by the program name
		path = append(path, cmd.Name)
		resolved = append(resolved, cmd)
		cmds = cmd.Subcommands
		arity = a.leadingFlagArity(resolved)
		i++
	}

	// Splice from the back so the offsets of the earlier tokens stay valid.
	for idx := len(toks) - 1; idx >= 1; idx-- {
		name, ok := full[idx]
		if !ok {
			continue
		}
		line = line[:toks[idx].start] + name + line[toks[idx].end:]
	}
	return line, path
}

// explainGeometry reports the terminal size the explanation must fit, taking it
// from the environment the shell binding sets, then from stderr (which stays a
// terminal even when stdout is captured), then from a conventional default.
func explainGeometry() (columns, lines int) {
	columns = envInt(envTermColumns)
	lines = envInt(envTermLines)
	if columns <= 0 || lines <= 0 {
		if w, h, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
			if columns <= 0 {
				columns = w
			}
			if lines <= 0 {
				lines = h
			}
		}
	}
	if columns <= 0 {
		columns = defaultTermColumns
	}
	if lines <= 0 {
		lines = defaultTermLines
	}
	return columns, lines
}

func envInt(name string) int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// explainBudget is how many lines an explanation may occupy on a screen of the
// given height, never fewer than three: the command, one line of help, and the
// note saying where the rest is.
func explainBudget(lines int) int {
	if lines <= 0 {
		lines = defaultTermLines
	}
	budget := int(float64(lines) * explainHeightFraction)
	if budget < 3 {
		return 3
	}
	return budget
}

// Explain writes the expanded form of a typed command line followed by the help
// for the command it names, trimmed to fit budget lines in total.
//
// The first line of the output is always the expanded command line, so a shell
// key binding can put it back on the prompt and print the remainder.
func (a *App) Explain(w io.Writer, line string, columns, budget int) {
	expanded, path := a.expandCommandLine(line)
	fmt.Fprintln(w, expanded)

	var buf bytes.Buffer
	o := Options{Writer: &buf, Theme: a.Theme, Width: columns, Concise: true}
	if len(path) == 0 {
		a.RenderGlobal(o)
	} else {
		a.RenderCommand(o, path...)
	}
	writeWithinBudget(w, buf.String(), budget-1, columns, a.explainMoreHint(path))
}

// explainMoreHint names the command that prints the help in full.
func (a *App) explainMoreHint(path []string) string {
	if len(path) == 0 {
		return fmt.Sprintf("run '%s help' for the rest", appName(a))
	}
	return fmt.Sprintf("run '%s help %s' for the rest", appName(a), strings.Join(path, " "))
}

// writeWithinBudget writes at most budget lines of text, replacing whatever did
// not fit with a single line saying how much was left and where to read it.
func writeWithinBudget(w io.Writer, text string, budget, columns int, hint string) {
	lines := splitLines(strings.TrimRight(text, "\n"))
	if budget < 1 {
		return
	}
	if len(lines) <= budget {
		for _, l := range lines {
			fmt.Fprintln(w, l)
		}
		return
	}
	for _, l := range lines[:budget-1] {
		fmt.Fprintln(w, l)
	}
	note := fmt.Sprintf("… %d more lines — %s", len(lines)-(budget-1), hint)
	fmt.Fprintln(w, runewidth.Truncate(note, columns, "…"))
}

// handleExplain serves the __explain protocol call: one argument, the command
// line as the user has typed it so far.
func (a *App) handleExplain(args []string) error {
	line := ""
	if len(args) > 0 {
		line = args[0]
	}
	columns, lines := explainGeometry()
	a.Explain(a.stdout(), line, columns, explainBudget(lines))
	return nil
}

// Key-binding snippets. These cannot live in the completion script: every shell
// loads that lazily, on the first completion of the command, so a binding
// written there would not exist until the user had already pressed Tab once.
// They are printed for the user to source from their shell's rc file.

const bashKeysTemplate = `# clihelp key bindings for {{app}}
# Printed for inspection or manual setup. "{{app}} completion install" writes
# this into a file your shell sources, so nothing runs at shell start.

# One dispatcher serves every clihelp program on the machine. A key binding is
# global to the shell, so a per-program binding is replaced by the next program
# that installs one — and that program's own name check then refuses every other
# program's command line, leaving Alt-H silently dead for all but the last one
# installed. The highest dispatcher version wins whatever order the snippets are
# sourced in; the registry of program names is shared and its name never changes.
if [ "${_clihelp_dispatcher_version:-0}" -lt {{dispatcher}} ]; then
    _clihelp_dispatcher_version={{dispatcher}}
    _clihelp_explain() {
        # Trim leading blanks before taking the first word: without this a line
        # beginning with a space (deliberate under HISTCONTROL=ignorespace)
        # yields an empty word, which then matches the space-padded registry, so
        # the key was swallowed and nothing ran.
        local line=$READLINE_LINE
        while [ "${line#[[:space:]]}" != "$line" ]; do line=${line#[[:space:]]}; done
        local word=${line%% *}
        # A path is not a registered program. The registry holds bare names, and
        # matching on the basename meant "./alpha" from whatever directory the
        # user had cd'd into was authorized and then executed — on a key the user
        # presses precisely because they have not decided to run the line.
        case $word in */*) return ;; esac
        [ -n "$word" ] || return
        # The registry is a wire format shared with clihelp programs built
        # against other library versions, so an entry carries the protocol it
        # speaks: "name:protocol". A bare name was written before the suffix
        # existed and means protocol 1. Reading it here is what lets a newer
        # dispatcher refuse a call it would get wrong rather than make it.
        local entry proto=
        for entry in ${_clihelp_apps:-}; do
            case $entry in
                "$word":*) proto=${entry#*:} ;;
                "$word") proto=1 ;;
                *) continue ;;
            esac
            break
        done
        [ "$proto" = {{proto}} ] || return
        local out
        out=$( CLIHELP_TERM_LINES="${LINES:-}" CLIHELP_TERM_COLUMNS="${COLUMNS:-}" \
               "$word" __explain "$READLINE_LINE" 2>/dev/null ) || return
        [ -n "$out" ] || return
        READLINE_LINE="${out%%$'\n'*}"
        READLINE_POINT=${#READLINE_LINE}
        printf '\n'
        case $out in *$'\n'*) printf '%s\n' "${out#*$'\n'}" ;; esac
    }
    # A key binding belongs to one keymap. Binding only the current one meant
    # that a user who ran "set -o vi" — before or after sourcing this — had no
    # Alt-H at all, silently. Set CLIHELP_NO_KEY_BINDINGS to keep the key.
    if [ -z "${CLIHELP_NO_KEY_BINDINGS:-}" ]; then
        for _clihelp_keymap in emacs-standard vi-insert vi-command; do
            bind -m "$_clihelp_keymap" -x '"\eh": _clihelp_explain' 2>/dev/null
        done
        unset _clihelp_keymap
    fi
fi
# Both forms, for one release: the suffixed entry is what this and every later
# dispatcher reads, and the bare one keeps an older dispatcher — one another
# program installed before the suffix existed — able to authorize this program.
for _clihelp_entry in "{{app}}:{{proto}}" "{{app}}"; do
    case " ${_clihelp_apps:-} " in
        *" $_clihelp_entry "*) ;;
        *) _clihelp_apps="${_clihelp_apps:-} $_clihelp_entry " ;;
    esac
done
unset _clihelp_entry
`

const zshKeysTemplate = `# clihelp key bindings for {{app}}
# Printed for inspection or manual setup. "{{app}} completion install" writes
# this into a file your shell sources, so nothing runs at shell start.

# See the bash snippet for why the dispatcher is shared rather than per program.
# Alt-H is zsh's own run-help key, so a command line that belongs to no clihelp
# program is handed straight back to it.
if [[ ${_clihelp_dispatcher_version:-0} -lt {{dispatcher}} ]]; then
    typeset -g _clihelp_dispatcher_version={{dispatcher}}
    _clihelp_explain() {
        # ${(z)} is zsh's own shell-word splitter, so leading blanks and tabs are
        # handled for us. A path-bearing word is not a registered program; see the
        # bash snippet.
        local word=${${(z)BUFFER}[1]}
        # See the bash snippet for the registry's "name:protocol" wire format.
        local entry proto=
        if [[ -n $word && $word != */* ]]; then
            for entry in ${=_clihelp_apps:-}; do
                case $entry in
                    ("$word":*) proto=${entry#*:} ;;
                    ("$word") proto=1 ;;
                    (*) continue ;;
                esac
                break
            done
        fi
        if [[ $proto != {{proto}} ]]; then
            zle run-help
            return
        fi
        local out
        out=$(CLIHELP_TERM_LINES=$LINES CLIHELP_TERM_COLUMNS=$COLUMNS \
              $word __explain "$BUFFER" 2>/dev/null) || return
        [[ -n $out ]] || return
        BUFFER=${out%%$'\n'*}
        CURSOR=${#BUFFER}
        zle -I
        print -r -- ""
        [[ $out == *$'\n'* ]] && print -r -- "${out#*$'\n'}"
        zle reset-prompt
    }
    zle -N _clihelp_explain
    # Every keymap, not just the current one; see the bash snippet.
    if [[ -z ${CLIHELP_NO_KEY_BINDINGS:-} ]]; then
        bindkey -M emacs '^[h' _clihelp_explain
        bindkey -M viins '^[h' _clihelp_explain
        bindkey -M vicmd '^[h' _clihelp_explain
    fi
fi
# Both forms; see the bash snippet.
for _clihelp_entry in '{{app}}:{{proto}}' '{{app}}'; do
    if [[ " ${_clihelp_apps:-} " != *" $_clihelp_entry "* ]]; then
        typeset -g _clihelp_apps="${_clihelp_apps:-} $_clihelp_entry "
    fi
done
unset _clihelp_entry
`

const fishKeysTemplate = `# clihelp key bindings for {{app}}
# Printed for inspection or manual setup. "{{app}} completion install" writes
# this into a file your shell sources, so nothing runs at shell start.

# See the bash snippet for why the dispatcher is shared rather than per program.
# alt-h is fish's own key for the man page of the command being typed, so a
# command line that belongs to no clihelp program is handed straight back to it.
if not set -q _clihelp_dispatcher_version; or test $_clihelp_dispatcher_version -lt {{dispatcher}}
    set -g _clihelp_dispatcher_version {{dispatcher}}
    function __clihelp_explain
        # string collect keeps the buffer as one value. Plain (commandline)
        # splits a multi-line buffer into one element per line, and those then
        # went to __explain as separate arguments, of which only the first was
        # read — so the continuation lines of a long command were dropped.
        set -l line (commandline | string collect)
        set -l word (string split -m 1 ' ' -- (string trim -l -- "$line"))[1]
        # A path is not a registered program, and see the bash snippet for the
        # registry's "name:protocol" wire format.
        set -l proto ""
        for entry in $_clihelp_apps
            set -l parts (string split -m 1 ':' -- $entry)
            if test "$parts[1]" = "$word"
                if set -q parts[2]
                    set proto $parts[2]
                else
                    set proto 1
                end
                break
            end
        end
        if test -z "$word"; or string match -q -- '*/*' $word; or test "$proto" != {{proto}}
            __fish_man_page
            return
        end
        set -lx CLIHELP_TERM_LINES $LINES
        set -lx CLIHELP_TERM_COLUMNS $COLUMNS
        set -l out ($word __explain "$line" 2>/dev/null)
        test (count $out) -gt 0; or return
        commandline -r -- $out[1]
        echo
        test (count $out) -gt 1; and printf '%s\n' $out[2..-1]
        commandline -f repaint
    end
    # Every keymap, not just the current one; see the bash snippet.
    if not set -q CLIHELP_NO_KEY_BINDINGS
        bind -M default \eh __clihelp_explain
        bind -M insert \eh __clihelp_explain
    end
end
# Both forms; see the bash snippet.
for _clihelp_entry in {{app}}:{{proto}} {{app}}
    contains -- $_clihelp_entry $_clihelp_apps
    or set -g _clihelp_apps $_clihelp_apps $_clihelp_entry
end
set -e _clihelp_entry
`

// GenKeyBindings writes the shell snippet that binds Alt-H to "expand this
// command line and explain it". The binding acts only on command lines that
// begin with the application's own name, and zsh hands anything else back to its
// standard run-help.
func GenKeyBindings(app *App, shell string, w io.Writer) error {
	if app == nil {
		return errors.New("key bindings: app is nil")
	}
	name, err := safeAppName(app)
	if err != nil {
		return err
	}
	shell, err = resolveShell(shell)
	if err != nil {
		return err
	}

	var tmpl string
	switch shell {
	case "bash":
		tmpl = bashKeysTemplate
	case "zsh":
		tmpl = zshKeysTemplate
	case "fish":
		tmpl = fishKeysTemplate
	default:
		return fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(SupportedShells, ", "))
	}

	script := strings.NewReplacer(
		"{{app}}", name,
		"{{dispatcher}}", strconv.Itoa(keyDispatcherVersion),
		"{{proto}}", strconv.Itoa(explainProtocolVersion),
	).Replace(tmpl)
	_, err = io.WriteString(w, script)
	return err
}
