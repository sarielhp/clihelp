//go:build unix

package clihelp

import (
	"os"
	"syscall"
)

// preserveOwner copies the old file's ownership onto the replacement.
//
// An atomic replace installs a new inode, created by the calling process, so a
// run under a different effective uid silently takes the user's own startup file
// away from them. maybeRefreshIntegration runs before command dispatch on
// every invocation, so `sudo -E myapp anything` is enough to do it. Best effort:
// an unprivileged process cannot chown, and failing to is not a reason to
// abandon the write.
func preserveOwner(from, to string) {
	info, err := os.Stat(from)
	if err != nil {
		return
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	_ = os.Chown(to, int(st.Uid), int(st.Gid))
}

// refuseForeignOwner reports whether writing path would take a file away from
// the user who owns it. Writing another user's dotfile as root is never what
// AutoRefreshIntegration means.
func refuseForeignOwner(path string) bool {
	if os.Geteuid() != 0 {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid != 0
}
