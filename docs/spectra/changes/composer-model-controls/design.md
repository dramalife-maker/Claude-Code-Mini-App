## Context

聊天頁的 model / effort / 權限模式選擇器現在由 `ChatSessionHeader`（`chat-header.js`）渲染：桌面是右上一排原生 `<select>`／分段開關，手機要先按頂欄的展開鈕（`HeaderToggleButton`，狀態由 `useChatHeaderCollapsed` 存在 localStorage）才看得到。三者的狀態（`modelSel`、`effortSel`、`mode`）與送出時才下 `set_model` / `set_effort` / `set_mode` 的流程都在 `useChatSocket`，`ChatView` 只負責把值與 handler 傳給頂欄。輸入列在 `ChatView` 的 `ra-cmd-bar`：左邊是 `ModeToggleBtn`（agent/shell）、附件迴紋針，中間 textarea，右邊中斷與送出。

## Goals / Non-Goals

**Goals:**

- 三個控制出現在輸入列、附件按鈕旁，桌面與手機都一次點得到，不需先展開頂欄。
- 行為與資料流完全沿用：只改「在哪裡渲染」與「長什麼樣」。
- 頂欄不再重複顯示這些控制，並清掉因此失去用途的元件與狀態。

**Non-Goals:**

- 不改後端、WebSocket 格式、`set_*` 語意。
- 不改新建 session 表單與轉發彈窗的選擇器。
- 不新增自訂 popover／選單元件。
- 不改授權對話框流程。

## Decisions

### chip 沿用原生 select 並加精簡樣式

`ModelSelect`、`EffortSelect`、`PermModeSelect` 各加一個 `compact` 屬性：套用精簡 chip 樣式（無邊框底色、h-8、小字、chevron 縮小），內部仍是 `<select>`。理由：手機會彈系統選擇器、鍵盤可操作、無障礙免費，且不用新寫 popover 的定位／關閉／焦點管理。替代方案：自訂 popover（像 Claude Desktop）→ 工作量大且輸入列在窄螢幕下 popover 易被裁切，列為後續。

### 權限模式在輸入列一律用下拉，不用分段開關

`PermModeSwitch`（分段開關）在輸入列寬度放不下，輸入列一律用 `PermModeSelect` 的 compact 形式。`usePermModeDropdown` 旗標原本決定頂欄用哪種；頂欄控制移除後，這個旗標對聊天頁不再有作用，但 `ForwardModal` 仍使用，所以旗標與 prop 保留，只是 `ChatView` 不再用它決定頂欄。

### 控制放在附件按鈕之後，空間不足時換行而非截斷

按鈕列順序：`ModeToggleBtn`、附件、權限、Model、Effort、（空白撐開）、中斷、送出。列允許 flex-wrap；窄螢幕時三個 chip 換到 textarea 下方一行，避免與送出鈕擠壓。shell 模式與訪客不渲染。

### 移除頂欄收合機制

頂欄收合只為了在手機隱藏這些控制。控制移走後，`HeaderToggleButton`、手機頂欄的 `PermModeIconChip`、`useChatHeaderCollapsed` 與 `readChatHeaderCollapsedPref` / `writeChatHeaderCollapsedPref` 無使用處，一併刪除；apply 前需再 grep 確認沒有其他引用。

## Implementation Contract

**Behavior**

- 非訪客、`inputMode === 'agent'` 時，輸入列在附件按鈕右側依序顯示：權限模式（`agentType` 非 `codex`、非 `kiro`）、Model、Effort（`agentType` 非 `cursor`）。
- 每個 chip 顯示目前選擇的標籤；Model 值為空時顯示「預設」。
- 選擇只改畫面草稿（`setMode`／`setModelSel`／`setEffortSel`）；下一次送出訊息時才送 `set_*`，與現況相同。
- `modeSwitchDisabled` 為真（THINKING、STREAMING、AWAITING_CONFIRM、各 SHELL 執行態）時三個 chip 皆 disabled。
- 權限為 `bypassPermissions` / `yolo` 時，權限 chip 以琥珀色強調（沿用 `permModeIsDanger`）。
- 頂欄桌面與手機皆不再顯示這三個控制；副標列的 repo / model / quota 資訊保留。

**Interface**

- `ModelSelect` / `EffortSelect` / `PermModeSelect` 新增選填 boolean prop `compact`，預設 false；既有呼叫處（新建 session、轉發）不傳即維持原樣。
- `ChatSessionHeader` 移除 props：`showPermModeSelect`、`inputMode`、`usePermModeDropdown`、`mode`、`modeSwitchDisabled`、`onPermModeChange`、`showEffortSelect`、`modelSel`、`effortSel`、`onModelChange`、`onEffortChange`、`headerCollapsed`、`onToggleHeader`、`onExpandHeader`。

**Failure modes**

- 讀不到 `/model-options/:agentType` 時，Model chip 只剩「預設」選項（現況行為）。
- 連線中斷時選擇仍可改草稿，送出時走既有的離線提示。

**Acceptance criteria**

- 手動：桌面與手機寬度（<640px）下，輸入列可見三個 chip，頂欄無對應控制；切換 model 後送訊息，後端收到 `set_model`（可由 `get_status` 或 session 列表的 `active_model` 確認）。
- 手動：Cursor session 無 Effort chip；Codex、Kiro session 無權限 chip；shell 模式與訪客頁面三者皆不顯示。
- 手動：訊息執行中三個 chip 不可操作。
- 手動：新建 session 表單與轉發彈窗的選擇器外觀與行為不變。
- 瀏覽器 console 無因移除 props 造成的 `undefined` 錯誤。

**Scope**

- In：`ChatView` 輸入列、`ChatSessionHeader`、三個 select 元件的 `compact` 樣式、頂欄收合機制清理。
- Out：後端、`useChatSocket` 狀態邏輯、`NewSessionComposer`、`ForwardModal`、授權對話框、popover 元件。

## Risks / Trade-offs

- [輸入列變擠，窄螢幕換行後輸入區變矮] → 允許 chip 換行並在 textarea 下方獨立一行，手機實測確認送出鈕不被擠出。
- [移除手機頂欄收合後，舊使用者 localStorage 殘留偏好] → 殘留值無害，不主動清除。
- [原生 select 在輸入列外觀受限，不如 Claude Desktop 的彈出選單精緻] → 先求位置與操作路徑到位，popover 另案。
- [刪除 `HeaderToggleButton` 等元件可能漏掉其他引用] → apply 時先 grep 確認再刪，並以瀏覽器 console 檢查。

## Open Questions

- 無。
