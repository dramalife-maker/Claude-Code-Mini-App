package tgclient

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

const (
	// 以下為 API 契約常數，不是效能量測值。
	DefaultHistoryLimit = 50
	MaxHistoryLimit     = 100

	SyncStateReady   = "ready"
	SyncStateOffline = "offline"

	historyRPCTimeout = 20 * time.Second
)

// 呼叫端（API 層）以 errors.Is／As 對應 HTTP 狀態，不依錯誤文字判斷。
var (
	ErrHistoryInvalidLimit    = errors.New("history limit must be 1..100")      // 400
	ErrHistoryInvalidCursor   = errors.New("invalid history cursor")            // 400
	ErrHistoryNotTelegram     = errors.New("session is not a telegram session") // 400
	ErrHistorySessionNotFound = errors.New("session not found")                 // 404
	ErrHistoryNoBinding       = errors.New("telegram session has no chat target")
	ErrHistoryUnavailable     = errors.New("telegram history unavailable") // 503
)

// HistoryRateLimitedError 對應 FLOOD_WAIT（429）；RetryAfter 為上游要求的等待時間。
type HistoryRateLimitedError struct{ RetryAfter time.Duration }

func (e *HistoryRateLimitedError) Error() string {
	return fmt.Sprintf("telegram rate limited, retry after %s", e.RetryAfter)
}

// HistoryPageResult 是 pagination=telegram 的回應主體。訊息不落庫。
type HistoryPageResult struct {
	Messages   []*db.Message `json:"messages"`
	NextCursor *string       `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
	SyncState  string        `json:"sync_state"`
}

// historyFunc 是 messages.getHistory 的注入點。
type historyFunc func(ctx context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error)

// rpcGetHistory 是正式環境的 historyFunc。
func (m *Manager) rpcGetHistory(ctx context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
	if m.client == nil {
		return nil, errTGNotConnected
	}
	return m.client.API().MessagesGetHistory(ctx, req)
}

// parseHistoryCursor 解出排他的遠端訊息 ID（只取 ID < before）。游標就是遠端訊息 ID，不帶 peer。
func parseHistoryCursor(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 || n > math.MaxInt32 {
		return 0, ErrHistoryInvalidCursor
	}
	return n, nil
}

// HistoryPage 直接向 Telegram 取一頁歷史並轉成 DTO，不寫入資料庫。
// limit 0＝預設 50，允許 1–100；beforeCursor 為空代表最新頁。每次只查這一頁。
// 未登入或上游失敗回 ErrHistoryUnavailable，不把未知終點當成空頁。
func (m *Manager) HistoryPage(ctx context.Context, sessionID string, limit int, beforeCursor string) (*HistoryPageResult, error) {
	if limit == 0 {
		limit = DefaultHistoryLimit
	}
	if limit < 1 || limit > MaxHistoryLimit {
		return nil, ErrHistoryInvalidLimit
	}
	if m.db == nil {
		return nil, ErrHistoryUnavailable
	}
	s, err := m.db.GetSession(sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHistorySessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.AgentType != agent.TypeTelegram {
		return nil, ErrHistoryNotTelegram
	}

	var before int64
	if beforeCursor != "" {
		if before, err = parseHistoryCursor(beforeCursor); err != nil {
			return nil, err
		}
	}
	b, err := m.liveHistoryPeer(ctx, s)
	if err != nil {
		return nil, err
	}
	return m.fetchHistoryPage(ctx, sessionID, b, limit, before)
}

// livePeer 是目前已授權帳號解析出的私聊對象。游標不能改它。
type livePeer struct {
	account    string
	peerID     int64
	accessHash int64
}

// liveHistoryPeer 只接受目前已授權帳號與 session 綁定的 PeerUser。離線沒有快取頁。
func (m *Manager) liveHistoryPeer(ctx context.Context, s *db.Session) (livePeer, error) {
	target := m.sessionTarget(s)
	if target == "" {
		return livePeer{}, ErrHistoryNoBinding
	}
	m.mu.Lock()
	authorized := m.authState == AuthStateAuthorized && m.accountID != 0
	account := strconv.FormatInt(m.accountID, 10)
	m.mu.Unlock()
	if !authorized {
		return livePeer{}, ErrHistoryUnavailable
	}
	peerID, hash, err := m.historyPeer(ctx, target)
	if err != nil || peerID == 0 || hash == 0 {
		return livePeer{}, ErrHistoryUnavailable
	}
	return livePeer{account: account, peerID: peerID, accessHash: hash}, nil
}

// historyPeer 只回傳 session target 對應的 PeerUser；快取缺 access hash 時才向 Telegram 解析。
func (m *Manager) historyPeer(ctx context.Context, target string) (int64, int64, error) {
	m.mu.Lock()
	id := m.userIDs[target]
	m.mu.Unlock()
	if id == 0 {
		if n, err := strconv.ParseInt(target, 10, 64); err == nil && n > 0 {
			id = n
		}
	}
	m.mu.Lock()
	hash := m.userAccess[id]
	m.mu.Unlock()
	if id != 0 && hash != 0 {
		return id, hash, nil
	}
	if m.client == nil {
		return 0, 0, ErrHistoryUnavailable
	}
	peer, err := m.resolveTarget(ctx, target)
	if err != nil {
		return 0, 0, err
	}
	return peer.UserID, peer.AccessHash, nil
}

// fetchHistoryPage 對上游查詢恰好一頁，轉成 DTO 後回傳。不寫資料庫。
func (m *Manager) fetchHistoryPage(ctx context.Context, sessionID string, b livePeer, limit int, before int64) (*HistoryPageResult, error) {
	if m.historyFn == nil {
		return nil, ErrHistoryUnavailable
	}
	rctx, cancel := context.WithTimeout(ctx, historyRPCTimeout)
	defer cancel()
	resp, err := m.historyFn(rctx, &tg.MessagesGetHistoryRequest{
		Peer:     &tg.InputPeerUser{UserID: b.peerID, AccessHash: b.accessHash},
		OffsetID: int(before), // 排他：上游只回 ID < offset_id；0＝最新
		Limit:    limit,
	})
	if err != nil {
		if d, ok := tgerr.AsFloodWait(err); ok {
			return nil, &HistoryRateLimitedError{RetryAfter: d}
		}
		return nil, fmt.Errorf("%w: %v", ErrHistoryUnavailable, err)
	}
	// 請求途中換了帳號：這頁屬於舊帳號，不能交給現在的連線。
	if m.currentAccount() != b.account {
		return nil, fmt.Errorf("%w: account changed during request", ErrHistoryUnavailable)
	}

	var raw []tg.MessageClass
	var users []tg.UserClass
	complete := false // messages.messages＝上游宣告沒有更多
	switch r := resp.(type) {
	case *tg.MessagesMessages:
		raw, users, complete = r.Messages, r.Users, true
	case *tg.MessagesMessagesSlice:
		raw, users = r.Messages, r.Users
	default:
		return nil, fmt.Errorf("%w: unexpected response %T", ErrHistoryUnavailable, resp)
	}
	m.recordUsers(users)

	msgs, lo, err := scanHistoryDTO(sessionID, b, raw, before)
	if err == nil {
		for _, msg := range msgs {
			if msg.Role != "user" {
				msg.Author = m.peerDisplayName(b.peerID)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 && lo == 0 {
		return nil, fmt.Errorf("%w: page made no progress", ErrHistoryUnavailable)
	}
	// 到頂只在上游確認後成立：空回應、messages.messages，或已掃到 ID 1。
	end := lo == 0 || complete || lo <= 1
	res := &HistoryPageResult{Messages: msgs, HasMore: !end, SyncState: SyncStateReady}
	if res.Messages == nil {
		res.Messages = []*db.Message{}
	}
	if res.HasMore {
		c := strconv.FormatInt(lo, 10)
		res.NextCursor = &c
	}
	return res, nil
}

// scanHistoryDTO 把一頁 getHistory 轉成正序 DTO。
// service／空訊息不出現在 messages，但它們的遠端 ID 仍推進游標，避免空頁卡住。
func scanHistoryDTO(sessionID string, b livePeer, raw []tg.MessageClass, before int64) ([]*db.Message, int64, error) {
	peerKey := strconv.FormatInt(b.peerID, 10)
	var msgs []*db.Message
	var lo int64
	for _, mc := range raw {
		id := int64(mc.GetID())
		if id <= 0 || (before > 0 && id >= before) {
			continue
		}
		var peer tg.PeerClass
		var visible *db.Message
		switch x := mc.(type) {
		case *tg.Message:
			peer = x.PeerID
			if text := telegramText(x); text != "" {
				role, author := "claude", ""
				if x.Out {
					role = "user"
				}
				visible = &db.Message{
					ID:        id,
					SessionID: sessionID,
					Role:      role,
					Content:   text,
					Status:    db.MessageStatusDone,
					CreatedAt: time.Unix(int64(x.Date), 0).UTC().Format("2006-01-02 15:04:05"),
					Author:    author,
					Telegram:  &db.TelegramRef{AccountID: b.account, PeerID: peerKey, MessageID: id},
					Delivery:  "sent",
				}
				fillMedia(sessionID, x, visible)
			}
		case *tg.MessageService:
			peer = x.PeerID
		}
		if p, ok := peer.(*tg.PeerUser); peer != nil && (!ok || p.UserID != b.peerID) {
			return nil, 0, fmt.Errorf("%w: response contains a different peer", ErrHistoryUnavailable)
		}
		if lo == 0 || id < lo {
			lo = id
		}
		if visible != nil {
			msgs = append(msgs, visible)
		}
	}
	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].CreatedAt != msgs[j].CreatedAt {
			return msgs[i].CreatedAt < msgs[j].CreatedAt
		}
		return msgs[i].ID < msgs[j].ID
	})
	return msgs, lo, nil
}

// telegramText 取出可顯示文字。純媒體先用既有佔位，真正的附件在媒體路由補上。
func telegramText(msg *tg.Message) string {
	if strings.TrimSpace(msg.Message) != "" {
		return msg.Message
	}
	if msg.Media != nil {
		return "[檔案/多媒體]"
	}
	return ""
}
