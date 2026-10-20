package ws

import (
	"context"
	"testing"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

// fakeTG 讓 ws 測試二進位裡的 telegram runner 不連真的 Telegram。
type fakeTG struct{}

func (fakeTG) Name() string { return agent.TypeTelegram }

func (fakeTG) Run(_ context.Context, _ agent.RunOptions, cb agent.EventCallback) error {
	cb(agent.Event{Type: agent.EventDone})
	return nil
}

func init() { agent.Register(agent.TypeTelegram, func() agent.Runner { return fakeTG{} }) }

func TestTelegramSyncCarriesScopeOnly(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/tg.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	s, err := database.CreateSession("tg", "tg_target:@alice", "", "default", agent.TypeTelegram, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := database.AddMessageWithID(s.ID, "claude", "舊鏡像", "Alice"); err != nil {
		t.Fatal(err)
	}
	SetTelegramScopeResolver(func(string) (string, string, bool) { return "1", "101", true })
	t.Cleanup(func() { SetTelegramScopeResolver(nil) })

	p, err := buildSyncPayload(database, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.TelegramScope == nil || p.TelegramScope.AccountID != "1" || p.TelegramScope.PeerID != "101" {
		t.Fatalf("sync 應帶目前 scope，got %+v", p.TelegramScope)
	}
	if string(p.Messages) != "[]" && string(p.Messages) != "null" {
		t.Fatalf("sync 不應帶資料庫裡的舊訊息，got %s", p.Messages)
	}
}
