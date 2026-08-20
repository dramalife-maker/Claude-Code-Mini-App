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
