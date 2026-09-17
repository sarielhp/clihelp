//go:build !unix

package clihelp

// Ownership is a POSIX concept; elsewhere the atomic replace carries everything
// the platform exposes.
func preserveOwner(from, to string)       {}
func refuseForeignOwner(path string) bool { return false }
