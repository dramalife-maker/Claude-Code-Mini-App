## Why

Model、Effort、權限模式目前放在聊天頁頂欄（桌面是右上一排下拉，手機要先展開頂欄），而送出訊息的輸入框在畫面底部。使用者每次想「這則訊息換個 model / 調高 effort / 開權限」都得視線和手指在頂欄與輸入框間來回，手機上還要多點一次展開。Claude Desktop 把這三項放在輸入列、附件按鈕旁，設定就在要送出的訊息旁邊，這個專案也該一致。

## What Changes

- 在輸入框的按鈕列（目前 `ModeToggleBtn`、附件迴紋針、送出鈕所在列）附件按鈕旁新增三個精簡控制：權限模式、Model、Effort，外觀為小型 chip（顯示目前值＋chevron），不是整排寬下拉。
- 沿用既有行為與資料流：選擇只更新畫面草稿（`modelSel` / `effortSel` / `mode`），送出訊息時才送 `set_model` / `set_effort` / `set_mode`；執行中（`modeSwitchDisabled` 為真的狀態）仍然 disabled；Cursor 不顯示 Effort；Codex / Kiro 不顯示權限模式。
- 從頂欄移除這三個控制（桌面右上排與手機展開區），頂欄只留標題、狀態、分支、分享、開啟 VSCode/目錄、分割相關按鈕與 repo/model/quota 副標。
- 手機頂欄原本的「展開/收合」按鈕與權限 icon chip 只服務這些控制，一併移除。
- Shell 輸入模式（`inputMode === 'shell'`）不顯示這三個控制；訪客（guest）不顯示，與現況一致。

## Non-Goals

- 不改後端、WebSocket 訊息格式或 `set_model` / `set_effort` / `set_mode` 的語意。
- 不改「新建 session」表單（`NewSessionComposer`）與「轉發」彈窗（`ForwardModal`）裡的選擇器。
- 不做新的自訂彈出選單元件；chip 內部沿用原生 `<select>`（手機可用系統選擇器），日後要換成 popover 再另案。
- 不改權限授權對話框（`AWAITING_CONFIRM` 流程）。

## Capabilities

### New Capabilities

- `composer-model-controls`: 輸入框按鈕列的 model / effort / 權限模式選擇器的顯示條件、位置、互動與停用規則，以及頂欄不再重複顯示。

### Modified Capabilities

(無；既有 specs 皆未規範這些控制項。)

## Impact

- `internal/static/js/chat/ChatView.js`：輸入列新增三個控制並接上既有 state；停止把相關 props 傳給頂欄。
- `internal/static/js/chat/chat-header.js`：移除 model / effort / 權限控制、手機展開區與 `PermModeIconChip` 使用處。
- `internal/static/js/ui/ui-atoms.js`：`ModelSelect`、`EffortSelect`、`PermModeSelect` 增加精簡 chip 樣式選項；`HeaderToggleButton`、`PermModeIconChip` 若不再有使用處則移除。
- `internal/static/js/core/core.js`（`useChatHeaderCollapsed`）：頂欄收合狀態不再有用途時一併移除。
- 無後端、API、依賴變更。
