//go:build unix

package clihelp

import (
	"os"
	"syscall"
)

// pagerSignals are the terminal-generated signals the pager owns while it runs.
//
// The pager is in the caller's foreground process group, so Ctrl-C reaches both.
// Go's default action kills the parent immediately, while a pager that traps
// SIGINT — less does — keeps running, keeps the terminal, and writes over the
// shell prompt of a program that has already exited. Holding these for the
// duration lets the pager handle the key and reset the terminal itself, which is
// what git does.
func pagerSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGQUIT}
}
