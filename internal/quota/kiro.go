package quota

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/usage"
)

type KiroFetcher struct{}

func (f *KiroFetcher) Provider() string { return agent.TypeKiro }

func (f *KiroFetcher) Fetch(ctx context.Context) (Snapshot, error) {
	cmd := exec.CommandContext(ctx, "kiro-cli", "chat", "/usage", "--no-interactive")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	cmd.SysProcAttr = proc.SysProcAttr()
	_ = cmd.Run()
	info := usage.FromKiroUsageText(stdout.String())
	return snapshotFromInfo(agent.TypeKiro, info), nil
}
