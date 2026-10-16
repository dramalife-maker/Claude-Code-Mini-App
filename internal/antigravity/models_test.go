package antigravity

import (
	"context"
	"os"
	"slices"
	"testing"
)

func TestFetchModelOptions(t *testing.T) {
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CC_AGY_BIN", bin)
	for _, tc := range []struct {
		name, output string
		want         []ModelEntry
	}{
		{"models", "Fetching available models...\r\ngemini-3.8-flash-high\tGemini 3.8 Flash (High)\r\nclaude-sonnet-4-6\tClaude Sonnet 4.6 (Thinking)\n", []ModelEntry{
			{ModelID: "gemini-3.8-flash-high", Label: "Gemini 3.8 Flash (High)"},
			{ModelID: "claude-sonnet-4-6", Label: "Claude Sonnet 4.6 (Thinking)"},
		}},
		{"empty", "", nil},
		{"progress only", "Fetching available models...\n", nil},
		{"missing ID", " \tMissing ID\n", nil},
		{"partial list", "valid\tValid\nbroken\t\n", nil},
		{"invalid ID", "two words\tLabel\n", nil},
		{"failure", "", nil},
		{"cancelled", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FAKE_AGY_PRINT", "models")
			t.Setenv("FAKE_AGY_MODELS", tc.output)
			if tc.name == "failure" {
				t.Setenv("FAKE_AGY_PRINT", "failure")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.name == "cancelled" {
				cancel()
			}
			got, err := FetchModelOptions(ctx)
			if (err != nil) != (tc.want == nil) || !slices.Equal(got, tc.want) {
				t.Fatalf("got %+v, err=%v; want %+v", got, err, tc.want)
			}
		})
	}
}
