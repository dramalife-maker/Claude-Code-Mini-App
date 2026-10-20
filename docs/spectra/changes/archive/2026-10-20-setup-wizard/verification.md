# setup-wizard 驗證紀錄

2026-10-20，Windows。本次接手時 HEAD 為 `aa96090`，實作進度 12/15；最初驗證完成至 14/15。使用者要求收尾時任務表已為 15/15；8.1 中第二台區網裝置的實測仍保留為未驗證限制，不宣稱通過。

## 實機結果

測試資料全部位於 `%TEMP%/miniapp-setup-e2e-w35ao9v7/`。為避開使用者現有的 8080，測試建置僅以 Go overlay 將首次設定的預設埠換成 18371；變更埠案例使用 18373。桌面測試沿用既有 `e2e` 建置標籤，隔離單一實例 ID、WebView2 資料與 CDP 埠；未修改正式設定或既有服務。

| 路徑 | 結果與觀察 |
| --- | --- |
| 純 server、Mini App 關閉 | Chrome 實際點選四步驟，CLI 五項版本正常顯示；網址憑證自動移除；產生 20 字元密碼；區網與 Mini App 預設關；儲存後切換至密碼登入，成功進入空白會話列表。設定沒有 Telegram 欄位。 |
| 純 server、Mini App 開啟 | 開啟後才顯示 Token、ID 與外部 HTTPS 說明；空 Token 阻止前進；填入測試用假 Token、兩個 ID 後可儲存。變更至 18373 時顯示重啟提示；重啟後該埠的 `/setup/status` 回 `needs_setup: false`，已有設定不再出現精靈。 |
| 桌面、Mini App 關閉 | 真正啟動 Wails/WebView2，由既有 CDP 驅動操作畫面；生成密碼、儲存、行程內切換與新密碼登入通過。從不同工作目錄啟動，設定仍寫在 exe 旁，捷徑工作目錄沒有設定檔。 |
| 桌面、Mini App 開啟 | 同樣經真實畫面填 Token、白名單，儲存後切換與密碼登入通過；設定正確包含 Telegram 欄位。沒有連線到真實 Telegram Bot。 |
| 桌面補充：查詢失敗、CLI 全缺、變更埠 | 以 CDP 在頁面內注入第一次 `/setup/status` 回 503，以及 CLI 五項全部缺失的回應；確實顯示錯誤與重試入口，重試後有五個安裝連結，仍可走完實際儲存。改至 18373 後顯示新網址，上一步與儲存均停用。這兩支查詢的故障是模擬，儲存、寫檔與桌面畫面是真實流程。結果在 `desktop-no-tools-restart/result.json`。 |
| 區網另一台裝置 | **not-run**：本次沒有可控制的第二台區網裝置。Go 測試已涵蓋非 loopback、偽造轉發標頭、錯誤 Host 與憑證，但不能取代此實機項目；封存時保留此限制，不記為實測通過。 |

桌面實測的 `result.json` 與登入後截圖分別在上述暫存目錄的 `desktop-off/`、`desktop-on/`。server 確認畫面的截圖是 `server-review.png`。暫存日誌含測試憑證，不加入版本庫。只終止本次啟動且執行檔路徑相符的 PID 與子行程。

## 自動檢查

| 指令／範圍 | 證據 |
| --- | --- |
| `go test -short ./...` | passed-current：全套通過；之後調整已有設定檔檢查順序，再以 `go test -short ./internal/server` 驗證受影響套件通過。 |
| `node --test scripts/*.test.cjs` | passed-current：57 pass、0 fail，含 24 項精靈檢查。 |
| `go vet ./internal/... ./cmd/...` | passed-current：通過。 |
| `go test ./internal/static`、`go test -tags production ./internal/static` | passed-current：開發與正式嵌入資源清單皆通過。 |
| esbuild JSX 解析 | passed-current：`setupForm.js`、`SetupWizard.js`、`app.js` 通過；沿用 npm 快取，沒有新增專案依賴。 |
| production server／desktop 建置 | passed-current：兩種正式建置通過；桌面另通過 `-ldflags "-H windowsgui"`。 |
| Postman 本機 HTTP | 兩次正常模式狀態檢查各 1 assertion 通過、exit 0；另觀察到缺憑證時 403 `setup_forbidden`，該負向請求的 CLI exit 為 1，不算通過的 Postman run。Go 的相同拒絕測試通過。 |
| 文件檢查 | 五份更新文件中舊的必填 Bot Token／A、B 方案敘述已移除；四份 Markdown 的相對連結目標均存在，中英文設定流程一致。 |

## 需求與情境對照

12 項需求、29 個 Scenario；沒有 `Example` 區塊。下表列出每個情境的測試依據，執行結果見上方。單頁間距、色彩與圓角不跨使用位置，依 TDD 範圍規則排除自動斷言；已查看 server 確認頁截圖。Unix 的 0600 權限斷言在 Windows 略過，Linux 實機與 SSH 轉發未執行。

| Requirement／涵蓋的 Scenario | 實作 | 測試或實測 |
| --- | --- | --- |
| CLI lookup：PATH、CODEX_BIN、Antigravity override、Missing tool | `internal/clicheck/clicheck.go` | `TestCheckResolution` 四種相應分支；單項缺失案例使用 Cursor，全部缺失案例同時涵蓋 Kiro。 |
| Version probing：Slow tool | 同上 | `TestCheckVersion`、`TestCheckVersionParallel`、`TestCheckVersionBoundedWithLingeringChild`。 |
| Install guidance：No tool found | `SetupWizard.js`、`setupForm.js` | `TestSetupCliMissingDoesNotBlock` 與工具安裝網址測試；額外桌面注入測試見補充結果。 |
| Missing config：Missing／Broken config | `internal/config/config.go`、`internal/server/server.go` | `TestLoadMissingConfig`、`TestStartRouting`、`TestStartBrokenConfigDoesNotEnterSetup`，以及四條首次啟動實測。 |
| Minimal surface：Other APIs／Normal mode hides endpoints | `internal/server/setup.go`、`server.go` | `TestSetupModeSurface`、`TestNormalModeHidesSetup`。 |
| Setup guard：Remote／Spoofed forwarding／Rebinding Host／Missing or wrong token／Token reaches user | `setupguard.go`、`setup.go`、桌面入口與精靈 | `TestSetupGuard`、`TestSetupGuardIgnoresForwardedFor`、`TestSetupModeSurface`、`TestWindowURL`、`TestReadSetupToken`、`TestStripSetupToken`；桌面與 Chrome 實際確認移除網址憑證。 |
| Optional Mini App：Password-only／No credentials／Forged initData | `config.go`、`server.go` | `TestLoadBotTokenOptional`、`TestEmptyBotTokenIgnoresInitData`（標頭與 query、ID 123456789）、`TestBotTokenSetStillVerifiesInitData`，以及密碼登入實測。 |
| Input validation：Example password／Mini App requires token／Unwritable DB directory | `internal/config/setup.go` | `TestValidateSetup`、`TestSetupSaveInvalid`、`TestValidateSetupForm`。 |
| Atomic write：Mini App off／Existing config／Concurrent saves | `setup.go`、`write.go` | `TestRenderConfigRoundTrip`、`TestWriteNewAtomic`、`TestWriteNewUnsupportedLinkLeavesNoConfig`、`TestSetupSaveAlreadyConfigured`、`TestSetupSaveConcurrent`。 |
| Switch mode：Same port／Changed port | `internal/server/setup.go`、`SetupWizard.js` | `TestSetupSaveSwitches`、`TestSetupSaveChangedPort`、`TestSetupSaveStartFailed`，以及切換、重啟實測。 |
| Desktop workdir：Launched from a shortcut | `cmd/desktop/workdir_windows.go` | `TestWorkDirForFirstRun` 與不同工作目錄的桌面實測。 |
| Guided wizard：Mini App off needs only password／Generated password | `setupForm.js`、`SetupWizard.js` | `TestSetupMiniAppOffCanSave`、`TestPayloadMiniAppOff`、`TestGeneratedPasswordValid`、`TestSetupGeneratePasswordAction`，以及四步驟實測。 |

## 本次審查修正

- 移除不支援硬連結時直接建立目標的退路：該路徑可能讓其他程序讀到半份設定。注入 `ENOTSUP` 的新測試先失敗（原來回成功），修正後失敗時不建立設定、也不留暫存檔。支援範圍限制已補入教學。
- 變更埠儲存成功或收到 `already_configured` 後停用重送；儲存中與終止狀態也停用上一步。新增兩個先失敗再通過的狀態測試。
- 刪除未被畫面使用、僅為測試回傳 false 的 `setupOffersNoAuth`，改檢查實際精靈來源沒有該欄位。
- `TestSetupSaveStartFailed` 改成驗證後將 DB 路徑建立為資料夾，實際呼叫正常啟動；因此抓到二次儲存先回 400 的問題。修正為先檢查已有設定檔，回 409，同時保留 `WriteNew` 的競爭保護。受影響測試先失敗再通過。
- 修正英文排解表殘留的 Bot Token 必填敘述。

審查涵蓋正確性、效率、重用與簡化、專案慣例。已知非阻擋觀察：CLI 檢查尚未回來時，歡迎頁會短暫顯示「尚未找到代理」的建議；結果回來後正常更新，不影響儲存。

## 封存前核對

| 面向 | 結果 |
| --- | --- |
| 完整性 | 任務表 15/15；12 項需求都有實作，29 個規格情境的依據見上方對照表，沒有 Example 區塊。 |
| 正確性 | 實作與已驗證的 v0.15.0 commit `542b54c` 相同。`go test ./internal/...`、`go test -tags production ./internal/...` 與 57 項 Node 測試沿用本次發布前的通過證據（passed-prior）；GitHub Actions run `39400953794` 在同一 commit 也通過一般與 production 測試、建置、資產比對與公開發布。 |
| 設計一致性 | 設定模式、原子寫入、空 Token 認證處理、CLI 偵測與精靈流程符合設計；保留前述非阻擋觀察。 |

使用者在 v0.15.0 發布後要求完成 Spectra 收尾；本次封存保留第二台區網裝置未實測的警告，不以勾選任務取代執行證據。後續有第二台裝置時，可在全新、無設定檔的隔離實例上，從該裝置連線主機區網 IP 的設定 API，預期無法連入。Unix 權限、Linux 實機與 SSH 轉發仍沿用上方的未執行標記。
