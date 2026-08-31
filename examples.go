package clihelp

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

// sprintColor applies c to text if c is non-nil and color is enabled.
func sprintColor(c *color.Color, text string) string {
	if c == nil || color.NoColor || text == "" {
		return text
	}
	return c.Sprint(text)
}

// ColorizeExampleLine applies ANSI syntax colors to a command-line example string.
// It recognizes comments, shell prompts, subcommands, flags, values, and operators.
func ColorizeExampleLine(line string, th Theme) string {
	return colorizeExampleLine(line, th)
}

func colorizeExampleLine(line string, th Theme) string {
	if line == "" {
		return ""
	}

	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	// Full comment line
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return indent + sprintColor(th.ExampleComment, trimmed)
	}

	// Shell prompt ($ or > or %)
	var promptPrefix string
	if strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "> ") || strings.HasPrefix(trimmed, "% ") {
		promptPrefix = sprintColor(th.ExampleComment, trimmed[:2])
		trimmed = trimmed[2:]
	}

	// Tokenize preserving spaces
	var b strings.Builder
	b.WriteString(indent)
	b.WriteString(promptPrefix)

	inCommand := true
	i := 0
	for i < len(trimmed) {
		// Whitespace
		if unicode.IsSpace(rune(trimmed[i])) {
			b.WriteByte(trimmed[i])
			i++
			continue
		}

		// Trailing inline comment (# ...)
		if trimmed[i] == '#' || (i+1 < len(trimmed) && trimmed[i:i+2] == "//") {
			comment := trimmed[i:]
			b.WriteString(sprintColor(th.ExampleComment, comment))
			break
		}

		// Shell operators: |, ||, &&, ;, >, >>, <
		if trimmed[i] == '|' || trimmed[i] == '&' || trimmed[i] == ';' || trimmed[i] == '>' || trimmed[i] == '<' {
			j := i + 1
			for j < len(trimmed) && (trimmed[j] == '|' || trimmed[j] == '&' || trimmed[j] == '>' || trimmed[j] == '<') {
				j++
			}
			op := trimmed[i:j]
			b.WriteString(sprintColor(th.Accent, op))
			i = j
			inCommand = true // reset command state after pipeline/chaining
			continue
		}

		// Extract single word/token (handling quotes)
		start := i
		inQuote := byte(0)
		for i < len(trimmed) && (!unicode.IsSpace(rune(trimmed[i])) || inQuote != 0) {
			if (trimmed[i] == '"' || trimmed[i] == '\'') && inQuote == 0 {
				inQuote = trimmed[i]
			} else if trimmed[i] == inQuote && inQuote != 0 {
				inQuote = 0
			}
			i++
		}
		token := trimmed[start:i]

		// Format token
		if strings.HasPrefix(token, "-") {
			inCommand = false
			// Flag (e.g. -o, --bitrate, --output=ep01.mp3)
			if eqIdx := strings.IndexByte(token, '='); eqIdx != -1 {
				flagPart := token[:eqIdx+1]
				valPart := token[eqIdx+1:]
				b.WriteString(sprintColor(th.ExampleFlag, flagPart))
				b.WriteString(sprintColor(th.ExampleArg, valPart))
			} else {
				b.WriteString(sprintColor(th.ExampleFlag, token))
			}
		} else if inCommand {
			// Check if token is obviously an argument (e.g. filename, quoted string, url, contains dots/slashes/numbers only)
			if strings.HasPrefix(token, "\"") || strings.HasPrefix(token, "'") ||
				strings.Contains(token, "/") || strings.Contains(token, ".") ||
				strings.Contains(token, "@") || strings.Contains(token, ":") ||
				isAllDigits(token) {
				inCommand = false
				b.WriteString(sprintColor(th.ExampleArg, token))
			} else {
				b.WriteString(sprintColor(th.ExampleCmd, token))
			}
		} else {
			b.WriteString(sprintColor(th.ExampleArg, token))
		}
	}

	return inline(b.String())
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// renderExamples writes formatted and colorized examples to w.
func renderExamples(w io.Writer, th Theme, o Options, termWidth int, examples []Example, lineIndent, descIndent int) {
	if len(examples) == 0 {
		return
	}
	descColor := th.ExampleDesc
	if descColor == nil {
		descColor = th.Body
	}
	for i, ex := range examples {
		if i > 0 && (ex.Description != "" || examples[i-1].Description != "" || strings.Contains(ex.Line, "\n") || strings.Contains(examples[i-1].Line, "\n")) {
			fmt.Fprintln(w)
		}
		lines := splitLines(ex.Line)
		for _, l := range lines {
			colored := colorizeExampleLine(l, th)
			reflow(w, th.Body, wrapWidth(termWidth, lineIndent, o.maxContent()), lineIndent, "", colored)
		}
		if ex.Description != "" {
			reflow(w, descColor, wrapWidth(termWidth, descIndent, o.maxContent()), descIndent, "", inline(ex.Description))
		}
	}
}

// SplitExampleCommandLine parses a shell command string into separate argument tokens,
// properly handling single quotes, double quotes, escape characters, prompt prefixes,
// and inline comments. If the command contains pipes or operators, the primary command
// segment before the pipe is tokenized for CLI validation.
func SplitExampleCommandLine(line string) ([]string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, nil
	}

	// Strip shell prompt if present
	if strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "> ") || strings.HasPrefix(trimmed, "% ") {
		trimmed = strings.TrimSpace(trimmed[2:])
	}

	// If pipeline or chained command, take the first command for CLI verification
	for _, sep := range []string{" | ", " || ", " && ", " ; ", "\n"} {
		if idx := strings.Index(trimmed, sep); idx != -1 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
	}

	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	tokenStarted := false

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if escaped {
			cur.WriteByte(ch)
			escaped = false
			tokenStarted = true
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			tokenStarted = true
			continue
		}

		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(ch)
			}
			tokenStarted = true
			continue
		}

		if inDouble {
			if ch == '"' {
				inDouble = false
			} else {
				cur.WriteByte(ch)
			}
			tokenStarted = true
			continue
		}

		// Comment outside quotes
		if ch == '#' && (!tokenStarted || unicode.IsSpace(rune(trimmed[i-1]))) {
			break
		}

		switch ch {
		case '\'':
			inSingle = true
			tokenStarted = true
		case '"':
			inDouble = true
			tokenStarted = true
		case ' ', '\t', '\n', '\r':
			if tokenStarted {
				tokens = append(tokens, cur.String())
				cur.Reset()
				tokenStarted = false
			}
		default:
			cur.WriteByte(ch)
			tokenStarted = true
		}
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote in example %q", line)
	}
	if escaped {
		return nil, fmt.Errorf("trailing escape backslash in example %q", line)
	}
	if tokenStarted {
		tokens = append(tokens, cur.String())
	}

	return tokens, nil
}

// ValidateExample statically validates that an Example can be parsed and accepted
// by the application. It verifies that commands exist, flags are recognized with valid
// syntax/values, mutually exclusive rules pass, and positional arguments satisfy constraints.
func ValidateExample(app *App, ex Example, cmd *Command) error {
	if app == nil {
		return errors.New("app is nil")
	}

	lines := splitLines(ex.Line)
	for _, rawLine := range lines {
		l := strings.TrimSpace(rawLine)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}

		tokens, err := SplitExampleCommandLine(l)
		if err != nil {
			return fmt.Errorf("example syntax error in %q: %w", rawLine, err)
		}
		if len(tokens) == 0 {
			continue
		}

		// Strip app name if present as the first token
		name := appName(app)
		if tokens[0] == name || tokens[0] == "./"+name || (app.Name != "" && tokens[0] == app.Name) {
			tokens = tokens[1:]
		}

		if len(tokens) == 0 {
			continue
		}

		// Resolve command path
		targetCmd, ancestors, path, remaining, handled, resolveErr := app.resolveCommand(tokens)
		if resolveErr != nil {
			// If resolving from root failed and cmd context is provided, try resolving with cmd name
			if cmd != nil && (len(tokens) == 0 || tokens[0] != cmd.Name) {
				tokensWithCmd := append([]string{cmd.Name}, tokens...)
				t2, a2, p2, r2, h2, err2 := app.resolveCommand(tokensWithCmd)
				if err2 == nil {
					targetCmd, ancestors, path, remaining, handled = t2, a2, p2, r2, h2
				} else {
					return fmt.Errorf("invalid command in example %q: %w", rawLine, resolveErr)
				}
			} else {
				return fmt.Errorf("invalid command in example %q: %w", rawLine, resolveErr)
			}
		}

		if handled {
			continue
		}

		// Bind flags for parsing
		cmdName := name
		if targetCmd != nil {
			cmdName = targetCmd.Name
		}
		fs := pflag.NewFlagSet(cmdName, pflag.ContinueOnError)

		var helpReq bool
		fs.BoolVarP(&helpReq, "help", "h", false, "help")
		_ = fs.MarkHidden("help")

		_ = bindAndMark(fs, app.PersistentOptions)
		_ = bindAndMark(fs, app.GlobalFlags)
		for _, anc := range ancestors {
			_ = bindAndMark(fs, anc.PersistentOptions)
		}
		if targetCmd != nil {
			_ = bindAndMark(fs, targetCmd.PersistentOptions)
			_ = bindAndMark(fs, targetCmd.Options)
		}

		if parseErr := fs.Parse(remaining); parseErr != nil {
			return fmt.Errorf("invalid flag in example %q: %w", rawLine, parseErr)
		}

		// Options validation
		if targetCmd != nil && targetCmd.OptionsValidator != nil {
			if err := targetCmd.OptionsValidator(fs); err != nil {
				return fmt.Errorf("option constraint failed in example %q: %w", rawLine, err)
			}
		}

		// Positional arguments validation
		cmdArgs := fs.Args()
		if targetCmd != nil && targetCmd.Args != nil {
			if err := targetCmd.Args(cmdArgs); err != nil {
				return fmt.Errorf("argument validation failed in example %q: %w", rawLine, err)
			}
		}
		_ = path
	}

	return nil
}

// ValidateExamples validates all examples defined on the application and all its commands.
// Returns a slice of all validation errors encountered.
func (a *App) ValidateExamples() []error {
	if a == nil {
		return nil
	}
	var errs []error

	// App-level examples
	for _, ex := range a.Examples {
		if err := ValidateExample(a, ex, nil); err != nil {
			errs = append(errs, fmt.Errorf("app %q: %w", appName(a), err))
		}
	}

	// Walk command tree
	var walk func(cmds []Command, path []string)
	walk = func(cmds []Command, path []string) {
		for i := range cmds {
			cmd := &cmds[i]
			cmdPath := append(path, cmd.Name)
			pathStr := strings.Join(cmdPath, " ")
			for _, ex := range cmd.Examples {
				if err := ValidateExample(a, ex, cmd); err != nil {
					errs = append(errs, fmt.Errorf("command %q: %w", pathStr, err))
				}
			}
			if len(cmd.Subcommands) > 0 {
				walk(cmd.Subcommands, cmdPath)
			}
		}
	}
	walk(a.Commands, nil)

	return errs
}

// ValidateAllExamples validates all examples and returns a single combined error if any fail.
func (a *App) ValidateAllExamples() error {
	errs := a.ValidateExamples()
	if len(errs) == 0 {
		return nil
	}
	var msgs []string
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return fmt.Errorf("example validation failed:\n  %s", strings.Join(msgs, "\n  "))
}

// CheckExample validates a single Example against a specific command context.
func (a *App) CheckExample(ex Example, cmd *Command) error {
	return ValidateExample(a, ex, cmd)
}
