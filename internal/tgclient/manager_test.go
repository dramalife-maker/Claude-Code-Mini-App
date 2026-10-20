package tgclient

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

func TestTelegramRunnerRegistered(t *testing.T) {
	if !agent.IsRegistered(agent.TypeTelegram) {
		t.Fatal("agent.TypeTelegram 應該已註冊")
	}

	runner, err := agent.NewRunner(agent.TypeTelegram)
	if err != nil {
		t.Fatalf("agent.NewRunner(telegram) 失敗: %v", err)
	}
	if runner.Name() != agent.TypeTelegram {
		t.Fatalf("runner.Name() = %q, 預期 %q", runner.Name(), agent.TypeTelegram)
	}
}

func TestManagerSendSystemMessage(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer database.Close()

	sess, err := database.CreateSession("test-tg", "", "", "default", agent.TypeTelegram, nil, "agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	mgr := NewManager(config.TelegramClient{
		Enabled: false,
	}, database)

	mgr.SendSystemMessage(sess.ID, "哈囉測試系統訊息")

	msgs, err := database.ListMessages(sess.ID)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("預期 1 則訊息，實際有 %d 則", len(msgs))
	}
	if msgs[0].Content != "哈囉測試系統訊息" {
		t.Fatalf("訊息內容不符: %s", msgs[0].Content)
	}

	// 測試未啟用時 runner.Run 不報錯且送出提示
	runner := NewRunner(mgr)
	doneCalled := false
	err = runner.Run(context.Background(), agent.RunOptions{
		SessionID: sess.ID,
		Prompt:    "hello",
	}, func(e agent.Event) {
		if e.Type == agent.EventDone {
			doneCalled = true
		}
	})
	if err != nil {
		t.Fatalf("runner.Run 失敗: %v", err)
	}
	if !doneCalled {
		t.Fatal("runner.Run 應回傳 EventDone")
	}
}

func TestManagerHandleUpdates(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test2.db")
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer database.Close()

	sess, err := database.CreateSession("TG: alex1151212", "tg_target:@alex1151212", "", "default", agent.TypeTelegram, nil, "agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	mgr := NewManager(config.TelegramClient{Enabled: true}, database)
	mgr.setAccount(1)
	mgr.SetActiveSession(sess.ID)

	// 1. 紀錄 User 資訊
	mgr.recordUsers([]tg.UserClass{
		&tg.User{
			ID:        123456789,
			FirstName: "Alex",
			LastName:  "Chen",
			Username:  "alex1151212",
		},
	})

	// 2. 測試 UpdateShort (包 UpdateNewMessage)
	var got []*db.Message
	orig := publishMessage
	publishMessage = func(sessionID string, m *db.Message, _ string) {
		if sessionID == sess.ID {
			got = append(got, m)
		}
	}
	t.Cleanup(func() { publishMessage = orig })

	err = mgr.handleUpdate(context.Background(), &tg.UpdateShort{
		Update: &tg.UpdateNewMessage{
			Message: &tg.Message{
				ID:      1,
				Out:     false,
				Message: "在嗎？兄弟！",
				FromID:  &tg.PeerUser{UserID: 123456789},
				PeerID:  &tg.PeerUser{UserID: 123456789},
				Date:    tgTestDate,
			},
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate 失敗: %v", err)
	}
	if len(got) != 1 || got[0].Content != "在嗎？兄弟！" || got[0].Author != "Alex Chen" || got[0].Role != "claude" {
		t.Fatalf("即時訊息應推成 DTO，got %+v", got)
	}
	stored, err := database.ListMessages(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 0 {
		t.Fatalf("即時訊息不應寫入資料庫，got %d", len(stored))
	}

	err = mgr.handleUpdate(context.Background(), &tg.UpdateShortMessage{
		ID:      2,
		Out:     false,
		UserID:  123456789,
		Message: "今晚吃熱炒嗎",
		Date:    tgTestDate,
	})
	if err != nil {
		t.Fatalf("handleUpdate 失敗: %v", err)
	}
	if len(got) != 2 || got[1].Content != "今晚吃熱炒嗎" {
		t.Fatalf("第二則應推播，got %+v", got)
	}
}

func TestManagerFiltersIncomingMessages(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/filter.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mgr := NewManager(config.TelegramClient{DefaultTarget: "@defaultuser"}, database)
	mgr.setAccount(1)
	sessions := make(map[string]*db.Session)
	for name, description := range map[string]string{
		"alex": "tg_target: @AlEx ", "numeric": "tg_target:202",
		"registered": "tg_target:@wrong", "default": "",
	} {
		s, err := database.CreateSession(name, description, "", "default", agent.TypeTelegram, nil, "agent")
		if err != nil {
			t.Fatal(err)
		}
		sessions[name] = s
	}
	mgr.SetActiveSession(sessions["alex"].ID)
	mgr.RegisterSessionTarget(sessions["registered"].ID, "@runtime")
	mgr.recordUsers([]tg.UserClass{
		&tg.User{ID: 101, FirstName: "Alex Chen", Username: "alex"},
		&tg.User{ID: 303, FirstName: "@alex", Username: "eve"},
		&tg.User{ID: 404, Username: "runtime"},
		&tg.User{ID: 505, Username: "defaultuser"},
	})

	// 每則協定訊息都要有唯一 ID 與真實 Date，避免舊 fixture 靠缺漏欄位繞過 upsert。
	nextID := 1000
	newID := func() int {
		nextID++
		return nextID
	}
	short := func(uid int64, text string, out bool) *tg.UpdateShortMessage {
		return &tg.UpdateShortMessage{ID: newID(), UserID: uid, Message: text, Out: out, Date: tgTestDate}
	}
	private := func(peer tg.PeerClass, out bool) *tg.UpdateNewMessage {
		return &tg.UpdateNewMessage{Message: &tg.Message{
			ID: newID(), PeerID: peer, FromID: &tg.PeerUser{UserID: 101}, Out: out, Message: "hello", Date: tgTestDate,
		}}
	}
	for _, tc := range []struct {
		name        string
		update      tg.UpdatesClass
		wantSession string
	}{
		{"private short", short(101, "hello", false), "alex"},
		{"private wrapped", &tg.UpdateShort{Update: private(&tg.PeerUser{UserID: 101}, false)}, "alex"},
		{"numeric target", short(202, "hello", false), "numeric"},
		{"registered target", short(404, "hello", false), "registered"},
		{"default target", short(505, "hello", false), "default"},
		{"updates with user data", &tg.Updates{
			Users:   []tg.UserClass{&tg.User{ID: 101, FirstName: "Alex Chen", Username: "alex"}},
			Updates: []tg.UpdateClass{private(&tg.PeerUser{UserID: 101}, false)},
		}, "alex"},
		{"combined private", &tg.UpdatesCombined{Updates: []tg.UpdateClass{private(&tg.PeerUser{UserID: 101}, false)}}, "alex"},
		{"unknown private", short(999, "unrelated", false), ""},
		{"display name impersonation", short(303, "unrelated", false), ""},
		{"peer rather than from ID", &tg.UpdateShort{Update: private(&tg.PeerUser{UserID: 999}, false)}, ""},
		{"group short", &tg.UpdateShortChatMessage{ID: newID(), FromID: 101, ChatID: 7, Message: "group", Date: tgTestDate}, ""},
		{"group wrapped", &tg.UpdateShort{Update: private(&tg.PeerChat{ChatID: 7}, false)}, ""},
		{"channel wrapped", &tg.UpdateShort{Update: &tg.UpdateNewChannelMessage{
			Message: private(&tg.PeerChannel{ChannelID: 8}, false).Message,
		}}, ""},
		{"channel in updates", &tg.Updates{Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{
			Message: private(&tg.PeerChannel{ChannelID: 8}, false).Message,
		}}}, ""},
		{"group in combined", &tg.UpdatesCombined{Updates: []tg.UpdateClass{private(&tg.PeerChat{ChatID: 7}, false)}}, ""},
		// Out 代表本帳號從其他裝置送給 PeerUser，應寫入該收件對象的會話（右側 user 訊息）。
		{"outgoing short", short(101, "hello", true), "alex"},
		{"outgoing wrapped", &tg.UpdateShort{Update: private(&tg.PeerUser{UserID: 101}, true)}, "alex"},
		{"missing peer", &tg.UpdateShort{Update: private(nil, false)}, ""},
		{"empty short", short(101, "  ", false), ""},
		{"zero sender", short(0, "hello", false), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits []string
			orig := publishMessage
			publishMessage = func(sessionID string, _ *db.Message, _ string) { hits = append(hits, sessionID) }
			t.Cleanup(func() { publishMessage = orig })
			if err := mgr.handleUpdate(context.Background(), tc.update); err != nil {
				t.Fatal(err)
			}
			want := ""
			if tc.wantSession != "" {
				want = sessions[tc.wantSession].ID
			}
			got := ""
			if len(hits) == 1 {
				got = hits[0]
			} else if len(hits) != 0 {
				t.Fatalf("應推播 0 或 1 則，got %d", len(hits))
			}
			if got != want {
				t.Fatalf("推播 session = %q, want %q", got, want)
			}
		})
	}
}
