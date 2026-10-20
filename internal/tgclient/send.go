package tgclient

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

// ArgTelegramLocalMessageID 是 ws 經 RunOptions.ExtraArgs 傳給 runner 的待傳識別（只存在這次連線的記憶體）。
const ArgTelegramLocalMessageID = "tg_local_message_id"

// sendFunc 是 messages.sendMessage 的注入點（測試以假實作替代，不送真實訊息）。
type sendFunc func(ctx context.Context, req *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error)

var errTGNotConnected = errors.New("Telegram 客戶端未連線")

// memSend 是尚未對上遠端 ID 的送出。只放記憶體：重啟後視為失敗，由使用者重送。
type memSend struct {
	sessionID string
	clientID  int64
	account   string
	peerID    int64
	access    int64
	text      string
}

// rpcSendMessage 是正式環境的 sendFunc。
func (m *Manager) rpcSendMessage(ctx context.Context, req *tg.MessagesSendMessageRequest) (tg.UpdatesClass, error) {
	if m.client == nil {
		return nil, errTGNotConnected
	}
	return m.client.API().MessagesSendMessage(ctx, req)
}

func newRandomID() (int64, error) {
	var buf [8]byte
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return 0, err
		}
		if id := int64(binary.BigEndian.Uint64(buf[:])); id != 0 {
			return id, nil
		}
	}
}

// sendTelegramText 送出一則文字。random_id 與待傳氣泡的對應只放記憶體。
// 失敗標 failed，不自動重送。回傳 sent 或 failed。
func (m *Manager) sendTelegramText(ctx context.Context, sessionID string, clientID int64, prompt string) (string, error) {
	fail := func(err error) (string, error) {
		slog.Warn(fmt.Sprintf("[telegram] 文字送出失敗: %v", err))
		if clientID > 0 {
			publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
		}
		return "failed", nil
	}
	if strings.TrimSpace(prompt) == "" {
		return fail(errors.New("空白訊息不發送"))
	}
	if m.db == nil {
		return fail(errTGNotConnected)
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil {
		return fail(err)
	}
	peer, err := m.liveHistoryPeer(ctx, s)
	if err != nil {
		return fail(err)
	}
	randomID, err := newRandomID()
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	if m.pending == nil {
		m.pending = map[int64]memSend{}
	}
	m.pending[randomID] = memSend{
		sessionID: sessionID, clientID: clientID, account: peer.account,
		peerID: peer.peerID, access: peer.accessHash, text: prompt,
	}
	m.mu.Unlock()
	return m.performSend(ctx, randomID)
}

// performSend 用記憶體裡的 random_id 送出。帳號不符時不呼叫 RPC，也不改別的帳號的待傳項目。
func (m *Manager) performSend(ctx context.Context, randomID int64) (string, error) {
	m.mu.Lock()
	p, ok := m.pending[randomID]
	m.mu.Unlock()
	if !ok {
		return "failed", nil
	}
	if p.account != m.currentAccount() {
		return "pending", nil
	}
	if m.sendFn == nil {
		m.dropPending(randomID)
		publishDelivery(p.sessionID, strconv.FormatInt(p.clientID, 10), "failed", "Telegram 未連線")
		return "failed", nil
	}
	result, err := m.sendFn(ctx, &tg.MessagesSendMessageRequest{
		Peer:     &tg.InputPeerUser{UserID: p.peerID, AccessHash: p.access},
		Message:  p.text,
		RandomID: randomID,
	})
	if err != nil {
		// 定義性拒絕與連線失敗都標 failed：沒有落庫的識別可安全重送。
		if _, ok := tgerr.AsFloodWait(err); ok {
			slog.Warn(fmt.Sprintf("[telegram] 送出被限流: %v", err))
		}
		m.dropPending(randomID)
		if p.clientID > 0 {
			publishDelivery(p.sessionID, strconv.FormatInt(p.clientID, 10), "failed", "訊息傳送失敗，請重新發送")
		}
		return "failed", nil
	}
	if short, ok := result.(*tg.UpdateShortSentMessage); ok {
		m.finishSend(randomID, int64(short.ID), int64(short.Date))
		return "sent", nil
	}
	if err := m.handleUpdate(ctx, result); err != nil {
		return "", err
	}
	m.mu.Lock()
	_, still := m.pending[randomID]
	m.mu.Unlock()
	if still {
		// 沒有對上遠端 ID：當成失敗，讓使用者重送，避免同一句話在不明狀態下再送一次。
		m.dropPending(randomID)
		if p.clientID > 0 {
			publishDelivery(p.sessionID, strconv.FormatInt(p.clientID, 10), "failed", "訊息傳送失敗，請重新發送")
		}
		return "failed", nil
	}
	return "sent", nil
}

// bindSendID 處理 UpdateMessageID：用 random_id 找到待傳氣泡並換成真實訊息 ID。
func (m *Manager) bindSendID(randomID, remoteID int64) {
	if remoteID <= 0 {
		return
	}
	m.mu.Lock()
	p, ok := m.pending[randomID]
	m.mu.Unlock()
	if !ok || p.account != m.currentAccount() {
		return
	}
	m.finishSend(randomID, remoteID, time.Now().Unix())
}

func (m *Manager) finishSend(randomID, remoteID, date int64) {
	m.mu.Lock()
	p, ok := m.pending[randomID]
	if ok {
		delete(m.pending, randomID)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	replaces := ""
	if p.clientID > 0 {
		replaces = strconv.FormatInt(p.clientID, 10)
	}
	msg := &db.Message{
		ID:        remoteID,
		SessionID: p.sessionID,
		Role:      "user",
		Content:   p.text,
		Status:    db.MessageStatusDone,
		CreatedAt: time.Unix(date, 0).UTC().Format("2006-01-02 15:04:05"),
		Telegram: &db.TelegramRef{
			AccountID: p.account,
			PeerID:    strconv.FormatInt(p.peerID, 10),
			MessageID: remoteID,
		},
		Delivery: "sent",
	}
	publishMessage(p.sessionID, msg, replaces)
}

func (m *Manager) dropPending(randomID int64) {
	m.mu.Lock()
	delete(m.pending, randomID)
	m.mu.Unlock()
}

// currentAccount 回傳目前已授權帳號；尚未取得時回傳空字串。
func (m *Manager) currentAccount() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.accountID == 0 {
		return ""
	}
	return strconv.FormatInt(m.accountID, 10)
}
