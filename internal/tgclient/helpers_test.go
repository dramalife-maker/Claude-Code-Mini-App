package tgclient

import (
	"testing"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

// tgTestDate 是 2026-10-19 08:00:00 UTC 的 Telegram 協定 Date（Unix 秒）。
const tgTestDate = 1792396800

const tgTestDateText = "2026-10-19 08:00:00"

func newTGSession(t *testing.T, database *db.DB, name, target string) string {
	t.Helper()
	s, err := database.CreateSession(name, "tg_target:"+target, "", "default", agent.TypeTelegram, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	return s.ID
}
