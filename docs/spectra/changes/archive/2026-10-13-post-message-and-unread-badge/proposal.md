## Why

目前系統的 MCP 僅提供 send_message 與 ask_session 工具，兩者均是模擬使用者輸入以喚醒目標 Session 的 Agent 執行，缺乏讓外部 Agent 主動回報進度或推送結果的工具；這會導致目標 Session 被強制喚起並浪費 Token，甚至引發互回迴圈。同時，前端會話列表的未讀狀態僅是一顆小圓點，無法像現代聊天軟體（Telegram/LINE/Slack）那樣直觀呈現未讀訊息數量。

## What Changes

- **新增 MCP `post_message` 工具**：外部 Agent 可主動向指定 Session 寫入訊息，支援指定角色（assistant/system）、署名（author）與可選 Telegram 推播（notify）。無論目標 Session 處於閒置或正在執行中皆可寫入，且絕不觸發目標 Session 的 Agent 子進程執行。
- **會話列表未讀訊息數量徽章（Unread Count Badge）**：後端在會話列表查詢中透過子查詢統計自 `last_read_at` 以來的非使用者訊息則數（`unread_count`）；前端會話卡片與收合側邊欄將原先單純的小圓點改造為鮮明的未讀數字角標（含超過 99 則時顯示 `99+`），並支援進入對話與全部標為已讀時的即時清零。

## Capabilities

### New Capabilities

- `mcp-post-message`: 提供 MCP post_message 工具，允許外部 Agent 主動將訊息（支援角色、署名、Telegram 推播）寫入指定 Session 並透過 WebSocket 即時廣播，不觸發目標 Session 的 Agent CLI 執行。
- `unread-message-badge`: 提供聊天軟體風格的未讀訊息數量徽章，計算每個會話自 last_read_at 以來的非使用者訊息數量，以數字角標呈現於會話卡片與側邊欄，並支援即時清零。

### Modified Capabilities

(none)

## Impact

- Affected code:
  - 后端：`internal/mcp/tools.go`, `internal/mcp/registry.go`, `internal/db/session.go`, `internal/api/session.go`
  - 前端：`internal/static/js/session/SessionView.js`, `internal/static/index.html` (CSS 樣式)
- Compatibility: 現有 API 與 WebSocket 通訊完全相容，既有 send_message 行為不受影響。
