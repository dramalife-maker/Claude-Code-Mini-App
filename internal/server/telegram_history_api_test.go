package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/tgclient"
)

func TestTelegramHistoryAPI(t *testing.T) {
	e := newGuestEnv(t)
	s, err := e.database.CreateSession("tg", "tg_target:@alice", "", "default", agent.TypeTelegram, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	other, err := e.database.CreateSession("claude", "", "", "default", agent.TypeClaude, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	e.sessions.TelegramHistory = func(_ context.Context, sessionID string, limit int, cursor string) (*tgclient.HistoryPageResult, error) {
		if cursor == "bad" {
			return nil, tgclient.ErrHistoryInvalidCursor
		}
		cur := "10"
		return &tgclient.HistoryPageResult{
			Messages:   []*db.Message{{ID: 11, SessionID: sessionID, Role: "claude", Content: "hi", Status: "done", CreatedAt: "2026-10-19 08:00:00"}},
			NextCursor: &cur,
			HasMore:    true,
			SyncState:  tgclient.SyncStateReady,
		}, nil
	}

	get := func(path string) (int, []byte) {
		t.Helper()
		res, err := http.Get(e.base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, b
	}

	code, body := get("/sessions/" + s.ID + "/messages?pagination=telegram&limit=50")
	if code != 200 {
		t.Fatalf("status %d body %s", code, body)
	}
	var page struct {
		Messages   []map[string]any `json:"messages"`
		NextCursor *string          `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
		SyncState  string           `json:"sync_state"`
	}
	if err := json.Unmarshal(body, &page); err != nil || len(page.Messages) != 1 || !page.HasMore || page.SyncState != "ready" {
		t.Fatalf("envelope = %s err=%v", body, err)
	}

	if code, body := get("/sessions/" + s.ID + "/messages?pagination=telegram&limit=0"); code != 400 {
		t.Fatalf("limit 0 應 400，got %d %s", code, body)
	}
	if code, body := get("/sessions/" + s.ID + "/messages?pagination=telegram&before=bad"); code != 400 {
		t.Fatalf("壞游標應 400，got %d %s", code, body)
	}
	if code, body := get("/sessions/" + other.ID + "/messages?pagination=telegram"); code != 400 {
		t.Fatalf("非 Telegram 應 400，got %d %s", code, body)
	}

	e.sessions.TelegramHistory = func(context.Context, string, int, string) (*tgclient.HistoryPageResult, error) {
		return nil, &tgclient.HistoryRateLimitedError{RetryAfter: 3 * time.Second}
	}
	res, err := http.Get(e.base + "/sessions/" + s.ID + "/messages?pagination=telegram")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 429 || res.Header.Get("Retry-After") != "3" {
		t.Fatalf("FLOOD_WAIT 應 429 Retry-After=3，got %d %q", res.StatusCode, res.Header.Get("Retry-After"))
	}

	e.sessions.TelegramHistory = func(context.Context, string, int, string) (*tgclient.HistoryPageResult, error) {
		return nil, tgclient.ErrHistoryUnavailable
	}
	if code, body := get("/sessions/" + s.ID + "/messages?pagination=telegram"); code != 503 {
		t.Fatalf("離線應 503，got %d %s", code, body)
	}

	for _, mode := range []string{"live", "snapshot"} {
		buf := bytes.NewBufferString(`{"role":"viewer","mode":"` + mode + `","ttl":3600000000000}`)
		req, _ := http.NewRequest(http.MethodPost, e.base+"/sessions/"+s.ID+"/shares", buf)
		req.Header.Set("Content-Type", "application/json")
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatalf("Telegram %s 分享應 400，got %d %s", mode, res.StatusCode, b)
		}
	}

	// 關閉前建立的舊分享：訪客讀訊息與媒體都被擋。
	tok, _ := e.guest(s.ID, db.ShareRoleViewer, db.ShareModeLive, "old")
	for _, path := range []string{"/messages?pagination=telegram", "/tg-media/1"} {
		req, _ := http.NewRequest(http.MethodGet, e.base+"/sessions/"+s.ID+path, nil)
		req.Header.Set("X-Share-Token", tok)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 403 {
			t.Fatalf("舊分享訪客讀 %s 應 403，got %d", path, res.StatusCode)
		}
	}
}
