//go:build unix

package clihelp

import (
	"os"
	"syscall"
)

// lockFile takes an exclusive advisory lock keyed on path, and returns the
// function that releases it. The lock is keyed on the *file being edited*, not
// on the application, because different applications contend for the same
// startup file.
//
// flock is advisory and unavailable on some filesystems, so a failure to acquire
// is not fatal: the caller proceeds unlocked, which is exactly the behaviour
// that existed before. A non-clihelp writer is not serialized either, and cannot
// be.
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path+".clihelp-lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return func() {}, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		_ = os.Remove(path + ".clihelp-lock")
	}, nil
}
