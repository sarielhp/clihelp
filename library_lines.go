package clihelp

import "strings"

// The library's own commands are written before anyone knows what the program
// is called or where its author will mount them, so their usage and example
// lines are relative to their own root: "completion install", "manpage > <app>.1".
// Every other command's lines begin with the program name, and a user who copies
// a relative one gets "unknown command". These helpers make such a line
// absolute at render time, from the path the command was actually reached by.

// absoluteLibraryLine rewrites a line the library wrote relative to its own
// root so that it starts with the program name and the real mount point. A
// command that is not the library's, or a line that does not open with the
// library's root name, is returned unchanged. "<app>" anywhere in a library line
// stands for the program name.
func (a *App) absoluteLibraryLine(cmd *Command, path []string, line string) string {
	if cmd == nil || !cmd.libraryOwned || cmd.libraryRoot == "" {
		return line
	}
	rootIdx := len(path) - 1 - cmd.libraryDepth
	if rootIdx < 0 {
		return strings.ReplaceAll(line, "<app>", appName(a))
	}
	line = strings.ReplaceAll(line, "<app>", appName(a))
	mountedAs := path[rootIdx]
	if line != cmd.libraryRoot && !strings.HasPrefix(line, cmd.libraryRoot+" ") {
		return line
	}
	prefix := append([]string{appName(a)}, path[:rootIdx]...)
	return strings.Join(append(prefix, mountedAs), " ") + strings.TrimPrefix(line, cmd.libraryRoot)
}

// absoluteExamples is absoluteLibraryLine for a command's examples. The slice is
// copied before it is changed, because the originals belong to the command.
func (a *App) absoluteExamples(cmd *Command, path []string, examples []Example) []Example {
	if cmd == nil || !cmd.libraryOwned {
		return examples
	}
	out := make([]Example, len(examples))
	for i, ex := range examples {
		lines := strings.Split(ex.Line, "\n")
		for j, l := range lines {
			lines[j] = a.absoluteLibraryLine(cmd, path, l)
		}
		out[i] = Example{
			Line:        strings.Join(lines, "\n"),
			Description: strings.ReplaceAll(ex.Description, "<app>", appName(a)),
		}
	}
	return out
}

// ForDisplay returns cmd as the help pages show it. The library's own commands
// (CompletionCommand, ManPageCommand) carry usage and example lines relative to
// their own root; here they are made absolute — program name and real mount
// point — exactly as the terminal pages do. Any other command is returned
// unchanged. path is the full path the command was reached by.
//
// It exists for generators in other packages (doc/), which cannot reach the
// unexported helpers and would otherwise print the lines as the library wrote them.
func (a *App) ForDisplay(cmd Command, path []string) Command {
	if cmd.UsageLine != "" {
		cmd.UsageLine = a.absoluteLibraryLine(&cmd, path, cmd.UsageLine)
	}
	cmd.Examples = a.absoluteExamples(&cmd, path, cmd.Examples)
	return cmd
}
