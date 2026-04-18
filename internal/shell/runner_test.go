package shell

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestDetectType(t *testing.T) {
	typ := DetectType()
	if runtime.GOOS == "windows" && typ != TypePowerShell {
		t.Fatalf("windows want powershell, got %s", typ)
	}
	if runtime.GOOS != "windows" && typ != TypeBash && typ != TypeSh {
		t.Fatalf("unix want bash or sh, got %s", typ)
	}
}

func TestRunEcho(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder
	ctx := context.Background()
	err := Run(ctx, RunOptions{Command: echoCmd(), WorkDir: dir, Timeout: 10}, func(e Event) {
		if e.Type == EventDeltaStdout || e.Type == EventDeltaStderr {
			out.WriteString(e.Text)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hi_shell") {
		t.Fatalf("output: %q", out.String())
	}
}

func echoCmd() string {
	if runtime.GOOS == "windows" {
		return "Write-Output hi_shell"
	}
	return "echo hi_shell"
}

func TestValidateWorkDirEmpty(t *testing.T) {
	err := Run(context.Background(), RunOptions{Command: "echo x", WorkDir: "", Timeout: 5}, func(Event) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
