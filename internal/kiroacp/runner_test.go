package kiroacp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
)

func TestExtractAgentText(t *testing.T) {
	body := sessionUpdateBody{
		SessionUpdate: "agent_message_chunk",
		Content:       json.RawMessage(`{"type":"text","text":"hello"}`),
	}
	if got := extractAgentText(body); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if extractAgentText(sessionUpdateBody{SessionUpdate: "tool_call"}) != "" {
		t.Fatal("tool_call should not yield text")
	}
}

func TestParseSessionResult(t *testing.T) {
	raw := json.RawMessage(`{"sessionId":"abc","models":{"currentModelId":"claude-sonnet-5"}}`)
	s, err := parseSessionResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.SessionID != "abc" || modelIDFrom(s) != "claude-sonnet-5" {
		t.Fatalf("%+v", s)
	}
	snap := modelSnapshot(s)
	if snap == nil || snap.Model != "claude-sonnet-5" {
		t.Fatalf("snapshot=%+v", snap)
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs(agent.RunOptions{
		ExtraArgs: map[string]string{agent.ArgModel: "claude-sonnet-5", "effort": "high"},
	})
	joined := ""
	for _, a := range args {
		joined += a + " "
	}
	if args[0] != "acp" {
		t.Fatalf("want acp first, got %v", args)
	}
	if !containsPair(args, "--model", "claude-sonnet-5") || !containsPair(args, "--effort", "high") {
		t.Fatalf("args=%v", args)
	}
}

func containsPair(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func contains(args []string, v string) bool {
	for _, a := range args {
		if a == v {
			return true
		}
	}
	return false
}

func TestBuildArgs_PermissionMode(t *testing.T) {
	responder := func(_ context.Context, _ agent.PermissionRequest) string { return "" }

	// 有回呼 + default → 互動模式，不加 --trust-all-tools
	if a := buildArgs(agent.RunOptions{RequestPermission: responder}); contains(a, "--trust-all-tools") {
		t.Fatalf("default+responder 應為互動模式（無 trust-all），got %v", a)
	}
	// 有回呼 + bypassPermissions → --trust-all-tools
	if a := buildArgs(agent.RunOptions{RequestPermission: responder, ExtraArgs: map[string]string{agent.ArgPermissionMode: "bypassPermissions"}}); !contains(a, "--trust-all-tools") {
		t.Fatalf("bypass 應加 trust-all，got %v", a)
	}
	// 無回呼 → 退回 --trust-all-tools（避免卡住）
	if a := buildArgs(agent.RunOptions{}); !contains(a, "--trust-all-tools") {
		t.Fatalf("無回呼應退回 trust-all，got %v", a)
	}
}

func TestClientRequestPermission(t *testing.T) {
	// 模擬 ACP server：srv 寫入 → client 讀（stdout）；client 寫回 → 由 cliOut 讀（stdin）。
	srvR, srvW := io.Pipe()
	cliR, cliW := io.Pipe()
	c := newClient(nil, srvR, cliW)
	defer c.close()

	var gotTitle string
	c.onPermission = func(p permissionParams) string {
		gotTitle = p.ToolCall.Title
		// 選 allow_once
		for _, o := range p.Options {
			if o.Kind == "allow_once" {
				return o.OptionID
			}
		}
		return ""
	}

	// server 發出 request_permission
	req := `{"jsonrpc":"2.0","id":7,"method":"session/request_permission","params":{"sessionId":"s1","toolCall":{"toolCallId":"tc1","title":"Creating hello.txt"},"options":[{"optionId":"allow_once","name":"Yes","kind":"allow_once"},{"optionId":"reject_once","name":"No","kind":"reject_once"}]}}` + "\n"
	go func() { _, _ = srvW.Write([]byte(req)) }()

	// 讀 client 回覆
	done := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(cliR)
		if sc.Scan() {
			done <- sc.Text()
		} else {
			done <- ""
		}
	}()

	select {
	case line := <-done:
		if gotTitle != "Creating hello.txt" {
			t.Fatalf("onPermission title=%q", gotTitle)
		}
		var resp struct {
			ID     int64 `json:"id"`
			Result struct {
				Outcome struct {
					Outcome  string `json:"outcome"`
					OptionID string `json:"optionId"`
				} `json:"outcome"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("回覆解析失敗: %v | %s", err, line)
		}
		if resp.ID != 7 {
			t.Fatalf("id=%d want 7", resp.ID)
		}
		if resp.Result.Outcome.Outcome != "selected" || resp.Result.Outcome.OptionID != "allow_once" {
			t.Fatalf("outcome=%+v", resp.Result.Outcome)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout：未收到 client 對 request_permission 的回覆")
	}
}
