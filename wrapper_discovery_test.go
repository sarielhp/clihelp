package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeWrapper generates a wrapper of app the way a user would, into dir.
func writeWrapper(t *testing.T, app *App, dir, name string, args ...string) {
	t.Helper()
	var b strings.Builder
	if err := GenWrapperScript(app, name, args, &b); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
}

// Registering a wrapper used to be lines the user pasted into a startup file,
// in the syntax of the shell $SHELL named. The installed integration registers
// every marked wrapper on $PATH itself — with completion, and with the Alt-H
// registry, without which the wrapper's __explain branch is unreachable.
func TestInstallRegistersWrappersFoundOnPath(t *testing.T) {
	for _, tt := range []struct {
		shell string
		want  []string
	}{
		{"bash", []string{"complete -o default -F _bare_complete pd", `_clihelp_apps="${_clihelp_apps:-} pd:`}},
		{"zsh", []string{"compdef _bare pd", "typeset -g _clihelp_apps=", "pd:"}},
		{"fish", []string{"complete -c pd --wraps 'bare deploy -2'", "set -g _clihelp_apps $_clihelp_apps pd:"}},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			sandboxHome(t)
			bin := t.TempDir()
			other := t.TempDir()
			t.Setenv("PATH", bin+string(os.PathListSeparator)+other)
			writeWrapper(t, bareApp(), bin, "pd", "deploy", "-2")
			// Shadowed by bin's pd, as the shell would shadow it.
			writeWrapper(t, bareApp(), other, "pd", "build")
			// Another program's wrapper, and a script that merely mentions bare.
			writeWrapper(t, &App{Name: "other"}, bin, "od", "x")
			if err := os.WriteFile(filepath.Join(bin, "plain"), []byte("#!/bin/sh\nbare build \"$@\"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			// Marked, but not executable: the shell would not run it.
			writeWrapper(t, bareApp(), bin, "noexec", "build")
			if err := os.Chmod(filepath.Join(bin, "noexec"), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := installShellIntegration(bareApp(), tt.shell, true); err != nil {
				t.Fatal(err)
			}
			path, err := IntegrationPath(bareApp(), tt.shell)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := string(raw)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("the integration does not contain %q:\n%s", want, got)
				}
			}
			for _, absent := range []string{"bare build'", " od:", " plain:", " noexec:"} {
				if strings.Contains(got, absent) {
					t.Errorf("the integration registers something it should not (%q):\n%s", absent, got)
				}
			}
		})
	}
}

// The marker is read from a file anyone may have written, and it ends up in
// generated shell source; a target that does not tokenize as a command line of
// this program is not taken.
func TestReadWrapperMarkerRejectsForeignAndMalformedTargets(t *testing.T) {
	dir := t.TempDir()
	for _, tt := range []struct {
		name, marker string
		ok           bool
	}{
		{"good", "bare deploy -2", true},
		{"other-app", "other deploy", false},
		{"prefix-app", "barely deploy", false},
		{"unterminated", "bare 'deploy", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name)
			body := "#!/bin/sh\n" + wrapperMarkerPrefix + tt.marker + "\n"
			if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
			target, ok := readWrapperMarker(path, "bare")
			if ok != tt.ok || (ok && target != tt.marker) {
				t.Errorf("readWrapperMarker = %q, %v; want ok=%v", target, ok, tt.ok)
			}
		})
	}
}

// "wrapper mt tui -2" rejected -2 as an option of the verb, and before that
// silently dropped it, so the wrapper ran "tui" without it.
func TestWrapperKeepsPresetArgumentsThatLookLikeFlags(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"after the name", []string{"mt", "build", "-2"}},
		{"after --", []string{"--", "mt", "build", "-2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := runProto(t, bareApp(), append([]string{"__clihelp", "wrapper"}, tt.args...)...)
			res.AssertNoError(t)
			res.AssertStdoutContains(t, "# clihelp-wraps: bare build -2")
			res.AssertStdoutContains(t, `exec "$__clihelp_app" build -2 "$@"`)
		})
	}
}

// "wrapper --completion-from X > X": the shell empties X before the program
// reads it. Saying so beats "no invocation found" about a file that had one.
func TestWrapperFromRefusesToReadItsOwnOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mt")
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	var stderr strings.Builder
	err = executeWrapperGen(bareApp(), path, nil, false, out, &stderr)
	if err == nil || !strings.Contains(err.Error(), "emptied") {
		t.Errorf("reading the file being written gave %v", err)
	}
}

func TestInteractiveShellName(t *testing.T) {
	for _, tt := range []struct {
		argv []string
		want string
	}{
		{[]string{"fish"}, "fish"},
		{[]string{"/usr/bin/fish"}, "fish"},
		{[]string{"-zsh"}, "zsh"},
		{[]string{"bash", "--login", "-i"}, "bash"},
		{[]string{"bash", "script.sh"}, ""},
		{[]string{"zsh", "-c", "app __clihelp install"}, ""},
		{[]string{"bash", "-lc", "app"}, ""},
		{[]string{"fish", "--command", "app"}, ""},
		{[]string{"ruby", "x.rb"}, ""},
		{[]string{"dash"}, ""},
	} {
		if got := interactiveShellName(tt.argv); got != tt.want {
			t.Errorf("interactiveShellName(%q) = %q, want %q", tt.argv, got, tt.want)
		}
	}
}
