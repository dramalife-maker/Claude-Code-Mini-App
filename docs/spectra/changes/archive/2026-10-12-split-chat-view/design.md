## Context

本專案目前於 `App.js` 中管理單一活動會話（`activeSessionId`），並渲染單一 `ChatView` 實例，該實例透過 `useChatSocket` 建立與伺服器的單一 WebSocket 連線。當使用者希望同時比對多個會話或即時監督多個並行運行的 Agent 時（例如 Claude Code 進行架構分析，同時 Antigravity 實作程式碼），必須不斷手動切換會話。

本設計旨在引入左右並排的雙欄聊天室佈局（Split View），同時完整保留現有的單欄操作體驗、狀態機穩定性與系統效能。

## Goals / Non-Goals

**Goals:**
- 在桌面視窗與平板寬螢幕上提供流暢的左右雙欄分割體驗。
- 維持兩欄完全隔離的 WebSocket 生命週期、串流狀態、訊息快取與工具授權確認流程。
- 支援滑鼠拖曳調整欄寬比例，並具備最小寬度保護與雙擊重設。
- 在窄螢幕寬度（< 900px）或 Telegram Mini App 手機模式下自動平滑降級為分頁切換。
- 在本機儲存（LocalStorage）中持久化記憶上次的分割偏好與開啟的會話。

**Non-Goals:**
- 不支援 3 個或更多面板的分割，亦不支援網格排列。
- 不支援上下垂直分割。
- 不進行跨面板自動文字導流（即不自動將左欄的回覆管線化送入右欄）。

## Decisions

### Decision: Independent React component instances for ChatView
不修改 `ChatView` 的內部資料結構使其同時承載雙會話，而是渲染兩個完全獨立的 `ChatView` 實例：
- 主面板（Primary Pane）：`<ChatView sessionId={primarySessionId} isFocused={activePane === 'primary'} ... />`
- 副面板（Secondary Pane）：`<ChatView sessionId={secondarySessionId} isFocused={activePane === 'secondary'} ... />`

**決策理由**：`ChatView` 及其子元件（`ChatMessages`, `ChatInput`, `useChatSocket`）本身已經對單一會話狀態具備極佳的封裝性。渲染兩個獨立實例能確保各面板狀態完全隔離，串流事件互不干擾。

**備選方案**：
- 將雙會話合併至單一全局 store：重構風險極大，且容易在 WebSocket 訊息分發時產生競態條件。

### Decision: Pane-scoped WebSocket state management
每個 `ChatView` 實例繼續沿用既有的 `useChatSocket(sessionId)` Hook。伺服器原生支援同一個前端客戶端針對不同會話同時建立多個 `/sessions/:id/ws` 連線。

**決策理由**：後端 Go 伺服器無需任何改動，各面板可完全自主處理重連、任務狀態更新與授權彈窗。

**備選方案**：
- 前端單一多工 WebSocket 連線：需重新定義後端多工通訊協定，破壞向後相容性。

### Decision: LocalStorage layout persistence with safe fallback
分割狀態持久化於 `localStorage` 的 `miniapp_split_chat_state`：
- `enabled`: boolean
- `secondarySessionId`: string 或 null
- `splitRatio`: number（預設 0.5，限制於 0.25 到 0.75 之間）

若還原時發現儲存的 `secondarySessionId` 已不存在於會話清單中（例如在其他裝置被刪除），應用程式會優雅清除副面板並退回單欄模式。

**決策理由**：防止因無效的歷史會話 ID 導致介面報錯或卡住。

### Decision: Responsive collapse below 900px breakpoint
當視窗寬度小於 900px（或處於 Telegram Mini App 手機環境）：
- 自動折疊左右並排佈局。
- 若分割模式為開啟，頂部提供分頁按鈕（Tab Switcher），允許使用者在兩者間一鍵切換。

**決策理由**：聊天室需要至少 320px–360px 才能良好排版程式碼區塊與工具卡片；在手機等小螢幕上強行並排會嚴重破壞可讀性。

## Implementation Contract

- **可觀察行為**：
  - 點擊頂部「分割視窗」按鈕即可在右側開啟副面板。
  - 點擊任一面板即可將焦點切換至該面板（帶有邊框或標題高亮指示）。
  - 滑鼠拖曳分割線即時調整左右寬度，雙擊重設為 50/50。
  - 在任一欄送出 prompt、中斷任務或批准工具授權，絕不影響另一欄的操作。
  - 關閉副面板時，主面板即刻恢復全寬顯示。
- **介面與資料結構**：
  - `SplitChatContainer`：Props `{ isSplit: boolean, ratio: number, onRatioChange: (r) => void, activePane: 'primary' | 'secondary', onFocusPane: (pane) => void }`。
  - 本地快取設定結構：`{ enabled: boolean, secondarySessionId: string, ratio: number }`。
- **驗收標準**：
  - 同時觸發兩邊 Agent 任務時，兩欄可並行且流暢進行文字與工具串流。
  - DevTools 主控台無事件衝突或 Key 重複警告。
  - 縮小視窗寬度至 900px 以下時自動平滑切換為單欄分頁，無橫向捲軸溢出。
  - 通過 ponytail-review 複雜度與過度工程審查，確保無投機性抽象與死代碼，維持程式碼精簡。

## Risks / Trade-offs

- [風險：同時雙串流增加渲染與記憶體負擔] → 緩解措施：`ChatMessages` 既有的虛擬化與筆數限制保持運作，關閉副欄時即時 unmount 釋放資源。
- [風險：全域快捷鍵在兩欄間衝突] → 緩解措施：由 `activePane` 狀態精準路由全域事件，僅對焦中的面板會響應鍵盤動作。
