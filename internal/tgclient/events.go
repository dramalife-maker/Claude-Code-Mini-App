package tgclient

import (
	"log/slog"
	"strconv"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/ws"
)

// 事件發布點：正式環境推到 ws；測試可替換以擷取事件（tgclient 已 import ws，反向不可）。
var (
	publishMessage  = ws.BroadcastTelegramMessage
	publishDelivery = ws.BroadcastTelegramDelivery
)

// syncState 回報目前連線：已授權為 ready，否則 offline。沒有補漏中的第三態。
func (m *Manager) syncState() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.authState == AuthStateAuthorized && m.accountID != 0 {
		return SyncStateReady
	}
	return SyncStateOffline
}

// sessionSyncState 供 WS sync 使用。歷史不在本機，狀態只反映有沒有登入。
func (m *Manager) sessionSyncState(string) string { return m.syncState() }

// syncScope 以目前已授權帳號與 session 的 PeerUser 快取驗證 scope。
// 未登入、帳號已切換（快取已清）或對象尚未解析時 ok=false，前端不得沿用舊帳號資料。
func (m *Manager) syncScope(sessionID string) (accountID, peerID string, ok bool) {
	if m.db == nil {
		return "", "", false
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil || s.AgentType != agent.TypeTelegram {
		return "", "", false
	}
	target := m.sessionTarget(s)
	if target == "" {
		return "", "", false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.authState != AuthStateAuthorized || m.accountID == 0 {
		return "", "", false
	}
	id := m.userIDs[target]
	if id == 0 {
		if n, e := strconv.ParseInt(target, 10, 64); e == nil && n > 0 {
			id = n
		}
	}
	if id == 0 {
		return "", "", false
	}
	return strconv.FormatInt(m.accountID, 10), strconv.FormatInt(id, 10), true
}

// announceAll 通知所有 Telegram session：帳號切換或斷線後前端要丟掉舊 scope。
func (m *Manager) announceAll() { m.announce(func(string, string) bool { return true }) }

func (m *Manager) announce(match func(account, peer string) bool) {
	if m.db == nil {
		return
	}
	sessions, err := m.db.ListSessions()
	if err != nil {
		slog.Warn("[telegram] 讀取會話失敗", "error", err)
		return
	}
	for _, s := range sessions {
		if s.AgentType != agent.TypeTelegram {
			continue
		}
		a, p, ok := m.syncScope(s.ID)
		if !ok || match(a, p) {
			ws.BroadcastTelegramSync(s.ID)
		}
	}
}

// publishLive 把一則即時私聊轉成與歷史頁相同的 DTO 推給綁定的 session。不寫資料庫。
func (m *Manager) publishLive(msg *db.Message, replaces string) {
	if msg == nil || msg.Telegram == nil {
		return
	}
	peerID, err := strconv.ParseInt(msg.Telegram.PeerID, 10, 64)
	if err != nil {
		return
	}
	ids, err := m.boundSessionIDs(peerID)
	if err != nil || len(ids) == 0 {
		return
	}
	for _, sid := range ids {
		out := *msg
		out.SessionID = sid
		if len(msg.Attachments) > 0 {
			cp := append([]db.Attachment(nil), msg.Attachments...)
			for i := range cp {
				cp[i].URL = tgMediaURL(sid, msg.Telegram.MessageID)
			}
			out.Attachments = cp
		}
		publishMessage(sid, &out, replaces)
	}
}
