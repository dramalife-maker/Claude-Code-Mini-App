package tgclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

func TestTelegramHistoryPagination(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(t.TempDir() + "/hist.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	alice := newTGSession(t, database, "alice", "@alice")
	mgr := NewManager(config.TelegramClient{Enabled: true}, database)
	mgr.setAccount(1)
	mgr.authState = AuthStateAuthorized
	mgr.recordUsers([]tg.UserClass{&tg.User{ID: 101, AccessHash: 555, FirstName: "Alice", Username: "alice"}})

	peer := &tg.PeerUser{UserID: 101}
	mk := func(id int, text string) tg.MessageClass {
		return &tg.Message{ID: id, PeerID: peer, Message: text, Date: tgTestDate + id}
	}
	// 55–57 同一秒；62 是 service（不可見但仍推進游標）。
	remote := []tg.MessageClass{
		mk(50, "old"),
		&tg.MessageService{ID: 62, PeerID: peer, Date: tgTestDate + 62, Action: &tg.MessageActionHistoryClear{}},
		&tg.Message{ID: 55, PeerID: peer, Message: "a", Date: tgTestDate + 55},
		&tg.Message{ID: 56, PeerID: peer, Message: "b", Date: tgTestDate + 55},
		&tg.Message{ID: 57, PeerID: peer, Message: "c", Date: tgTestDate + 55},
	}
	var reqs []tg.MessagesGetHistoryRequest
	mgr.historyFn = func(_ context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		reqs = append(reqs, *req)
		p := req.Peer.(*tg.InputPeerUser)
		if p.UserID != 101 || p.AccessHash != 555 {
			t.Fatalf("peer = %+v", p)
		}
		var matched []tg.MessageClass
		for i := len(remote) - 1; i >= 0; i-- {
			id := remote[i].GetID()
			if req.OffsetID > 0 && id >= req.OffsetID {
				continue
			}
			matched = append(matched, remote[i])
			if len(matched) == req.Limit {
				break
			}
		}
		if len(matched) < req.Limit {
			return &tg.MessagesMessages{Messages: matched}, nil
		}
		return &tg.MessagesMessagesSlice{Messages: matched, Count: len(remote)}, nil
	}

	page, err := mgr.HistoryPage(ctx, alice, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].OffsetID != 0 || reqs[0].Limit != 2 {
		t.Fatalf("最新頁應只查一次 offset=0 limit=2，got %+v", reqs)
	}
	if !page.HasMore || page.NextCursor == nil || *page.NextCursor != "56" {
		t.Fatalf("最新頁游標應停在掃到的最舊 ID 56，got %+v", page)
	}
	if len(page.Messages) != 2 || page.Messages[0].ID != 56 || page.Messages[1].ID != 57 {
		t.Fatalf("同秒應依 message id 由舊到新，got %+v", page.Messages)
	}
	var stored int
	if err := database.QueryRow(`SELECT COUNT(*) FROM messages WHERE session_id = ?`, alice).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("歷史不應寫入資料庫，stored=%d err=%v", stored, err)
	}

	older, err := mgr.HistoryPage(ctx, alice, 4, *page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if reqs[1].OffsetID != 56 {
		t.Fatalf("下一頁 offset 應為排他游標 56，got %d", reqs[1].OffsetID)
	}
	// 可見的是 55 與 50；62 是 service，不出現，但游標要跨過它。
	if len(older.Messages) != 2 || older.Messages[0].ID != 50 || older.Messages[1].ID != 55 {
		t.Fatalf("前頁應為 50、55（service 不顯示），got %+v", older.Messages)
	}
	if older.HasMore {
		t.Fatal("messages.messages 代表已到頂")
	}

	// 只剩 service 的頁：可見為空，游標仍往前。
	mgr.historyFn = func(_ context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		return &tg.MessagesMessagesSlice{Messages: []tg.MessageClass{
			&tg.MessageService{ID: 10, PeerID: peer, Date: tgTestDate, Action: &tg.MessageActionHistoryClear{}},
		}, Count: 99}, nil
	}
	empty, err := mgr.HistoryPage(ctx, alice, 1, "11")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Messages) != 0 || !empty.HasMore || empty.NextCursor == nil || *empty.NextCursor != "10" {
		t.Fatalf("空可見頁仍應推進游標，got %+v", empty)
	}

	mgr.historyFn = func(context.Context, *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		return nil, tgerr.New(420, "FLOOD_WAIT_3")
	}
	if _, err := mgr.HistoryPage(ctx, alice, 1, ""); err == nil {
		t.Fatal("FLOOD_WAIT 應回傳可判定的限流錯誤")
	} else {
		var rl *HistoryRateLimitedError
		if !errors.As(err, &rl) || rl.RetryAfter != 3*time.Second {
			t.Fatalf("retry after 應為 3s，got %v", err)
		}
	}

	calls := 0
	mgr.historyFn = func(context.Context, *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		calls++
		return nil, nil
	}
	if _, err := mgr.HistoryPage(ctx, alice, 0, "nope"); !errors.Is(err, ErrHistoryInvalidCursor) || calls != 0 {
		t.Fatalf("壞游標不應查上游，err=%v calls=%d", err, calls)
	}
	if _, err := mgr.HistoryPage(ctx, alice, 101, ""); !errors.Is(err, ErrHistoryInvalidLimit) {
		t.Fatal(err)
	}

	mgr.setAccount(2) // 換帳號會清掉 peer 快取
	mgr.authState = AuthStateAuthorized
	if _, err := mgr.HistoryPage(ctx, alice, 1, ""); !errors.Is(err, ErrHistoryUnavailable) {
		t.Fatalf("換帳號後沒有新的 access hash，應 503，got %v", err)
	}
}
