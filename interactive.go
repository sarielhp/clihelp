package clihelp

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
)

// getMissingRequiredFlags returns all Option.Required flags that have not been set.
func getMissingRequiredFlags(fs *pflag.FlagSet, allOptions []Option) []*pflag.Flag {
	var missing []*pflag.Flag
	for _, opt := range allOptions {
		if opt.Required {
			spec := parseFlagSpec(opt.Flags)
			name := ""
			if len(spec.longNames) > 0 {
				name = spec.longNames[0]
			} else if len(spec.shortNames) > 0 {
				name = "flag-" + spec.shortNames[0]
			}
			if name != "" {
				flg := fs.Lookup(name)
				if flg != nil && !fs.Changed(name) {
					missing = append(missing, flg)
				}
			}
		}
	}
	return missing
}

// promptForMissing prompts the user for each missing required flag using numbered choice or text input.
func promptForMissing(fs *pflag.FlagSet, missing []*pflag.Flag, stdin io.Reader, stderr io.Writer) error {
	reader := bufio.NewReader(stdin)
	for _, flg := range missing {
		cleanName := strings.TrimPrefix(flg.Name, "flag-")

		// If it's a boolean flag, present a numbered choice menu (True/False)
		if flg.Value.Type() == "bool" {
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
				if choice == "1" {
					if err := fs.Set(flg.Name, "true"); err != nil {
						return err
					}
					break
				} else if choice == "2" {
					if err := fs.Set(flg.Name, "false"); err != nil {
						return err
					}
					break
				}
				fmt.Fprintln(stderr, "Invalid choice. Please enter 1 or 2.")
			}
			continue
		}

		// Otherwise, prompt for standard text input
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
			break
		}
	}
	return nil
}

// escapeShellArg single-quotes a string to make it shell-safe.
func escapeShellArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.ContainsAny(arg, " \t\n\r&;`'\"|*?~<>^()!$") {
		return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return arg
}

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
