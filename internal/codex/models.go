package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ModelEntry 是一個可選模型。
type ModelEntry struct {
	ModelID string
	Label   string
}

type modelsCacheFile struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
	} `json:"models"`
}

// FetchModelOptions 讀取 codex CLI 自己維護的本機快取檔 ~/.codex/models_cache.json。
// 該檔案由 codex CLI 執行時自動刷新，本函式只負責讀取解析，不主動觸發刷新。
func FetchModelOptions() ([]ModelEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	if err != nil {
		return nil, err
	}
	var parsed modelsCacheFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	entries := make([]ModelEntry, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		entries = append(entries, ModelEntry{ModelID: m.Slug, Label: m.DisplayName})
	}
	return entries, nil
}
