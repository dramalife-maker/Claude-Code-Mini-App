package desktopupdate

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// MessagePrefix 是前端 → Go 的 raw message 前綴；其他 wails: 或無前綴訊息不屬於本功能。
const MessagePrefix = "desktop-update:"

// Action 是前端可要求的固定動作；不接受任何 URL、路徑或指令參數。
type Action string

const (
	ActionCheck    Action = "check"
	ActionDownload Action = "download"
	ActionStatus   Action = "status"
	// ActionRestart 只在 ready 時受理，且必須先取得工作閘門（見 Manager.Restart）。
	ActionRestart Action = "restart"
)

var (
	// ErrUntrusted 表示訊息來源不是「主視窗的主框架首頁」。
	ErrUntrusted = errors.New("desktopupdate: untrusted bridge message")
	// ErrUnknownAction 表示 action 不在固定清單內（含夾帶參數）。
	ErrUnknownAction = errors.New("desktopupdate: unknown bridge action")
)

// Inbound 是桌面端收到的一則原始訊息與其來源資訊（來自 Wails OriginInfo 與視窗名稱）。
type Inbound struct {
	Message     string
	Window      string
	Origin      string
	TopOrigin   string
	IsMainFrame bool
}

// Authorize 判斷訊息是否來自主視窗首頁，並解析 action。
//
// Windows 上 Wails beta.26 不會設定 IsMainFrame，因此以 Origin 與 TopOrigin 皆等於首頁 origin
// 且路徑為 "/" 作為「主框架」的近似判斷；IsMainFrame 為 true 時同樣接受。
// 已知限制：若首頁內有同 URL 的子框架，兩者的 Origin/TopOrigin 會相同而無法區分。
func Authorize(home *url.URL, in Inbound) (Action, error) {
	if in.Window != "main" {
		return "", ErrUntrusted
	}
	if !in.IsMainFrame && in.Origin != in.TopOrigin {
		return "", ErrUntrusted
	}
	if !isHomeOrigin(home, in.Origin) || !isHomeOrigin(home, in.TopOrigin) {
		return "", ErrUntrusted
	}
	if !strings.HasPrefix(in.Message, MessagePrefix) {
		return "", ErrUnknownAction
	}
	switch a := Action(strings.TrimPrefix(in.Message, MessagePrefix)); a {
	case ActionCheck, ActionDownload, ActionStatus, ActionRestart:
		return a, nil
	default:
		return "", ErrUnknownAction
	}
}

// isHomeOrigin 只接受與首頁相同 scheme/host，且路徑為根目錄（首頁）的來源；
// 分享頁、子路徑、query、fragment 與 userinfo 一律不算首頁。
func isHomeOrigin(home *url.URL, raw string) bool {
	if home == nil || raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == home.Scheme && u.Host == home.Host &&
		(u.Path == "" || u.Path == "/") &&
		u.RawQuery == "" && u.Fragment == "" && u.User == nil
}

// Bridge 把已授權的前端訊息轉成 Manager 操作。
type Bridge struct {
	m    *Manager
	home *url.URL
}

// NewBridge 建立 bridge；homeURL 為主視窗首頁的完整 URL（例如伺服器的 started.URL()）。
// homeURL 無法解析時，所有訊息都會被拒絕。
func NewBridge(m *Manager, homeURL string) *Bridge {
	home, err := url.Parse(homeURL)
	if err != nil {
		home = nil
	}
	return &Bridge{m: m, home: home}
}

// Handle 同步處理一則訊息；呼叫端應在獨立 goroutine 中呼叫，避免阻塞 Wails 訊息迴圈。
// 被拒絕的訊息不會呼叫 Manager 的任何更新操作。
func (b *Bridge) Handle(in Inbound) error {
	action, err := Authorize(b.home, in)
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch action {
	case ActionCheck:
		return b.m.Check(ctx)
	case ActionDownload:
		return b.m.Download(ctx)
	case ActionStatus:
		b.m.Publish()
		return nil
	case ActionRestart:
		return b.m.Restart(ctx)
	}
	return ErrUnknownAction
}
