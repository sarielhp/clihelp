//go:build !unix

package clihelp

// lockFile is a no-op where flock is unavailable: the read-modify-write proceeds
// unserialized, as it did everywhere before the lock existed.
func lockFile(path string) (func(), error) { return func() {}, nil }
