package clihelp

import (
	"errors"
	"fmt"
	"io"
	"strings"

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

// isSpaceByte reports whether b is shell whitespace. The scanners here walk
// bytes, and unicode.IsSpace(rune(b)) read the 0xA0 continuation byte of a rune
// such as "à" as a non-breaking space, splitting the token — and the color run
// around it — in the middle of a character.
func isSpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// startsComment reports whether a comment begins at s[i]. A "#" or "//" only
// starts one at the beginning of a token: inside "http://host/p#frag" they are
// part of the argument, and treating them as a comment greyed out the rest of
// the line.
func startsComment(s string, i, tokenStart int) bool {
	if i > tokenStart && i > 0 && !isSpaceByte(s[i-1]) {
		return false
	}
	return s[i] == '#' || strings.HasPrefix(s[i:], "//")
}

// colorizeExampleLine applies ANSI syntax colors to a command-line example string.
// It recognizes comments, shell prompts, subcommands, flags, values, and operators.
func colorizeExampleLine(line string, th Theme) string {
	return ColorizeExampleLineWithApp(nil, nil, line, th)
}

// ColorizeExampleLineWithApp applies ANSI syntax colors to an example string using the application
// command tree to accurately identify subcommands, flags, and arguments.
//
// The line comes back as it was written, only colored. It used to be returned
// through the inline-markdown renderer, which rewrote the command itself: it
// swallowed the backslashes of "--path C:\temp\x", turned the asterisks of
// "'*.go'" into emphasis, and ate the escape in "echo a\ b". A description is
// prose and gets its own inline() pass; a command line is not.
func ColorizeExampleLineWithApp(app *App, cmd *Command, line string, th Theme) string {
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

	var b strings.Builder
	b.WriteString(indent)
	b.WriteString(promptPrefix)

	i := 0
	for i < len(trimmed) {
		// Whitespace
		if isSpaceByte(trimmed[i]) {
			b.WriteByte(trimmed[i])
			i++
			continue
		}

		// Trailing inline comment (# ... or // ...)
		if startsComment(trimmed, i, i) {
			comment := trimmed[i:]
			b.WriteString(sprintColor(th.ExampleComment, comment))
			break
		}

		// Shell operators: |, ||, &&, ;, >, >>, <
		if trimmed[i] == '|' || trimmed[i] == '&' || trimmed[i] == ';' || trimmed[i] == '>' || trimmed[i] == '<' {
			j := i + 1
			for j < len(trimmed) && (trimmed[j] == '|' || trimmed[j] == '&' || trimmed[j] == '>' || trimmed[j] == '<' || trimmed[j] == ';') {
				j++
			}
			op := trimmed[i:j]
			b.WriteString(sprintColor(th.Accent, op))
			i = j
			continue
		}

		// Extract command segment up to next operator or comment
		segStart := i
		inQuote := byte(0)
		for i < len(trimmed) {
			ch := trimmed[i]
			if (ch == '"' || ch == '\'') && inQuote == 0 {
				inQuote = ch
			} else if ch == inQuote && inQuote != 0 {
				inQuote = 0
			} else if inQuote == 0 {
				if startsComment(trimmed, i, segStart) {
					break
				}
				if ch == '|' || ch == '&' || ch == ';' || ch == '>' || ch == '<' {
					break
				}
			}
			i++
		}
		segText := trimmed[segStart:i]
		coloredSeg := colorizeSegment(app, cmd, segText, th)
		b.WriteString(coloredSeg)
	}

	return b.String()
}

type segToken struct {
	text  string
	start int
	end   int
}

func extractSegmentTokens(seg string) []segToken {
	var tokens []segToken
	i := 0
	for i < len(seg) {
		if isSpaceByte(seg[i]) {
			i++
			continue
		}
		start := i
		inQuote := byte(0)
		for i < len(seg) {
			ch := seg[i]
			if (ch == '"' || ch == '\'') && inQuote == 0 {
				inQuote = ch
			} else if ch == inQuote && inQuote != 0 {
				inQuote = 0
			} else if inQuote == 0 && isSpaceByte(ch) {
				break
			}
			i++
		}
		tokens = append(tokens, segToken{
			text:  seg[start:i],
			start: start,
			end:   i,
		})
	}
	return tokens
}

func identifySegmentTokens(app *App, cmd *Command, toks []segToken, rawStrings []string) ([]bool, []bool) {
	isCmd := make([]bool, len(toks))
	isFlag := make([]bool, len(toks))

	if app != nil {
		name := appName(app)
		tokensToResolve := rawStrings
		offset := 0
		if len(tokensToResolve) > 0 && (tokensToResolve[0] == name || tokensToResolve[0] == "./"+name || (app.Name != "" && tokensToResolve[0] == app.Name)) {
			isCmd[0] = true
			tokensToResolve = tokensToResolve[1:]
			offset = 1
		}

		res, resolveErr := app.resolveCommand(tokensToResolve)
		if resolveErr == nil || res.cmd != nil {
			// indices are positions within tokensToResolve, which may hold
			// global flags before and between the command names.
			for _, idx := range res.indices {
				if offset+idx < len(isCmd) {
					isCmd[offset+idx] = true
				}
			}
		} else if cmd != nil {
			if len(tokensToResolve) > 0 && tokensToResolve[0] == cmd.Name {
				isCmd[offset] = true
			}
		}
	} else if cmd != nil {
		if len(rawStrings) > 0 && rawStrings[0] == cmd.Name {
			isCmd[0] = true
		} else if len(rawStrings) > 0 && !strings.HasPrefix(rawStrings[0], "-") {
			isCmd[0] = true
		}
	} else {
		if len(rawStrings) > 0 && !strings.HasPrefix(rawStrings[0], "-") {
			isCmd[0] = true
		}
	}

	for idx, t := range toks {
		if !isCmd[idx] && strings.HasPrefix(t.text, "-") {
			isFlag[idx] = true
		}
	}
	return isCmd, isFlag
}

func buildColorizedSegment(seg string, toks []segToken, isCmd, isFlag []bool, th Theme) string {
	var b strings.Builder
	lastPos := 0
	for idx, t := range toks {
		if t.start > lastPos {
			b.WriteString(seg[lastPos:t.start])
		}
		lastPos = t.end

		if isCmd[idx] {
			b.WriteString(sprintColor(th.ExampleCmd, t.text))
		} else if isFlag[idx] {
			if eqIdx := strings.IndexByte(t.text, '='); eqIdx != -1 {
				flagPart := t.text[:eqIdx+1]
				valPart := t.text[eqIdx+1:]
				b.WriteString(sprintColor(th.ExampleFlag, flagPart))
				b.WriteString(sprintColor(th.ExampleArg, valPart))
			} else {
				b.WriteString(sprintColor(th.ExampleFlag, t.text))
			}
		} else {
			b.WriteString(sprintColor(th.ExampleArg, t.text))
		}
	}
	if lastPos < len(seg) {
		b.WriteString(seg[lastPos:])
	}
	return b.String()
}

func colorizeSegment(app *App, cmd *Command, seg string, th Theme) string {
	toks := extractSegmentTokens(seg)
	if len(toks) == 0 {
		return seg
	}

	rawStrings := make([]string, len(toks))
	for i, t := range toks {
		rawStrings[i] = t.text
	}

	isCmd, isFlag := identifySegmentTokens(app, cmd, toks, rawStrings)
	return buildColorizedSegment(seg, toks, isCmd, isFlag, th)
}

// renderExamples writes formatted and colorized examples to w.
func renderExamples(w io.Writer, app *App, cmd *Command, th Theme, o Options, termWidth int, examples []Example, lineIndent, descIndent int) {
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
			writeExampleLine(w, th, lineIndent, ColorizeExampleLineWithApp(app, cmd, l, th))
		}
		if ex.Description != "" {
			reflow(w, descColor, wrapWidth(termWidth, descIndent, o.maxContent()), descIndent, "", inline(ex.Description))
		}
	}
}

// writeExampleLine emits one example command line at its indent and nothing
// more. Examples are meant to be copied, and the prose reflow broke a long one
// across two lines, neither of which could be pasted into a shell.
func writeExampleLine(w io.Writer, th Theme, indent int, colored string) {
	if strings.TrimSpace(colored) == "" {
		fmt.Fprintln(w)
		return
	}
	// The indent stays outside the colour, as it does everywhere else since
	// emitLine stopped wrapping a whole line: a nested colour closes with a full
	// SGR reset, so anything wrapped around already-coloured text ends up
	// colouring only the part before the first nested sequence.
	indentStr := strings.Repeat(" ", indent)
	if th.Body != nil {
		fmt.Fprintln(w, indentStr+th.Body.Sprint(colored))
		return
	}
	fmt.Fprintln(w, indentStr+colored)
}

func cleanExampleCommandLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "> ") || strings.HasPrefix(trimmed, "% ") {
		trimmed = strings.TrimSpace(trimmed[2:])
	}
	if idx := strings.IndexByte(trimmed, '\n'); idx != -1 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	return trimmed
}

type exampleTokenState struct {
	tokens       []string
	cur          strings.Builder
	inSingle     bool
	inDouble     bool
	escaped      bool
	tokenStarted bool
}

func (s *exampleTokenState) pushToken() {
	if s.tokenStarted {
		s.tokens = append(s.tokens, s.cur.String())
		s.cur.Reset()
		s.tokenStarted = false
	}
}

// dropDescriptorPrefix discards the token in progress when it is the file
// descriptor of the redirection that follows it, as the "2" of "2>/dev/null".
func (s *exampleTokenState) dropDescriptorPrefix() {
	if !s.tokenStarted {
		return
	}
	for _, r := range s.cur.String() {
		if r < '0' || r > '9' {
			return
		}
	}
	s.cur.Reset()
	s.tokenStarted = false
}

// isShellOperatorByte reports whether b begins a shell operator: a pipe, a
// redirection, or a list separator.
func isShellOperatorByte(b byte) bool {
	switch b {
	case '|', '&', ';', '<', '>':
		return true
	}
	return false
}

func tokenizeCommandLine(trimmed, line string) ([]string, error) {
	var s exampleTokenState
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if s.escaped {
			s.cur.WriteByte(ch)
			s.escaped = false
			s.tokenStarted = true
			continue
		}

		if ch == '\\' && !s.inSingle {
			s.escaped = true
			s.tokenStarted = true
			continue
		}

		if s.inSingle {
			if ch == '\'' {
				s.inSingle = false
			} else {
				s.cur.WriteByte(ch)
			}
			s.tokenStarted = true
			continue
		}

		if s.inDouble {
			if ch == '"' {
				s.inDouble = false
			} else {
				s.cur.WriteByte(ch)
			}
			s.tokenStarted = true
			continue
		}

		if ch == '#' && (!s.tokenStarted || isSpaceByte(trimmed[i-1])) {
			break
		}

		if isShellOperatorByte(ch) {
			// A pipe, a redirection or a list operator ends the command being
			// validated. Unquoted, it is the shell's, not the program's — and
			// "app logs > out.txt" used to be validated with "> out.txt" as two
			// positional arguments.
			s.dropDescriptorPrefix()
			break
		}

		switch ch {
		case '\'':
			s.inSingle = true
			s.tokenStarted = true
		case '"':
			s.inDouble = true
			s.tokenStarted = true
		case ' ', '\t', '\n', '\r':
			s.pushToken()
		default:
			s.cur.WriteByte(ch)
			s.tokenStarted = true
		}
	}

	if s.inSingle || s.inDouble {
		return nil, fmt.Errorf("unclosed quote in example %q", line)
	}
	if s.escaped {
		return nil, fmt.Errorf("trailing escape backslash in example %q", line)
	}
	s.pushToken()

	return s.tokens, nil
}

// splitExampleCommandLine parses a shell command string into separate argument tokens,
// properly handling single quotes, double quotes, escape characters, prompt prefixes,
// and inline comments. If the command contains pipes or operators, the primary command
// segment before the pipe is tokenized for CLI validation.
func splitExampleCommandLine(line string) ([]string, error) {
	trimmed := cleanExampleCommandLine(line)
	if trimmed == "" {
		return nil, nil
	}
	return tokenizeCommandLine(trimmed, line)
}

func resolveExampleCommand(app *App, cmd *Command, tokens []string, rawLine string) (*Command, []*Command, []string, bool, error) {
	res, resolveErr := app.resolveCommand(tokens)
	if resolveErr != nil {
		if cmd != nil && (len(tokens) == 0 || tokens[0] != cmd.Name) {
			tokensWithCmd := append([]string{cmd.Name}, tokens...)
			retry, err2 := app.resolveCommand(tokensWithCmd)
			if err2 == nil {
				return retry.cmd, retry.ancestors, retry.remaining, retry.isHelp, nil
			}
		}
		return nil, nil, nil, false, fmt.Errorf("invalid command in example %q: %w", rawLine, resolveErr)
	}
	return res.cmd, res.ancestors, res.remaining, res.isHelp, nil
}

func validateExampleTokens(app *App, targetCmd *Command, ancestors []*Command, remaining []string, rawLine string) error {
	cmdName := appName(app)
	if targetCmd != nil {
		cmdName = targetCmd.Name
	}
	fs := pflag.NewFlagSet(cmdName, pflag.ContinueOnError)

	// The same binder the real flag set uses, rather than this function's own
	// idea of what the help flags are. They had drifted: the real set binds
	// --help-concise with -h, and --help with -H when App.ExtendedHelpFlag is
	// on, while this bound only --help with -h. An example using a help flag the
	// application really accepts was reported as "invalid flag in example" — by
	// Audit, which the README recommends running in CI.
	_ = app.bindHelpFlags(fs, cmdName)

	allOptions := app.collectAllActiveOptions(targetCmd, ancestors)
	// Scratch binding: validating an example must not write through the pointers
	// the application runs on, and a bind error is a defect in the declarations,
	// not in the example, so it is reported as itself instead of resurfacing as
	// "unknown flag" on whichever example happens to come first.
	if err := bindScratchAll(fs, allOptions); err != nil {
		return fmt.Errorf("cannot validate example %q: %w", rawLine, err)
	}

	if parseErr := fs.Parse(remaining); parseErr != nil {
		return fmt.Errorf("invalid flag in example %q: %w", rawLine, parseErr)
	}

	if missing := getMissingRequiredFlags(fs, allOptions); len(missing) > 0 {
		var names []string
		for _, m := range missing {
			names = append(names, `"`+strings.TrimPrefix(m.Name, "flag-")+`"`)
		}
		return fmt.Errorf("required flag(s) %s not set in example %q", strings.Join(names, ", "), rawLine)
	}

	if targetCmd != nil && targetCmd.OptionsValidator != nil {
		if err := targetCmd.OptionsValidator(fs); err != nil {
			return fmt.Errorf("option constraint failed in example %q: %w", rawLine, err)
		}
	}

	cmdArgs := fs.Args()
	if targetCmd != nil && targetCmd.Args != nil {
		if err := targetCmd.Args(cmdArgs); err != nil {
			return fmt.Errorf("argument validation failed in example %q: %w", rawLine, err)
		}
	}
	return nil
}

// validateExample statically validates that an Example can be parsed and accepted
// by the application. It verifies that commands exist, flags are recognized with valid
// syntax/values, mutually exclusive rules pass, and positional arguments satisfy constraints.
func validateExample(app *App, ex Example, cmd *Command) error {
	if app == nil {
		return errors.New("app is nil")
	}

	lines := splitLines(ex.Line)
	for _, rawLine := range lines {
		l := strings.TrimSpace(rawLine)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}

		tokens, err := splitExampleCommandLine(l)
		if err != nil {
			return fmt.Errorf("example syntax error in %q: %w", rawLine, err)
		}
		if len(tokens) == 0 {
			continue
		}

		name := appName(app)
		if tokens[0] == name || tokens[0] == "./"+name || (app.Name != "" && tokens[0] == app.Name) {
			tokens = tokens[1:]
		}
		if len(tokens) == 0 {
			continue
		}

		targetCmd, ancestors, remaining, handled, err := resolveExampleCommand(app, cmd, tokens, rawLine)
		if err != nil {
			return err
		}
		if handled {
			continue
		}

		if err := validateExampleTokens(app, targetCmd, ancestors, remaining, rawLine); err != nil {
			return err
		}
	}

	return nil
}

// validateExamples validates all examples defined on the application and all its commands.
// Returns a slice of all validation errors encountered.
func (a *App) validateExamples() []error {
	if a == nil {
		return nil
	}
	var errs []error

	// App-level examples
	for _, ex := range a.Examples {
		if err := validateExample(a, ex, nil); err != nil {
			errs = append(errs, fmt.Errorf("app %q: %w", appName(a), err))
		}
	}

	// Walk command tree
	_ = a.Walk(func(path []string, cmd *Command) error {
		pathStr := strings.Join(path, " ")
		for _, ex := range cmd.Examples {
			if err := validateExample(a, ex, cmd); err != nil {
				errs = append(errs, fmt.Errorf("command %q: %w", pathStr, err))
			}
		}
		return nil
	})

	return errs
}

// ValidateAllExamples validates all examples and returns a single combined error if any fail.
func (a *App) ValidateAllExamples() error {
	errs := a.validateExamples()
	if len(errs) == 0 {
		return nil
	}
	var msgs []string
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return fmt.Errorf("example validation failed:\n  %s", strings.Join(msgs, "\n  "))
}
