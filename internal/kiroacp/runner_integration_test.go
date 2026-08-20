package kiroacp

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
)

// TestInteractivePermission_E2E 對真實 kiro-cli acp 驗證互動式授權端到端。
// 需登入的 kiro-cli（>= 2.16）。預設略過；設 KIROACP_E2E=1 才執行。
//
//	KIROACP_E2E=1 go test ./internal/kiroacp/ -run E2E -v
func TestInteractivePermission_E2E(t *testing.T) {
	if os.Getenv("KIROACP_E2E") != "1" {
		t.Skip("設 KIROACP_E2E=1 才跑真實 kiro-cli 整合測試")
	}
	dir := t.TempDir()
	var asked atomic.Bool

	r := &Runner{}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	opts := agent.RunOptions{
		Prompt:  "請用工具在目前工作目錄建立一個名為 hello.txt 的檔案，內容為 HELLO。完成後回覆 done。",
		WorkDir: dir,
		// 不設 bypassPermissions → 互動模式
		ExtraArgs: map[string]string{agent.ArgPermissionMode: "default"},
		RequestPermission: func(_ context.Context, req agent.PermissionRequest) string {
			asked.Store(true)
			t.Logf("收到授權請求 title=%q options=%d", req.Title, len(req.Options))
			for _, o := range req.Options {
				if o.Kind == "allow_once" {
					return o.OptionID
				}
			}
			if len(req.Options) > 0 {
				return req.Options[0].OptionID
			}
			return ""
		},
	}

	err := r.Run(ctx, opts, func(e agent.Event) {
		if e.Type == agent.EventError && e.Err != nil {
			t.Logf("event error: %v", e.Err)
		}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !asked.Load() {
		t.Fatal("未觸發 RequestPermission（互動授權未生效）")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "hello.txt")); statErr != nil {
		t.Fatalf("授權後檔案未建立: %v", statErr)
	}
}
