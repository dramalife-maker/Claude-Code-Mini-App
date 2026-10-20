## 1. 設定載入與認證前提

- [x] 1.1 完成 Setup mode only when the config file is missing 的偵測與 Telegram Mini App is optional 的設定驗證：`config.Load` 對「找不到 `config.yaml`」回傳可用 `errors.Is` 辨識的 `ErrConfigMissing`，檔案存在但 YAML 壞掉或欄位非法仍回原本的錯誤；`bot_token` 為空且 `no_auth` 為 false 時，只要 `web.password` 有效（非空、不等於 `change-me`、至少 8 個字元）就通過，否則錯誤訊息指出「需要 bot_token 或 web.password」，`no_auth: true` 行為不變。密碼規則放在可被精靈驗證共用的單一函式。驗證：`internal/config/config_test.go` 新增 `TestLoadMissingConfig`（空目錄得 `ErrConfigMissing`、壞 YAML 不是）與 `TestLoadBotTokenOptional`（僅密碼通過、兩者皆無失敗、`change-me` 失敗、`no_auth` 且 Token 為空通過），既有 `TestTrustedProxiesConfig` 不變。
- [x] 1.2 [after: 1.1] 完成 Telegram Mini App is optional 的認證面（空 Token 不接受 `initData`）：`bot_token` 為空時，`authMiddleware` 完全忽略 `X-Telegram-Init-Data` 標頭與 `initData` 查詢參數（不呼叫 `tg.Verify`），請求改走後續的訪客 token、MCP token、內網 IP 加 session 驗證；`bot_token` 非空時行為不變。因為目前沒有任何測試呼叫 `server.Start`，同時建立測試輔助函式：在暫存目錄寫入 `config.yaml`、切換工作目錄、用空閒連接埠呼叫 `Start`、結束時 `Shutdown`。驗證：`internal/server` 新增 `TestEmptyBotTokenIgnoresInitData`——白名單含 123456789、以空 Token 金鑰簽出該使用者的 `initData`，請求 `/sessions` 不得得到 200；另有 `TestBotTokenSetStillVerifiesInitData`，以正確 Token 簽出的 `initData` 仍可通過，證明未破壞原行為。

## 2. 設定檔產生

- [x] 2.1 [after: 1.1] 完成 Setup input validation 與 Config file is written minimally, atomically, and never overwritten 的內容部分（設計主題「設定檔以最小欄位原子寫入」）：新增 `config.SetupInput`、`ValidateSetup` 與 `RenderConfig`：驗證密碼（共用 1.1 的規則）、連接埠 1–65535、資料庫路徑的目錄存在且可寫、開啟 Mini App 時 `bot_token` 非空、白名單皆為正整數，錯誤帶欄位名稱；`RenderConfig` 只輸出 `web.password`、`web.allowed_cidrs`（預設僅 `127.0.0.0/8` 與 `::1/128`，勾選區網才加入三個私有網段）、`server.port`、`db.path`、`no_auth: false`，Mini App 開啟時才輸出 `bot_token` 與 `whitelist_tg_ids`。驗證：`internal/config` 新增 `TestValidateSetup`（逐欄位的表格案例，含 `change-me`、7 個字元的密碼、不存在的資料庫目錄、`bot_token` 為空）與 `TestRenderConfigRoundTrip`（輸出寫入暫存目錄後經 `config.Load` 載入皆有效；Mini App 關閉時內容沒有 `bot_token`；`allow_lan` 影響 `allowed_cidrs`）。
- [x] 2.2 [after: 2.1] 完成 Config file is written minimally, atomically, and never overwritten 的寫入部分（設計主題「設定檔以最小欄位原子寫入」）：新增 `config.WriteNew(path, content)`：先寫同目錄暫存檔並同步，再以「目標已存在即失敗」的方式放到目標路徑，已存在回 `ErrConfigExists` 且不改動原檔，失敗時不留暫存檔，Unix 權限 0600。驗證：`TestWriteNewAtomic`——已存在檔案內容不變且得 `ErrConfigExists`；8 個 goroutine 同時寫入恰好 1 個成功且檔案內容等於成功者的內容；失敗後目錄內沒有殘留暫存檔；Unix 上權限為 0600（Windows 略過此斷言）。

## 3. 代理 CLI 偵測

- [x] 3.1 完成 Detect each agent CLI with the app's own lookup rules 與 Install guidance without blocking setup 的偵測與提示資料：新增 `internal/clicheck`：對 claude、codex、cursor-agent、kiro-cli、agy 回傳 `{agent_type, label, command, found, path, install_url, note}`，沿用既有解析規則：codex 呼叫 `codex.ResolveBin`（`CODEX_BIN`、PATH、Windows 預設路徑），agy 讀 `CC_AGY_BIN` 後才查 PATH（目前 `agyBinary` 未匯出，匯出為 `antigravity.Binary`，原呼叫處改用它，不複製邏輯），其餘查 PATH；找不到只標 `found: false`；cursor-agent 的 `note` 說明官方命令是 `agent` 但本專案呼叫 `cursor-agent`。驗證：`internal/clicheck` 新增 `TestCheckResolution`——以測試用 PATH 目錄放入假執行檔，涵蓋 PATH 找到、`CODEX_BIN` 指定、`CC_AGY_BIN` 指定、全部缺失（`found: false` 且各有 `install_url`）、單項缺失不影響其他項。
- [x] 3.2 [after: 3.1] 完成 Version probing is bounded and non-fatal：為已找到的工具以 `--version` 取得版本：各項並行執行、單項逾時 3 秒、子行程使用 `proc.SysProcAttr()` 避免 Windows 彈出黑窗，失敗、逾時或沒有輸出只讓該項 `version` 為空而 `found` 維持 true，不影響其他項。驗證：`TestCheckVersion`——正常輸出被解析；其中一個假工具睡眠超過逾時，整體在略多於逾時（測試中用縮短的逾時值）內回傳，該項 `version` 為空，其他項版本正確。

## 4. 設定模式伺服器

- [x] 4.1 完成 Setup API is restricted to the local machine with a one-time token（設計主題「設定 API 的安全限制（三道，缺一不可）」）：新增設定模式的請求檢查函式：只接受連線實際遠端位址為 loopback、`Host` 為 `127.0.0.1`／`localhost`／`[::1]` 加目前連接埠，對受保護端點另要求 `X-Setup-Token` 與啟動時產生的 32 位元組十六進位憑證相符（常數時間比較），完全忽略轉發標頭。驗證：`TestSetupGuard` 表格案例——非 loopback 位址（含帶 `X-Forwarded-For: 127.0.0.1`）被拒、`Host: attacker.example:8080` 被拒、缺憑證與錯誤憑證被拒、正確組合通過，被拒者回 `setup_forbidden`。
- [x] 4.2 [after: 1.1, 4.1, 3.2] 完成 Setup mode exposes a minimal surface（設計主題「設定模式是獨立、最小的 HTTP 服務」）：`server.Start` 遇到 `ErrConfigMissing` 時進入設定模式：只提供前端靜態資源、`GET /setup/status`、`GET /setup/cli-check`（呼叫 3.2 的檢查）與 `POST /setup/save` 的占位，其餘路徑回 503 `setup_required`；不開資料庫、不啟動模型清單同步與 MCP；在 log 印出含憑證的設定網址並提供 `Server.EntryURL()`（設定模式為含憑證的網址，正常模式為 `URL()`）；正常模式下 `/setup/status` 回 `{needs_setup: false}`，另兩支回 404。驗證：`internal/server` 新增 `TestSetupModeSurface`（`/sessions` 得 503、無憑證的 `cli-check` 得 403、有憑證得工具清單、啟動後工作目錄沒有產生 `claude-miniapp.db`）與 `TestNormalModeHidesSetup`。
- [x] 4.3 [after: 4.2, 2.2] 完成 `POST /setup/save`：驗證輸入（失敗回 400 `invalid_input` 與欄位）、以 `RenderConfig` 與 `WriteNew` 寫檔（已存在回 409 `already_configured`，寫入失敗回 500 `write_failed`），成功且連接埠與目前不同時回 `applied: "restart_required"` 與新網址，並保持設定模式不切換。驗證：`TestSetupSaveInvalid`（各欄位 400 且沒有檔案產生）、`TestSetupSaveWritesConfig`（寫出的檔案可被 `config.Load` 載入，Mini App 關閉時無 `bot_token`）、`TestSetupSaveAlreadyConfigured`（檔案先存在得 409 且內容不變）、`TestSetupSaveChangedPort`（回 `restart_required`）。
- [x] 4.4 [after: 4.3, 1.2] 完成 Switch to normal mode after saving：連接埠不變時完成行程內切換：寫檔成功後關閉設定模式的監聽，以同一連接埠啟動正常模式並換進同一個 `*Server`（`Wait`、`Shutdown`、`URL` 持續有效），回 `applied: "switched"`；正常模式啟動失敗回 500 `start_failed` 與錯誤原因且保留已寫入的設定檔。驗證：`TestSetupSaveSwitches`——在暫存目錄以設定模式啟動，POST 有效設定得 `switched`，隨後以新密碼 `POST /auth/login` 得 200 且 `/setup/status` 回 `needs_setup: false`，對同一個 `Server` 呼叫 `Shutdown` 可正常結束；`TestSetupSaveStartFailed`——資料庫路徑在驗證後被改成不可開啟時得 `start_failed` 且設定檔仍在。

## 5. 桌面版啟動

- [x] 5.1 [after: 4.2] 完成 Desktop first run writes next to the executable：桌面版在目前目錄沒有 `config.yaml` 但 exe 旁邊有 `config.example.yaml` 時，將工作目錄切到 exe 旁（現有的 `chdirToExeIfConfigured` 只處理 exe 旁已有 `config.yaml` 的情況），並改用 `Server.EntryURL()` 開啟視窗；`go run` 的暫存 exe 旁邊沒有該檔，維持原行為。驗證：把判斷抽成可測的純函式，新增 `TestWorkDirForFirstRun`（Windows 建置標籤）涵蓋「exe 旁有 config.yaml」「exe 旁僅有 config.example.yaml」「兩者皆無」三種情況；`go build -tags production -ldflags "-H windowsgui" ./cmd/desktop` 通過。

## 6. 設定精靈前端

- [x] 6.1 新增 `internal/static/js/setup/setupForm.js`（不依賴 DOM 的純函式）：與伺服器相同的欄位驗證規則、以 `crypto.getRandomValues` 產生符合規則的隨機密碼、白名單文字解析（接受逗號與換行分隔的正整數）、組出 `POST /setup/save` 的請求內容（Mini App 關閉時不含 `bot_token` 與白名單）、從網址讀取設定憑證。驗證：新增 `scripts/setup-wizard.test.cjs` 的 `TestPasswordRules`、`TestGeneratedPasswordValid`（重複產生 200 次皆通過規則）、`TestPayloadMiniAppOff`、`TestWhitelistParse`、`TestReadSetupToken`，由 `node --test scripts/*.test.cjs` 執行。
- [x] 6.2 [after: 6.1, 4.2] 完成 Guided wizard flow 與 Install guidance without blocking setup 的畫面部分：新增 `internal/static/js/setup/SetupWizard.js` 並在 `app.js` 啟動時先查 `/setup/status`：`needs_setup` 為真顯示精靈，否則維持原流程。步驟依序為歡迎與 CLI 檢查結果（僅提示，不擋儲存）、密碼與連接埠與資料庫路徑與「允許區網裝置登入」（預設關）、Telegram Mini App 開關（預設關，開啟才顯示 Token 與白名單欄位與需要外部 HTTPS 網址的說明）、確認儲存；讀到憑證後用 `history.replaceState` 清掉網址列；`switched` 時導向首頁登入，`restart_required` 時顯示新網址與重新啟動提示；不提供 `no_auth`。`index.html` 加入新檔，`static.IsPublic` 已允許 `js/` 底下的檔案所以不需改白名單。驗證：以 esbuild 以 JSX 載入器解析兩個新檔與 `app.js` 皆無語法錯誤；`go test ./internal/static/` 通過（含 production 建置的資源清單）；瀏覽器實測列在 8.1。

## 7. 文件

- [x] 7.1 [after: 6.2] 改寫 `docs/getting-started.zh-TW.md` 與 `docs/getting-started.md` 的「建立設定檔」段落，以精靈為主、手動編輯為輔：說明首次啟動會出現精靈、Mini App 為可選、純 server 在遠端機器需用 SSH 連接埠轉發並從 log 取得含憑證的網址、Windows 設定檔權限提醒；同步更新 `README.md`、`README.zh-TW.md` 的快速開始與 `config.example.yaml` 開頭註解（`bot_token` 現為可選）。驗證：文件內容審閱，確認不再要求「只用網頁也要填 Bot Token」，所有指令與連結可點擊／可執行，中英文版本內容一致。

## 8. 實機驗證

- [x] 8.1 [after: 1.2, 3.2, 4.4, 5.1, 6.2] 在乾淨資料夾（沒有 `config.yaml`）完整走一次：桌面版與純 server 版各一次，每次各走 Mini App 關閉與開啟兩條路徑；另驗證連接埠變更的提示、已有設定檔時精靈不出現、從區網另一台裝置無法開啟設定 API。驗證：逐項記錄實測結果與觀察到的限制；沒有條件的項目保留為未驗證，不宣稱完成。

