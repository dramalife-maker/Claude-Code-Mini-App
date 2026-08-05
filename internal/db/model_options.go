package db

// ModelOption 是某個 agent_type 底下的一個可選模型。
type ModelOption struct {
	ModelID string `json:"model_id"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

// ListModelOptions 回傳指定 agent_type 底下已啟用的模型選項。
func (db *DB) ListModelOptions(agentType string) ([]ModelOption, error) {
	rows, err := db.Query(
		`SELECT model_id, label, enabled FROM model_options WHERE agent_type = ? AND enabled = 1 ORDER BY id`,
		agentType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []ModelOption
	for rows.Next() {
		var o ModelOption
		var enabled int
		if err := rows.Scan(&o.ModelID, &o.Label, &enabled); err != nil {
			return nil, err
		}
		o.Enabled = enabled != 0
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

// SyncModelOptions 用最新抓到的 live 清單同步 model_options：
// live 裡有、DB 沒有的 → 新增（enabled 預設開）；
// DB 有、live 裡沒有的 → 刪除；
// 兩邊都有的 → 不動（不覆寫 label/enabled，保留手動調整）。
// 只在成功抓到 live 清單時呼叫；抓取失敗時呼叫端不應呼叫本函式。
func (db *DB) SyncModelOptions(agentType string, live []ModelOption) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	liveIDs := make(map[string]bool, len(live))
	for _, o := range live {
		liveIDs[o.ModelID] = true
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO model_options (agent_type, model_id, label, enabled) VALUES (?, ?, ?, 1)`,
			agentType, o.ModelID, o.Label,
		); err != nil {
			return err
		}
	}

	rows, err := tx.Query(`SELECT model_id FROM model_options WHERE agent_type = ?`, agentType)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var modelID string
		if err := rows.Scan(&modelID); err != nil {
			rows.Close()
			return err
		}
		if !liveIDs[modelID] {
			stale = append(stale, modelID)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, modelID := range stale {
		if _, err := tx.Exec(
			`DELETE FROM model_options WHERE agent_type = ? AND model_id = ?`,
			agentType, modelID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
