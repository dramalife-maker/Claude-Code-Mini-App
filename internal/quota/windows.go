package quota

import (
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/usage"
)

// Window 是結構化用量窗口（供前端依 key 畫進度條）；key 取 usage.QuotaWindow.Kind。
type Window struct {
	Key      string   `json:"key"`
	Label    string   `json:"label,omitempty"`
	Percent  *float64 `json:"percent,omitempty"`
	ResetsAt string   `json:"resets_at,omitempty"`
	Used     *float64 `json:"used,omitempty"`
	Limit    *float64 `json:"limit,omitempty"`
}

// windowsFromInfo 保留 info.Windows 順序，略過沒有 percent 也沒有 used/limit 的窗口。
// resets_at 能解析時統一轉成 RFC3339（Claude /usage 的人類可讀字串前端無法解析）；解析不了就原樣送出。
func windowsFromInfo(info *usage.QuotaInfo) []Window {
	if info == nil {
		return nil
	}
	now := time.Now()
	var out []Window
	for _, w := range info.Windows {
		if w.Percent == nil && w.Used == nil && w.Limit == nil {
			continue
		}
		resets := w.ResetsAt
		if t, ok := usage.ParseResetsAt(resets, now); ok {
			resets = t.UTC().Format(time.RFC3339)
		}
		out = append(out, Window{Key: w.Kind, Label: w.Label, Percent: w.Percent, ResetsAt: resets, Used: w.Used, Limit: w.Limit})
	}
	return out
}

// snapshotFromInfo 給各 fetcher 共用：display_text 仍由 FormatDisplay 產生，再附上結構化 windows / plan。
func snapshotFromInfo(provider string, info *usage.QuotaInfo) Snapshot {
	s := Snapshot{
		Provider:    provider,
		DisplayText: FormatDisplay(provider, info),
		Windows:     windowsFromInfo(info),
		UpdatedAt:   time.Now(),
	}
	if info != nil {
		s.Plan = info.Plan
	}
	return s
}
