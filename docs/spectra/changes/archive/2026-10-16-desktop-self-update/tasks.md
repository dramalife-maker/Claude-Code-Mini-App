## 1. 統一前端資源來源

- [x] 1.1 實作「Go embed 與開發磁碟資源共用 fs.FS」的雙模式資源來源，達成 Production embedded assets 與 Development disk assets：internal/static/ 的 production 建置只嵌入 index.html、favicon.ico、js/，非 production 讀磁碟；不把 config、DB、附件、日誌、Go 原始碼及開發文件收入嵌入檔案。驗證：新增 TestStaticAssetModes，分別以 go test ./internal/static 與 go test -tags production ./internal/static 檢查資源集合、production 不依賴／不被舊磁碟副本遮蔽，以及開發檔案變動可立即讀取；兩種模式皆可建置。
- [x] 1.2 遷移資源消費端，讓 Production embedded assets 的首頁、JavaScript、ShareHandler.Page 與 favicon 全部來自同一 fs.FS；保留分享頁 base 注入與權限／安全 header，維持 Development disk assets 的正確 MIME、API 路由與路徑邊界。定位：internal/server/server.go、internal/api/share.go、cmd/desktop/main_windows.go。驗證：TestSharePageAssets 涵蓋分享網址有／無結尾斜線；TestStaticAssetModes 涵蓋已知 API、未知 API、路徑逃逸與 JS MIME；移走磁碟前端後 production 桌面首頁、分享頁與圖示仍可用，開發模式改 JS 後重整生效。[after: 1.1]

## 2. 桌面檢查、下載與操作入口

- [x] 2.1 依「helper 優先與任務維護閘門」先實作 Helper startup precedes application side effects：桌面 main 第一個動作執行 updater.HandleHelperMode，一般啟動維持原行為，helper 不讀 config、不初始化 log／DB、不綁 port、不進 single-instance。驗證：TestDesktopUpdateHelperStartup 以隔離路徑及已佔用 port 啟動 helper，確認收到 ready 而且原服務及資料不被觸碰；Windows production desktop 建置通過。[after: 1.2]
- [x] 2.2 依「正式版檢查與強制 checksum」實作 Stable release checks 的固定 GitHub provider 與精確桌面資產選擇；比對 version.Version，拒絕同版、舊版、draft、prerelease、server 與 ZIP，且下載前必須有相符檔名的有效 SHA256 Verification。驗證：TestDesktopUpdateSelection 用 fixture HTTP server 檢查 v1.0.0 → v1.0.1、同版／舊版／預覽版排除、錯平台、僅 ZIP、缺資產、缺 checksum 檔／對應行及格式錯誤，並確認程式不帶 PAT、不接受外部傳入 repository 或下載位置。[after: 2.1]
- [x] 2.3 依「原生更新協調與設定入口」實作 Explicit serialized download and verification 與「使用者資料保留與失敗復原界線」的檢查／下載部分：用 WindowNone 與明確 Check／DownloadAndInstall 維持單一操作，只有使用者 download 才下載；進度、錯誤與 ready 狀態可查詢，ready 不被重複操作或排程覆寫；不自行改寫 Wails 的下載、驗證或檔案替換。驗證：TestDesktopUpdateFlow 涵蓋確認前零下載、確認後進度與 ready、hash 不符、網路／限流／不可寫目錄、錯誤可重試、重複操作只有一次下載、錯誤不偽裝為已是最新，失敗時原 EXE 未修改。[after: 2.2]
- [x] 2.4 實作 Desktop update entry points 與 Restricted native update commands：系統匣與 SettingsModal 版號旁共用更新區，保留版本／SHA，呈現純文字 release notes、狀態、進度、下載、稍後及重試；注入設計定義的 request(action) 與 desktop-update-state 快照，Go 驗證 main window／主 frame／Origin／TopOrigin／首頁導航，非桌面及訪客無橋接，未知 action 與呼叫者指定 URL／路徑均拒絕，不新增安裝／重啟 HTTP API。此階段 restart 僅顯示尚未接線的停用動作，直到 3.3 才啟用。驗證：TestDesktopUpdateFlow 的信任來源及 action 案例，瀏覽器實測兩個入口同頁、HTML notes 不執行、遠端瀏覽器／Telegram／訪客無控制、外部與分享連結交外部瀏覽器；未完成 restart 不可觸發 helper。[after: 2.3]
- [x] 2.5 完成 Stable release checks 的啟動及六小時背景排程，僅呼叫 Check，不用會進入安裝流程的 CheckInterval／CheckAndInstall；背景只更新快照，不開設定、不下載，下載／驗證／ready／重啟期間跳過，關閉程序時停止排程。驗證：TestDesktopUpdateFlow 用假時鐘確認初始化與間隔各觸發一次、背景錯誤可在設定查看、各忙碌狀態零重入，無需實際等待六小時。[after: 2.4]

## 3. 任務保護與確認重啟

- [x] 3.1 實作「helper 優先與任務維護閘門」的共用 admission 與任務生命週期追蹤，支援 Idle-only update restart：ws 的 TryBeginDesktopUpdate 在同一同步邊界檢查 agent、shell、取消後仍在清理、待確認與 queued 工作；失敗回 ErrUpdateBusy，成功回可釋放的維護權；維護中不得承認新的工作。定位：internal/ws/taskmanager.go、internal/ws/shell.go 與必要的 queue 查詢。驗證：新增 TestDesktopUpdateAdmission，覆蓋空閒、各種忙碌、取消 map 已刪但清理未完、兩個重啟競爭、釋放後可正常 admission，並以 go test -race ./internal/ws ./internal/db 檢查同步。
- [x] 3.2 把 Idle-only update restart 的 admission 接到所有 agent／shell、WS／MCP loopback 與 queue dispatch 路徑，追蹤從第一次取消／執行副作用前到真正結束的生命週期；維護時在寫 running／消耗 queue 前回 update_in_progress，不刪訊息或強停任務。定位：internal/ws/handler.go、internal/ws/taskmanager.go、internal/ws/shell.go、internal/db/queue.go。驗證：TestDesktopUpdateAdmission 覆蓋逐一路徑、排隊與開始任務之間的競爭、維護被拒後沒有子程序及 DB running 副作用，原 queue 保留；既有 taskmanager 與 queue 測試通過。[after: 3.1]
- [x] 3.3 完成 Idle-only update restart 及 Data preservation and bounded failure recovery，接上 Server 暴露的維護操作與使用者「重新啟動並更新」：先取得閘門再 Restart，busy 保留 ready 且不生成 helper；helper 失敗釋放閘門並保留 ready，成功後封鎖新工作直到 HTTP／DB 正常關閉；只替換當前 desktop EXE、保持 cwd／資料及 server 檔案，重用 Wails 替換／程序建立失敗的備份復原，不用 ShouldQuit 當唯一檢查。驗證：TestDesktopUpdateFlow 的 busy、helper_failed、閘門釋放與資料保留案例，TestDesktopUpdateAdmission 的新工作競爭；啟用 2.4 的 restart 動作後手動確認只有明確點擊才產生 helper。[after: 2.5, 3.2]

## 4. 發佈方式與使用文件

- [x] 4.1 實作「保留安裝包並新增桌面更新資產」，達成 Compatible release artifacts、Release identity and integrity 與 Complete release publication and bootstrap 的 CI 合約：Windows ZIP／Linux tar.gz 保持原用途，desktop production bytes 同時放 ZIP 與版本化 EXE，checksums.txt 涵蓋所有程式資產；tag／VERSION／version.Version 與建置 SHA 必須一致；草稿資產齊備且 checksum 驗證後才公開為 latest，重跑只重用同 tag 草稿，已公開版本禁止覆寫；build.bat 使用 production，go run 保留開發模式。定位：.github/workflows/release.yml、build.bat、internal/version/version.go。驗證：在隔離 stage／fixture API 檢查套件清單、兩份 desktop bytes 相同、缺一資產不公開、既有草稿重用、公開 Release 不改寫、版號不符即失敗，SHA256SUMS 全部驗證通過；不要為測試在公開 GitHub 發佈 Release。[after: 3.3]
- [x] 4.2 補齊 Complete release publication and bootstrap 與「使用者資料保留與失敗復原界線」文件：README.md、README.zh-TW.md、.github/release-header.md 說明首次手動升級一次、後續檢查／下載／確認重啟、工作忙碌與可寫目錄限制、config／DB／附件／日誌保留、只更新 desktop、手動恢復舊 EXE；internal/static/AGENT.md 說明 embed production／開發磁碟模式，CHANGELOG.md 記錄功能；明確區分 SHA256 完整性與簽章身分驗證，不宣稱離線完整可用或新程序崩潰後可自動回滾。驗證：逐項比對兩語 README 與 Release header、建置命令及實際包內容，範例不覆寫使用者資料。[after: 4.1]

## 5. 整合驗證

- [x] 5.1 完成 Windows 真程序更新及全部規格驗收：用隔離 fixture 1.0.0／1.0.1 與本機更新來源，在無磁碟前端且原 port 被桌面服務使用的環境走檢查→下載→確認重啟；比對新 PID、版本／SHA、新嵌入前端、原 port、config、DB 記錄、附件、保留的 log 內容與未改 server 檔案；分別注入 checksum、連線、不可寫目錄、busy、helper、替換／建立程序失敗，確認與兩份 spec 相符，記錄通過與實際限制。驗證：go test ./internal/...、go test -tags production ./internal/...、go test -race ./internal/ws ./internal/db、Windows production desktop／既有 server 建置及隔離端到端腳本通過；所有測試只用暫存資料，提交可重跑的檢查與結果，不建立公開測試 Release。[after: 4.2]

