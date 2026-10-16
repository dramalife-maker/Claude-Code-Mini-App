## 1. 後端結構化窗口

- [x] 1.1 Quota payload carries structured windows：`quota.Snapshot` 與 `Payload` 新增選填的 `windows`（key、label、percent、resets_at、used、limit）與 `plan`，並有一個共用 helper 把 `usage.QuotaInfo` 轉成窗口清單（保留順序、略過沒有 percent 也沒有 used/limit 的窗口）；`ToPayload` 帶出新欄位，`display_text` 不變。驗證：新增 Go 單元測試，對 Claude、Codex、Cursor、Kiro、Antigravity 各一份 `usage.QuotaInfo` 斷言 key 順序與 percent；`go test ./internal/quota/...` 通過且既有 `display_test.go` 斷言未改。
- [x] 1.2 五個 fetcher 填入窗口資料 [after: 1.1]：Claude、Codex、Cursor、Kiro、Antigravity 的 fetcher 建立 `Snapshot` 時呼叫 helper 填 `windows` 與 `plan`，取不到資料的路徑維持 `windows` 空。驗證：`go build ./cmd/server` 通過；用 `get_quota`（MCP）或 GET quota API 實際看 claude 的 payload 含 `windows` 且 `display_text` 與先前相同。

## 2. 前端進度條元件

- [x] 2.1 Header renders runner-specific usage bars（依「版型由前端依 runner 的設定表決定」）：在 `ui-atoms.js` 新增依 runner 設定表（claude/codex 兩條、cursor 只 Total、kiro Credits 加 used/limit、antigravity 兩組各兩條窄條無文字）渲染的用量元件，未知 key 略過。驗證：在瀏覽器注入五種 runner 的假 quota 物件，逐一確認形態；Antigravity 在 1280px 寬時用量區單行不換行。
- [x] 2.2 Usage bar color follows thresholds and state [after: 2.1]：每條依自己的 percent 套 80 橘、100 紅，`stale` 灰、`error` 琥珀。驗證：注入 percent 85 與 100 的窗口，確認顏色分別為橘與紅。
- [x] 2.3 Tooltip shows full usage detail [after: 2.1]（依「Antigravity 的數字與倒數放進 tooltip」與「重置倒數由前端用 resets_at 計算」）：用量區的 `title` 列出各窗口名稱、百分比、可解析時的重置倒數，最後附完整 `display_text`；`resets_at` 解析失敗時只省略該窗口倒數。驗證：注入有效與無效的 `resets_at`，滑過確認 tooltip 內容，console 無錯誤。

## 3. 整合與 fallback [after: 2.3]

- [x] 3.1 頂欄桌面副標列改用新元件：`ChatSessionHeader` 的 quota 區改渲染新元件，刷新鈕與 `quotaRefreshing` 旋轉、訪客不顯示的行為維持。驗證：開啟 claude 與 antigravity 的 session，頂欄顯示進度條，按刷新鈕仍會刷新。
- [x] 3.2 Mobile chip shows the highest window（依「手機晶片顯示最高使用率窗口」）：手機 compact 晶片取 percent 最大的窗口畫小進度條加百分比，點擊仍刷新；沒有 `windows` 時沿用 `formatQuotaCompact`。驗證：375px 寬注入 percent 43 與 16，確認晶片顯示 43%，點擊觸發刷新。
- [x] 3.3 Text fallback when windows are missing（依「display_text 保留並作為 fallback」）：`windows` 缺失、空或全部是未知 key 時顯示原本的文字段落。驗證：注入只有 `display_text` 的 quota，頂欄顯示原文字且 console 無錯誤。

## 4. 收尾 [after: 3.3]

- [x] 4.1 整體驗收：`go build ./cmd/server` 與 `go test ./internal/quota/...` 通過；以暫存埠起服務，五種 runner 的形態與 fallback 手動走過一遍，暫存服務結束後關閉。驗證：上述指令輸出與手動結果記錄在回報中。
