package db

import (
	"testing"
)

func TestSyncModelOptions(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// 初次同步：全部新增
	err = database.SyncModelOptions("claude", []ModelOption{
		{ModelID: "sonnet", Label: "Sonnet 5"},
		{ModelID: "opus", Label: "Opus 5"},
	})
	if err != nil {
		t.Fatalf("SyncModelOptions #1: %v", err)
	}
	opts, err := database.ListModelOptions("claude")
	if err != nil {
		t.Fatalf("ListModelOptions: %v", err)
	}
	if len(opts) != 2 {
		t.Fatalf("want 2 options, got %d: %+v", len(opts), opts)
	}

	// 手動關閉 sonnet
	if _, err := database.Exec(`UPDATE model_options SET enabled = 0 WHERE agent_type = 'claude' AND model_id = 'sonnet'`); err != nil {
		t.Fatalf("manual disable: %v", err)
	}

	// 第二次同步：opus 消失(該刪)，新增 fable，sonnet 仍在 live 清單裡但已手動關閉 → enabled 不應被覆寫回開啟
	err = database.SyncModelOptions("claude", []ModelOption{
		{ModelID: "sonnet", Label: "Sonnet 5"},
		{ModelID: "fable", Label: "Fable 5"},
	})
	if err != nil {
		t.Fatalf("SyncModelOptions #2: %v", err)
	}

	rows, err := database.Query(`SELECT model_id, enabled FROM model_options WHERE agent_type = 'claude' ORDER BY model_id`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var id string
		var enabled int
		if err := rows.Scan(&id, &enabled); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		got[id] = enabled != 0
	}
	if _, exists := got["opus"]; exists {
		t.Fatal("opus 應該已被刪除（live 清單裡消失了）")
	}
	if enabled, exists := got["sonnet"]; !exists || enabled {
		t.Fatalf("sonnet 應仍存在且維持手動關閉狀態，got exists=%v enabled=%v", exists, enabled)
	}
	if enabled, exists := got["fable"]; !exists || !enabled {
		t.Fatalf("fable 應是新增且預設開啟，got exists=%v enabled=%v", exists, enabled)
	}

	// ListModelOptions 只回傳 enabled=1（fable），sonnet 因手動關閉被排除
	opts, err = database.ListModelOptions("claude")
	if err != nil {
		t.Fatalf("ListModelOptions #2: %v", err)
	}
	if len(opts) != 1 || opts[0].ModelID != "fable" {
		t.Fatalf("want only fable enabled, got %+v", opts)
	}
}
