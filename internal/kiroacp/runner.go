package kiroacp

import (
	"context"
	"fmt"

	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/media"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/model"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
	"log/slog"
)

func init() {
	agent.Register(agent.TypeKiroACP, func() agent.Runner {
		return &Runner{}
	})
}

// Runner 以 kiro-cli acp（JSON-RPC over stdio）實作 agent.Runner。
//
// 每則訊息 spawn 一次子進程：initialize → session/new|load → session/prompt → kill。
// 不重用 internal/kiro 的 --list-sessions / TTY 行解析。
//
// session/load：kiro-cli 2.16.0+ 跨進程 resume 已通過 PoC（同／異 cwd 皆可）；
// 需 kiro-cli >= 2.16.0（2.12.1 會 timeout，見 GitHub kirodotdev/Kiro#6753）。
type Runner struct{}

func (r *Runner) Name() string { return agent.TypeKiroACP }

func (r *Runner) Run(ctx context.Context, opts agent.RunOptions, cb agent.EventCallback) error {
	cwdRes, err := resolveWorkDir(opts.WorkDir)
	if err != nil {
		cb(agent.Event{Type: agent.EventError, Err: err})
		return err
	}
	cwd := cwdRes.Path
	logWorkDirWarning(cwdRes)

	agentName := ""
	if opts.ExtraArgs != nil {
		agentName = strings.TrimSpace(opts.ExtraArgs["agent"])
	}
	mcpServers, mcpMeta, err := buildACPMcpServers(cwd, agentName)
	if err != nil {
		cb(agent.Event{Type: agent.EventError, Err: err})
		return err
	}
	logMcpLoad(mcpMeta)

	args := buildArgs(opts)
	slog.Info(fmt.Sprintf("[kiroacp] 執行: kiro-cli %s (prompt len=%d)", strings.Join(args, " "), len(opts.Prompt)))
	slog.Info(fmt.Sprintf("[kiroacp] 工作目錄: %s", cwd))

	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	cmd.Dir = cwd
	cmd.SysProcAttr = proc.SysProcAttr()
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return proc.GracefulStop(cmd.Process.Pid, 3*time.Second)
		}
		return nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		slog.Info(fmt.Sprintf("[kiroacp] 啟動失敗: %v", err))
		return err
	}
	slog.Info(fmt.Sprintf("[kiroacp] 子進程 PID=%d", cmd.Process.Pid))

	cl := newClient(cmd, stdout, stdin)
	defer func() {
		cl.close()
		_ = cmd.Wait()
	}()

	// 互動式授權：把 ACP session/request_permission 轉給呼叫端（WS）決定。
	if opts.RequestPermission != nil {
		cl.onPermission = func(p permissionParams) string {
			req := agent.PermissionRequest{
				ToolCallID: p.ToolCall.ToolCallID,
				Title:      p.ToolCall.Title,
			}
			for _, o := range p.Options {
				req.Options = append(req.Options, agent.PermissionOption{
					OptionID: o.OptionID, Name: o.Name, Kind: o.Kind,
				})
			}
			return opts.RequestPermission(ctx, req)
		}
	}

	streamStarted := false
	emitStreamStart := func() {
		if streamStarted {
			return
		}
		streamStarted = true
		cb(agent.Event{Type: agent.EventStreamStart})
	}

	// ACP v1：session/load 必須以 session/update 回放歷史。
	// 本專案 UI／DB 已有完整對話，若把回放當成 EventDelta 會寫進「本則新回覆」，
	// 造成複誦且隨回合雪球變大。load 期間關閉轉發；prompt 階段再開。
	var acceptUpdates atomic.Bool
	acceptUpdates.Store(true)

	cl.onUpdate = func(body sessionUpdateBody) {
		if ctx.Err() != nil || !acceptUpdates.Load() {
			return
		}
		switch body.SessionUpdate {
		case "agent_message_chunk":
			text := extractAgentText(body)
			if text == "" {
				return
			}
			emitStreamStart()
			cb(agent.Event{Type: agent.EventDelta, Text: text})
		case "tool_call", "tool_call_update":
			label := strings.TrimSpace(body.Title)
			if label == "" {
				label = body.Kind
			}
			if body.Status != "" {
				label = label + " (" + body.Status + ")"
			}
			if label == "" {
				label = body.SessionUpdate
			}
			cb(agent.Event{Type: agent.EventActivity, Text: label})
			// 僅在 completed 時落地圖片，避免 started/in_progress 的空 rawOutput 或半成品重覆寫入。
			if body.SessionUpdate != "tool_call_update" || !strings.EqualFold(body.Status, "completed") {
				break
			}
			for _, img := range body.images() {
				url, err := media.SaveBase64Image(img.MediaType, img.Data)
				if err != nil {
					slog.Info(fmt.Sprintf("[kiroacp] 圖片存檔失敗: %v", err))
					continue
				}
				emitStreamStart()
				cb(agent.Event{Type: agent.EventDelta, Text: fmt.Sprintf("\n![screenshot](%s)\n", url)})
			}
		}
	}

	if _, err := cl.call(ctx, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs": map[string]any{"readTextFile": false, "writeTextFile": false},
		},
		"clientInfo": map[string]any{"name": "claude-miniapp", "version": "0.1.0"},
	}); err != nil {
		cb(agent.Event{Type: agent.EventError, Err: err})
		return err
	}
	_ = cl.notify("initialized", map[string]any{})

	sessionID := strings.TrimSpace(opts.SessionID)
	var sess sessionNewResult

	if sessionID == "" {
		raw, err := cl.call(ctx, "session/new", map[string]any{
			"cwd":        cwd,
			"mcpServers": mcpServers,
		})
		if err != nil {
			cb(agent.Event{Type: agent.EventError, Err: err})
			return err
		}
		sess, err = parseSessionResult(raw)
		if err != nil {
			cb(agent.Event{Type: agent.EventError, Err: err})
			return err
		}
		sessionID = sess.SessionID
		if sessionID == "" {
			err := fmt.Errorf("kiroacp: session/new 未回傳 sessionId")
			cb(agent.Event{Type: agent.EventError, Err: err})
			return err
		}
		slog.Info(fmt.Sprintf("[kiroacp] session/new id=%s model=%s", sessionID, modelIDFrom(sess)))
		cb(agent.Event{
			Type:      agent.EventSessionInit,
			SessionID: sessionID,
			Model:     modelSnapshot(sess),
		})
	} else {
		// session/load：2.16.0 實測 ~0.5s；20s 足夠容錯慢磁碟／冷啟動。
		// 關閉 update 轉發，避免歷史回放寫入本則回覆。
		acceptUpdates.Store(false)
		loadCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		raw, err := cl.call(loadCtx, "session/load", map[string]any{
			"sessionId":  sessionID,
			"cwd":        cwd,
			"mcpServers": mcpServers,
		})
		cancel()
		acceptUpdates.Store(true)
		if err != nil {
			err = fmt.Errorf("kiroacp: session/load 失敗（需 kiro-cli >= 2.16.0）: %w", err)
			slog.Info(fmt.Sprintf("[kiroacp] %v", err))
			cb(agent.Event{Type: agent.EventError, Err: err})
			return err
		}
		sess, _ = parseSessionResult(raw)
		slog.Info(fmt.Sprintf("[kiroacp] session/load id=%s model=%s", sessionID, modelIDFrom(sess)))
		if m := modelSnapshot(sess); m != nil {
			cb(agent.Event{Type: agent.EventSessionInit, SessionID: sessionID, Model: m})
		}
	}

	if _, err := cl.call(ctx, "session/prompt", map[string]any{
		"sessionId": sessionID,
		"prompt": []map[string]string{
			{"type": "text", "text": opts.Prompt},
		},
	}); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cb(agent.Event{Type: agent.EventError, Err: err})
		return err
	}

	cb(agent.Event{Type: agent.EventDone, SessionID: sessionID})
	return nil
}

func buildArgs(opts agent.RunOptions) []string {
	args := []string{"acp"}
	// 權限模式對應：
	//   bypassPermissions / yolo → --trust-all-tools（全自動放行）
	//   其他（default/plan/acceptEdits）且有互動授權回呼 → 不加旗標（走 session/request_permission）
	//   無回呼時退回 --trust-all-tools，避免無人回覆導致工具卡住。
	mode := ""
	if opts.ExtraArgs != nil {
		mode = strings.TrimSpace(opts.ExtraArgs[agent.ArgPermissionMode])
	}
	interactive := opts.RequestPermission != nil && mode != "bypassPermissions" && mode != "yolo"
	if !interactive {
		args = append(args, "--trust-all-tools")
	}
	if opts.ExtraArgs != nil {
		if effort := strings.TrimSpace(opts.ExtraArgs["effort"]); effort != "" {
			args = append(args, "--effort", effort)
		}
		if agentProfile := strings.TrimSpace(opts.ExtraArgs["agent"]); agentProfile != "" {
			args = append(args, "--agent", agentProfile)
		}
		if m := strings.TrimSpace(opts.ExtraArgs[agent.ArgModel]); m != "" {
			args = append(args, "--model", m)
		}
	}
	return args
}

func modelIDFrom(s sessionNewResult) string {
	if s.Models == nil {
		return ""
	}
	return strings.TrimSpace(s.Models.CurrentModelID)
}

func modelSnapshot(s sessionNewResult) *agent.ModelSnapshot {
	id := modelIDFrom(s)
	if id == "" {
		return nil
	}
	return model.AgentSnapshot(model.Info{
		Provider:    agent.TypeKiroACP,
		Model:       id,
		DisplayText: model.FormatDisplay(id),
		Source:      model.SourceInitEvent,
		Ok:          true,
	})
}
