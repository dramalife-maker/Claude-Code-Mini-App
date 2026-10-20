package clicheck

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/antigravity"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/codex"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
)

const cursorNote = "官方命令是 agent，但本專案呼叫 cursor-agent；只有 agent 不代表此整合可用"

// Tool 是一個代理 CLI 的偵測結果。找不到只把 Found 設為 false，不當成錯誤。
type Tool struct {
	AgentType  string `json:"agent_type"`
	Label      string `json:"label"`
	Command    string `json:"command"`
	Found      bool   `json:"found"`
	Path       string `json:"path"`
	Version    string `json:"version"`
	InstallURL string `json:"install_url"`
	Note       string `json:"note"`
}

// Checker 的空欄位用真實查找與 --version。測試可注入，避免執行本機的 CLI。
type Checker struct {
	Timeout      time.Duration
	LookPath     func(name string) (string, error)
	ResolveCodex func() (string, error)
	AgyBin       func() string
	Run          func(ctx context.Context, path string) (stdout, stderr string, err error)
}

// Check 依固定順序回報五個工具。找不到或版本探測失敗都不讓整體失敗。
func Check(ctx context.Context) []Tool {
	return Checker{}.Check(ctx)
}

func (c Checker) Check(ctx context.Context) []Tool {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.Timeout <= 0 {
		c.Timeout = 3 * time.Second
	}
	if c.LookPath == nil {
		c.LookPath = exec.LookPath
	}
	if c.ResolveCodex == nil {
		c.ResolveCodex = codex.ResolveBin
	}
	if c.AgyBin == nil {
		c.AgyBin = antigravity.Binary
	}
	if c.Run == nil {
		c.Run = runVersion
	}
	specs := []Tool{
		{AgentType: agent.TypeClaude, Label: "Claude", Command: "claude", InstallURL: "https://code.claude.com/docs/en/quickstart"},
		{AgentType: agent.TypeCodex, Label: "Codex", Command: "codex", InstallURL: "https://developers.openai.com/codex/cli/"},
		{AgentType: agent.TypeCursor, Label: "Cursor", Command: "cursor-agent", InstallURL: "https://cursor.com/docs/cli/installation", Note: cursorNote},
		{AgentType: agent.TypeKiroACP, Label: "Kiro", Command: "kiro-cli", InstallURL: "https://kiro.dev/docs/cli/"},
		{AgentType: agent.TypeAntigravity, Label: "Antigravity", Command: "agy", InstallURL: "https://antigravity.google/docs/overview?tab=cli"},
	}
	for i, spec := range specs {
		switch spec.Command {
		case "codex":
			if p, err := c.ResolveCodex(); err == nil && p != "" {
				specs[i].Found = true
				specs[i].Path = p
			}
		case "agy":
			if p, ok := resolveAgy(c.AgyBin(), c.LookPath); ok {
				specs[i].Found = true
				specs[i].Path = p
			}
		default:
			if p, err := c.LookPath(spec.Command); err == nil && p != "" {
				specs[i].Found = true
				specs[i].Path = p
			}
		}
	}
	// 已找到的才探測版本。各項並行，單一逾時不拖住其他項，也不把 Found 改回 false。
	var wg sync.WaitGroup
	for i := range specs {
		if !specs[i].Found {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(ctx, c.Timeout)
			defer cancel()
			stdout, stderr, err := c.Run(pctx, specs[i].Path)
			if err != nil {
				return
			}
			specs[i].Version = versionText(stdout, stderr)
		}(i)
	}
	wg.Wait()
	return specs
}

func runVersion(ctx context.Context, path string) (string, string, error) {
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.SysProcAttr = proc.SysProcAttr()
	// 逾時只殺父行程。孫行程若繼承管線，Wait 會等到管線關閉。
	// WaitDelay 讓 Wait 在父行程結束後最多再等一會兒就放棄，避免整次檢查被拖住。
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func versionText(stdout, stderr string) string {
	if line := firstLine(stdout); line != "" {
		return clip(line)
	}
	return clip(firstLine(stderr))
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line != "" {
			return line
		}
	}
	return ""
}

func clip(s string) string {
	r := []rune(s)
	if len(r) > 100 {
		return string(r[:100])
	}
	return s
}

// resolveAgy：CC_AGY_BIN 若是路徑就看檔案在不在；若只是名稱再走 LookPath。
func resolveAgy(bin string, look func(string) (string, error)) (string, bool) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return "", false
	}
	if filepath.IsAbs(bin) || strings.ContainsAny(bin, `/\`) {
		st, err := os.Stat(bin)
		if err != nil || st.IsDir() {
			return "", false
		}
		return bin, true
	}
	p, err := look(bin)
	if err != nil || p == "" {
		return "", false
	}
	return p, true
}
