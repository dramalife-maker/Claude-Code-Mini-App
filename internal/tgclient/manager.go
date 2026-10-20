package tgclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/ws"
)

// AuthState 代表 MTProto 客戶端目前認證狀態。
type AuthState string

const (
	AuthStateUnconfigured AuthState = "unconfigured"
	AuthStateDisconnected AuthState = "disconnected"
	AuthStateNeedPhone    AuthState = "need_phone"
	AuthStateNeedCode     AuthState = "need_code"
	AuthStateNeedPassword AuthState = "need_password"
	AuthStateAuthorized   AuthState = "authorized"
	AuthStateError        AuthState = "error"
)

// Manager 管理 Telegram MTProto 客戶端連線、認證與雙向訊息轉發。
type Manager struct {
	cfg    config.TelegramClient
	db     *db.DB
	client *telegram.Client
	cancel context.CancelFunc

	mu              sync.Mutex
	authState       AuthState
	lastAuthError   string
	phoneChan       chan string
	codeChan        chan string
	passwordChan    chan string
	activeSessionID string
	accountID       int64

	sessionTargets map[string]string // sessionID -> target
	userNames      map[int64]string  // userID -> Display Name / Username
	userIDs        map[string]int64  // target (lowercase) -> userID
	userAccess     map[int64]int64   // userID -> access hash（發送時組 InputPeerUser 用）
	sendFn         sendFunc
	historyFn      historyFunc // messages.getHistory 注入點（測試以假實作替代）
	uploadFn       uploadFunc
	sendMediaFn    sendMediaFunc
	fileFn         fileFunc
	pending        map[int64]memSend // random_id → 尚未對上遠端 ID 的送出（只在記憶體）
}

// setAccount 記錄目前授權的 Telegram 帳號；帳號變更時清除舊帳號的 peer 快取，避免跨帳號路由。
func (m *Manager) setAccount(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.accountID != id {
		m.userIDs = make(map[string]int64)
		m.userNames = make(map[int64]string)
		m.userAccess = make(map[int64]int64)
	}
	m.accountID = id
}

var defaultManager *Manager
var defaultManagerMu sync.Mutex

// SetDefaultManager 設定全域 Manager。
func SetDefaultManager(m *Manager) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()
	defaultManager = m
}

// GetDefaultManager 取得全域 Manager。
func GetDefaultManager() *Manager {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()
	return defaultManager
}

// NewManager 建立新的 Manager 實例。
func NewManager(cfg config.TelegramClient, database *db.DB) *Manager {
	m := &Manager{
		cfg:            cfg,
		db:             database,
		authState:      AuthStateUnconfigured,
		sessionTargets: make(map[string]string),
		userNames:      make(map[int64]string),
		userIDs:        make(map[string]int64),
		userAccess:     make(map[int64]int64),
	}
	m.sendFn = m.rpcSendMessage
	m.historyFn = m.rpcGetHistory
	m.uploadFn = m.rpcUpload
	m.sendMediaFn = m.rpcSendMedia
	m.fileFn = m.rpcSaveFile
	m.pending = map[int64]memSend{}
	ws.SetTelegramSyncState(m.sessionSyncState)
	ws.SetTelegramScopeResolver(m.syncScope)
	SetDefaultManager(m)
	return m
}

// Start 啟動 Telegram 連線與監聽 Goroutine。
func (m *Manager) Start(ctx context.Context) error {
	if !m.cfg.Enabled || m.cfg.APIID == 0 || strings.TrimSpace(m.cfg.APIHash) == "" {
		m.mu.Lock()
		m.authState = AuthStateUnconfigured
		m.mu.Unlock()
		slog.Info("[telegram] MTProto 客戶端未啟用（enabled=false 或未設定 api_id / api_hash）")
		return nil
	}

	sessionFile := m.cfg.SessionFile
	if sessionFile == "" {
		sessionFile = "./telegram.session"
	}

	m.client = telegram.NewClient(m.cfg.APIID, m.cfg.APIHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionFile},
		UpdateHandler:  telegram.UpdateHandlerFunc(m.handleUpdate),
	})

	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	go func() {
		defer cancel()
		for {
			select {
			case <-runCtx.Done():
				return
			default:
			}

			err := m.client.Run(runCtx, func(cctx context.Context) error {
				status, err := m.client.Auth().Status(cctx)
				if err != nil {
					return err
				}

				if !status.Authorized {
					m.mu.Lock()
					m.authState = AuthStateNeedPhone
					m.mu.Unlock()

					flow := auth.NewFlow(&authHelper{m: m}, auth.SendCodeOptions{})
					if err := flow.Run(cctx, m.client.Auth()); err != nil {
						m.mu.Lock()
						m.authState = AuthStateError
						m.lastAuthError = err.Error()
						m.mu.Unlock()
						return err
					}
				}

				m.mu.Lock()
				m.authState = AuthStateAuthorized
				active := m.activeSessionID
				m.mu.Unlock()

				self, err := m.client.Self(cctx)
				if err != nil {
					return err
				}
				m.setAccount(self.ID)
				slog.Info("[telegram] MTProto 客戶端已授權並開始監聽更新")
				go m.resolveAllSessionTargets(cctx)
				go m.announceAll()
				if active != "" {
					m.SendSystemMessage(active, "[系統] 🎉 Telegram 登入成功！已鎖定秘密通訊，可直接開始聊天。")
				}

				<-cctx.Done()
				return cctx.Err()
			})

			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn(fmt.Sprintf("[telegram] client.Run 結束: %v，5 秒後嘗試重連", err))
				m.mu.Lock()
				m.authState = AuthStateDisconnected
				m.lastAuthError = err.Error()
				m.mu.Unlock()
				m.announceAll() // 斷線：通知既有 WS 客戶端 sync_state 變 offline
				select {
				case <-time.After(5 * time.Second):
				case <-runCtx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// Stop 停止連線。
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// SetActiveSession 記錄最近使用者互動的 session。
func (m *Manager) SetActiveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeSessionID = sessionID
}

// RegisterSessionTarget 綁定 sessionID 與對象 target。
func (m *Manager) RegisterSessionTarget(sessionID, target string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionTargets[sessionID] = target
}

// SendSystemMessage 向 session 寫入系統提示訊息並推播給前端。
func (m *Manager) SendSystemMessage(sessionID string, content string) {
	if sessionID == "" || m.db == nil {
		return
	}
	msgID, createdAt, err := m.db.AddMessageWithID(sessionID, "claude", content, "System")
	if err != nil {
		slog.Warn(fmt.Sprintf("[telegram] 寫入系統訊息失敗: %v", err))
		return
	}
	_ = m.db.TouchSessionLastActive(sessionID)
	ws.BroadcastSessionMessage(sessionID, "message", content, "claude", "System", msgID, createdAt)
	ws.NotifySessionsChanged()
}

func (m *Manager) resolveTarget(ctx context.Context, target string) (*tg.InputPeerUser, error) {
	target = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(target), "@"))
	var resolved *tg.ContactsResolvedPeer
	var err error
	if strings.HasPrefix(target, "+") {
		resolved, err = m.client.API().ContactsResolvePhone(ctx, strings.TrimPrefix(target, "+"))
	} else {
		resolved, err = m.client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: target})
	}
	if err != nil {
		return nil, err
	}
	peer, ok := resolved.Peer.(*tg.PeerUser)
	if !ok {
		return nil, errors.New("Telegram 聊天對象必須是使用者，不能是群組或頻道")
	}
	m.recordUsers(resolved.Users)
	for _, uc := range resolved.Users {
		if user, ok := uc.(*tg.User); ok && user.ID == peer.UserID {
			m.mu.Lock()
			m.userIDs[target] = user.ID
			m.mu.Unlock()
			return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}, nil
		}
	}
	return nil, errors.New("Telegram 未回傳聊天對象資訊")
}

func (m *Manager) resolveAllSessionTargets(ctx context.Context) {
	if m.db == nil || m.client == nil {
		return
	}
	sessions, err := m.db.ListSessions()
	if err != nil {
		slog.Warn("[telegram] 讀取聊天對象失敗", "error", err)
		return
	}
	seen := make(map[string]bool)
	for _, s := range sessions {
		if s.AgentType != "telegram" {
			continue
		}
		target := m.sessionTarget(s)
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		if _, err := strconv.ParseInt(target, 10, 64); err == nil && !strings.HasPrefix(target, "+") {
			continue // Numeric IDs can be matched without resolving a username.
		}
		m.mu.Lock()
		cached := m.userIDs[target] != 0 && m.userAccess[m.userIDs[target]] != 0
		m.mu.Unlock()
		if cached {
			continue
		}
		resolveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err := m.resolveTarget(resolveCtx, target)
		cancel()
		if err != nil {
			slog.Warn("[telegram] 解析聊天對象失敗", "error", err)
		}
	}
}

func (m *Manager) sessionTarget(s *db.Session) string {
	m.mu.Lock()
	target := m.sessionTargets[s.ID]
	m.mu.Unlock()
	if target == "" && strings.HasPrefix(s.Description, "tg_target:") {
		target = strings.TrimPrefix(s.Description, "tg_target:")
	}
	if target == "" {
		target = m.cfg.DefaultTarget
	}
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(target), "@"))
}

func (m *Manager) recordUsers(users []tg.UserClass) {
	if len(users) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, uc := range users {
		if user, ok := uc.(*tg.User); ok {
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name == "" && user.Username != "" {
				name = "@" + user.Username
			}
			if name != "" {
				m.userNames[user.ID] = name
			}
			if user.Username != "" {
				m.userIDs[strings.ToLower(user.Username)] = user.ID
			}
			if user.AccessHash != 0 {
				m.userAccess[user.ID] = user.AccessHash
			}
		}
	}
}

// processMessage 把一則私聊更新轉成 DTO 推給已綁定的 session。不寫資料庫。
func (m *Manager) processMessage(msgClass tg.MessageClass) {
	msg, ok := msgClass.(*tg.Message)
	if !ok || m.currentAccount() == "" {
		return
	}
	peer, ok := msg.PeerID.(*tg.PeerUser)
	if !ok || peer.UserID <= 0 || msg.ID <= 0 || msg.Date <= 0 {
		return
	}
	text := telegramText(msg)
	if text == "" {
		return
	}
	role, author := "claude", m.peerDisplayName(peer.UserID)
	if msg.Out {
		role, author = "user", ""
	}
	dto := &db.Message{
		ID:        int64(msg.ID),
		Role:      role,
		Content:   text,
		Status:    db.MessageStatusDone,
		CreatedAt: time.Unix(int64(msg.Date), 0).UTC().Format("2006-01-02 15:04:05"),
		Author:    author,
		Telegram: &db.TelegramRef{
			AccountID: m.currentAccount(),
			PeerID:    strconv.FormatInt(peer.UserID, 10),
			MessageID: int64(msg.ID),
		},
		Delivery: "sent",
	}
	fillMedia("", msg, dto)
	m.publishLive(dto, "")
}

// handleUpdate 處理來自 Telegram 的訊息更新。只轉發已綁定私聊，不落庫。
func (m *Manager) handleUpdate(_ context.Context, u tg.UpdatesClass) error {
	slog.Info(fmt.Sprintf("[telegram] 收到 Telegram 更新: %T", u))
	switch upd := u.(type) {
	case *tg.UpdateShort:
		if inner, ok := upd.Update.(*tg.UpdateNewMessage); ok {
			m.processMessage(inner.Message)
		}

	case *tg.UpdateShortMessage:
		m.processMessage(&tg.Message{
			ID:      upd.ID,
			PeerID:  &tg.PeerUser{UserID: upd.UserID},
			Out:     upd.Out,
			Message: upd.Message,
			Date:    upd.Date,
		})

	case *tg.Updates:
		m.recordUsers(upd.Users)
		m.processUpdateList(upd.Updates)

	case *tg.UpdatesCombined:
		m.recordUsers(upd.Users)
		m.processUpdateList(upd.Updates)

	default:
		slog.Debug(fmt.Sprintf("[telegram] 略過非訊息更新: %T", u))
	}
	return nil
}

func (m *Manager) processUpdateList(updates []tg.UpdateClass) {
	for _, update := range updates {
		switch upd := update.(type) {
		case *tg.UpdateNewMessage:
			m.processMessage(upd.Message)
		case *tg.UpdateMessageID:
			m.bindSendID(upd.RandomID, int64(upd.ID))
		}
	}
}

// boundSessionIDs 回傳所有綁定到該 PeerUser 的 Telegram 會話；不回退到活躍會話或其他對象。
func (m *Manager) boundSessionIDs(peerID int64) ([]string, error) {
	sessions, err := m.db.ListSessions()
	if err != nil {
		slog.Warn("[telegram] 讀取會話失敗", "error", err)
		return nil, err
	}
	numeric := strconv.FormatInt(peerID, 10)
	var ids []string
	for _, s := range sessions {
		if s.AgentType != "telegram" {
			continue
		}
		target := m.sessionTarget(s)
		if target == "" {
			continue
		}
		m.mu.Lock()
		targetID := m.userIDs[target]
		m.mu.Unlock()
		if targetID == peerID || target == numeric {
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}

func (m *Manager) peerDisplayName(peerID int64) string {
	m.mu.Lock()
	name := m.userNames[peerID]
	m.mu.Unlock()
	if name == "" {
		return strconv.FormatInt(peerID, 10)
	}
	return name
}

// authHelper 實作 auth.UserAuthenticator 介面，透過聊天框交互完成首次登入。
type authHelper struct {
	m *Manager
}

func (h *authHelper) Phone(ctx context.Context) (string, error) {
	h.m.mu.Lock()
	h.m.authState = AuthStateNeedPhone
	h.m.phoneChan = make(chan string, 1)
	activeSession := h.m.activeSessionID
	h.m.mu.Unlock()

	if activeSession != "" {
		h.m.SendSystemMessage(activeSession, "[系統] 首次使用，請在輸入框輸入你的 Telegram 註冊電話號碼（例如 +886912345678）：")
	}

	select {
	case p := <-h.m.phoneChan:
		return strings.TrimSpace(p), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (h *authHelper) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	h.m.mu.Lock()
	h.m.authState = AuthStateNeedCode
	h.m.codeChan = make(chan string, 1)
	activeSession := h.m.activeSessionID
	h.m.mu.Unlock()

	if activeSession != "" {
		h.m.SendSystemMessage(activeSession, "[系統] 驗證碼已發送至您的 Telegram，請在輸入框輸入 5 位數驗證碼：")
	}

	select {
	case c := <-h.m.codeChan:
		return strings.TrimSpace(c), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (h *authHelper) Password(ctx context.Context) (string, error) {
	h.m.mu.Lock()
	h.m.authState = AuthStateNeedPassword
	h.m.passwordChan = make(chan string, 1)
	activeSession := h.m.activeSessionID
	h.m.mu.Unlock()

	if activeSession != "" {
		h.m.SendSystemMessage(activeSession, "[系統] 請在輸入框輸入您的 Telegram 兩步驟驗證密碼（2FA Password）：")
	}

	select {
	case pwd := <-h.m.passwordChan:
		return strings.TrimSpace(pwd), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (h *authHelper) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (h *authHelper) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signup not supported; please register on Telegram first")
}
