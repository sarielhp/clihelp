module github.com/sarielhp/clihelp

go 1.26.5

require (
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/fatih/color v1.19.0
	github.com/mattn/go-runewidth v0.0.28
	github.com/spf13/pflag v1.0.10
	golang.org/x/term v0.45.0
)

require (
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

// The generated zsh completion script passed the words the user had typed
// through zsh's _call_program, whose body ends in `eval ... "$argv[2,-1]"` —
// it re-parses its arguments as shell code. Pressing <Tab> on a command line
// containing $(...) or backticks executed it, with nothing shown on screen.
// Present from v0.2.2, when the zsh generator was added, and fixed in v0.3.15.
retract [v0.2.2, v0.3.14]
