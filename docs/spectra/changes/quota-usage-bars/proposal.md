## Why

聊天頁頂欄的帳戶用量目前是後端組好的一長串文字，前端再用 regex 拆段上色。Antigravity 會輸出 `Gemini 5h 43% · 3h 1m · Gemini Week 16% · Claude/GPT 5h x% · … · Claude/GPT Week y%` 這種四個窗口加倒數的文字，一行就佔滿頂欄，在桌面副標列與手機都擠不下。文字格式各 runner 不同（Claude/Codex 的 5h 與 Week、Cursor 的 Total/Auto/API、Kiro 的 Credits 加方案、Antigravity 的兩組模型各兩個窗口），靠前端 regex 猜格式脆弱，改一個文案就壞。

## What Changes

- 後端 `quota` 快照新增結構化的 `windows`（每個窗口：key、label、percent、resets_at、used、limit）與 `plan`，隨既有的 API／WS／MCP quota payload 一起送出。`display_text` 保留不動。
- 前端新增依 runner 決定形態的精簡用量顯示，取代頂欄的長文字：
  - Claude、Codex：5h 與 Week 兩條細進度條，各帶百分比。
  - Cursor：只顯示 Total 一條，Auto／API 放進提示（tooltip）。
  - Kiro：Credits 一條，旁邊顯示已用／上限，方案放進提示。
  - Antigravity：兩組（Gemini、Claude/GPT）各兩條細條，條上不放文字，數字與重置倒數放進提示。
- 顏色門檻沿用現行規則：80% 以上橘、100% 以上紅、資料過期灰、取得失敗琥珀。
- 手機的精簡晶片改為「使用率最高的那個窗口」一條小進度條加百分比，取代 `formatQuotaCompact` 的文字啟發式。
- 後端沒送 `windows`（舊版後端或取不到資料）時，前端退回現行文字顯示，不報錯。

## Non-Goals

- 不改用量抓取方式（各 CLI／OAuth／API 來源不動）、不改 quota 快取 TTL 與刷新流程。
- `display_text` 的內容與格式不變，所有沿用它的文字輸出（API、WS、MCP）行為不變。
- 不做用量歷史圖表、不做額度預警通知。
- 不新增使用者可設定的版型選項；各 runner 的形態固定寫在前端。

## Capabilities

### New Capabilities

- `quota-usage-bars`: 後端 quota 快照的結構化窗口資料契約，以及前端依 runner 呈現精簡進度條（含手機晶片、顏色門檻、文字 fallback）的規則。

### Modified Capabilities

(無；既有 specs 皆未規範 quota 顯示。)

## Impact

- `internal/quota/result.go`：`Snapshot`、`Payload` 新增 `windows`、`plan` 欄位與轉換。
- `internal/quota/display.go` 與各 fetcher（`claude.go`、`codex.go`、`cursor.go`、`kiro.go`、`antigravity.go`）：由 `usage.QuotaInfo` 填入窗口資料，需保留 `display_text` 原輸出。
- `internal/quota/display_test.go` 或新增測試：窗口正規化的 Go 單元測試。
- `internal/static/js/ui/ui-atoms.js`、`internal/static/js/chat/chat-header.js`、`internal/static/js/core/core.js`：新增進度條元件、頂欄與手機晶片改用結構化資料，`formatQuotaCompact` 保留為 fallback。
- API／WS／MCP 的 quota payload 多兩個選填欄位，屬向下相容的新增，無依賴變更。
