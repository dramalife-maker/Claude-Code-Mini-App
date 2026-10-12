# Antigravity 審查問題修復計劃

> 狀態：修復完成，回歸測試與 Chrome 元件驗證通過
> 日期：2026-10-12
> 範圍：修復本輪審查已重現的三個 P2 問題，沿用既有 runner、WebSocket 與 quota 架構。

## 1. 空白輸出的錯誤收尾

**問題**：設定 `CC_AGY_FORCE_PRINT_TEXT=1` 後，CLI 若 exit 0 且 stdout 為空白，`Run()` 回傳 nil，也不發送完成或錯誤事件，會話停留在執行中。

**修改位置**：`internal/antigravity/runner.go`、`runner_test.go`。

- [x] 在 `runPrintText()` 中補回空白輸出檢查：程序成功退出、未取消、輸出為空白時，發送 `EventError`，錯誤訊息明確說明 CLI 沒有回傳內容。
- [x] 保留 stderr 作為診斷資訊，讓既有 WebSocket 錯誤收尾流程完成訊息並恢復閒置狀態。
- [x] 沿用程序失敗及取消的既有處理，避免同一失敗重複發送錯誤或被判為成功。
- [x] 在現有測試檔使用假 CLI，重現 exit 0／空白 stdout，確認收到錯誤且沒有成功完成事件；同一組測試涵蓋正常文字輸出與取消情境。

**完成條件**：空白輸出會收到明確錯誤，訊息能收尾，會話不再卡在執行中。

## 2. 模型標籤與執行參數一致

**問題**：執行參數優先使用 `sess.Model`，但 Antigravity 的標籤解析忽略該欄位；切換模型後仍顯示原始 CLI 引數中的模型或固定預設模型。

**修改位置**：`internal/ws/model.go`、`handler.go`、`model_test.go`。

- [x] 將 Antigravity 與舊 `gemini` 別名納入既有「串流沒有模型資訊」判斷，重用 `resolveStreamless()`。
- [x] 模型解析與執行保持同一優先序：`sess.Model` → `cli_extra_args` 的 `--model` → 目前的 fallback。明確選擇必須覆蓋過期的 `ActiveModel`。
- [x] 在 `set_model` 時更新並廣播標籤；執行前也解析及持久化模型資訊，讓重新連線後保持一致。沿用既有 `model_update` 訊息。
- [x] 執行前的 provider 判斷使用共用判斷函式，保留目前 Kiro ACP 的處理。
- [x] 擴充現有回歸測試，涵蓋 Antigravity／gemini 的選擇優先序、過期 `ActiveModel`，以及清除選擇後退回 CLI 引數或 fallback。

**完成條件**：切換模型、開始執行及重新連線時，標籤都與傳給 CLI 的模型選擇一致。

## 3. 顯示兩組帳戶配額

**問題**：解析器已取得 Gemini 與 Claude/GPT 的四個用量窗口，但 formatter 只顯示 Gemini，可能掩蓋 Claude/GPT 已耗盡的額度。

**修改位置**：`internal/quota/display.go`、`display_test.go`、`internal/static/js/core/core.js`、`scripts/quota-compact.test.cjs`。

- [x] 保留並顯示 `session`、`weekly`、`session_claude`、`weekly_claude`，使用 Gemini／Claude/GPT 標籤區分。
- [x] 沿用已使用百分比、四捨五入及重置倒數規則；缺少的窗口略過，兩組都無資料才回傳 `—`。
- [x] 配額仍為帳戶級快取，以現有 `display_text` 傳遞兩組資料。
- [x] 擴充 formatter 測試：四個窗口同時存在、Claude/GPT 已耗盡、只有其中一組、空資料。
- [x] Chrome 驗證桌面與手機徽章：完整資訊可查看，耗盡的配額呈現既有警示色；手機精簡徽章改為顯示兩組窗口的最高使用率。

**完成條件**：Gemini 為 0% 已使用、Claude/GPT 為 100% 已使用時，畫面能辨識 Claude/GPT 已耗盡。

## 執行順序與驗證

1. 依序處理空白輸出、模型標籤、配額顯示；每項先加入重現測試，再修改實作。
2. 每項修復執行對應套件測試，三項完成後執行全專案測試及建置：

   ```powershell
   go test ./internal/antigravity ./internal/ws ./internal/quota ./internal/usage
   go test ./...
   go build ./cmd/server ./cmd/desktop
   node scripts/quota-compact.test.cjs
   ```

3. 手動驗收空白輸出錯誤收尾、模型切換及重新連線、兩組配額顯示與刷新。
4. 驗收完成後勾選清單，將此計劃移至 `docs/plan/done/` 並更新工作清單。

## 完成紀錄

- 新增回歸測試在修復前確實失敗，修復後均通過。
- `go test ./...`、伺服器／桌面版建置及 Node 配額檢查均通過。
- 假 CLI 驗證空白／純空白字元 stdout、正常文字、非零退出及取消情境。
- 本地 WebSocket 整合測試驗證模型切換廣播、執行前推送、重新連線及清除選擇。
- Chrome 使用專案原有 QuotaBadge／QuotaSegments 元件，在桌面 1280px 及手機 390px 視窗驗證兩組配額、紅色耗盡警示、完整 tooltip、手機文字不裁切及刷新回呼。
- 本輪使用假 CLI 與獨立元件頁驗證；真實 CLI 帳戶配額、正式聊天頁及 Telegram WebView 實機驗收尚未執行。
- 後續 Ponytail 簡化：移除空的停用表及無效分支、刪除未使用的模型解析欄位、以 Go `min`／`max` 限制百分比、以 `slices.Equal` 比較測試結果，淨減少 33 行；保留新建會話限制，測試與建置再次通過。
