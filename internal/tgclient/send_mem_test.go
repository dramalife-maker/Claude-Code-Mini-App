package tgclient

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

func TestTelegramTextSendMemory(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(t.TempDir() + "/send.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	alice := newTGSession(t, database, "alice", "@alice")
	mgr := NewManager(config.TelegramClient{Enabled: true}, database)
	mgr.setAccount(1)
	mgr.authState = AuthStateAuthorized
	mgr.recordUsers([]tg.UserClass{&tg.User{ID: 101, AccessHash: 555, Username: "alice"}})

	var reqs []*tg.MessagesSendMessageRequest
	var published []*db.Message
	var replaces []string
	var deliveries []string
	origM, origD := publishMessage, publishDelivery
	publishMessage = func(_ string, m *db.Message, rep string) {
		published = append(published, m)
		replaces = append(replaces, rep)
	}
	publishDelivery = func(_, id, state, _ string) { deliveries = append(deliveries, id+":"+state) }
	t.Cleanup(func() { publishMessage, publishDelivery = origM, origD })

	mgr.sendFn = func(_ context.Context, req *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error) {
		reqs = append(reqs, req)
		return &tg.UpdateShortSentMessage{ID: 910, Date: tgTestDate}, nil
	}
	state, err := mgr.sendTelegramText(ctx, alice, 42, "你好")
	if err != nil || state != "sent" {
		t.Fatalf("state=%q err=%v", state, err)
	}
	if len(reqs) != 1 || reqs[0].RandomID == 0 || reqs[0].Message != "你好" {
		t.Fatalf("request = %+v", reqs)
	}
	if len(published) != 1 || published[0].ID != 910 || replaces[0] != "42" || published[0].Telegram.MessageID != 910 {
		t.Fatalf("應以遠端 ID 取代待傳 42，got %+v replaces=%v", published, replaces)
	}
	var rows int
	if err := database.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("送出不應寫入 messages，rows=%d err=%v", rows, err)
	}

	// 帳號 B 不能把 A 留在記憶體的 random_id 送出去。
	mgr.setAccount(1)
	mgr.authState = AuthStateAuthorized
	mgr.recordUsers([]tg.UserClass{&tg.User{ID: 101, AccessHash: 555, Username: "alice"}})
	mgr.pending[7] = memSend{sessionID: alice, clientID: 1, account: "1", peerID: 101, access: 555, text: "A"}
	mgr.setAccount(2)
	calls := 0
	mgr.sendFn = func(context.Context, *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error) {
		calls++
		return nil, nil
	}
	if st, err := mgr.performSend(ctx, 7); err != nil || st != "pending" || calls != 0 {
		t.Fatalf("帳號 B 不得送出 A 的待傳，state=%q calls=%d err=%v", st, calls, err)
	}
	if len(deliveries) != 0 {
		t.Fatalf("不得改動 A 的待傳狀態，got %v", deliveries)
	}
}
