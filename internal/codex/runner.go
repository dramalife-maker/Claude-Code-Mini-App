package codex

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"

	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
	"log/slog"
)

func init() {
	agent.Register(agent.TypeCodex, func() agent.Runner {
		return &Runner{}
	})
}

// Runner 是 codex exec --json 的 agent.Runner 實作。
// 參考：docs/spec/codex-cli.md
type Runner struct{}

func (r *Runner) Name() string { return agent.TypeCodex }

func (r *Runner) Run(ctx context.Context, opts agent.RunOptions, cb agent.EventCallback) error {
	start := time.Now()
	codexBin, err := ResolveBin()
	if err != nil {
		cb(agent.Event{Type: agent.EventError, Err: err})
		return err
	}
	if !HasAuthConfig() {
		slog.Info(fmt.Sprintf("[codex] 警告：未找到 CODEX_API_KEY 或 ~/.codex/auth.json，仍嘗試執行 exec"))
	}

	args := buildArgs(opts)
	slog.Info(fmt.Sprintf("[codex] 執行指令: codex %s (prompt len=%d)", strings.Join(redactArgs(args), " "), len(opts.Prompt)))
	if opts.WorkDir != "" {
		slog.Info(fmt.Sprintf("[codex] 工作目錄: %s", opts.WorkDir))
	}

	cmd := exec.CommandContext(ctx, codexBin, args...)
	if opts.SessionID == "" && opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	} else if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	cmd.Env = codexEnv()
	cmd.SysProcAttr = proc.SysProcAttr()
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return proc.KillTree(cmd.Process.Pid)
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

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		slog.Error(fmt.Sprintf("[codex] 子進程啟動失敗: %v", err))
		return err
	}
	_ = stdin.Close()
	slog.Info(fmt.Sprintf("[codex] 子進程已啟動，PID=%d", cmd.Process.Pid))
	if opts.OnStart != nil {
		opts.OnStart(cmd.Process.Pid)
	}

	var st dispatchState
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineCount := 0

	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		lineCount++
		slog.Debug(fmt.Sprintf("[codex] 收到第 %d 行: %s", lineCount, truncate(string(raw), 200)))
		ev, parseErr := ParseEvent(raw)
		if parseErr != nil {
			slog.Warn(fmt.Sprintf("[codex] 解析失敗: %v", parseErr))
			continue
		}
		r.dispatch(ev, cb, &st)
	}
	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("[codex] scanner 錯誤: %v", err))
	}

	waitErr := cmd.Wait()
	if stderr := stderrBuf.String(); stderr != "" {
		slog.Debug(fmt.Sprintf("[codex] stderr:\n%s", truncate(stderr, 500)))
	}

	sessionID := opts.SessionID
	if st.sessionID != "" {
		sessionID = st.sessionID
	}
	duration := time.Since(start)

	if waitErr != nil && ctx.Err() == nil {
		if !st.sawTurnCompleted {
			cb(agent.Event{Type: agent.EventError, Err: classifyRunnerError(stderrBuf.String(), waitErr)})
		}
		slog.Error("[codex] run 結束", "session_id", sessionID, "duration", duration, "lines", lineCount, "ok", false, "err", waitErr)
		return waitErr
	}

	if !st.sawTurnCompleted && ctx.Err() == nil {
		cb(agent.Event{Type: agent.EventError, Err: errors.New("codex 串流中斷：未收到 turn.completed")})
	}
	slog.Info("[codex] run 結束", "session_id", sessionID, "duration", duration, "lines", lineCount, "ok", true)
	cb(agent.Event{Type: agent.EventDone, SessionID: sessionID})
	return waitErr
}

type dispatchState struct {
	sawTurnCompleted bool
	sessionID        string
}

func (r *Runner) dispatch(ev *StreamEvent, cb agent.EventCallback, st *dispatchState) {
	switch ev.Type {
	case "thread.started":
		if ev.ThreadID != "" {
			st.sessionID = ev.ThreadID
			cb(agent.Event{Type: agent.EventSessionInit, SessionID: ev.ThreadID})
		}
	case "item.started":
		if ev.Item != nil {
			if label := ActivityLabel(ev.Item.Type); label != "" {
				cb(agent.Event{Type: agent.EventActivity, Text: label})
			}
		}
	case "item.completed":
		if text := AgentMessageText(ev.Item); text != "" {
			cb(agent.Event{Type: agent.EventDelta, Text: text})
		}
	case "turn.completed":
		st.sawTurnCompleted = true
	case "turn.failed", "error":
		cb(agent.Event{Type: agent.EventError, Err: errors.New(ErrorMessage(ev))})
	}
}

func buildArgs(opts agent.RunOptions) []string {
	if opts.SessionID != "" {
		args := []string{
			"exec", "resume", opts.SessionID,
			"--json", "--skip-git-repo-check",
			"--yolo",
		}
		if opts.ExtraArgs != nil {
			if m := strings.TrimSpace(opts.ExtraArgs[agent.ArgModel]); m != "" {
				args = append(args, "-m", m)
			}
			if effort := strings.TrimSpace(opts.ExtraArgs[agent.ArgEffort]); effort != "" {
				args = append(args, "-c", "model_reasoning_effort="+effort)
			}
		}
		args = append(args, opts.Prompt)
		return args
	}

	args := []string{
		"exec", "--json", "--skip-git-repo-check",
		"--yolo",
		"-C", opts.WorkDir,
	}
	if opts.ExtraArgs != nil {
		if m := strings.TrimSpace(opts.ExtraArgs[agent.ArgModel]); m != "" {
			args = append(args, "-m", m)
		}
		if effort := strings.TrimSpace(opts.ExtraArgs[agent.ArgEffort]); effort != "" {
			args = append(args, "-c", "model_reasoning_effort="+effort)
		}
	}
	args = append(args, opts.Prompt)
	return args
}

func codexEnv() []string {
	out := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "TERM=") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func classifyRunnerError(stderr string, waitErr error) error {
	s := strings.TrimSpace(stderr)
	if strings.Contains(strings.ToLower(s), "unauthorized") ||
		strings.Contains(strings.ToLower(s), "not logged in") ||
		strings.Contains(strings.ToLower(s), "authentication") {
		return fmt.Errorf("codex 認證失敗：請在本機執行 codex login，或設定 CODEX_API_KEY / CODEX_BIN。原始訊息：%s", s)
	}
	if s != "" {
		return fmt.Errorf("codex failed: %s", s)
	}
	return waitErr
}

func redactArgs(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
