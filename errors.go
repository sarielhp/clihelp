package clihelp

import "errors"

// ErrUsage indicates a command-line syntax, argument, flag, or resolution error.
var ErrUsage = errors.New("usage error")

// IsUsageError reports whether err was caused by a command-line usage or syntax error.
func IsUsageError(err error) bool {
	return errors.Is(err, ErrUsage)
}
