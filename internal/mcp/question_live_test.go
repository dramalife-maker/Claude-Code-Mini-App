package mcp

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/tg"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/ws"
)

// Exercise production runner + MCP + WebSocket handlers together, with an isolated database/workdir.
func TestLiveQuestionRoundTrip(t *testing.T) {
	if os.Getenv("CLAUDE_QUESTION_LIVE_TEST") != "1" {
		t.Skip("set CLAUDE_QUESTION_LIVE_TEST=1")
	}
	database, err := db.Open(filepath.Join(t.TempDir(), "questions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	bridge := httptest.NewServer(NewPermHTTPHandler(ws.AskClaudePermission, ws.AskClaudeQuestion))
	defer bridge.Close()
	ws.SetClaudePermMCP(bridge.URL, "isolated-test")
	defer ws.SetClaudePermMCP("", "")
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/sessions/:id/ws", fiberws.New(ws.NewHandler(database, "", ws.ShellOpts{}, nil, tg.NotifyConfig{})))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer app.Shutdown()
	s, err := database.CreateSession("live question", "", t.TempDir(), "default", "claude", []string{"--no-session-persistence", "--strict-mcp-config", "--tools", "AskUserQuestion", "--max-turns", "3"}, "agent")
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/sessions/"+s.ID+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	finished := false
	defer func() {
		if !finished {
			_ = c.WriteJSON(map[string]string{"type": "interrupt"})
		}
	}()
	if err := c.WriteJSON(map[string]string{"type": "input", "data": "Call AskUserQuestion exactly once asking which colors I prefer. Set multiSelect true, with options Red, Blue, Green. Then repeat my chosen colors."}); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(90 * time.Second))
	answered := false
	idle, ended := false, false
	var reply strings.Builder
	for {
		var event struct {
			Type     string `json:"type"`
			Value    string `json:"value"`
			Content  string `json:"content"`
			Question struct {
				RequestID string            `json:"request_id"`
				Questions []json.RawMessage `json:"questions"`
			} `json:"question"`
		}
		if err := c.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		switch event.Type {
		case "permission_request", "error":
			t.Fatalf("unexpected %s: %s", event.Type, event.Content)
		case "question_request":
			if len(event.Question.Questions) != 1 {
				t.Fatalf("expected one question: %+v", event.Question)
			}
			if err := c.WriteJSON(map[string]any{"type": "question_answer", "request_id": event.Question.RequestID, "answers": []map[string]any{{"selected": []string{"Blue", "Green"}}}}); err != nil {
				t.Fatal(err)
			}
			answered = true
		case "delta":
			reply.WriteString(event.Content)
		case "question_error":
			t.Fatal(event.Content)
		case "lingering":
			ended = event.Value == "off"
		case "status":
			if event.Value == "IDLE" && answered {
				idle = true
			}
		case "queue_update":
			if idle && ended {
				if !strings.Contains(reply.String(), "Blue") || !strings.Contains(reply.String(), "Green") {
					t.Fatalf("answer did not reach Claude: %s", reply.String())
				}
				finished = true
				return
			}
		}
	}
}

// Opt in: uses the installed, authenticated Claude CLI and consumes a model turn.
func TestLiveClaudeQuestion(t *testing.T) {
	if os.Getenv("CLAUDE_QUESTION_LIVE_TEST") != "1" {
		t.Skip("set CLAUDE_QUESTION_LIVE_TEST=1")
	}
	for _, mode := range []string{"default", "bypassPermissions", "plan"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(NewPermHTTPHandler(func(context.Context, string, string, json.RawMessage, string) (bool, string) {
				return false, "Only questions are allowed"
			}, func(_ context.Context, _ string, input json.RawMessage, _ string) (json.RawMessage, error) {
				var in map[string]any
				if err := json.Unmarshal(input, &in); err != nil {
					return nil, err
				}
				answers := map[string]string{}
				for _, q := range in["questions"].([]any) {
					answers[q.(map[string]any)["question"].(string)] = "Blue, Green"
				}
				in["answers"] = answers
				calls.Add(1)
				return json.Marshal(in)
			}))
			defer srv.Close()
			dir := t.TempDir()
			cfg, _ := json.Marshal(map[string]any{"mcpServers": map[string]any{"miniapp_perm": map[string]any{"type": "http", "url": srv.URL, "headers": map[string]string{PermSessionHeader: "live-test"}}}})
			path := filepath.Join(dir, "mcp.json")
			if err := os.WriteFile(path, cfg, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "claude", "-p", "--output-format", "json", "--no-session-persistence", "--strict-mcp-config", "--mcp-config", path, "--permission-prompt-tool", "mcp__miniapp_perm__claude_permission", "--permission-mode", mode, "--tools", "AskUserQuestion", "--max-turns", "3")
			cmd.Dir = dir
			cmd.Stdin = strings.NewReader("Call AskUserQuestion exactly once to ask which colors I prefer, with multiSelect true and options Red, Blue, Green. After receiving my answer, repeat exactly my selected colors. Do not answer the question yourself.")
			out, err := cmd.CombinedOutput()
			if err != nil || calls.Load() == 0 || !strings.Contains(string(out), "Blue") || !strings.Contains(string(out), "Green") {
				t.Fatalf("calls=%d err=%v output=%s", calls.Load(), err, out)
			}
			t.Logf("question answered through MCP (%s)", mode)
		})
	}
}
