## 1. 精簡樣式的選擇器元件

- [x] 1.1 `ModelSelect`、`EffortSelect`、`PermModeSelect` 新增選填 `compact` prop；傳入時渲染為輸入列用的精簡 chip 樣式（chip 樣式的選擇器，內部仍是原生 `<select>`，權限為 `bypassPermissions` / `yolo` 時用琥珀色），未傳時外觀與現在完全相同。驗證：新建 session 表單與轉發彈窗開啟後選擇器外觀不變（手動），瀏覽器 console 無錯誤。

## 2. 輸入列加入控制 [after: 1.1]

- [x] 2.1 Composer action row hosts model, effort, and permission controls：依「權限模式在輸入列一律用下拉，不用分段開關」與「控制放在附件按鈕之後，空間不足時換行而非截斷」兩項設計決定，在 `ChatView` 輸入列附件按鈕之後渲染權限、Model、Effort 三個 compact 選擇器（權限一律用 `PermModeSelect` 而非分段開關），接上既有 `mode` / `modelSel` / `effortSel` 與 `handlePermModeDraftChange` / `handleModelDraftChange` / `handleEffortDraftChange`；非訪客、agent 模式才顯示，shell 模式與訪客不顯示；窄螢幕 chip 可換行不擠掉送出鈕。驗證：桌面與 <640px 兩種寬度手動檢查可見性與換行。
- [x] 2.2 Selector availability follows agent type：Cursor session 不顯示 Effort，Codex / Kiro session 不顯示權限（沿用 `showEffortSelect` / `showPermModeSelect`）。驗證：分別開 cursor、codex、kiroacp session 手動確認。
- [x] 2.3 Selectors disabled while the agent is busy 與 Selections apply on next send：三個 chip 以 `modeSwitchDisabled` 停用；切換後只改草稿，送訊息才送 `set_*`。驗證：訊息串流中 chip 無法操作；切換 model 後送訊息，用 `mcp__miniapp__get_status` 或 session 列表確認 `active_model` 已變、未送訊息時不變。
- [x] 2.4 Dangerous permission mode is visually emphasized：權限為 `bypassPermissions` 或 `yolo` 時權限 chip 呈琥珀色，其他模式為一般色。驗證：手動切到 bypassPermissions 確認顏色，切回 default 確認恢復。

## 3. 移除頂欄控制 [after: 2.1]

- [x] 3.1 Chat header no longer duplicates the controls：從 `ChatSessionHeader` 移除桌面右上的 Model / Effort / 權限控制、手機展開區與手機的 `PermModeIconChip`、`HeaderToggleButton`，並移除對應 props 與 `ChatView` 傳入處。驗證：桌面與手機頂欄都沒有這些控制，repo / model / quota 副標仍在，console 無 `undefined` 錯誤。
- [x] 3.2 依「移除頂欄收合機制」清理失去用途的程式：先 grep 確認無其他引用，再刪除 `HeaderToggleButton`、`PermModeIconChip`、`useChatHeaderCollapsed` 及其 localStorage 讀寫 helper；`usePermModeDropdown` 因 `ForwardModal` 仍用，保留。驗證：grep 這些名稱在 `internal/static` 無殘留引用，頁面載入無 console 錯誤。

## 4. 收尾 [after: 3.2]

- [x] 4.1 Selectors in other screens are unchanged：完整手動走一遍——新建 session 表單、轉發彈窗外觀與行為不變；`go build ./cmd/server` 通過（前端是內嵌靜態檔，確認內嵌後仍可啟動並載入聊天頁）。驗證：`go build -o claude-miniapp.exe ./cmd/server` 成功，開啟聊天頁無 console 錯誤。
