## Context

在目前的聊天視窗中（`ChatView.js`），使用者能閱讀使用者的提問、Agent 的 Markdown 與程式碼回應，以及 Shell 執行結果。當需要針對 Agent 回應中的特定段落、某一段函數實作或特定錯誤訊息進行細部追問時，使用者必須手動選取、複製、點擊輸入框、手動鍵入 `>` 符號貼上，流程繁瑣且容易打斷思緒。

主流 AI 介面（如 Claude Desktop 與 Codex）普遍提供「選取文字浮出引言回覆」的直覺操作。本設計在不更動後端架構的前提下，於前端對話訊息檢視區引入選取文字監聽、浮動操作列（Selection Quote Popover）以及 Markdown 引言自動填入與焦點管理。

## Goals / Non-Goals

**Goals:**

- 在使用者訊息、Agent 回應內容及程式碼區塊內選取文字時，自動計算幾何位置並浮出輕量操作列。
- 提供「引言」操作按鈕，點擊後自動將所選文字格式化為標準 Markdown 引言區塊（`> {文字}\n\n`）並填入聊天輸入框。
- 填入引言後自動聚焦輸入框游標至引言末端，支援立即鍵入後續問題。
- 支援與雙欄分割視窗（Split View）完整相容，精準將引言導流至選取來源所屬的面板輸入框。
- 具備視窗邊界限制（Clamping）、捲動或點擊外部即時關閉、以及 `Esc` 鍵快捷關閉。

**Non-Goals:**

- 不引入複雜的富文字引用卡片或附件型態，維持標準 Markdown 語法以相容所有 Agent 後端。
- 不接管或阻擋原生瀏覽器右鍵選單。
- 不提供跨 Session 的選字轉發（跨 Session 轉發維持由長按動作選單提供）。

## Decisions

### Selection event listening and boundary detection

在訊息捲動容器（`chatScrollRef`）層級監聽 `mouseup`、`keyup` 與 `selectionchange` 事件，僅當選取範圍（Selection Range）落在訊息本文節點（如 `.bubble-user`、`.bubble-claude`、`.bubble-shell` 等內部，且排除輸入框自身）時觸發。

- **Rationale**: 容器層級監聽避免在每筆訊息 DOM 上綁定獨立事件，降低長對話時的記憶體與重新渲染負擔，且能精準過濾輸入框內選取與純空白字元。
- **Alternatives considered**: 在每筆訊息元件上個別綁定 `onMouseUp`，實作較為分散且無法捕捉鍵盤 Shift+方向鍵選字行為。

### Floating popover positioning and viewport clamping

使用 `window.getSelection().getRangeAt(0).getBoundingClientRect()` 計算選取文字的幾何包圍盒，將浮動選單水平置中於選取頂部上方 8px 處；若靠近視窗頂部邊緣則自動翻轉至選取底部下方，並確保水平位置不超出螢幕左右邊界。

- **Rationale**: 貼近選取文字位置符合使用者視覺直覺與 Claude Desktop / Codex 之操作慣例。
- **Alternatives considered**: 固定於視窗底部或頂部固定工具列，遠離使用者視覺焦點。

### Markdown quote formatting and input focus injection

點擊引言按鈕後，將選取文字進行多行分解，每行前方加上 `> ` 前綴，並於引言末尾追加空行：
```markdown
> 第一行
> 第二行

```
若當前輸入框已含有未送出的草稿文字，則於現有文字末端插入（以換行分隔），並自動調用 `chatInputRef.current.focus()`，將游標設置於引言文字之後，最後清除瀏覽器選取狀態（`window.getSelection().removeAllRanges()`）。

- **Rationale**: Markdown blockquote 是所有主流大模型原生理解的引用標準，無需任何後端協定擴充。
- **Alternatives considered**: 引入獨立 Quote 附件物件，會需要修改訊息送出協定與後端 WebSocket 格式。

### Multi-pane routing for split chat view

在雙欄分割模式（`SplitLayout`）下，左右兩欄各擁有獨立的 `ChatView` 執行個體與 `chatInputRef`。引言元件由各自的 `ChatView` 局部維護狀態，確保從右欄 Agent 回應框選文字時，引言只會寫入右欄對應的輸入框，絕不干擾左欄面板。

- **Rationale**: 避免跨面板誤填，維護兩側對話上下文的獨立性。
- **Alternatives considered**: 全域單一實例（Global Singleton），容易因焦點切換造成引言填入錯誤面板。

## Implementation Contract

- **Behavior**:
  - 使用者以滑鼠拖曳或鍵盤選取對話訊息內文字時，若選取長度 > 0 且非純空白，於選取文字周邊顯示浮動「引言」按鈕。
  - 點擊「引言」按鈕：選取文字被格式化為 `> ...\n\n` 填入該面板輸入框，游標移至引言末尾，輸入框獲得焦點，浮動按鈕與瀏覽器反白選取清除。
  - 當使用者點擊訊息外任意區域、按下 `Esc` 鍵、或容器發生捲動時，浮動按鈕平滑消失。
  - 在輸入框（`textarea[data-chat-input]`）內部的選取不觸發引言按鈕。
- **Interface / data shape**:
  - 元件：`internal/static/js/chat/SelectionQuotePopover.js`，包含浮動引言工具列。
  - 輔助函式：`formatMarkdownQuote(text: string): string`，負責換行處理與 `> ` 前綴組裝。
  - 狀態介面：`selectionQuote: { text: string, rect: { top, left, width, height } } | null`。
- **Failure modes**:
  - 當選取包含不合法字元或空字串時，靜默關閉選單而不拋錯。
  - 當前輸入框處於唯讀或不可編輯狀態（如正在執行強制指令），引言按鈕顯示為禁用狀態或不提供操作。
- **Acceptance criteria**:
  - 在 Agent 回應中選取文字後顯示引言按鈕。
  - 點擊按鈕後輸入框正確顯示引言格式，游標位於引言後方。
  - 輸入框已有草稿文字時，新引言追加至輸入框且不覆蓋現有內容。
  - 在雙欄分割視窗下，在右欄選字點擊引言，僅寫入右欄輸入框。
- **Scope boundaries**:
  - **In scope**: 訊息框選事件處理、幾何定位與視窗邊界限制、Markdown 引用格式化、輸入框內容注入與游標控制、多面板隔離。
  - **Out of scope**: 後端 WebSocket 協定變更、自訂富文字引言卡片、跨 Session 轉發訊息。

## Risks / Trade-offs

- **[Risk]** 觸控裝置上的原生選字放大鏡或 Callout 可能與浮動按鈕重疊。
  → **Mitigation**: 針對觸控環境（`pointer: coarse`）設定適當邊距偏移，並保留既有的長按動作選單（MessageContextMenu）不受影響。
- **[Risk]** 捲動聊天視窗時浮動選單未跟隨而滯留在空中。
  → **Mitigation**: 在聊天捲動容器（`chatScrollRef`）觸發 `onScroll` 事件時立即關閉或重新定位浮動選單。
