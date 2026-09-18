package main

import (
	"os"
	"testing"
)

// The example app sets AutoInstallCompletion, and RenderCommand honours it — so
// merely rendering a help page in a test wrote a completion script into whoever
// ran the suite. Point every path clihelp can install into at a temp directory
// and switch the auto path off, for the whole package, before any test runs.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "clihelp-example-home")
	if err != nil {
		panic(err)
	}
	for k, v := range map[string]string{
		"HOME":                       home,
		"XDG_CONFIG_HOME":            home + "/.config",
		"XDG_DATA_HOME":              home + "/.local/share",
		"CLIHELP_NO_AUTO_COMPLETION": "1",
	} {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
