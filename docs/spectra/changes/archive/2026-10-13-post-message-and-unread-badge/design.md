## Context

目前系統提供 MCP 供外部 Agent 與客戶端呼叫，但訊息傳遞僅有 `send_message` 與 `ask_session`，這兩個工具皆是作為使用者 Prompt 注入以喚醒目標 Session 的 Agent CLI 執行。外部 Agent 若僅是完成背景工作欲主動回報或推送通知至使用者的會話中，目前無工具可用，強行使用 `send_message` 會導致目標 Session 被強制喚起回答，產生浪費與循環。
同時，前端會話列表的未讀指示僅是一顆小圓點，無法像現代聊天軟體（Telegram/LINE/Slack）那樣讓使用者一目了然得知累積了幾則未讀訊息。

## Goals / Non-Goals

**Goals:**
- 實作 MCP `post_message` 工具，允許外部 Agent 主動寫入訊息（支援 assistant/system 角色、author 署名、Telegram notify）至指定 Session，透過 WebSocket 即時呈現於畫面，且絕不觸發目標 Session 的 Agent CLI 執行。
- 改造會話列表為聊天軟體風格的未讀數字徽章（Unread Count Badge），在後端以子查詢精確計算自 `last_read_at` 以來的非使用者訊息數量，並於前端呈現鮮明膠囊數字角標（含 99+），支援進入會話時即時樂觀清零。

**Non-Goals:**
- 不改變既有 `send_message` 與 `ask_session` 的語意與行為。
- 不修改既有 Agent CLI 的子進程呼叫與 stream-json 事件協議。
- 不引入外部通知佇列或額外第三方推播系統（通知沿用現有 Telegram Bot 機制）。

## Decisions

### 1. MCP post_message 獨立工具實作
- **決策**：在 `internal/mcp/tools.go` 與 `registry.go` 新增獨立工具 `post_message`，直接操作資料庫寫入訊息並觸發 WebSocket 廣播，而非走 loopback 發送 prompt。
- **原因**：若擴充 `send_message` 加入旗標容易破壞原有工具契約；新增獨立工具職責分明，且不經由 loopback WebSocket 送出 `prompt` 訊息，從根源保證不會觸發目標 Session 的 Agent 子進程。
- **替代方案評估**：曾考慮在 `send_message` 中新增 `trigger: false` 參數，但工具描述會變得晦澀，外部 LLM 調用時容易誤用。

### 2. 資料庫未讀訊息子查詢統計 (Unread Count Subquery)
- **決策**：在 `db.Session` 結構加入 `UnreadCount int`（JSON: `unread_count`），並在 `ListSessions()` 的 SQL 查詢中加入子查詢：
  ```sql
  (SELECT COUNT(*) FROM messages m
   WHERE m.session_id = s.id
     AND m.created_at > s.last_read_at
     AND m.role != 'user') AS unread_count
  ```
- **原因**：使用者自行發出的訊息不屬於「未讀」，只有 AI 回覆、Shell 輸出或外部 Agent 發布的訊息才計入未讀。透過 SQLite 子查詢在單次查詢中計算，避免 N+1 查詢。

### 3. 前端未讀數字徽章視覺與樂觀清零 (Unread Count Badge UI)
- **決策**：在 `SessionView.js` 將原有的狀態小圓點替換為 Telegram 風格的紫色圓角膠囊角標（`bg-violet-600 text-white`）。未讀數介於 1~99 顯示真實數字，超過 99 則顯示 `99+`。使用者點擊會話時，前端立即 optimistic update 將 `unread_count` 歸零。
- **原因**：與現代通訊軟體心智模型完全一致，大幅提升多會話背景運行時的感知度。

## Implementation Contract

- **MCP post_message 介面契約**：
  - 工具名稱：`post_message`
  - 輸入參數：
    - `session_id` (string, 必填)
    - `text` (string, 必填)
    - `role` (string, 選填，"assistant" 或 "system"，預設 "assistant")
    - `author` (string, 選填)
    - `notify` (bool, 選填，預設 false)
  - 成功回傳：`{ ok: true, id: <message_id> }`
  - 行為：直接呼叫 `db.AddMessage` 或 `db.CreatePendingMessage` 寫入完成的訊息，透過 `ws.hub.Broadcast` 廣播至該 session 的 WebSocket，並呼叫 `ws.NotifySessionsChanged()`。若 `notify=true` 且系統有綁定使用者與 BotToken，則調用 `tg.Notify` 送出推播。
- **Unread Count 資料形狀**：
  - `GET /sessions` 回傳的陣列中，每個 Session 物件均帶有 `unread_count: <number>` 欄位。
- **驗收準則**：
  - 測試 `TestMCPPostMessage` 證明 `post_message` 寫入成功後目標 Session 狀態維持為 `IDLE`。
  - 測試 `TestSessionUnreadCount` 證明未讀數精確匹配在 `last_read_at` 之後的非 user 訊息。
- **範疇界線**：
  - 範圍僅限於 `internal/mcp`、`internal/db`、`internal/api` 與前端 `SessionView.js`。

## Risks / Trade-offs

- [Risk] 大量歷史訊息時子查詢統計可能變慢 -> [Mitigation] `messages` 表已有 `idx_messages_session` 索引，若有需要可建立 `(session_id, created_at)` 覆蓋索引。
- [Risk] `notify: true` 推播失敗可能阻斷 tool call -> [Mitigation] Telegram 發送採用非阻斷或忽略錯誤（只記 log），不影響訊息寫入成功之結果。
