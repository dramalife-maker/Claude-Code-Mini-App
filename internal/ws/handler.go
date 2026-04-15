package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"

	"claude-miniapp/internal/claude"
	"claude-miniapp/internal/db"
	"claude-miniapp/internal/tg"
)

// connRegistry 記錄每個 session 當前活著的連線
// key: sessionID, value: activeConn
var connRegistry sync.Map

type activeConn struct {
	token int64
	send  func(serverMsg) bool
}

// clearPendingDenials 清除 DB 中的待授權紀錄
func clearPendingDenials(database *db.DB, sessionID string) {
	if err := database.UpdatePendingDenials(sessionID, ""); err != nil {
		log.Printf("[ws] 清除 pending_denials 失敗: %v", err)
	}
}

const (
	StateIdle            = "IDLE"
	StateThinking        = "THINKING"
	StateStreaming       = "STREAMING"
	StateAwaitingConfirm = "AWAITING_CONFIRM"
)

type clientMsg struct {
	Type  string   `json:"type"`
	Data  string   `json:"data,omitempty"`
	Tools []string `json:"tools,omitempty"`
	Mode  string   `json:"mode,omitempty"`
}

type serverMsg struct {
	Type    string      `json:"type"`
	Value   string      `json:"value,omitempty"`
	Content string      `json:"content,omitempty"`
	Tools   interface{} `json:"tools,omitempty"`
}

func NewHandler(database *db.DB, botToken string) func(*fiberws.Conn) {
	return func(c *fiberws.Conn) {
		sessionID := c.Params("id")
		tgUserID, _ := c.Locals("tg_id").(int64)

		sess, err := database.GetSession(sessionID)
		if err != nil {
			log.Printf("[ws] session %s 不存在: %v", sessionID, err)
			c.Close()
			return
		}

		log.Printf("[ws] session %s 已連線 (claudeID=%q mode=%s)", sessionID, sess.ClaudeID, sess.PermissionMode)
		defer log.Printf("[ws] session %s 已斷線", sessionID)

		var mu sync.Mutex
		var cancelFn context.CancelFunc
		claudeID := sess.ClaudeID
		permMode := sess.PermissionMode
		allowedTools := sess.AllowedTools

		// 此連線的 send（直接寫入當前 c）
		send := func(msg serverMsg) bool {
			b, _ := json.Marshal(msg)
			return c.WriteMessage(1, b) == nil
		}

		// 註冊到 registry，用 token 確保斷線時不誤刪後來的連線
		token := time.Now().UnixNano()
		connRegistry.Store(sessionID, activeConn{token: token, send: send})
		defer func() {
			if v, ok := connRegistry.Load(sessionID); ok && v.(activeConn).token == token {
				connRegistry.Delete(sessionID)
			}
		}()

		// relaySend：查 registry，送給當前活著的連線（不限於本連線）
		relaySend := func(msg serverMsg) bool {
			if v, ok := connRegistry.Load(sessionID); ok {
				return v.(activeConn).send(msg)
			}
			return false
		}

		// 還原未處理的 pending_denials
		if sess.PendingDenials != "" {
			send(serverMsg{Type: "status", Value: StateAwaitingConfirm})
			send(serverMsg{Type: "permission_request", Tools: json.RawMessage(sess.PendingDenials)})
			log.Printf("[ws] 還原 pending_denials for session %s", sessionID)
		} else {
			send(serverMsg{Type: "status", Value: StateIdle})
		}

		// runClaude 啟動子進程並串流結果
		runClaude := func(prompt string) {
			mu.Lock()
			if cancelFn != nil {
				cancelFn()
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancelFn = cancel
			opts := claude.RunOptions{
				Prompt:         prompt,
				SessionID:      claudeID,
				WorkDir:        sess.WorkDir,
				PermissionMode: permMode,
				AllowedTools:   allowedTools,
			}
			mu.Unlock()

			log.Printf("[ws] 啟動 claude.Run claudeID=%q mode=%s", opts.SessionID, opts.PermissionMode)
			relaySend(serverMsg{Type: "status", Value: StateThinking})

			go func(opts claude.RunOptions) {
				streaming := false
				var responseBuf strings.Builder

				err := claude.Run(ctx, opts, func(e *claude.StreamEvent) {
					switch e.Type {
					case "stream_event":
						if e.Event == nil {
							return
						}
						switch e.Event.Type {
						case "content_block_start":
							if e.Event.ContentBlock != nil && e.Event.ContentBlock.Type == "text" && !streaming {
								streaming = true
								relaySend(serverMsg{Type: "status", Value: StateStreaming})
							}
						case "content_block_delta":
							if e.Event.Delta != nil && e.Event.Delta.Type == "text_delta" && e.Event.Delta.Text != "" {
								responseBuf.WriteString(e.Event.Delta.Text)
								relaySend(serverMsg{Type: "delta", Content: e.Event.Delta.Text})
							}
						}

					case "assistant":
						text := e.TextContent()
						if text != "" {
							log.Printf("[ws] assistant 整包回覆，長度=%d", len(text))
							if !streaming {
								streaming = true
								relaySend(serverMsg{Type: "status", Value: StateStreaming})
							}
							responseBuf.WriteString(text)
							relaySend(serverMsg{Type: "delta", Content: text})
						}

					case "result":
						mu.Lock()
						if e.SessionID != "" && e.SessionID != claudeID {
							claudeID = e.SessionID
							if err := database.UpdateClaudeID(sessionID, claudeID); err != nil {
								log.Printf("[ws] 更新 claude_id 失敗: %v", err)
							}
						}
						mu.Unlock()

						if resp := responseBuf.String(); resp != "" {
							if err := database.AddMessage(sessionID, "claude", resp); err != nil {
								log.Printf("[ws] 儲存 claude 訊息失敗: %v", err)
							}
						}

						if len(e.PermissionDenials) > 0 {
							if raw, err := json.Marshal(e.PermissionDenials); err == nil {
								if err := database.UpdatePendingDenials(sessionID, string(raw)); err != nil {
									log.Printf("[ws] 儲存 pending_denials 失敗: %v", err)
								}
							}
							relaySend(serverMsg{Type: "status", Value: StateAwaitingConfirm})
							relaySend(serverMsg{Type: "permission_request", Tools: e.PermissionDenials})
							go tg.Notify(botToken, tgUserID, fmt.Sprintf("⚠️ *%s* 需要授權確認，請開啟 App", sess.Name))
						} else {
							clearPendingDenials(database, sessionID)
							relaySend(serverMsg{Type: "status", Value: StateIdle})
							go tg.Notify(botToken, tgUserID, fmt.Sprintf("✅ *%s* 任務完成", sess.Name))
						}
					}
				})

				if err != nil {
					if ctx.Err() != nil {
						log.Println("[ws] claude.Run 被 context 取消")
					} else {
						log.Printf("[ws] claude.Run 執行錯誤: %v", err)
						relaySend(serverMsg{Type: "error", Content: err.Error()})
					}
					relaySend(serverMsg{Type: "status", Value: StateIdle})
				} else {
					log.Println("[ws] claude.Run 正常結束")
				}
			}(opts)
		}

		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				break
			}

			var msg clientMsg
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "input":
				log.Printf("[ws] 收到 input，prompt 長度=%d", len(msg.Data))
				if err := database.AddMessage(sessionID, "user", msg.Data); err != nil {
					log.Printf("[ws] 儲存 user 訊息失敗: %v", err)
				}
				runClaude(msg.Data)

			case "allow_once":
				clearPendingDenials(database, sessionID)
				mu.Lock()
				existing := make(map[string]bool)
				for _, t := range allowedTools {
					existing[t] = true
				}
				for _, t := range msg.Tools {
					existing[t] = true
				}
				merged := make([]string, 0, len(existing))
				for t := range existing {
					merged = append(merged, t)
				}
				allowedTools = merged
				mu.Unlock()
				if err := database.UpdateAllowedTools(sessionID, merged); err != nil {
					log.Printf("[ws] 更新 allowed_tools 失敗: %v", err)
				}
				runClaude("please retry the previous operation")

			case "set_mode":
				clearPendingDenials(database, sessionID)
				mu.Lock()
				permMode = msg.Mode
				mu.Unlock()
				if err := database.UpdatePermissionMode(sessionID, msg.Mode); err != nil {
					log.Printf("[ws] 更新 permission_mode 失敗: %v", err)
				}
				send(serverMsg{Type: "status", Value: StateIdle})
				log.Println("[ws] permission mode 切換為:", msg.Mode)

			case "reset_context":
				mu.Lock()
				if cancelFn != nil {
					cancelFn()
					cancelFn = nil
				}
				claudeID = ""
				mu.Unlock()
				if err := database.UpdateClaudeID(sessionID, ""); err != nil {
					log.Printf("[ws] 清除 claude_id 失敗: %v", err)
				}
				if err := database.ClearMessages(sessionID); err != nil {
					log.Printf("[ws] 清除訊息失敗: %v", err)
				}
				clearPendingDenials(database, sessionID)
				send(serverMsg{Type: "reset"})
				send(serverMsg{Type: "status", Value: StateIdle})
				log.Printf("[ws] session %s context 已重置", sessionID)

			case "interrupt":
				mu.Lock()
				if cancelFn != nil {
					cancelFn()
					cancelFn = nil
				}
				mu.Unlock()
				send(serverMsg{Type: "status", Value: StateIdle})
			}
		}
	}
}
