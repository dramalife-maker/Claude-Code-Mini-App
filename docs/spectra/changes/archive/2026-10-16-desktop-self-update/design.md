## Context

- .github/workflows/release.yml 由 v* tag 觸發，打包 Windows 的 server 與 desktop EXE、Linux server 及磁碟前端，產生 checksums.txt 後用 gh release create 發佈。
- internal/server/server.go 的 Start 使用 app.Static 提供磁碟前端；internal/api/share.go 的 ShareHandler.Page 直接讀取 index.html；cmd/desktop/main_windows.go 的 favicon 亦直接讀磁碟。單改首頁會漏掉分享頁及圖示。
- 桌面 main 在 application.New 前啟動 server.Start；目前 Wails helper 由 application.New 處理，重執行時會先撞 port。Wails v3.0.0-beta.26 已有可直接呼叫的 updater.HandleHelperMode。
- 已檢視目前依賴的 pkg/updater/config.go、updater.go、helper.go、providers/github/github.go 與原生訊息範例：CheckInterval 會呼叫 CheckAndInstall；GitHub provider 的 checksum 是選填；helper 只替換目標 EXE，能復原替換或建立新程序失敗，沒有新程序啟動後的健康檢查回滾。
- SettingsModal 已有版本與 SHA 區塊。現有 spec 沒有桌面發佈或更新能力，因此新增 desktop-distribution 與 desktop-self-update，保留其他規格。
- 工作樹另有 session-tabs 及既有版本顯示修改。本提案只寫文件，apply 前須重讀相關程式並整合已存在的改動，不覆蓋其他變更。

## Goals / Non-Goals

**Goals:**

- In scope：Windows amd64 桌面版完整自我更新；Go production 資源嵌入；既有安裝包相容；穩定版選擇、下載驗證、明確確認、安全重啟與失敗診斷。
- 更新只動目前桌面 EXE，保留所有使用者資料及檔案位置；使用既有 GitHub Release 與 Wails，不增加 updater 套件。
- 開發仍可直接修改磁碟前端並重整；server 既有啟動方式維持可用。

**Non-Goals:**

- Out of scope：Linux／macOS 桌面更新、純 server 的 self-update、強制更新、背景自動下載／重啟、差分更新、安裝服務與管理員提權。
- 不將資料搬到 AppData、不做資料庫 schema migration、不新增更新安裝／重啟 HTTP API、不引入前端建置系統或取代現有 CDN。
- 不承諾新程序已啟動卻稍後崩潰時的自動回滾，不加入健康檢查 launcher。
- 本期信任固定的公開 GitHub repository、HTTPS 與其 Release 權限。SHA256 是下載完整性驗證，不是獨立發佈者身分驗證；Ed25519 更新簽章與 Windows Authenticode 簽章不在本期範圍，兩者也不能互相替代。

## Decisions

### Go embed 與開發磁碟資源共用 fs.FS

在 internal/static/ 新增互斥的 production 與非 production Go 檔，提供同一個 Assets fs.FS。production 僅 embed index.html、favicon.ico 與 js/；不把 AGENT.md、Go 原始碼或使用者資料收入資源。非 production 使用 os.DirFS("./internal/static")。

server 靜態服務、ShareHandler.Page 與 favicon 皆讀此資源來源。ShareHandler 接受 fs.FS，保持既有 base 注入、no-store、no-referrer 與訪客權限。靜態 middleware 僅處理存在的公開資源，找不到時交還既有 API 路由，不以 SPA fallback 吃掉 API 404；JS MIME 必須正確。production 不因旁邊仍有舊 internal/static 而退回磁碟。

替代方案是下載 ZIP 後協調多檔更新，需自製檔案替換交易，因此選擇更新單一 EXE。開發模式保留磁碟資源，避免每次改 JSX 都重編譯。

### 保留安裝包並新增桌面更新資產

保持 claude-miniapp-v<VERSION>-windows-amd64.zip 與 claude-miniapp-v<VERSION>-linux-amd64.tar.gz 名稱及用途。Windows ZIP 內 server 保留磁碟資源；desktop 以 -tags production 編譯，自帶前端。另上傳相同 desktop bytes 為 claude-miniapp-desktop-v<VERSION>-windows-amd64.exe，checksums.txt 加入其 SHA256。

build.bat 改為 production 桌面建置；go run ./cmd/desktop 仍為開發模式。CI 檢查 tag、根目錄 VERSION、internal/version/version.go 的 Version 一致，且非 unknown 的建置 SHA 對應觸發 tag 的 commit。自動選檔須精確匹配桌面名稱，不能用內建平台字串 matcher 誤選舊 Windows ZIP。

所有資產與 checksums 先在草稿 Release 上傳並驗證齊備，再發佈為 latest 正式版；任一步失敗保留草稿且不讓 updater 看見半成品。重跑時只重用相同 tag 的草稿，若已公開發佈則停止，禁止覆寫公開版本資產。替代方案是另設更新 manifest 伺服器，本期由 GitHub metadata 與 checksums 滿足需求。

### 原生更新協調與設定入口

以 app.Updater 為核心，WindowNone、CheckInterval=0，透過明確的 Check、DownloadAndInstall、Restart 操作。應用協調器只管理一次一個操作、狀態快照與重啟閘門；不重寫下載、hash、替換或 helper。避免 CheckAndInstall 與內建 Restart 按鈕直接跳過下載確認或任務保護。

系統匣「檢查更新」顯示主視窗並開啟設定的更新區；設定版號旁的按鈕開啟相同區域。使用既有 React 與樣式，提供目前版本／SHA、可用版本、純文字更新內容、進度、下載、稍後、重新啟動並更新、重試；更新內容不作為 HTML 執行。

由桌面主視窗注入 window.__desktopUpdater.request(action)，送出固定前綴的原生 raw message；action 僅允許 check、download、restart、status。Go 端將 JSON 快照送到該視窗的 desktop-update-state DOM event。欄位為 state、currentVersion、currentSha、availableVersion、releaseNotes、downloadedBytes、totalBytes、errorCode、errorMessage、busyReason；無可用版本時 availableVersion 為空字串。busyReason 只在 errorCode 為 update_busy 時有值：running_task、shell_task、pending_confirm、queued，或 maintenance（另一個更新操作已持有閘門）。JS 不傳入 repository、URL、目的路徑或程序命令。

原生端驗證發起視窗是 main、主 frame、Origin 與 TopOrigin 等於 started.URL 的 origin，且主視窗維持擁有者首頁；拒絕空 origin、其他視窗與未知 action。限制主視窗導航在本機擁有者首頁，外部及分享連結交給外部瀏覽器，禁止在訪客頁注入更新橋接。瀏覽器、Telegram、訪客頁與純 server 沒有此橋接，也不顯示更新控制。不在共用 /config 回傳 desktop=true，避免遠端網頁誤認為具有本機權限。

替代方案是 HTTP 更新端點，但會新增遠端管理程序權限；本期沿用 Wails 原生訊息通道，不開此 API。

### 正式版檢查與強制 checksum

固定 repository 為 jerry12122/Claude-Code-Mini-App，使用公開 GitHub API，程式內不帶 PAT。CurrentVersion 取 version.Version；Go 原生 VCS metadata 只作 SHA 顯示，不用來比較新版。用 provider 的 semver 判斷，只接受非 draft、非 prerelease 且較新的版本。

初始化完成後檢查一次，之後六小時檢查一次。六小時是排程產品值，不是量測／估計；以假時鐘測試觸發次數，不等實際六小時。背景只 Check，不自動下載或彈出設定。downloading、verifying、ready、restarting 期間略過排程檢查，ready 保留已驗證的待套用版本；手動重複操作不重入。

GitHub provider 設 ChecksumAsset="checksums.txt" 並精確選擇桌面 asset。接收 provider 結果後在允許下載前確認 SHA256 digest 存在且長度正確，缺少 checksum 檔、缺少對應行、格式錯誤或 hash 不符一律拒絕。不能僅設 ChecksumAsset 就假設缺失時必然拒絕，因為已檢視版本仍容許空 Verification。這個應用驗證層只補強必填合約，保持 Wails 的下載與驗證實作。

### helper 優先與任務維護閘門

main 的第一個動作呼叫 updater.HandleHelperMode，早於 chdir、logging.Init、config.Load、server.Start 及 application.New 的單一實例處理。一般程序接著維持原有啟動順序；helper 不碰使用者 DB、log 或 port。

新增由 ws 擁有的共用維護閘門，暴露 TryBeginDesktopUpdate() (release func(), err error) 給 server，再由 Server 暴露給桌面端。錯誤區分 ErrUpdateBusy 與內部檢查失敗。取得閘門前檢查所有 session 的 agent、shell、等待確認、待執行 queue 與尚未完成清理的任務；任何一項存在都保留 ready，使用者處理工作後再按重啟，不設自動閒置重啟。

不要只讀 DB status 或 taskIsActive：目前 taskCancel／shellTaskCancel 會在子程序清理完之前移除 map 項目。任務的 admission 與 in-flight lifetime 必須涵蓋首次取消／副作用之前到真正返回的 defer；所有 WS、MCP loopback 與 queue dispatch 入口共用此閘門。維護狀態下，新工作在寫入 running 或消耗 queue 前被拒絕，既有 queue 不刪除，前端收到可重試的 update_in_progress 錯誤。

取得閘門後才能呼叫 Restart；helper 啟動失敗或未 ready 時釋放閘門、保留程序與待更新狀態。成功則維持封鎖直到 OnShutdown 完成 HTTP／DB 關閉並退出。不能只在 ShouldQuit 否決，因為此時 helper 已生成，可能稍後意外套用更新。

### 使用者資料保留與失敗復原界線

更新目標是正在執行的 desktop EXE，即使最初下載檔含版本名也替換原本路徑；不按下載檔名建立第二套安裝、不切換 cwd 或移動 config。純 server EXE 與其前端也不跟著變更。

下載暫存遵循 Wails staging 規則，安裝目錄必須可寫；不可寫顯示明確錯誤，不要求管理員提權。下載、驗證及 helper 啟動失敗保留舊程序；實際替換或建立新程序失敗重用已安裝 Wails helper 的備份復原。新程序啟動後的應用錯誤不在自動復原承諾內，README 提供重新下載舊版 EXE 的手動恢復方式，不回寫或降版 DB。

## Implementation Contract

- In scope：兩份新 capability spec 所列情境、Windows portable desktop、相同來源的資源服務、GitHub Release 與文件。Out of scope：上述 Non-Goals，不修改 session-tabs 狀態或聊天版面設計。
- 狀態：idle → checking → up-to-date／available／error；available 經使用者 download → downloading → verifying → ready；ready 經使用者 restart 且閘門成功 → restarting（Restart 已呼叫，前端一律忙碌、不可再操作；成功時程序退出，helper 失敗則回到 ready）。busy、閘門內部檢查失敗或 helper 失敗返回 ready 並帶 errorCode；檢查／下載失敗可重試。重複命令不產生額外下載或 helper。
- errorCode：check_failed、asset_missing、checksum_missing、checksum_mismatch、download_failed、install_not_writable、update_busy（附 busyReason）、update_check_failed、update_in_progress、helper_failed；驗證缺失與網路錯誤不得顯示為已是最新。update_check_failed 表示閘門內部無法確認工作狀態（例如 DB 查詢失敗），已取消重啟、保持 ready、不啟動 helper。背景失敗記錄並供設定顯示，不中斷聊天；手動錯誤顯示原因。
- 驗證名稱為計畫新增：TestStaticAssetModes（含本機路徑逃逸拒絕）、TestSharePageAssets、TestDesktopUpdateSelection、TestDesktopUpdateFlow、TestDesktopUpdateAdmission、TestDesktopUpdateHelperStartup。使用本機 fixture HTTP server、假時鐘與隔離的暫存 config／DB，不讀寫使用者實際資料，不對公開 GitHub 發佈測試版。
- 開發／正式資源驗證：go test ./internal/... 與 go test -tags production ./internal/...；Windows 的 go build -tags production ./cmd/desktop 成功。正式桌面在移走磁碟 internal/static 後仍能顯示首頁、JS、分享頁及 icon；開發模式修改磁碟 JS 並重整仍可看見變更。
- UI 驗證：桌面設定與系統匣同一流程，版本與 SHA 保留；外部瀏覽器／Telegram／訪客沒有更新控制且不能觸發 native action；release notes 含 script 字串仍只顯示文字。
- Release 驗證：隔離套件建置的清單、checksum 與 tag／VERSION／Version／建置 SHA 一致；草稿未齊備不發佈，正式版禁止重寫；ZIP 內 desktop 與單檔資產 bytes 相同，Linux 套件保留原啟動方式。
- Windows 端到端：fixture 版本由 1.0.0 升到 1.0.1（測試輸入，非實際發佈版本），確認 PID 改變、版本與 SHA 更新、前端屬於新 EXE、原 port 可用且資料不變；注入 checksum、網路、目錄不可寫、busy、helper、替換失敗並觀察對應結果。

## Risks / Trade-offs

- [Wails beta 更新器的行為與新文件不同] → 以上以 go.mod 鎖定版本原始碼為準，不升依賴；以 Windows 真程序測試證明 helper 與復原路徑，避免直接採用文件中的自動下載流程。
- [重啟與新任務之間競爭] → admission 與維護取得共用同步，lifetime 跨越取消清理；測試競爭時只能「任務取得執行權」或「更新取得維護權」其中之一。
- [公開 API 斷線／限流] → 一個背景排程與單一操作協調，失敗保留舊版並可手動重試；不內嵌 GitHub token。
- [移除磁碟前端造成分享頁／圖示漏改] → 統一 fs.FS，production 測試不提供磁碟副本；舊目錄即使存在也不能遮蔽嵌入資源。
- [更新中資料損毀] → idle gate、正常 Shutdown、不遷移 DB，端到端前後比較 config、DB 記錄與附件／日誌內容；正常追加的日誌不要求 bytes 完全相同。
- [使用者放在不可寫位置或 EXE 被其他程式鎖住] → 不提權，顯示可寫目錄／手動更新指引；重用 Wails 的有限重試與原 EXE 復原。
- [SHA256 與 GitHub 權限不是獨立簽章信任根] → 明確記錄本期信任模型，不宣稱防止 GitHub 維護者帳號失陷；未將未簽章下載說成已驗證發佈者。
- [CDN 依賴仍需連網] → embed 只保證本專案前端與 EXE 同版，不承諾離線啟動。

## 已知限制

- **外部與分享連結不保證在外部瀏覽器開啟**：Wails v3.0.0-beta.26 的 Windows 沒有導航攔截 API，無法在原生層攔截主視窗的外部連結。更新控制的安全性由注入腳本（橋接只掛在本機擁有者首頁的主框架）與 Go 端驗證（視窗身分、主框架、Origin、TopOrigin）保證，不交給外部頁面；但連結是否在外部瀏覽器開啟，不能以原生攔截保證。
- **同 URL 的子框架無法區分**：原生驗證依視窗與主框架判斷身分。頁面內若有與擁有者首頁相同 URL 的 iframe，目前無法與主框架區分。
- **背景檢查失敗會清除「可更新」提示**：背景 Check 失敗時快照變為 error（`check_failed`），原本的 availableVersion 會被清空，要等下一次成功檢查才會回到可用更新。錯誤會顯示在設定中，不中斷聊天。
- **「稍後」只存在前端記憶體**：重新整理或重新啟動後會再次提示，不寫入設定或資料庫。
- **安裝階段錯誤代碼對映有限**：verify 階段失敗為 `checksum_mismatch`；install 階段的權限錯誤為 `install_not_writable`；其餘 Wails 錯誤（含 install 階段的非權限錯誤）統一為 `download_failed`，細節需看 `logs/server.log`。
- **新增的代碼與欄位**：`update_check_failed`（閘門內部檢查失敗，取消重啟）、`busyReason`（僅 `update_busy` 時有值）、`restarting`（Restart 已呼叫的狀態）。
- **佇列也會阻擋更新**：佇列有項目（即使暫停）就拒絕重啟（`update_busy`，`busyReason: queued`），既有佇列不會刪除，使用者需先清空或完成。
- **關閉逾時強制結束**：Restart 成功後，HTTP 與 DB 正常關閉超過 `ShutdownDeadline`（20 秒）會強制 `os.Exit(0)`，閘門不釋放。SQLite WAL 在強制結束後的復原需在 5.1 以真程序驗證。
- **SHA256 不是身分驗證**：SHA256 只證明下載的位元組與同一 Release 的 `checksums.txt` 相符（完整性）。它不證明發佈者身分，也不等於 Ed25519 更新簽章或 Windows Authenticode 簽章；本期兩者都不做，且不能互相替代。信任根是固定的公開 GitHub repository、HTTPS 與其 Release 權限。
- **桌面頁面需自行送出 runtime 就緒訊號**：桌面視窗載入的是外部 URL（本機 server 首頁），不經 Wails assetserver，因此 Wails 自己的 runtime JS 不會載入，也就不會送出 `wails:runtime:ready`。Wails beta.26 的 `ExecJS` 在收到該訊號前只把 JS 排進佇列，若沒有補送，bridge 腳本與 `desktop-update-state` 都無法送達頁面（5.1 e2e 發現的阻斷缺陷）。因此 `internal/static/js/wails-runtime-ready.js` 在首頁（路徑 `/`）且 `window.chrome.webview.postMessage` 可用時，最先送出一次該訊號；一般瀏覽器與分享頁為 no-op。這只觸發佇列 flush，flush 的 bridge 腳本仍自帶頂層、Origin 與路徑檢查。
- **不承諾離線可用與自動回滾**：更新需要網路。新程序已成功啟動後發生的應用程式錯誤，不會自動回滾，依 README 手動還原舊 EXE。
- **雙重故障不會自動 relaunch**：Wails helper 在所有替換嘗試用盡後，會先還原 `<exe>.bak`（`restoreFromBackup` 先移除目標再改名）。若目標仍被其他程序鎖定，兩步都會因同一把鎖失敗（`restore failed`），helper 不會 launch，結束碼 14。此時原 EXE 內容不變、`<exe>.bak` 保留，使用者需手動恢復：關閉占用該檔的程序（或重新開機）後直接啟動應用程式；若原 EXE 已損毀或遺失，將 `<exe>.bak` 改回原檔名後再啟動。替換成功路徑（happy 場景）已由真實 1.0.0→1.0.1 驗證。
- **無公開正式版時回報已是最新**：GitHub `releases/latest` 回 404 代表尚無公開正式版，產品顯示 `up-to-date` 且無錯誤碼，這是已知行為。它與「連線、限流、下載失敗不得偽裝為已是最新」不衝突，後者針對失敗而非查無版本。
- **e2e 覆蓋範圍**：設定頁以 `desktop-update-open` 事件開啟，而非點擊系統匣圖示；`helper-failed` 注入在 manager 層（Restart 之前），不驗證真實 helper 啟動失敗；`watchdog` 走 `os.Exit` 而非 TerminateProcess 崩潰；真實 GitHub 發佈路徑未驗證（不存取公開 GitHub）。
- **e2e fixture 埠非 18xxx**：fixture HTTP server 使用 OS 分配的臨時埠，僅綁定 `127.0.0.1`；應用程式測試埠仍為 18xxx 範圍。

## Migration Plan

1. 先加共用資源讀取與可選 production 來源，遷移所有資源消費端，保留開發及 server 套件的磁碟模式；每步皆能建置。
2. 加入 helper 優先處理、更新協調器與原生入口，再接入任務閘門及確認重啟；在 fixture server 上驗證，無公開發佈副作用。
3. CI 保留原資產再新增 desktop EXE，更新 checksum 與草稿發佈順序，補文件；正式發佈時沿用既有 tag 流程。
4. 既有版本使用者手動下載一次含 updater 的版本，保留原 config／DB；後續才支援應用程式內更新。伺服器使用者繼續手動更新原套件。
5. 回復時重新放回舊 desktop EXE；資料檔維持原位。若新版已正常啟動而應用錯誤，依文件手動恢復，不承諾自動健康檢查或資料格式降版。

## Open Questions

無阻擋實作的未定事項。平台、排程、穩定版通道、信任模型、資產名稱及重啟條件皆已在本提案定義；不需要先建立雲端更新服務。

