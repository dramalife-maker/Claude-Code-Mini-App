## 1. 浮動引言元件與格式化邏輯

- [x] 1.1 實作 Markdown 引言格式化工具函式 `formatMarkdownQuote`，將單行與多行文字分解並在各行前綴 `> `，處理末尾雙換行；建立驗證腳本以 node 斷言格式化輸出符合 Markdown quote formatting and input focus injection 與 Markdown quote insertion and input focus。
- [x] 1.2 建立 `SelectionQuotePopover` 元件（於 `internal/static/js/chat/SelectionQuotePopover.js`）並在 `index.html` 引入樣式，接收選取範圍矩形座標並進行水平垂直螢幕防溢出計算；以瀏覽器驗證彈出鈕座標置中與 Floating popover positioning and viewport clamping 正確運作。 [after: 1.1]

## 2. 框選事件監聽與輸入框注入

- [x] 2.1 在 `ChatView.js` 實作訊息容器選取事件監聽器，偵測使用者訊息、Agent 回應或程式碼內容之文字反白，過濾輸入框內選取與純空白，並在捲動、點擊外部或按下 Esc 鍵時自動收合；以手動選取訊息本文驗證 Selection event listening and boundary detection 與 Text selection detection and quote popover display。 [after: 1.2]
- [x] 2.2 串接引言按鈕點擊處理器，將格式化引言寫入 `chatInputRef`（支援空輸入框與已有草稿文字末端追加），並自動觸發 `focus()` 將游標移至引言末尾且清除選取；以手動選字點擊引言驗證輸入框內容更新與游標位置，確認滿足 Markdown quote insertion and input focus。 [after: 2.1]

## 3. 多欄分割路由與整體驗證

- [x] 3.1 支援雙欄分割聊天室（Split View），確保引言操作精確注入選取文字來源之面板輸入框，避免跨面板誤寫；在雙欄視窗下分別框選左欄與右欄訊息，驗證 Multi-pane routing for split chat view 與 Target pane isolation in split view 正確運作。 [after: 2.2]
- [x] 3.2 執行前端 JS 語法檢查（使用 `poc/babel-syntax-check.js` 或 Node 驗證靜態檔案），並進行端到端手動驗證，確認觸控裝置長按選單與桌面滑鼠框選引言回覆無衝突。 [after: 3.1]
