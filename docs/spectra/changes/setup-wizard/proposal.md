## Why

下載發布包的人第一次啟動時沒有 `config.yaml`，程式直接以「讀取 config.yaml 失敗」結束（桌面版只跳出一個訊息框），使用者必須自己讀 `docs/getting-started`、複製 `config.example.yaml`、手動編輯。對只想在本機使用桌面版或網頁的人，門檻太高。

另外，目前 `bot_token` 即使只用桌面版或網頁密碼登入也是必填（`config.Load` 在 `bot_token` 為空且 `no_auth` 為 false 時直接失敗）。大多數人沒有網域、也很少用 Telegram Mini App，只能填假的 Token，或改用 `no_auth: true`。後者仍監聽所有網路介面，不安全。

需要一個首次使用的設定精靈：沒有設定檔時引導使用者完成設定並存檔，Telegram Mini App 做成可關閉的選項，並順便檢查本機有哪些代理 CLI 可用。

## What Changes

- 沒有 `config.yaml` 時，程式進入「僅設定模式」：只提供設定精靈頁與設定 API，不開資料庫、不載入代理、不提供其他 API。桌面版直接顯示精靈；純 server 版在同一台電腦開瀏覽器進入。
- 精靈引導設定：網頁／桌面登入密碼（可一鍵產生隨機密碼，拒絕空白與範例值 `change-me`）、服務連接埠、資料庫路徑；不提供 `no_auth` 選項。
- 精靈內有「使用 Telegram Mini App」開關，預設關閉。關閉時不需要 `bot_token`、白名單與網域；開啟時才引導填 Bot Token 與白名單 Telegram ID，並說明需要可從外部連到的 HTTPS 網址（例如 Tunnel），但精靈不負責建立 Tunnel。
- 設定驗證調整：`bot_token` 為空時，只要有設定有效的網頁密碼即可啟動；`bot_token` 為空時 Telegram 驗證與通知自動停用。
- 精靈顯示本機 CLI 檢查結果：claude、codex、cursor-agent、kiro-cli、agy 是否找得到、版本，找不到時顯示安裝連結。檢查只是提示，不阻擋完成設定。
- 儲存時以最小必要欄位原子寫入 `config.yaml`；已有 `config.yaml` 時精靈不出現、設定 API 拒絕覆寫。儲存成功後，連接埠未變時在行程內直接切換到正常模式，連接埠有變時提示重新啟動。
- 安全：設定 API 只接受本機連線，並要求每次啟動隨機產生的一次性設定憑證，防止區網裝置或網頁上的惡意腳本搶先設定。
- 文件：`docs/getting-started` 與 README 的設定步驟改為以精靈為主、手動編輯為輔。

## Non-Goals

- 事後編輯設定的頁面：精靈只在沒有 `config.yaml` 時出現；既有使用者改設定仍手動編輯檔案。
- 建立或管理 Tunnel、申請網域。
- Telegram MTProto 個人帳號客戶端（`telegram_client`）、`shell`、`notify`、`mcp_token` 的設定引導：這些維持預設值，進階使用者自行編輯。
- 自動安裝代理 CLI 或替使用者登入代理。
- 連接埠變更後的行程內切換（只在連接埠不變時切換）。

## Capabilities

### New Capabilities

- `first-run-setup`: 沒有設定檔時的僅設定模式、本機限定與一次性憑證的安全限制、精靈引導、Telegram Mini App 開關、原子寫入設定檔、儲存後切換到正常模式，以及 `bot_token` 變為可選的設定驗證。
- `agent-cli-detection`: 偵測各代理 CLI 的可用性與版本，並回報安裝連結，供精靈顯示。

### Modified Capabilities

無。現有 spec 沒有涵蓋設定檔載入與啟動驗證的需求。

## Impact

- 設定與啟動：`internal/config/config.go`（缺檔錯誤的分類、`bot_token` 驗證、新增寫檔）、`internal/server/server.go`（`Start` 在缺設定檔時進入設定模式並可切換）、`cmd/desktop/main_windows.go` 與 `cmd/server/main.go`（設定模式的啟動與視窗網址）。
- 新增：設定 API（`/setup/status`、`/setup/cli-check`、`/setup/save`）、`internal/clicheck/`（CLI 偵測，沿用 `internal/codex` 的 `ResolveBin` 與 `CC_AGY_BIN` 等既有解析規則）。
- 前端：`internal/static/js/app.js` 啟動時查詢是否需要設定；新增 `internal/static/js/setup/` 精靈元件與純邏輯，`internal/static/index.html` 引入新檔（前端為執行時 Babel，無建置步驟，新檔會隨 production 建置的 `js/` 一併嵌入）。
- 文件：`docs/getting-started.zh-TW.md`、`docs/getting-started.md`、`README.md`、`README.zh-TW.md`、`config.example.yaml` 的註解。
- 行為相容性：既有已有 `config.yaml` 的使用者啟動行為不變；唯一放寬是 `bot_token` 為空但有有效網頁密碼時不再啟動失敗。
