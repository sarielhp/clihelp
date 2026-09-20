package clihelp

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParamComplete_EdgeCases(t *testing.T) {
	app := &App{
		Name: "testcli",
		Commands: []Command{
			{
				Name: "none",
				// Parameters is nil/empty
				Run: func(*Context) error { return nil },
			},
			{
				Name: "mixed",
				Parameters: []Param{
					{Name: "<first>"}, // Complete is nil
					{
						Name: "<second>",
						Complete: func(toComplete string) []string {
							all := []string{"apple\tRed fruit", "apricot\tOrange fruit", "banana\tYellow fruit"}
							var matched []string
							for _, item := range all {
								if strings.HasPrefix(item, toComplete) {
									matched = append(matched, item)
								}
							}
							return matched
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
			{
				Name: "variadic-nil",
				Parameters: []Param{
					{
						Name: "<first>",
						Complete: func(_ string) []string {
							return []string{"init"}
						},
					},
					{
						Name:     "<rest...>",
						Variadic: true,
						// Complete is nil
					},
				},
				Run: func(*Context) error { return nil },
			},
			{
				Name: "variadic-active",
				Parameters: []Param{
					{
						Name: "<fixed>",
						Complete: func(_ string) []string {
							return []string{"base"}
						},
					},
					{
						Name:     "<extra...>",
						Variadic: true,
						Complete: func(toComplete string) []string {
							return []string{"item-" + toComplete}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	tests := []struct {
		name       string
		args       []string
		wantSubstr string
		dontWant   string
	}{
		{
			name:       "empty parameters produces nothing",
			args:       []string{"__complete", "none", ""},
			wantSubstr: "",
		},
		{
			name:       "slot 0 with nil Complete produces nothing",
			args:       []string{"__complete", "mixed", ""},
			wantSubstr: "",
		},
		{
			name:       "slot 1 with Complete produces candidates",
			args:       []string{"__complete", "mixed", "first-arg", "ap"},
			wantSubstr: "apple\tRed fruit",
			dontWant:   "banana",
		},
		{
			name:       "slot 2 out of bounds when non-variadic produces nothing",
			args:       []string{"__complete", "mixed", "first-arg", "second-arg", "extra"},
			wantSubstr: "",
		},
		{
			name:       "variadic with nil Complete produces nothing on tail",
			args:       []string{"__complete", "variadic-nil", "init", "tail1", "tail2"},
			wantSubstr: "",
		},
		{
			name:       "variadic active invokes callback across all tail positions",
			args:       []string{"__complete", "variadic-active", "fixed-val", "tailA"},
			wantSubstr: "item-tailA",
		},
		{
			name:       "variadic active invokes callback on further tail positions",
			args:       []string{"__complete", "variadic-active", "fixed-val", "tailA", "tailB"},
			wantSubstr: "item-tailB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			app.Stdout = &out

			if err := app.ExecuteContext(context.Background(), tt.args); err != nil {
				t.Fatalf("ExecuteContext failed: %v", err)
			}

			got := out.String()
			if tt.wantSubstr != "" && !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("got %q, want substring %q", got, tt.wantSubstr)
			}
			if tt.dontWant != "" && strings.Contains(got, tt.dontWant) {
				t.Errorf("got %q, unexpected substring %q", got, tt.dontWant)
			}
			if tt.wantSubstr == "" && len(got) > 0 {
				t.Errorf("expected empty output, got %q", got)
			}
		})
	}
}

func TestParamComplete_Unicode(t *testing.T) {
	app := &App{
		Name: "unicodecli",
		Commands: []Command{
			{
				Name: "city",
				Parameters: []Param{
					{
						Name: "<name>",
						Complete: func(toComplete string) []string {
							cities := []string{
								"東京\tTokyo capital",
								"京都\tKyoto historic capital",
								"大阪\tOsaka commerce center",
							}
							var matches []string
							for _, c := range cities {
								if strings.HasPrefix(c, toComplete) {
									matches = append(matches, c)
								}
							}
							return matches
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	var out bytes.Buffer
	app.Stdout = &out

	// Completing prefix "東"
	if err := app.ExecuteContext(context.Background(), []string{"__complete", "city", "東"}); err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "東京\tTokyo capital") {
		t.Errorf("expected Japanese unicode candidate, got: %q", got)
	}
	if strings.Contains(got, "京都") {
		t.Errorf("unexpected candidate 京都 for prefix 東, got: %q", got)
	}
}

func TestParamComplete_ContextCancellation(t *testing.T) {
	app := &App{
		Name: "cancelcli",
		Commands: []Command{
			{
				Name: "run",
				Parameters: []Param{
					{
						Name: "<param>",
						Complete: func(string) []string {
							return []string{"res1", "res2"}
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var out bytes.Buffer
	app.Stdout = &out

	err := app.ExecuteContext(ctx, []string{"__complete", "run", "r"})
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if out.Len() > 0 {
		t.Fatalf("expected no output on cancelled context, got: %q", out.String())
	}
}

func TestParamComplete_FlagInteractions(t *testing.T) {
	var targetFormat string
	var targetVerbose bool

	app := &App{
		Name: "flagcli",
		Commands: []Command{
			{
				Name: "process",
				Options: []Option{
					String(&targetFormat, "--format <fmt>", "text", "Output format"),
					Bool(&targetVerbose, "-v, --verbose", false, "Verbose output"),
				},
				Parameters: []Param{
					{
						Name: "<target>",
						Complete: func(toComplete string) []string {
							items := []string{"target1\tFirst target", "target2\tSecond target"}
							var res []string
							for _, item := range items {
								if strings.HasPrefix(item, toComplete) {
									res = append(res, item)
								}
							}
							return res
						},
					},
				},
				Run: func(*Context) error { return nil },
			},
		},
	}

	tests := []struct {
		name       string
		args       []string
		wantSubstr string
	}{
		{
			name:       "flag with value consumes next token, then completes positional slot 0",
			args:       []string{"__complete", "process", "--format", "json", "tar"},
			wantSubstr: "target1\tFirst target",
		},
		{
			name:       "toggle flag does not consume next token, completes positional slot 0",
			args:       []string{"__complete", "process", "-v", "tar"},
			wantSubstr: "target1\tFirst target",
		},
		{
			name:       "dash-dash boundary terminates flag scanning and completes positional slot 0",
			args:       []string{"__complete", "process", "--", "tar"},
			wantSubstr: "target1\tFirst target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			app.Stdout = &out

			if err := app.ExecuteContext(context.Background(), tt.args); err != nil {
				t.Fatalf("ExecuteContext failed: %v", err)
			}

			got := out.String()
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("got %q, want substring %q", got, tt.wantSubstr)
			}
		})
	}
}
