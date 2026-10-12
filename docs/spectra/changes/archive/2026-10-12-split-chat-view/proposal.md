## Why

使用者在同時監督多個 Coding Agent 時（例如使用 Claude Code 進行架構規劃、使用 Antigravity 或 Codex 撰寫程式碼、使用 Kiro 進行測試驗證），目前必須透過側邊欄或快速切換面板頻繁跳轉。這會造成嚴重的注意力切換成本，且無法即時並排觀察不同 Agent 之間的執行進度。

雙欄分割聊天室（Split View）允許使用者在桌面或平板螢幕上同時並排檢視兩個會話，即時比對不同 Agent 的產出與對話，並在同一個畫面中各自下達指令與監督進度。

## What Changes

- 在導覽列增加分割視窗切換按鈕，並支援鍵盤快捷鍵（`Ctrl+\` / `Cmd+\`）。
- 引入雙欄佈局容器（`SplitChatContainer`），支援拖曳調整左右欄寬（Draggable divider）與最小寬度限制。
- 實作獨立的面板焦點管理（Active focus），讓鍵盤快捷鍵、Quick Switcher 與文字輸入能精準路由至當前選定面板。
- 讓左右兩欄各自獨立掛載 `ChatView` 與 `useChatSocket` 連線，支援並行串流輸出與獨立的工具授權確認。
- 支援透過右鍵選單、Quick Switcher（`Alt+Enter`）或面板頂部下拉選單在副面板開啟指定會話。
- 提供自適應小螢幕降級機制：當視窗寬度小於 900px 或在 Telegram Mini App 行動環境時，自動折疊為分頁（Tab）切換模式。
- 在 `localStorage` 持久化儲存分割設定（啟用狀態、欄寬比例、副面板會話 ID）。

## Non-Goals (optional)

- 本次不支援 3 個或更多視窗分割，亦不支援網格佈局。
- 僅支援左右並排分割，不支援上下垂直分割。
- 跨 Agent 自動對話管道（例如自動將左欄輸出作為右欄輸入）不屬於本次範疇，留待後續 Agent 協作規劃。

## Capabilities

### New Capabilities

- `split-chat-layout`: 提供雙欄容器佈局、可拖曳欄寬分隔條、活動面板焦點指示，以及小螢幕自適應折疊降級。
- `dual-chat-session`: 提供各欄獨立會話選擇、並行 WebSocket 串流通訊、獨立 prompt 送出與授權確認，以及分割狀態持久化。

### Modified Capabilities

## Impact

- 前端元件結構（`internal/static/js/`）：
  - `App.js`：擴充根狀態以記錄主副會話 ID、當前活動焦點與分割開關。
  - `chat/ChatView.js`：封裝並支援面板級屬性與焦點外框。
  - `chat/chat-header.js`：新增分割視窗切換按鈕與面板狀態指示。
  - `ui/SplitLayout.js`：新建可自訂比例的可拖曳分割容器元件。
- 後端 WebSocket：無需修改後端協定，伺服器已原生支援同客戶端對不同會話建立多個獨立 WebSocket 連線。
