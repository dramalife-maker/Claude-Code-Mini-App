package ws

import (
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

// telegramSyncStateFn 由 tgclient 設定；ws 不能 import tgclient（tgclient 已 import ws）。
var telegramSyncStateFn = func(string) string { return "offline" }

// SetTelegramSyncState 設定 sync 時回報的同步狀態來源（ready／offline）。
func SetTelegramSyncState(fn func(sessionID string) string) {
	if fn != nil {
		telegramSyncStateFn = fn
	}
}

// telegramScopeFn 由 tgclient 設定（ws 不能 import tgclient）：以 manager 目前帳號與 session 實際 target
// 驗證出 account/peer；無法驗證時回傳 ok=false，sync 保守地不帶任何歷史。
var telegramScopeFn func(sessionID string) (accountID, peerID string, ok bool)

// SetTelegramScopeResolver 設定 sync 使用的 scope 驗證來源。
func SetTelegramScopeResolver(fn func(sessionID string) (accountID, peerID string, ok bool)) {
	telegramScopeFn = fn
}

// tgScope 是 sync 帶給前端的目前已驗證 account/peer。
type tgScope struct {
	AccountID string `json:"account_id"`
	PeerID    string `json:"peer_id"`
}

// BroadcastTelegramSync 推播目前已驗證的 scope 與同步狀態（type=telegram_scope）。
// 歷史不走這條：前端自己向 Telegram 要最新頁。帳號切換後前端要丟掉舊 scope 的資料。
func BroadcastTelegramSync(sessionID string) {
	if !hub.has(sessionID) {
		return
	}
	ev := serverMsg{Type: "telegram_scope", SyncState: telegramSyncStateFn(sessionID)}
	if telegramScopeFn != nil {
		if acct, peer, ok := telegramScopeFn(sessionID); ok {
			ev.TelegramScope = &tgScope{AccountID: acct, PeerID: peer}
		}
	}
	hub.Broadcast(sessionID, ev)
}

// BroadcastTelegramMessage 推播完整 Telegram 訊息 upsert（與 HTTP 分頁同一份 DTO）。
// replaces 非空時，前端用遠端訊息取代同一個 client_message_id 的待傳氣泡。
func BroadcastTelegramMessage(sessionID string, m *db.Message, replaces string) {
	hub.Broadcast(sessionID, serverMsg{
		Type:                    "message",
		ID:                      m.ID,
		Role:                    m.Role,
		Content:                 m.Content,
		CreatedAt:               m.CreatedAt,
		Author:                  m.Author,
		Attachments:             m.Attachments,
		Telegram:                m.Telegram,
		Delivery:                m.Delivery,
		ClientMessageID:         m.ClientMessageID,
		ReplacesClientMessageID: replaces,
	})
}

// BroadcastTelegramDelivery 推播送達狀態變化；錯誤呈現在 client_message_id 對應的原項目上。
func BroadcastTelegramDelivery(sessionID, clientMessageID, delivery, errText string) {
	hub.Broadcast(sessionID, serverMsg{Type: "delivery", ClientMessageID: clientMessageID, Delivery: delivery, Error: errText})
}
