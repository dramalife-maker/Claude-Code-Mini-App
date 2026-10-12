package quota

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/usage"
)

type AntigravityFetcher struct{}

func (f *AntigravityFetcher) Provider() string { return agent.TypeAntigravity }

func (f *AntigravityFetcher) Fetch(ctx context.Context) (Snapshot, error) {
	text, err := runAgyUsageText(ctx)
	if err != nil {
		return Snapshot{
			Provider:    agent.TypeAntigravity,
			DisplayText: "—",
			UpdatedAt:   time.Now(),
		}, err
	}
	info := usage.FromAgyUsageText(text)
	snap := Snapshot{
		Provider:    agent.TypeAntigravity,
		DisplayText: FormatDisplay(agent.TypeAntigravity, info),
		UpdatedAt:   time.Now(),
	}
	if snap.DisplayText == "—" {
		return snap, fmt.Errorf("antigravity quota: no session/week data")
	}
	return snap, nil
}

func runAgyUsageText(ctx context.Context) (string, error) {
	bin := "agy"
	if b := strings.TrimSpace(os.Getenv("CC_AGY_BIN")); b != "" {
		bin = b
	}
	cmd := exec.CommandContext(ctx, bin, "-p", "/usage")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = proc.SysProcAttr()
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run agy usage: %w (stderr: %s)", err, stderr.String())
	}
	return stdout.String(), nil
}
