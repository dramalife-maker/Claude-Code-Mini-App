## Why

使用者在與 AI 進行長篇對話或檢視程式碼時，常需要針對特定語句、錯誤訊息或某幾行程式碼進行提問與追問。目前使用者必須手動複製文字、切換焦點到輸入框、手動加上 `>` 引用符號再貼上，操作步驟繁瑣且中斷思考流程。

導入類似 Claude Desktop 與 Codex 的框選聊天回覆功能（Selection Quote Reply），允許使用者在對話訊息中選取任意文字後，直接透過浮出的「引言」快捷按鈕將選取內容格式化為 Markdown 引言填入對話輸入框，使細緻追問更加流暢直覺。

## What Changes

- 在對話檢視區實作選取範圍監聽與幾何座標計算，當使用者在訊息正文（含 Markdown 內容與程式碼區塊）選取文字時觸發。
- 提供輕量浮動引言工具列（Selection Quote Popover），在選取文字周邊顯示「引言」動作按鈕；在選取取消、點擊外部或按下 Esc 鍵時自動收合。
- 點擊「引言」按鈕時，將選取文字格式化為 Markdown 引言（`> {選取文字}\n\n`）寫入當前面板的輸入框，並將游標聚焦於輸入框引言後方，方便使用者立即鍵入追問內容。
- 支援與雙欄分割視窗（Split View）相容：引言動作精準導向當前選取文字所在的焦點面板與輸入框。
- 處理邊界情境：排除輸入框內選取、避免與程式碼區塊複製按鈕衝突、過濾純空白選取，並在行動觸控環境維持長按選單操作不互相干擾。

## Non-Goals (optional)

- 不處理跨會話（跨 Session）的文字引言轉發（此場景由現有的長按「轉發到其他會話」處理）。
- 不引入富文字結構化引用卡片，保持標準 Markdown 引言格式以確保與所有支援的 Agent 模型（Claude, Codex, Antigravity, Kiro）相容。
- 不覆寫或阻擋桌面環境的原生瀏覽器右鍵選單。

## Capabilities

### New Capabilities

- `selection-quote-reply`: 提供訊息內容文字選取偵測、浮動引言按鈕彈出、自動格式化為 Markdown 引言並填入聊天輸入框與焦點定位。

### Modified Capabilities

## Impact

- 前端元件結構：
  - `internal/static/js/chat/ChatView.js`: 整合框選監聽、引言插入與焦點控制。
  - `internal/static/js/chat/SelectionQuotePopover.js`: 新增浮動引言彈出元件與位置計算。
  - `internal/static/index.html`: 新增浮動引言工具列的樣式規則與淡入淡出動畫。
- 後端服務：
  - 無後端 API 或資料庫變更，純前端使用者體驗增強功能。
