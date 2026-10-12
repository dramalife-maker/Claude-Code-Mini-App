## Why

現有雙欄分割功能僅支援左右水平並排。在直立螢幕（如旋轉 90 度的直屏顯示器、直向持握的平板）或視窗寬度有限但縱向空間充足的情境下，左右分割會導致各欄寬度過於狹窄，嚴重影響程式碼塊與 Markdown 閱讀體驗。提供直觀的「上下分屏」模式與即時方向切換，讓直屏使用者能充分利用縱向高度同時監控多個 Agent 會話。

## What Changes

- **直向（上下）分屏佈局**：支援將主副聊天視窗垂直堆疊排列（上欄主會話、下欄副會話），高度預設 50/50。
- **水平／垂直拖曳切換**：`SplitDivider` 根據當前分割方向自動切換為水平分隔線或垂直分隔線，支援滑鼠／觸控垂直拖曳調整上下高度比例，並施加最小 200px 高度邊界保護。
- **直觀方向切換控制**：在頂部工具列的分割按鈕選單、分隔線操作點或快速鍵（`Ctrl+Alt+O`）提供方向切換功能，可一鍵在「左右並排」與「上下分割」之間無縫切換。
- **方向偏好持久化**：在 LocalStorage 儲存使用者的分割方向設定（`orientation: 'horizontal' | 'vertical'`），重新整理頁面時自動維持所選方向。
- **自適應窄螢幕相容**：當視窗高度過低（如 < 600px）或處於行動 TMA 模式時，仍維持分頁切換降級機制，避免畫面過度壓縮。

## Capabilities

### New Capabilities

### Modified Capabilities

- `split-chat-layout`: 擴充分割版面需求以涵蓋垂直（上下）排列模式、方向切換互動、直向分隔條拖曳邊界限制與方向設定持久化。

## Impact

- 前端元件：
  - `internal/static/js/ui/SplitLayout.js`：更新 `SplitChatContainer` 支援 `flex-col` / `flex-row` 方向切換，擴充 `SplitDivider` 支援水平分隔線樣式與 Y 軸拖曳計算，更新 `SplitViewButton` 支援方向切換選單或提示。
  - `internal/static/js/app.js`：狀態管理新增 `splitOrientation`，更新快捷鍵監聽器（支援方向切換），以及 LocalStorage 儲存與讀取邏輯。
  - `internal/static/js/chat/chat-header.js`：傳遞方向切換回調至頂欄操作區。
- 後端／API：無破壞性變更，純前端版面與狀態擴充。
