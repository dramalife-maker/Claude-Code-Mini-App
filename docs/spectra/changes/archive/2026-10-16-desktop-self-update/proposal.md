## Why

目前 GitHub Actions 已可發佈 Windows 與 Linux 套件，但桌面使用者仍須自行下載、解壓及替換檔案；桌面 EXE 與外部前端資源也可能來自不同版本。將正式桌面版前端嵌入 EXE，並接上現有 Wails GitHub 更新器，讓發佈與使用者更新形成完整流程。

## What Changes

- 正式桌面建置以 Go embed 內含 index.html、JavaScript 與圖示，首頁、分享頁及系統匣圖示使用同一份資源；開發建置繼續從磁碟讀取。
- 保留既有 Windows ZIP 與 Linux server tar.gz，再新增版本化的 Windows desktop 單一 EXE 更新資產；checksums.txt 涵蓋所有可下載程式套件。
- 使用已安裝的 Wails v3 updater 及 GitHub provider，提供啟動檢查、每六小時背景檢查、系統匣與設定版號旁的手動入口。六小時為產品排程設定，非效能估計。
- 僅追蹤較新的正式版，精確選擇 desktop 資產、強制驗證 SHA256，顯示更新內容與下載狀態，由使用者確認下載與重新啟動。
- helper 在讀取設定、初始化日誌、開啟 DB、監聽 port 與單一實例處理前執行；更新只替換桌面 EXE。
- 更新重啟須先取得任務維護閘門；進行中的 agent、shell、待確認與待執行工作會阻止重啟，失敗時保留目前程序或透過 Wails 的替換失敗復原流程恢復舊 EXE。
- 保留 config.yaml、資料庫與附件、日誌及原有資料路徑，補上首次手動升級、可寫入安裝位置及後續桌面更新操作文件。

## Capabilities

### New Capabilities

- `desktop-distribution`: 正式桌面嵌入資源、開發磁碟模式、相容的初次下載套件與可驗證的桌面更新資產。
- `desktop-self-update`: Windows 桌面檢查、選版、下載驗證、原生操作入口、閒置重啟與失敗處理。

### Modified Capabilities

無。既有聊天、分享權限與雙欄規格的對外合約保持不變。

## Impact

- 資源來源：internal/static/ 的 Go 資源宣告、internal/server/server.go、internal/api/share.go、cmd/desktop/main_windows.go。
- 更新與生命週期：cmd/desktop/、internal/version/version.go、internal/ws/taskmanager.go、internal/ws/shell.go、internal/ws/handler.go、internal/db/queue.go，以及相應測試。
- 設定入口：internal/static/js/ui/SettingsModal.js；僅桌面主視窗具有原生更新橋接，不新增遠端安裝／重啟 HTTP API。
- 發佈：.github/workflows/release.yml、.github/release-header.md、build.bat、VERSION 與 internal/version/version.go 的一致性檢查。
- 文件：README.md、README.zh-TW.md、CHANGELOG.md、internal/static/AGENT.md。
- 重用 Go 標準庫與現有 github.com/wailsapp/wails/v3 v3.0.0-beta.26，不增加 updater 依賴或額外更新伺服器。

