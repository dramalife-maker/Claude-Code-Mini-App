package tgclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
)

func init() {
	agent.Register(agent.TypeTelegram, func() agent.Runner {
		return NewRunner(GetDefaultManager())
	})
}

// Runner 實作 agent.Runner，專門負責將會話輸入轉發給 Telegram 朋友。
type Runner struct {
	manager *Manager
}

// NewRunner 建立 Runner。
func NewRunner(m *Manager) *Runner {
	return &Runner{manager: m}
}

// Name 回傳工具名稱 "telegram"。
func (r *Runner) Name() string {
	return agent.TypeTelegram
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// Run 處理使用者在會話中送出的訊息。
func (r *Runner) Run(ctx context.Context, opts agent.RunOptions, cb agent.EventCallback) error {
	reply := func(text string) {
		cb(agent.Event{Type: agent.EventDelta, Text: text})
		cb(agent.Event{Type: agent.EventDone})
	}

	sessionID := opts.SessionID
	if sessionID == "" && opts.ExtraArgs != nil {
		sessionID = opts.ExtraArgs["miniapp_session_id"]
	}

	m := r.manager
	if m == nil || !m.cfg.Enabled || m.cfg.APIID == 0 || strings.TrimSpace(m.cfg.APIHash) == "" {
		reply("[系統] ⚠️ 尚未在 config.yaml 完整設定 Telegram MTProto 憑證。\n\n請在 config.yaml 的 `telegram_client` 區塊填入從 my.telegram.org 申請的 `api_id` 與 `api_hash` 後重啟程式。\n\n範例：\n```yaml\ntelegram_client:\n  enabled: true\n  api_id: 12345678\n  api_hash: \"0123456789abcdef0123456789abcdef\"\n```")
		return nil
	}

	if sessionID != "" {
		m.SetActiveSession(sessionID)
	}

	m.mu.Lock()
	state := m.authState
	m.mu.Unlock()

	switch state {
	case AuthStateNeedPhone:
		trimmed := strings.TrimSpace(opts.Prompt)
		isPhone := strings.HasPrefix(trimmed, "+") || (len(trimmed) >= 8 && isDigits(trimmed))
		if isPhone && m.phoneChan != nil {
			select {
			case m.phoneChan <- trimmed:
				reply(fmt.Sprintf("[系統] 已送出手機號碼 `%s`，Telegram 正在發送驗證碼，請在下方輸入收到的 5 位數驗證碼：", trimmed))
			default:
				reply("[系統] 正在等待 Telegram 伺服器回應，請稍候...")
			}
			return nil
		}
		reply("[系統] 首次使用 Telegram，請在下方輸入框輸入您的 Telegram 註冊手機號碼（格式如 `+886912345678`）：")
		return nil

	case AuthStateNeedCode:
		trimmed := strings.TrimSpace(opts.Prompt)
		if m.codeChan != nil {
			select {
			case m.codeChan <- trimmed:
				reply(fmt.Sprintf("[系統] 正在驗證登入碼 `%s`... 請稍候。", trimmed))
			default:
				reply("[系統] 正在等待驗證結果，請稍候...")
			}
			return nil
		}
		reply("[系統] 請在輸入框輸入 Telegram 發送給您的 5 位數驗證碼：")
		return nil

	case AuthStateNeedPassword:
		trimmed := strings.TrimSpace(opts.Prompt)
		if m.passwordChan != nil {
			select {
			case m.passwordChan <- trimmed:
				reply("[系統] 正在驗證兩步驟驗證（2FA）密碼... 請稍候。")
			default:
			}
			return nil
		}
		reply("[系統] 請在輸入框輸入您的 Telegram 兩步驟驗證（2FA）密碼：")
		return nil

	case AuthStateAuthorized:
		// 已授權：random_id 只留在記憶體（無本機訊息 ID 時 localID 為 0）。
		var localID int64
		if raw := opts.ExtraArgs[ArgTelegramLocalMessageID]; raw != "" {
			localID, _ = strconv.ParseInt(raw, 10, 64)
		}
		var state string
		var err error
		if raw := opts.ExtraArgs["tg_attachment_ids"]; strings.TrimSpace(raw) != "" {
			state, err = m.sendTelegramMedia(ctx, sessionID, localID, opts.Prompt, strings.Split(raw, ","))
		} else {
			state, err = m.sendTelegramText(ctx, sessionID, localID, opts.Prompt)
		}
		if err != nil {
			reply(fmt.Sprintf("[系統] 訊息傳送失敗: %v", err))
			return nil
		}
		if localID > 0 || state == "sent" || state == "failed" {
			// 待傳氣泡上的 delivery／取代事件已說明結果，不再插一則系統泡泡。
			cb(agent.Event{Type: agent.EventDone})
			return nil
		}
		reply("[系統] 訊息傳送失敗，未送出，請確認聊天對象後重新發送。")
		return nil

	default:
		reply("[系統] Telegram 連線準備中，若需登入驗證請稍候系統提示...")
		return nil
	}
}
