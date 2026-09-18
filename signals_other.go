//go:build !unix

package clihelp

import "os"

// pagerSignals is the portable subset: SIGQUIT is not defined off unix.
func pagerSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
