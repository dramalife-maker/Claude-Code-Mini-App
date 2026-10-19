# Windows 真程序端到端驗證（desktop-self-update 5.1）

以隔離的 1.0.0 → 1.0.1 fixture 與本機更新來源，驗證桌面自我更新在真實 Windows 程序上的行為：
檢查 → 下載 → 確認重啟，以及各項失敗注入。

## 前置需求

- Windows 10/11，WebView2 Runtime。
- Go（與 `go.mod` 相同版本），可 `go build`。
- Python 3.10+，並安裝 `websocket-client`（`pip install websocket-client`），供 WebView2 CDP 驅動使用。
- 驗證 ACL 與替換失敗注入會呼叫 `icacls` 與 PowerShell（系統內建）。

## 安全規則（腳本內已實作）

- 所有資料、程序與應用程式埠都在暫存目錄與 `18100–18899` 範圍；不碰 port `8080`，不碰使用者的 `config.yaml`、DB 或 workspace。fixture HTTP server 例外：使用 OS 分配的臨時埠，僅綁定 `127.0.0.1`。
- 只結束本腳本啟動的 PID，且執行檔路徑必須吻合；不以程序名稱結束任何程序。
- 替換失敗注入只在測試目錄的 exe 上持有檔案鎖，不會替換任何 repo 內或使用者安裝的 exe。
- 不建立 GitHub Release、不 push tag、不寫入 repo。更新來源是本機 `127.0.0.1` 的 fixture HTTP server。
- 目錄 ACL 注入會在 `finally` 中以 `icacls /remove:d` 還原。

## 步驟

```powershell
# 1. 建置 fixture（1.0.0／1.0.1 的 e2e 版、server、正式版），並執行守門檢查
python scripts/e2e/build_fixtures.py --work "$env:TEMP\cma-e2e"

# 2. 執行全部場景
python scripts/e2e/run_e2e.py --work "$env:TEMP\cma-e2e" [--keep]

# 2b. 只跑指定群組：happy,cleanup,matrix,not-writable,busy,helper-fail,watchdog,replacement-fail
python scripts/e2e/run_e2e.py --work "$env:TEMP\cma-e2e" --only replacement-fail

# 2c. 只跑 matrix 的指定子項（多個以 + 連接）
python scripts/e2e/run_e2e.py --work "$env:TEMP\cma-e2e" --only matrix:api-404
```

輸出：`<work>\report.json`（每個場景的檢查項、限制與程序清單）與 `<work>\evidence\`（helper 日誌）。
每次執行都會覆寫 `report.json`，需要保留時請先複製。
退出碼 `0` 表示全部 PASS；`1` 表示有 FAIL；`2` 表示參數錯誤或缺少 manifest；`3` 表示守門失敗（正式 EXE 含 e2e hook）。

## 場景

| 場景 | 注入 | 預期（規格） |
|---|---|---|
| `happy` | 無 | 真實 1.0.0→1.0.1 替換並重啟；新 PID、同 port、版本／SHA、嵌入前端、config／DB／附件／log 保留、server 檔不變 |
| `matrix` | fixture 故障 | checksum_missing／checksum_mismatch、連線／限流／500、下載斷線與中途中斷、server 拒絕連線；`api-404` 見下方說明 |
| `matrix:api-404` | fixture 回 404 | 尚無公開正式版：state=`up-to-date` 且無錯誤碼（已知行為，非失敗） |
| `not-writable` | 安裝目錄 ACL 拒絕建立檔案 | install_not_writable，且未下載 EXE |
| `busy` | 注入一筆 queued 訊息 | update_busy／busyReason=queued，未啟動 helper，ready 保留 |
| `helper-fail` | `CMA_E2E_FAIL_RESTART=1`（manager 層） | helper_failed，ready 保留，閘門已釋放 |
| `watchdog` | `CMA_E2E_HANG_SHUTDOWN=1` | ShutdownDeadline 後強制結束，新程序仍能開啟 DB 並保有資料 |
| `cleanup` | 啟動前在 exe 旁放假的 `<exe>.old.<數字>`（含最新一份與名稱相近的其他檔案） | 啟動後只清掉較舊的 `.old.*`，最新一份與其他檔案都不動，日誌有記錄 |
| `replacement-fail` | 以 FileShare.Read 鎖住目標 exe 約 45 秒 | 雙重故障：原 exe SHA 不變、`.bak` 與 helper 日誌保留；**不要求** relaunch |

### `replacement-fail` 雙重故障說明

替換（rename-aside）與還原（移除目標後改名 `.bak`）都被同一把鎖擋住。Wails helper 還原失敗（`restore failed`，結束碼 14）後不會 launch，因此不會自動重新啟動舊程序。這是 spec 中「替換失敗」的實際行為，見 `docs/spectra/changes/desktop-self-update/design.md` 已知限制。

手動恢復步驟（使用者端）：

1. 關閉占用 `claude-miniapp-desktop.exe` 的程序（或重新開機）。
2. 若 `claude-miniapp-desktop.exe` 仍可執行，直接啟動即可。
3. 若目標檔損毀或遺失，先確認 `claude-miniapp-desktop.exe.bak` 存在，將其改回 `claude-miniapp-desktop.exe` 後再啟動。
4. 配置、DB 與附件不會因此被修改。

## e2e 專用 hook（只在 `-tags e2e` 編入）

- `internal/desktopupdate/endpoint_e2e.go`：`CMA_E2E_API_BASE`（僅接受 `127.0.0.1`）、`CMA_E2E_FAIL_RESTART`。
- `cmd/desktop/e2e_hooks_windows.go`：`CMA_E2E_UNIQUE_ID`、`CMA_E2E_WEBVIEW_DATA`、`CMA_E2E_CDP_PORT`、`CMA_E2E_HANG_SHUTDOWN`。
- 正式與一般建置使用 `endpoint_default.go` 與 `e2e_default_windows.go` 的 no-op，`build_fixtures.py` 會檢查正式 EXE 不含 `CMA_E2E_`。

## 已知限制

- 驅動以 WebView2 CDP 點擊實際按鈕；設定頁以 `desktop-update-open` 事件開啟（與系統匣 OpenScript 相同），未點擊系統匣圖示本身。
- `helper-fail` 的注入在 manager 層（Restart 之前），不驗證真實 Wails helper 程序啟動失敗。
- `watchdog` 走的是 `os.Exit` 強制結束路徑，不是 TerminateProcess 崩潰。
- 真實 GitHub 發佈路徑未驗證（不存取公開 GitHub）；`api-404` 代表尚無公開正式版。
- fixture HTTP server 使用 OS 分配的臨時埠（僅綁定 `127.0.0.1`），非 18xxx；應用程式測試埠為 18xxx。
- 所有 fixture 的 `/config` 與 revision 來自 `vcs.revision`；fixture 副本在暫存目錄內 `git init`，不影響 repo。
