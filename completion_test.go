package clihelp

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestShellCompletionProtocol(t *testing.T) {
	var outBuf bytes.Buffer
	var podcastVal string
	var fillVal bool

	app := &App{
		Name:   "podcli",
		Stdout: &outBuf,
		Commands: []Command{
			{
				Name:        "scan",
				Aliases:     []string{"rescan"},
				Description: "Scan podcasts",
				Options: []Option{
					{
						Flags:       "-p, --podcast <id>",
						Description: "Podcast ID",
						Complete: func(toComplete string) []string {
							candidates := []string{"pod1\tHistory podcast", "pod2\tTech podcast"}
							var res []string
							for _, c := range candidates {
								if strings.HasPrefix(c, toComplete) {
									res = append(res, c)
								}
							}
							return res
						},
						Binder: String(&podcastVal, "-p, --podcast <id>", "", "Podcast ID").Binder,
					},
					Bool(&fillVal, "-f, --fill", false, "Fill gaps"),
				},
			},
			{
				Name:        "download",
				Description: "Download episodes",
			},
		},
	}

	// 1. Root command completion
	outBuf.Reset()
	err := app.ExecuteContext(context.Background(), []string{"__complete", "sc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "scan\tScan podcasts") {
		t.Errorf("expected scan suggestion, got: %q", outBuf.String())
	}

	// 2. Subcommand flag completion
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"__complete", "scan", "--"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "--podcast\tPodcast ID") || !strings.Contains(outBuf.String(), "--fill\tFill gaps") {
		t.Errorf("expected flag suggestions, got: %q", outBuf.String())
	}

	// 3. Dynamic flag value completion
	outBuf.Reset()
	err = app.ExecuteContext(context.Background(), []string{"__complete", "scan", "-p", "pod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "pod1\tHistory podcast") || !strings.Contains(outBuf.String(), "pod2\tTech podcast") {
		t.Errorf("expected dynamic value completions, got: %q", outBuf.String())
	}
}

func TestShellCompletionGenerators(t *testing.T) {
	app := &App{Name: "my-app"}

	var bashBuf bytes.Buffer
	if err := GenBashCompletion(app, &bashBuf); err != nil {
		t.Fatalf("GenBashCompletion error: %v", err)
	}
	if !strings.Contains(bashBuf.String(), "_my_app_complete") || !strings.Contains(bashBuf.String(), "complete -o default -F _my_app_complete my-app") {
		t.Errorf("invalid bash completion script: %s", bashBuf.String())
	}

	var zshBuf bytes.Buffer
	if err := GenZshCompletion(app, &zshBuf); err != nil {
		t.Fatalf("GenZshCompletion error: %v", err)
	}
	if !strings.Contains(zshBuf.String(), "#compdef my-app") || !strings.Contains(zshBuf.String(), "_my_app") {
		t.Errorf("invalid zsh completion script: %s", zshBuf.String())
	}

	var fishBuf bytes.Buffer
	if err := GenFishCompletion(app, &fishBuf); err != nil {
		t.Fatalf("GenFishCompletion error: %v", err)
	}
	if !strings.Contains(fishBuf.String(), "__my_app_complete") || !strings.Contains(fishBuf.String(), "complete -c my-app") {
		t.Errorf("invalid fish completion script: %s", fishBuf.String())
	}
}
