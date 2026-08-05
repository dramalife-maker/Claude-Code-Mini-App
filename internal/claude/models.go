package claude

// ModelEntry 是一個可選模型（含空字串代表「預設」）。
type ModelEntry struct {
	ModelID string
	Label   string
}

// ModelOptions 回傳 Claude CLI 可用的模型別名。
// ponytail: Claude CLI 無 list-models 指令，手動維護；Anthropic 出新一代模型要手動加。
func ModelOptions() []ModelEntry {
	return []ModelEntry{
		{"sonnet", "Sonnet 5"},
		{"opus", "Opus 5"},
		{"fable", "Fable 5"},
		{"haiku", "Haiku 4.5"},
	}
}

// EffortOptions 回傳 --effort 支援的等級。
func EffortOptions() []string {
	return []string{"low", "medium", "high", "xhigh", "max"}
}
