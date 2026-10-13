package db

import "testing"

func TestMarkSessionRead_DoesNotTouchLastActive(t *testing.T) {
	path := t.TempDir() + "/smoke.db"
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	s, err := database.CreateSession("t", "", "", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if s.LastReadAt == "" {
		t.Fatal("CreateSession 應寫入 last_read_at")
	}
	beforeActive := s.LastActive

	if err := database.MarkSessionRead(s.ID); err != nil {
		t.Fatal(err)
	}
	got, err := database.GetSession(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastActive != beforeActive {
		t.Fatalf("MarkSessionRead 不應動到 last_active: before=%q after=%q", beforeActive, got.LastActive)
	}
	if got.LastReadAt == "" {
		t.Fatal("MarkSessionRead 後 last_read_at 不應為空")
	}
}

func TestMarkAllSessionsRead(t *testing.T) {
	path := t.TempDir() + "/smoke_all.db"
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	a, err := database.CreateSession("a", "", "", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	b, err := database.CreateSession("b", "", "", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	// 模擬未讀：把 a、b 的 last_read_at 清空，last_active 維持建立時的時間。
	if _, err := database.Exec(`UPDATE sessions SET last_read_at = '' WHERE id IN (?, ?)`, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}

	if err := database.MarkAllSessionsRead(); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{a.ID, b.ID} {
		got, err := database.GetSession(id)
		if err != nil {
			t.Fatal(err)
		}
		if got.LastReadAt == "" {
			t.Fatalf("session %s: MarkAllSessionsRead 後 last_read_at 不應為空", id)
		}
	}
}

func TestUnreadCountCalculation(t *testing.T) {
	path := t.TempDir() + "/unread_count.db"
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	sess, err := database.CreateSession("test_unread", "", "", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}

	// 初始無訊息，未讀應為 0
	got, err := database.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnreadCount != 0 {
		t.Fatalf("expected initial unread_count=0, got %d", got.UnreadCount)
	}

	// 使用者發送訊息，role='user' 不應計入未讀
	if err := database.AddMessage(sess.ID, "user", "hello", ""); err != nil {
		t.Fatal(err)
	}
	got, err = database.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnreadCount != 0 {
		t.Fatalf("user message should not count as unread, got %d", got.UnreadCount)
	}

	// 寫入兩則 assistant 訊息與一則 shell 訊息（created_at > last_read_at）
	// 將 last_read_at 設為稍微過去的時間，確保後續訊息大於 last_read_at
	if _, err := database.Exec(`UPDATE sessions SET last_read_at = '2020-01-01 00:00:00' WHERE id = ?`, sess.ID); err != nil {
		t.Fatal(err)
	}

	_, _, err = database.AddMessageWithID(sess.ID, "claude", "response 1", "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = database.AddMessageWithID(sess.ID, "claude", "response 2", "Worker")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = database.AddMessageWithID(sess.ID, "shell", "echo done", "")
	if err != nil {
		t.Fatal(err)
	}

	// 查詢未讀數應為 3
	got, err = database.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnreadCount != 3 {
		t.Fatalf("expected unread_count=3, got %d", got.UnreadCount)
	}

	// 驗證 ListSessions 亦正確統計 unread_count
	list, err := database.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 || list[0].UnreadCount != 3 {
		t.Fatalf("ListSessions expected unread_count=3, got %d", list[0].UnreadCount)
	}

	// 標記已讀後，未讀數應歸零
	if err := database.MarkSessionRead(sess.ID); err != nil {
		t.Fatal(err)
	}
	got, err = database.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnreadCount != 0 {
		t.Fatalf("after MarkSessionRead expected unread_count=0, got %d", got.UnreadCount)
	}
}
