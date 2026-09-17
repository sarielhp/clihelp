//go:build unix

package clihelp

import (
	"os"
	"syscall"
	"testing"
)

func ownerOf(t *testing.T, info os.FileInfo) (int, int) {
	t.Helper()
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("no POSIX ownership on this platform")
	}
	return int(st.Uid), int(st.Gid)
}
