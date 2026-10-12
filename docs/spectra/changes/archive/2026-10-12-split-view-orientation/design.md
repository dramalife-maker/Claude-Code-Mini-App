## Context

目前已實作的 `split-chat-view` 支援橫向雙欄分割（左右並排），然而直屏使用者（如程式開發者常用的縱向副螢幕、旋轉 90 度顯示器）在左右分割下各欄水平寬度嚴重不足，不利於閱讀多行程式碼與 diff。為了滿足直屏場景，需要擴充支援上下分割（垂直並排）與直觀的切換方式。

## Goals / Non-Goals

**Goals:**
- 提供直向（上下）雙面板分割佈局，上方面板為主會話，下方面板為副會話。
- 支援直觀的分割方向切換（頂欄按鈕與快捷鍵 `Ctrl+Alt+O`）。
- `SplitDivider` 根據方向自動切換為水平分隔線或垂直分隔線，支援 Y 軸高度拖曳（最小高度 200px）與雙擊 50/50 重設。
- 持久化儲存分割方向設定至 LocalStorage，重整後無縫還原。
- 維持窄螢幕自適應降級與 TMA 相容性，不破壞手機操作體驗。

**Non-Goals:**
- 支援三欄以上分割或九宮格多分割。
- 獨立子視窗或浮動彈窗（PIP）。
- 更改後端 WebSocket 協定或路由架構。

## Decisions

### Decision: Dual-direction Flexbox container in SplitChatContainer
**Rationale**: `SplitChatContainer` 透過 `orientation` 屬性（`'horizontal' | 'vertical'`）動態切換容器 class（`flex-row` 或 `flex-col`）。在 `vertical` 模式下，主面板位於上方，副面板位於下方；各面板高度以 `calc(${splitRatio * 100}% - 3px)` 及 `calc(${(1 - splitRatio) * 100}% - 3px)` 計算，並確保設定 `min-height: 200px`。
**Alternatives considered**:
- 使用 CSS Grid：動態拖曳時樣式計算比 flex-basis 複雜且易閃爍，維持 Flexbox 更精簡。

### Decision: Orientation-aware SplitDivider with dynamic drag plane
**Rationale**: `SplitDivider` 依據 `orientation` 決定外觀與拖曳軸向：
- `horizontal`：寬度 6px，`cursor-col-resize`，監聽 X 軸位移，以容器寬度與 320px 邊界計算比率。
- `vertical`：高度 6px，`cursor-row-resize`，監聽 Y 軸位移，以容器高度與 200px 邊界計算比率。
雙擊事件在兩種方向皆重設比率為 0.5。
**Alternatives considered**:
- 分離為兩個獨立 Divider 元件：增加重複程式碼，單一元件內部切換軸向更乾淨。

### Decision: Explicit orientation toggle control and shortcut Ctrl+Alt+O
**Rationale**: 在頂欄分割按鈕旁提供方向切換選單或圖示按鈕（依據當前方向顯示「切換為上下分割」或「切換為左右分割」圖示），同時註冊全域快捷鍵 `Ctrl+Alt+O`（Orientation），讓鍵盤使用者能瞬間切換佈局。
**Alternatives considered**:
- 自動依螢幕長寬比判定：使用者常常自訂視窗大小，強迫自動判斷會造成非預期的佈局翻轉，以明確設定與手動切換最可靠。

### Decision: LocalStorage state persistence for orientation
**Rationale**: 將 `orientation` 欄位納入 `miniapp_split_chat_state`（JSON 物件：`{ enabled, secondarySessionId, ratio, orientation }`），預設值為 `'horizontal'`。重新整理時優先讀取該偏好，會話若失效亦不會遺失方向偏好。

### Decision: Ponytail-review simplicity criteria for orientation handling
**Rationale**: 延續極簡架構原則：
1. 嚴禁引進額外佈局函式庫或拖曳套件。
2. 切換方向純粹透過 CSS flex-direction 與尺寸運算，完全不重新掛載子元件或中斷 WebSocket 連線。
3. 程式碼必須直觀且無多餘抽象層。

## Implementation Contract

- **Behavior**:
  - 當分割模式開啟時，使用者可透過頂欄圖示按鈕或 `Ctrl+Alt+O` 切換「左右並排」與「上下分割」。
  - 上下分割時，上欄為主會話，下欄為副會話，水平分隔線可上下拖曳調整高度比例（最小 200px）。
  - 兩欄獨立 WebSocket、狀態、輸入框與焦點完全保持隔離。
  - 重新整理網頁時，方向設定與比例正確還原。
- **Interface & Shapes**:
  - `SplitChatContainer` props 新增：`orientation = 'horizontal' | 'vertical'`, `onToggleOrientation`。
  - `SplitDivider` props 新增：`orientation = 'horizontal' | 'vertical'`。
  - LocalStorage key `miniapp_split_chat_state` 新增欄位：`orientation: 'horizontal' | 'vertical'`。
- **Scope Boundaries**:
  - In Scope: `SplitLayout.js`, `app.js`, `chat-header.js`。
  - Out of Scope: 後端 API、多於 2 欄之佈局、浮動視窗。

## Risks / Trade-offs

- [Risk] 垂直高度過低時上下分割可能導致輸入框或訊息區可視範圍過小。 → [Mitigation] 設定最小面板高度 clamp（200px），若視窗高度小於 600px 則自動提示或降級為分頁切換。
- [Risk] 快捷鍵與瀏覽器原生行為衝突。 → [Mitigation] `Ctrl+Alt+O` 並非瀏覽器標準保留快捷鍵，並在 keydown 事件調用 `e.preventDefault()`。
