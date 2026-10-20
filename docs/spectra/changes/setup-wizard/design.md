## Context

目前沒有 `config.yaml` 時，`config.Load` 回傳「讀取 config.yaml 失敗」，`server.Start` 失敗，桌面版只彈出訊息框後結束。使用者必須手動複製 `config.example.yaml` 並編輯。`bot_token` 在 `no_auth` 為 false 時永遠必填，即使只用桌面版或網頁密碼登入。

現況的關鍵事實（設計依據）：

- `config.Load` 用全域 viper，設定檔搜尋路徑是目前工作目錄 `.`。
- `server.Start` 一次完成：讀設定、開資料庫、載入代理相關服務、註冊所有路由、綁定連接埠，回傳 `*Server`（有 `URL()`、`Wait()`、`Shutdown()`）。桌面版用 `started.URL()` 建立視窗，並在關閉時呼叫 `Shutdown`。
- 桌面版的 `chdirToExeIfConfigured` 只在 exe 旁邊已經有 `config.yaml` 時才切換工作目錄。
- `authMiddleware` 的 Telegram 路徑以 `tg.Verify(initData, cfg.BotToken)` 驗證簽章；`tg.Verify` 以 Bot Token 推導 HMAC 金鑰。**Token 為空時金鑰是公開可計算的**，任何人都能偽造出通過簽章驗證的 `initData`，再靠白名單 ID 通過授權。目前因為 Token 必填而不會發生。
- 通知函式在 `botToken` 為空時直接略過；網頁密碼登入（`/auth/login`）不依賴 `bot_token`。
- 各代理 CLI 的解析規則不一：`codex` 用 `CODEX_BIN`、PATH 與 Windows 預設安裝路徑（`codex.ResolveBin`）；`agy` 用 `CC_AGY_BIN` 或 PATH；`claude`、`cursor-agent`、`kiro-cli` 用 PATH。

## Goals / Non-Goals

**Goals：**

- 首次啟動（沒有 `config.yaml`）就能在畫面上完成設定並開始使用，不必手動編輯檔案。
- Telegram Mini App 是可選功能：不選就不需要 `bot_token`、白名單、網域。
- 設定頁在沒有任何驗證的狀態下也不會被區網裝置或網頁上的惡意腳本搶先設定。
- 顯示本機各代理 CLI 的可用性與版本，幫使用者知道下一步。

**Non-Goals：**

- 事後編輯設定頁、Tunnel 建立、`telegram_client`／`shell`／`notify`／`mcp_token` 的引導、自動安裝或登入代理。
- 連接埠變更後的行程內切換；已存在設定檔的使用者不會看到精靈。

## Decisions

### 只在「設定檔不存在」時進入設定模式

`config.Load` 對「找不到 `config.yaml`」回傳可辨識的錯誤（`ErrConfigMissing`），其他錯誤（檔案存在但 YAML 壞掉、欄位非法）維持原本的啟動失敗。這樣壞掉的設定檔不會被精靈悄悄當成「首次使用」而被覆寫。

### 設定模式是獨立、最小的 HTTP 服務

`server.Start` 偵測到 `ErrConfigMissing` 時啟動設定模式：只註冊前端靜態資源與 `/setup/*` 三支 API，不開資料庫、不啟動代理偵測與 MCP、不註冊其他路由；其他路徑回 503 `{code: "setup_required"}`。連接埠使用預設值 8080。設定模式回傳的仍是同一個 `*Server` 型別，桌面版與純 server 版都不需要分兩套啟動流程。

### 儲存後在行程內切換（連接埠不變時）

評估結果：可行。設定模式從未初始化資料庫、代理服務或其他全域狀態，所以「儲存後第一次完整啟動」與一般啟動等價；`config.Load` 在檔案寫入後再次讀取即可。流程：寫檔成功 → 關閉設定模式的監聽 → 以同一連接埠啟動正常模式 → 換進同一個 `*Server`（`Wait`、`Shutdown`、`URL` 維持有效）→ 前端導向首頁。

連接埠變更時不做行程內切換（桌面版的自我更新橋接綁定啟動當下的網址來源，換埠會使其失效），儲存後回報「需要重新啟動」並說明新網址。行程內切換失敗（例如資料庫路徑無法開啟）時，設定已寫入，回報錯誤原因並要求修正 `config.yaml` 後重新啟動；為降低此情況，儲存前先驗證資料庫路徑的目錄存在且可寫。

### 設定 API 的安全限制（三道，缺一不可）

1. **只接受本機連線**：以連線的實際遠端位址判斷是否為 loopback，不信任任何轉發標頭（設定模式下沒有可信代理）。
2. **Host 標頭檢查**：Host 必須是 `127.0.0.1`、`localhost` 或 `[::1]` 加上目前連接埠，防止 DNS rebinding。
3. **一次性設定憑證**：每次啟動隨機產生 32 位元組（十六進位字串），`/setup/cli-check` 與 `/setup/save` 必須在 `X-Setup-Token` 標頭帶上，用常數時間比較。憑證只出現在 log 的設定網址與桌面版視窗網址的查詢參數；前端讀到後立刻用 `history.replaceState` 從網址列移除，只留在記憶體。

沒有憑證就只有 loopback 保護，網頁上的惡意腳本可對 `http://127.0.0.1:8080/setup/save` 發 POST（CSRF）而搶先設定，所以憑證是必要的。

實作時加上兩道縱深防禦（不取代上面三道）：設定模式的監聽只綁 `127.0.0.1`，區網連不到；所有回應帶 `Referrer-Policy: no-referrer`，避免帶憑證的設定網址被當成 Referer 送給頁面載入的第三方腳本。純 server 版在遠端無桌面的機器上，使用者需用 SSH 連接埠轉發（`ssh -L 8080:127.0.0.1:8080 ...`）後，在自己電腦開啟 log 印出的網址；文件需寫明。

### 已有設定檔時設定 API 一律拒絕

正常模式下 `/setup/status` 回 `{needs_setup: false}`，`/setup/cli-check` 與 `/setup/save` 回 404。設定模式內 `save` 在寫檔前再檢查一次檔案是否存在，已存在回 409 `already_configured`。

### Telegram Mini App 開關 = `bot_token` 是否存在

不新增設定欄位：精靈關閉開關就不寫 `bot_token` 與 `whitelist_tg_ids`。`config.Load` 的驗證改為：`bot_token` 為空且 `no_auth` 為 false 時，必須有有效的 `web.password`（非空、不等於 `change-me`、至少 8 個字元），否則啟動失敗並說明。

**`bot_token` 為空時，`authMiddleware` 必須忽略 `initData`**（`X-Telegram-Init-Data` 標頭與 `initData` 查詢參數都視為未提供，繼續走後續驗證），不得呼叫 `tg.Verify`；否則會以公開可算的金鑰通過簽章驗證，白名單 ID 一旦非空就是認證繞過。

### 設定檔以最小欄位原子寫入

寫出的內容：開關開啟時有 `bot_token` 與 `whitelist_tg_ids`；一律有 `web.password`、`web.allowed_cidrs`、`server.port`、`db.path`；`no_auth` 明確寫 `false`。`web.allowed_cidrs` 預設只含本機（`127.0.0.0/8`、`::1/128`），精靈提供「允許區網裝置登入」選項，勾選時再加入三個私有網段。其餘欄位不寫，沿用程式預設值。

寫入方式：同目錄暫存檔寫完並同步後，以「目標已存在即失敗」的方式放到 `config.yaml`，不會留下寫到一半的檔案；Unix 權限 0600。Windows 不另設存取控制，沿用資料夾繼承的權限，文件需提醒。密碼以明文存放，與既有行為一致，不在此變更內改變。

### 工作目錄：設定檔寫在 exe 旁邊

桌面版沒有 `config.yaml` 時，若 exe 旁邊有 `config.example.yaml`（發布包的結構），啟動時切換工作目錄到 exe 旁邊，避免從捷徑啟動時把設定寫到別處。`go run` 的暫存 exe 旁邊沒有該檔，維持目前工作目錄。

### CLI 偵測沿用既有解析規則，僅提示不阻擋

新增 `internal/clicheck`，對五個代理各回傳 `{agent_type, label, command, found, path, version, install_url, note}`。`codex` 呼叫 `codex.ResolveBin`；`agy` 讀 `CC_AGY_BIN`（匯出現有的解析函式，不重複實作）；其餘用 PATH 查找。版本以 `--version` 執行取得，逾時 3 秒、並行、子行程使用 `proc.SysProcAttr()` 避免彈出黑窗；任何失敗只影響該項的 `found` 或 `version` 欄位，不使整體失敗。`cursor-agent` 的 `note` 說明官方命令是 `agent` 但本專案呼叫 `cursor-agent`。檢查結果不影響儲存。

## Implementation Contract

**API（僅設定模式，且必須通過三道安全限制；`/setup/status` 只需要前兩道）：**

- `GET /setup/status` → `{needs_setup: boolean, port: number, config_path: string}`；正常模式回 `{needs_setup: false}`。
- `GET /setup/cli-check` → `{tools: [{agent_type, label, command, found, path, version, install_url, note}]}`。
- `POST /setup/save`，請求 `{password, port, db_path, allow_lan, miniapp: {enabled, bot_token, whitelist_tg_ids}}`：
  - 200 `{ok: true, applied: "switched" | "restart_required", url}`。
  - 400 `{code: "invalid_input", field, error}`：密碼少於 8 字元或為 `change-me`、連接埠不在 1–65535、資料庫路徑的目錄不存在或不可寫、開啟 Mini App 但 `bot_token` 為空、白名單含非正整數。
  - 403 `{code: "setup_forbidden"}`：非本機、Host 不符、憑證缺少或錯誤。
  - 409 `{code: "already_configured"}`；500 `{code: "write_failed" | "start_failed", error}`。
- 設定模式的其他路徑：503 `{code: "setup_required"}`。

**前端：** `app.js` 啟動時先查 `/setup/status`；`needs_setup` 為真就顯示精靈，否則走原本流程。精靈步驟：歡迎與 CLI 檢查、登入密碼與連接埠與資料庫路徑（含「產生隨機密碼」，用瀏覽器 `crypto.getRandomValues`）、Telegram Mini App 開關（預設關；開啟後才出現 Token 與白名單欄位與網域說明）、確認並儲存。儲存成功且為 `switched` 時導向首頁登入，為 `restart_required` 時顯示新網址與重新啟動提示。表單驗證與請求內容組裝放在不依賴 DOM 的純函式，供 Node 測試。

**驗收：**

- `go test ./internal/config/ ./internal/clicheck/ ./internal/server/`，涵蓋：缺檔與壞檔的錯誤分類、`bot_token` 驗證、`bot_token` 為空時偽造的 `initData` 被拒（401）、非本機與錯誤 Host 與錯誤憑證被拒、已有設定檔時 404／409、寫檔的原子性與併發只有一次成功、儲存後切換並可用密碼登入。
- `node --test scripts/*.test.cjs` 含精靈表單邏輯測試。
- 手動：乾淨資料夾下桌面版與純 server 版各完整走一次（含 Mini App 關閉與開啟兩條路徑），記錄結果。

**範圍：** 範圍內為上述 API、精靈、設定驗證調整、CLI 偵測、桌面版工作目錄與視窗網址、文件；範圍外見 Non-Goals。

## Risks / Trade-offs

- [`bot_token` 可為空使 `initData` 簽章金鑰公開] → 空 Token 時完全忽略 `initData`，並用專屬測試防止回歸。
- [本機其他程式或網頁在設定模式下搶先設定] → loopback 限制、Host 檢查、一次性憑證；憑證只在 log 與桌面視窗網址出現。同一台電腦上能讀 log 的其他使用者仍可取得憑證，這在單機單使用者的使用情境下接受。
- [密碼以明文存在 `config.yaml`] → 沿用既有行為；Unix 設 0600，Windows 沿用資料夾權限並在文件提醒。
- [行程內切換失敗] → 儲存前先驗證資料庫路徑；失敗時設定已存在，錯誤訊息要求修正後重新啟動。
- [連接埠變更需重新啟動] → 明確提示，不嘗試半套切換。
- [前端為執行時 Babel，精靈檔案語法錯誤會使整頁無法載入] → 純邏輯放獨立檔並用 Node 測試，元件以 esbuild 解析驗證語法。
