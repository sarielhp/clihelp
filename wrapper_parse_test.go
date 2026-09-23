package clihelp

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractWrapperArgs(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		app     string
		want    []string
		wantErr string
	}{
		{
			name:   "standard simple wrapper",
			script: "#!/bin/bash\nmail_cli -2 tui \"$@\"\n",
			app:    "mail_cli",
			want:   []string{"-2", "tui"},
		},
		{
			name:   "exec and absolute path",
			script: "#!/bin/sh\nexec /usr/local/bin/mail_cli --account \"personal\" \"$@\"\n",
			app:    "mail_cli",
			want:   []string{"--account", "personal"},
		},
		{
			name:   "single quotes",
			script: "mail_cli 'deploy' '--tag=v1' \"$@\"",
			app:    "mail_cli",
			want:   []string{"deploy", "--tag=v1"},
		},
		{
			name:   "bare forwarding",
			script: "mail_cli \"$@\"",
			app:    "mail_cli",
			want:   []string(nil),
		},
		{
			name:   "curly braces forwarding",
			script: "mail_cli run \"${@}\"",
			app:    "mail_cli",
			want:   []string{"run"},
		},
		{
			name:   "comments and blank lines before invocation",
			script: "#!/bin/sh\n# setup\n\n# another comment\nmail_cli deploy \"$@\"\n",
			app:    "mail_cli",
			want:   []string{"deploy"},
		},
		{
			name:    "script with control flow fails fast",
			script:  "if [ -f config ]; then\n  mail_cli run \"$@\"\nfi\n",
			app:     "mail_cli",
			wantErr: "control flow",
		},
		{
			name:    "script with shell variable fails fast",
			script:  "mail_cli \"$ACCOUNT\" \"$@\"",
			app:     "mail_cli",
			wantErr: "unexpanded shell variable",
		},
		{
			name:    "missing argument forwarding",
			script:  "mail_cli -2 tui\n",
			app:     "mail_cli",
			wantErr: "does not end with",
		},
		{
			name:    "no matching invocation",
			script:  "echo 'running'\nother_tool \"$@\"\n",
			app:     "mail_cli",
			wantErr: "could not find a line invoking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractWrapperArgs(strings.NewReader(tt.script), tt.app)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("extractWrapperArgs() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractWrapperArgs() unexpected error: %v", err)
			}
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractWrapperArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveWrapperTarget(t *testing.T) {
	t.Run("resolves existing file path", func(t *testing.T) {
		dir := t.TempDir()
		script := filepath.Join(dir, "my-wrapper")
		if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		path, name, err := resolveWrapperTarget(script)
		if err != nil {
			t.Fatalf("resolveWrapperTarget(%q) error: %v", script, err)
		}
		if name != "my-wrapper" {
			t.Errorf("name = %q, want %q", name, "my-wrapper")
		}
		if path != script {
			t.Errorf("path = %q, want %q", path, script)
		}
	})

	t.Run("resolves binary in PATH", func(t *testing.T) {
		path, name, err := resolveWrapperTarget("sh")
		if err != nil {
			t.Fatalf("resolveWrapperTarget(\"sh\") error: %v", err)
		}
		if name != "sh" {
			t.Errorf("name = %q, want %q", name, "sh")
		}
		if !filepath.IsAbs(path) {
			t.Errorf("path = %q should be absolute", path)
		}
	})

	t.Run("fails for non-existent target", func(t *testing.T) {
		_, _, err := resolveWrapperTarget("definitely_not_a_real_command_xyz123")
		if err == nil {
			t.Errorf("expected error for non-existent target")
		}
	})
}
