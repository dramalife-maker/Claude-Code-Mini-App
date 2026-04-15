package claude

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"os/exec"
	"strings"
)

type RunOptions struct {
	Prompt         string
	SessionID      string // 空字串表示新 session
	WorkDir        string
	PermissionMode string   // default | acceptEdits | bypassPermissions
	AllowedTools   []string // 允許的工具清單
}

// EventCallback 每收到一個解析好的事件就呼叫一次
type EventCallback func(e *StreamEvent)

// Run 啟動 claude -p 子進程，逐行解析 stream-json，透過 callback 回傳事件
// 子進程結束後函式返回
func Run(ctx context.Context, opts RunOptions, cb EventCallback) error {
	args := []string{
		"-p", opts.Prompt,
		"--output-format", "stream-json",
		"--verbose",
	}

	if opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
	}

	mode := opts.PermissionMode
	if mode == "" {
		mode = "default"
	}
	args = append(args, "--permission-mode", mode)

	for _, tool := range opts.AllowedTools {
		args = append(args, "--allowedTools", tool)
	}

	log.Printf("[claude] 執行指令: claude %s", strings.Join(args, " "))
	if opts.WorkDir != "" {
		log.Printf("[claude] 工作目錄: %s", opts.WorkDir)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[claude] 取得 stdout pipe 失敗: %v", err)
		return err
	}

	// 同時捕捉 stderr
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		log.Printf("[claude] 子進程啟動失敗: %v", err)
		return err
	}
	log.Printf("[claude] 子進程已啟動，PID=%d", cmd.Process.Pid)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		lineCount++
		log.Printf("[claude] 收到第 %d 行 (len=%d): %s", lineCount, len(line), truncate(string(line), 200))

		e, err := ParseEvent(line)
		if err != nil {
			log.Printf("[claude] 解析失敗: %v | 原始內容: %s", err, truncate(string(line), 200))
			continue
		}
		log.Printf("[claude] 事件 type=%s subtype=%s", e.Type, e.Subtype)
		if e.Event != nil {
			log.Printf("[claude]   └─ API event type=%s", e.Event.Type)
		}
		if e.SessionID != "" {
			log.Printf("[claude]   └─ session_id=%s", e.SessionID)
		}
		if e.IsError {
			log.Printf("[claude]   └─ IS_ERROR result=%s", e.Result)
		}
		if len(e.PermissionDenials) > 0 {
			log.Printf("[claude]   └─ permission_denials=%d 項", len(e.PermissionDenials))
		}
		cb(e)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[claude] scanner 錯誤: %v", err)
	}

	waitErr := cmd.Wait()
	stderr := stderrBuf.String()
	if stderr != "" {
		log.Printf("[claude] stderr 輸出:\n%s", stderr)
	}
	if waitErr != nil {
		log.Printf("[claude] 子進程結束，exit error: %v", waitErr)
	} else {
		log.Printf("[claude] 子進程正常結束，共處理 %d 行", lineCount)
	}
	return waitErr
}

// truncate 截斷過長字串，避免 log 爆炸
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}
