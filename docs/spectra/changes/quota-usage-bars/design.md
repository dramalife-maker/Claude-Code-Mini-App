## Context

`quota.Snapshot` 目前只有 `display_text`（`FormatDisplay` 依 provider 組字串），前端 `QuotaBadge`／`QuotaSegments` 以 ` · ` 拆段並用 regex 取 `N%` 上色，手機晶片由 `formatQuotaCompact` 以啟發式縮短。各 fetcher 其實已先把用量正規化成 `usage.QuotaInfo`（含 `Windows`：kind、label、percent、used、limit、resets_at，以及 `Plan`），`display_text` 只是它的文字化結果，所以結構化資料不需要新的抓取。窗口 kind 現況：Claude／Codex 為 `session`、`weekly`；Cursor 為 `billing_total`、`billing_auto`、`billing_api`；Kiro 為 `credits`；Antigravity 為 `session`、`weekly`、`session_claude`、`weekly_claude`。

## Goals / Non-Goals

**Goals:**

- 頂欄用量顯示的寬度大幅縮短，Antigravity 不再撐滿一行。
- 版型依 runner 決定，格式由結構化資料驅動，不依賴文字 regex。
- 舊資料（沒有 `windows`）仍能顯示。

**Non-Goals:**

- 不改抓取來源、快取 TTL、刷新流程。
- 不改 `display_text` 內容。
- 不做使用者可調版型、歷史圖表、預警通知。

## Decisions

### 在 Snapshot 與 Payload 新增 windows 與 plan

`Snapshot`／`Payload` 新增 `windows`（每項：`key`、`label`、`percent`、`resets_at`、`used`、`limit`）與 `plan`，皆選填（omitempty）。`key` 直接取 `usage.QuotaWindow.Kind`，前端以 key 對應版型，不解析 label。由一個共用 helper 把 `usage.QuotaInfo` 轉成窗口清單，各 fetcher 呼叫它，避免五處各寫一份。替代方案：前端 regex 拆 `display_text` → 格式一變就壞，且 Antigravity 的 `3h 1m` 倒數無法可靠對回窗口，不採用。

### display_text 保留並作為 fallback

`display_text` 不移除、不改格式：既是舊前端與其他消費者（MCP、API）的契約，也是前端在 `windows` 缺失或為空時的 fallback，以及 tooltip 的完整文字。

### 版型由前端依 runner 的設定表決定

前端用一張以 agent type 為 key 的設定表描述版型，而非各元件寫 if：

- `claude`、`codex`：條 `session`（標籤 5h）、`weekly`（標籤 Week），各帶百分比。
- `cursor`：條 `billing_total`（標籤 Total）；`billing_auto`、`billing_api` 只進 tooltip。
- `kiro`、`kiroacp`：條 `credits`，旁邊顯示 `used/limit`；`plan` 進 tooltip。
- `antigravity`：兩組，Gemini（`session`、`weekly`）與 Claude/GPT（`session_claude`、`weekly_claude`），各組一個短標籤加兩條窄條，條上不放文字。

設定表中找不到對應 key 的窗口一律略過，不報錯；整張表對不到任何窗口時走文字 fallback。

### Antigravity 的數字與倒數放進 tooltip

四條窄條無法同時放數字；完整數字、各窗口重置倒數與完整文字放在 tooltip（沿用元素 `title`），不新增彈出元件。

### 重置倒數由前端用 resets_at 計算

`resets_at` 以後端既有的格式送出（`usage.ParseResetsAt` 能解析的字串）；前端只在 tooltip 顯示，若解析失敗就省略該窗口的倒數，不顯示錯誤。倒數不是即時更新，隨 quota 刷新或重繪時重算。

### 手機晶片顯示最高使用率窗口

手機 compact 晶片取所有窗口中 `percent` 最大者，畫一條小進度條加 `NN%`；點擊刷新的既有行為保留。沒有 `windows` 時沿用 `formatQuotaCompact`。

## Implementation Contract

**Behavior**

- 後端：每個 provider 的 `Snapshot` 與 `Payload` 帶 `windows`（取不到資料時為空或省略）；`plan` 有值才送。`display_text` 與現況逐字相同。
- 前端桌面：頂欄副標列依 runner 版型顯示進度條；顏色依窗口自己的 percent（80 以上橘、100 以上紅），資料 `stale` 時灰，`error` 時琥珀；滑過條顯示 tooltip，內容含各窗口名稱、百分比、重置倒數，以及 `display_text`。
- 前端手機：晶片顯示最高窗口的小進度條與百分比；點擊刷新。
- 刷新按鈕、`quotaRefreshing` 的旋轉、訪客不顯示 quota 的行為不變。

**Interface / data shape**

- `quota.Payload` 新增 `windows: [{key: string, label?: string, percent?: number, resets_at?: string, used?: number, limit?: number}]` 與 `plan?: string`，欄位皆 omitempty；Go 型別名稱為 `Window`。
- 窗口 key 取值與上面 Context 所列的 `usage.QuotaWindow.Kind` 一致；不在設定表內的 key 前端略過。
- 轉換 helper 輸入 `*usage.QuotaInfo`，輸出 `[]Window`，保留 `info.Windows` 順序，略過 `percent` 與 `used/limit` 都沒有的窗口。

**Failure modes**

- `windows` 缺失、空、或全部對不到版型 → 前端顯示現行 `display_text` 文字。
- `percent` 為空的窗口 → 該條不畫。
- `resets_at` 無法解析 → tooltip 省略該窗口倒數。
- 抓取失敗時行為不變：沿用上一份快取並帶 `error`。

**Acceptance criteria**

- Go 單元測試：給定 Claude、Codex、Cursor、Kiro、Antigravity 各一份 `usage.QuotaInfo`，轉出的 `windows` 的 key、順序、percent 與預期相符；`display_text` 與既有 `display_test.go` 斷言一致（既有斷言不得刪改）。
- `go build ./cmd/server`、`go test ./internal/quota/...` 通過。
- 手動：以暫存服務或在瀏覽器注入假的 quota 物件，確認五種 runner 的形態；Antigravity 的用量區在桌面 1280px 寬時為單行、不換行。
- 手動：移除 `windows` 後頂欄退回文字且無 console 錯誤。

**Scope**

- In：`internal/quota`（payload、helper、fetcher 填值、測試）、前端 quota 顯示元件與頂欄／手機晶片。
- Out：用量抓取、TTL、`display_text` 格式、使用者設定。

## Risks / Trade-offs

- [新舊前後端版本不一致] → `windows` 為選填、前端有 fallback，順序任一邊先上線都不壞。
- [Antigravity 窄條看不到數字] → tooltip 與顏色補足；若實測不夠用，再加最高值的小數字（另案）。
- [設定表寫死 runner 與 key，日後新增窗口要改前端] → 略過未知 key 不會壞，只是不顯示新窗口，可接受。
- [倒數不即時更新] → 只放 tooltip，需求低；要即時再加計時器。

## Open Questions

- 無。
