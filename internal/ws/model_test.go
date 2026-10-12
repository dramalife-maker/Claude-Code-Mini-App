package ws

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fasthttp/websocket"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

func TestSessionModelPayload_AntigravityDefault(t *testing.T) {
	sess := &db.Session{AgentType: agent.TypeAntigravity}
	p := sessionModelPayload(sess)
	if p == nil || p.DisplayText != "gemini-3.8-flash-high" {
		t.Fatalf("antigravity should expose default model payload, got %+v", p)
	}
}

func TestSessionModelPayload_AntigravitySelection(t *testing.T) {
	for _, typ := range []string{agent.TypeAntigravity, agent.TypeGemini} {
		t.Run(typ, func(t *testing.T) {
			sess := &db.Session{AgentType: typ, Model: "selected-model", ActiveModel: "stale-model", CliExtraArgs: []string{"--model", "original-model"}}
			for _, tc := range []struct{ selected, want string }{
				{"selected-model", "selected-model"},
				{"", "original-model"},
			} {
				sess.Model = tc.selected
				if p := sessionModelPayload(sess); p == nil || p.DisplayText != tc.want || p.Source != "cli_flag" {
					t.Fatalf("selected=%q: got %+v; want %s", tc.selected, p, tc.want)
				}
				if info := resolveStreamless(sess, typ); info.Model != tc.want {
					t.Fatalf("run model: %+v", info)
				}
			}
			sess.CliExtraArgs = nil
			if p := sessionModelPayload(sess); p == nil || p.DisplayText != "gemini-3.8-flash-high" {
				t.Fatalf("cleared selection should discard stale model: %+v", p)
			}
			if !isStreamless(typ) {
				t.Fatal("model changes must update streamless badge")
			}
		})
	}
}

func TestAntigravityModelUpdatesAndReconnect(t *testing.T) {
	original, err := agent.NewRunner(agent.TypeAntigravity)
	if err != nil {
		t.Fatal(err)
	}
	agent.Register(agent.TypeAntigravity, func() agent.Runner { return fakeRunner{} })
	t.Cleanup(func() { agent.Register(agent.TypeAntigravity, func() agent.Runner { return original }) })
	database, wsURL := startTestServer(t)
	sess, err := database.CreateSession("model", "", t.TempDir(), "default", agent.TypeAntigravity, []string{"--model", "original-model"}, "agent")
	if err != nil {
		t.Fatal(err)
	}
	readModel := func(conn *websocket.Conn, typ, want string) {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			var msg serverMsg
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatalf("waiting for %s/%s: %v", typ, want, err)
			}
			if msg.Type == typ && msg.Model != nil {
				if msg.Model.DisplayText != want {
					t.Fatalf("%s model: got %s; want %s", typ, msg.Model.DisplayText, want)
				}
				return
			}
		}
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(sess.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	readModel(conn, "sync", "original-model")
	if err := conn.WriteJSON(clientMsg{Type: "set_model", Model: "selected-model"}); err != nil {
		t.Fatal(err)
	}
	readModel(conn, "model_update", "selected-model")
	if err := conn.WriteJSON(clientMsg{Type: "input", Data: "go"}); err != nil {
		t.Fatal(err)
	}
	readModel(conn, "model_update", "selected-model")
	waitFor(t, "model run complete", func() bool { return lastReply(database, sess.ID) == "ok:go" && !taskIsActive(sess.ID) })
	conn.Close()
	reconnected, _, err := websocket.DefaultDialer.Dial(wsURL(sess.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reconnected.Close()
	readModel(reconnected, "sync", "selected-model")
	if err := reconnected.WriteJSON(clientMsg{Type: "set_model", Model: ""}); err != nil {
		t.Fatal(err)
	}
	readModel(reconnected, "model_update", "original-model")
}

func TestSessionModelPayload_CursorDefault(t *testing.T) {
	sess := &db.Session{AgentType: agent.TypeCursor, CliExtraArgs: []string{}}
	p := sessionModelPayload(sess)
	if p == nil || p.DisplayText != "auto" {
		t.Fatalf("got %+v", p)
	}
}

func TestPersistModelUpdate(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	sess, err := database.CreateSession("m", "", "", "default", agent.TypeClaude, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}

	p := persistModelUpdate(database, sess.ID, &agent.ModelSnapshot{
		Model:       "claude-sonnet-5",
		DisplayText: "claude-sonnet-5",
		Source:      "init_event",
	})
	if p.DisplayText != "claude-sonnet-5" || p.Source != "init_event" {
		t.Fatalf("got %+v", p)
	}

	updated, err := database.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ActiveModel != "claude-sonnet-5" || updated.ActiveModelSource != "init_event" {
		t.Fatalf("db: %+v", updated)
	}
}

// 回歸：下拉選的 sess.Model 必須壓過舊版寫壞的 ActiveModel 與 config.toml 預設。
func TestSessionModelPayload_StreamlessPrefersSelectedModel(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte("model = \"gpt-6-astra\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)

	sess := &db.Session{AgentType: agent.TypeCodex, Model: "gpt-6.1-sol", ActiveModel: "gpt-6-astra", ActiveModelSource: "global_config"}
	if p := sessionModelPayload(sess); p == nil || p.DisplayText != "gpt-6.1-sol" {
		t.Fatalf("selected model: got %+v", p)
	}
	if info := resolveStreamless(&db.Session{AgentType: agent.TypeCodex, Model: "gpt-6.1-sol", CliExtraArgs: []string{"--model", "x"}}, agent.TypeCodex); info.Model != "gpt-6.1-sol" {
		t.Fatalf("sess.Model should beat cli_extra_args: got %+v", info)
	}
	// 未選擇時退回全域設定。
	if info := resolveStreamless(&db.Session{AgentType: agent.TypeCodex}, agent.TypeCodex); info.Model != "gpt-6-astra" {
		t.Fatalf("fallback: got %+v", info)
	}
	// claude 不受影響：stream 回填的 ActiveModel 照舊優先。
	c := &db.Session{AgentType: agent.TypeClaude, Model: "opus", ActiveModel: "claude-sonnet-5", ActiveModelSource: "init_event"}
	if p := sessionModelPayload(c); p == nil || p.DisplayText != "claude-sonnet-5" {
		t.Fatalf("claude: got %+v", p)
	}
}
