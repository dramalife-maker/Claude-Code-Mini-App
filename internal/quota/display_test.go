package quota

import (
	"testing"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/usage"
)

func TestFormatClaudeDisplay(t *testing.T) {
	now := time.Date(2026, 6, 30, 11, 45, 0, 0, time.FixedZone("CST", 8*3600))
	info := usage.FromClaudeUsageText(`Current session: 16% used · resets Jun 30, 3:10pm (Asia/Taipei)
Current week (all models): 9% used · resets Jul 3, 9:59pm (Asia/Taipei)`)
	got := FormatDisplayAt("claude", info, now)
	if got != "5h 16% · Resets in 3 hr 25 min · Week 9%" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatClaudeDisplayOAuth(t *testing.T) {
	now := time.Date(2026, 6, 30, 11, 45, 0, 0, time.FixedZone("CST", 8*3600))
	info, err := usage.FromClaudeOAuthUsageJSON([]byte(`{"five_hour":{"utilization":16,"resets_at":"2026-06-30T15:09:59+08:00"},"seven_day":{"utilization":9,"resets_at":"2026-07-03T22:00:00+08:00"}}`))
	if err != nil {
		t.Fatal(err)
	}
	got := FormatDisplayAt("claude", info, now)
	if got != "5h 16% · Resets in 3 hr 25 min · Week 9%" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatCursorDisplay(t *testing.T) {
	raw := `{"planUsage":{"autoPercentUsed":8.05,"apiPercentUsed":100,"totalPercentUsed":30.28}}`
	info, err := usage.FromCursorPeriodUsageJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	got := FormatDisplay("cursor", info)
	if got != "Total 30% · Auto 8% · API 100%" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatKiroDisplay(t *testing.T) {
	info := usage.FromKiroUsageText("KIRO FREE\nCredits (4.52 of 50)\n9%\n")
	got := FormatDisplay("kiro", info)
	if got != "FREE · Credits 9% (4.52/50)" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatCodexDisplay(t *testing.T) {
	now := time.Unix(1790700000, 0)
	info := usage.FromCodexRolloutLine([]byte(sampleCodexRolloutLine), now)
	got := FormatDisplay("codex", info)
	if got != "5h 2% · Week 13%" {
		t.Fatalf("got %q", got)
	}
}

// codex-cli 0.159.2 實測 rollout token_count 行（週用量改為 13 以驗證非零）。
const sampleCodexRolloutLine = `{"timestamp":"2026-09-30T06:44:51.387Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":38134}},"rate_limits":{"limit_id":"codex","primary":{"used_percent":2.0,"window_minutes":300,"resets_at":1790767809},"secondary":{"used_percent":13,"window_minutes":10080,"resets_at":1791354609},"credits":{"has_credits":false},"plan_type":"plus"}}}`

func TestFormatAntigravityDisplay(t *testing.T) {
	now := time.Date(2026, 10, 12, 3, 30, 0, 0, time.UTC)
	raw := "Gemini Models\tWeekly Limit Remaining\t84%\t2026-10-14T08:22:23Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t98%\t2026-10-12T06:52:28Z\n"
	info := usage.FromAgyUsageText(raw)
	got := FormatDisplayAt("antigravity", info, now)
	if got != "Gemini 5h 2% · Resets in 3 hr 22 min · Gemini Week 16%" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatAntigravityQuotaGroups(t *testing.T) {
	now := time.Date(2026, 10, 12, 3, 30, 0, 0, time.UTC)
	gemini := "Gemini Models\tWeekly Limit Remaining\t100%\nGemini Models\tFive Hour Limit Remaining\t100%\n"
	claude := "Claude and GPT models\tWeekly Limit Remaining\t0%\nClaude and GPT models\tFive Hour Limit Remaining\t0%\t2026-10-12T06:52:28Z\n"
	for _, tc := range []struct{ name, raw, want string }{
		{"both", gemini + claude, "Gemini 5h 0% · Gemini Week 0% · Claude/GPT 5h 100% · Resets in 3 hr 22 min · Claude/GPT Week 100%"},
		{"gemini", gemini, "Gemini 5h 0% · Gemini Week 0%"},
		{"claude", claude, "Claude/GPT 5h 100% · Resets in 3 hr 22 min · Claude/GPT Week 100%"},
		{"weekly-only", "Claude and GPT models\tWeekly Limit Remaining\t84%\n", "Claude/GPT Week 16%"},
		{"empty", "", "—"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDisplayAt("antigravity", usage.FromAgyUsageText(tc.raw), now)
			if got != tc.want {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestServiceGetAntigravity(t *testing.T) {
	s := NewService()
	snap := s.Get("antigravity")
	if snap.DisplayText != "—" {
		t.Fatalf("got %q, expected '—'", snap.DisplayText)
	}
	// 舊 agent_type 別名
	snap = s.Get("gemini")
	if snap.DisplayText != "—" {
		t.Fatalf("gemini alias got %q, expected '—'", snap.DisplayText)
	}
}
