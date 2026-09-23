package clihelp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveWrapperTarget locates the wrapper script file and determines its name.
// It checks if target exists as a file, or searches target in PATH via exec.LookPath.
func resolveWrapperTarget(target string) (path, name string, err error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("empty wrapper target")
	}

	if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
		abs, absErr := filepath.Abs(target)
		if absErr == nil {
			return abs, filepath.Base(target), nil
		}
		return target, filepath.Base(target), nil
	}

	if resolved, lookErr := exec.LookPath(target); lookErr == nil {
		return resolved, filepath.Base(target), nil
	}

	return "", "", fmt.Errorf("wrapper target %q not found as file or command in PATH", target)
}

var controlFlowKeywords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"case": true, "esac": true, "for": true, "while": true, "until": true,
	"do": true, "done": true, "|": true, "||": true, "&&": true,
}

// extractWrapperArgs parses a script from r and extracts preset arguments passed
// to appName prior to "$@" (or "${@}").
func extractWrapperArgs(r io.Reader, appName string) ([]string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		words, err := tokenizeShellWords(line)
		if err != nil {
			return nil, fmt.Errorf("syntax error parsing wrapper script: %w", err)
		}
		if len(words) == 0 {
			continue
		}

		for _, w := range words {
			if controlFlowKeywords[w] {
				return nil, fmt.Errorf("script contains shell control flow (%s); specify arguments manually", w)
			}
		}

		idx := 0
		if words[0] == "exec" {
			idx = 1
		}
		if idx >= len(words) {
			continue
		}

		cmdWord := words[idx]
		if cmdWord != appName && !strings.HasSuffix(cmdWord, "/"+appName) {
			continue
		}

		rest := words[idx+1:]
		if len(rest) == 0 {
			return nil, fmt.Errorf("line invoking %q has no argument forwarding (\"$@\")", appName)
		}

		last := rest[len(rest)-1]
		if last != "$@" && last != "${@}" && last != "$*" {
			return nil, fmt.Errorf("line invoking %q does not end with \"$@\"", appName)
		}

		preset := rest[:len(rest)-1]
		for _, arg := range preset {
			if strings.Contains(arg, "$") {
				return nil, fmt.Errorf("script contains unexpanded shell variable %q; specify arguments manually", arg)
			}
		}
		return preset, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading wrapper script: %w", err)
	}
	return nil, fmt.Errorf("could not find a line invoking %q with \"$@\" in script", appName)
}

// tokenizeShellWords splits line into shell words, respecting quotes and escapes.
func tokenizeShellWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inSingle, inDouble, escaped := false, false, false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			cur.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}
		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(ch)
			}
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			} else {
				cur.WriteByte(ch)
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case ' ', '\t', '\r', '\n':
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(ch)
		}
	}
	if inSingle || inDouble || escaped {
		return nil, errors.New("unmatched quote or trailing escape")
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words, nil
}
