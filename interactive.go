package clihelp

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

// getMissingRequiredFlags returns all Option.Required flags that have not been
// set. An option set through any of its spellings counts as set: pflag keeps a
// Changed bit per flag, and an alias is a flag of its own.
func getMissingRequiredFlags(fs *pflag.FlagSet, allOptions []Option) []*pflag.Flag {
	var missing []*pflag.Flag
	for _, opt := range allOptions {
		if !opt.Required {
			continue
		}
		name := parseFlagSpec(opt.Flags).primaryFlagName()
		if name == "" {
			continue
		}
		flg := fs.Lookup(name)
		if flg == nil || optionChanged(fs, name) {
			continue
		}
		missing = append(missing, flg)
	}
	return missing
}

func promptBoolChoice(fs *pflag.FlagSet, flg *pflag.Flag, cleanName string, reader *bufio.Reader, stderr io.Writer) error {
	for {
		fmt.Fprintf(stderr, "\nPlease select a value for required flag --%s:\n", cleanName)
		fmt.Fprintln(stderr, "  [1] true")
		fmt.Fprintln(stderr, "  [2] false")
		fmt.Fprint(stderr, "Select option (1-2): ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		choice := strings.TrimSpace(line)
		switch choice {
		case "1":
			return fs.Set(flg.Name, "true")
		case "2":
			return fs.Set(flg.Name, "false")
		default:
			fmt.Fprintln(stderr, "Invalid choice. Please enter 1 or 2.")
		}
	}
}

func promptTextInput(fs *pflag.FlagSet, flg *pflag.Flag, cleanName string, reader *bufio.Reader, stderr io.Writer) error {
	for {
		defaultPrompt := ""
		if flg.DefValue != "" {
			defaultPrompt = fmt.Sprintf(" [default: %s]", flg.DefValue)
		}
		fmt.Fprintf(stderr, "\nEnter value for required flag --%s%s: ", cleanName, defaultPrompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input := strings.TrimSpace(line)
		if input == "" && flg.DefValue != "" {
			input = flg.DefValue
		}
		if input == "" {
			fmt.Fprintf(stderr, "Error: flag --%s is required and cannot be empty.\n", cleanName)
			continue
		}
		if err := fs.Set(flg.Name, input); err != nil {
			fmt.Fprintf(stderr, "Error: invalid value: %v\n", err)
			continue
		}
		return nil
	}
}

// promptForMissing prompts the user for each missing required flag using numbered choice or text input.
func promptForMissing(fs *pflag.FlagSet, missing []*pflag.Flag, stdin io.Reader, stderr io.Writer) error {
	reader := bufio.NewReader(stdin)
	for _, flg := range missing {
		cleanName := strings.TrimPrefix(flg.Name, "flag-")
		if flg.Value.Type() == "bool" {
			if err := promptBoolChoice(fs, flg, cleanName, reader, stderr); err != nil {
				return err
			}
			continue
		}
		if err := promptTextInput(fs, flg, cleanName, reader, stderr); err != nil {
			return err
		}
	}
	return nil
}

// escapeShellArg quotes a string as a single POSIX shell word.
//
// The test is an allow-list, not a deny-list. A deny-list has to be complete to
// be correct, and this one was not: it missed the glob characters, so an
// argument like "a[1]" was emitted bare and matched a file in whatever directory
// the script ran from, and it missed the backslash.
func escapeShellArg(arg string) string {
	if arg != "" && shellSafeArg.MatchString(arg) {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// shellSafeArg matches the characters that need no quoting in any POSIX shell.
var shellSafeArg = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// constructCommand builds the equivalent full CLI command for presentation.
func constructCommand(a *App, path []string, fs *pflag.FlagSet, positionalArgs []string) string {
	var parts []string
	parts = append(parts, appName(a))
	parts = append(parts, path...)

	// Collect set flags in order
	fs.Visit(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		cleanName := strings.TrimPrefix(f.Name, "flag-")
		val := escapeShellArg(f.Value.String())
		if f.Value.Type() == "bool" && f.Value.String() == "true" {
			if len(cleanName) == 1 {
				parts = append(parts, "-"+cleanName)
			} else {
				parts = append(parts, "--"+cleanName)
			}
		} else {
			if len(cleanName) == 1 {
				parts = append(parts, fmt.Sprintf("-%s %s", cleanName, val))
			} else {
				parts = append(parts, fmt.Sprintf("--%s %s", cleanName, val))
			}
		}
	})

	for _, arg := range positionalArgs {
		parts = append(parts, escapeShellArg(arg))
	}

	return strings.Join(parts, " ")
}
