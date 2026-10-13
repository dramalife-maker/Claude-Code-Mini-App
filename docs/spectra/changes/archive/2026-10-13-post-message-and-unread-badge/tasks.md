## 1. MCP post_message 獨立工具實作

- [x] 1.1 在 internal/mcp/tools.go 與 registry.go 新增並註冊 post_message 工具（落實 Post message tool execution），直接寫入完成狀態訊息至資料庫並透過 WebSocket hub.Broadcast 廣播，且支援 notify: true 時透過 Telegram Bot 推播通知，以單元測試 TestPostMessageTool 驗證訊息成功寫入且目標 Session 不會被切換為 running 狀態。
- [x] 1.2 驗證當目標 Session 正在執行其他任務時，post_message 仍能即時將訊息寫入並推播至連線客戶端，不造成進程中斷或衝突，以單元測試 TestPostMessageWhileRunning 驗證並發寫入安全性。[after: 1.1]

## 2. 資料庫未讀訊息子查詢統計 (Unread Count Subquery)

- [x] 2.1 在 internal/db/session.go 的 Session 結構新增 UnreadCount int 欄位，並在 ListSessions() 查詢中加入計算自 last_read_at 以來的非使用者訊息子查詢（落實 Unread message count calculation），以單元測試 TestUnreadCountCalculation 驗證使用者發送的訊息不計入未讀，且新產生的 assistant/shell 訊息精確統計。[after: 1.1]
- [x] 2.2 在 internal/api/session.go 的 List API 確保序列化回傳包含 unread_count，以單元測試 TestSessionListReturnsUnreadCount 驗證 HTTP API 輸出符合預期資料格式。[after: 2.1]

## 3. 前端未讀數字徽章視覺與樂觀清零 (Unread Count Badge UI)

- [x] 3.1 在 internal/static/js/session/SessionView.js 與 internal/static/index.html 將會話列表項目的未讀狀態小圓點升級為聊天軟體風格的未讀數字徽章（落實 Unread count badge rendering in UI），在 unread_count > 0 時渲染紫色圓角膠囊角標（超過 99 則顯示 99+），並在收合側邊欄上呈現角標，以瀏覽器驗證徽章正常顯示且對齊正確。[after: 2.2]
- [x] 3.2 在 SessionView.js 實現使用者點擊進入會話或點擊全部標為已讀時的樂觀清零（optimistic update），立即將 unread_count 歸零並隱藏角標，以手動操作驗證點擊後無閃爍且標題未讀總數同步更新。[after: 3.1]
