## Context

桌面版（`isWideScreen`，≥1024px）現在由 `App`（`app.js`）以多個分散的 state 管雙欄：`session`（主欄）、`secondarySession`（副欄）、`isSplit`、`splitRatio`、`splitOrientation`、`activePane`（'primary'｜'secondary'），持久化在 `miniapp_split_chat_state`（`enabled`、`secondarySessionId`、`ratio`、`orientation`）。`SplitChatContainer`（`SplitLayout.js`）接收 `primaryContent`／`secondaryContent` 與分隔條拖曳，窄螢幕走另一條單欄／手機路徑。側欄 `SessionView` 點擊會話呼叫 `selectSession`，原本另有「在副視窗開啟」按鈕（本 change 期間已移除，改以拖放或 Ctrl+\ 開第二群組）；原本沒有任何 HTML5 拖放。每個掛載的 `ChatView` 會開自己的 WebSocket。

## Goals / Non-Goals

**Goals:**

- 以單一分頁群組狀態取代上述分散 state，成為桌面與窄螢幕共同的唯一資料來源。
- 分頁列、分頁開關、側欄與分頁的拖放、持久化與舊資料遷移。
- 窄螢幕與手機行為不變。

**Non-Goals:**

- 三個以上群組、巢狀分割、分頁釘選。
- 觸控拖曳、背景分頁保持 WebSocket。
- 改後端或 `ChatView` 內部。

## Decisions

### 單一分頁群組狀態與純 reducer

狀態形狀：`{ groups: [{ tabs: [sessionId...], active: sessionId|null }], focused: 0|1, orientation: 'horizontal'|'vertical', ratio: number }`，`groups` 長度 1 或 2。所有變更都經一個純函式 `tabGroupsReducer(state, action)`，動作包含：開啟會話（`open`：指定群組與插入位置，已存在則聚焦）、`close`、`move`（跨群組或同群組排序）、`splitDrop`（由邊緣建立第二群組）、`focus`、`setRatio`、`setOrientation`、`toggleSplit`、`prune`（移除已不存在的會話 id）。純函式便於用 Node 測試，不依賴 React。替代方案：在 `App` 內用多個 `useState` 疊加分頁 → 邏輯分散、無法單獨測試，不採用。

### 衍生值維持舊介面

`App` 由 state 衍生 `session`（群組 0 的作用中會話）、`secondarySession`（群組 1 的作用中會話）、`isSplit`（群組數為 2）、`activePane`（聚焦群組索引對應 'primary'｜'secondary'）。窄螢幕路徑、`selectSession`、`handleCreated`、`closeSecondaryPane`、`toggleSplit` 等既有呼叫點保留簽名，內部改派發 reducer 動作，這樣窄螢幕邏輯不必重寫。

### 一個會話只在一個分頁

`open` 動作先檢查會話是否已在任一群組：已存在就聚焦該群組並設為作用中，不重複。理由：每個作用中分頁各自一條 WebSocket 與一份輸入草稿，同一會話開兩份會互相搶草稿與授權對話框。

### 只有作用中分頁掛載 ChatView

每個群組只渲染作用中分頁的 `ChatView`（以 `key={session.id}`），切換分頁等同現在換會話。非作用中分頁的狀態點與未讀徽章沿用側欄既有輪詢資料（`sidebarSortedSessions` 的 `status`、未讀數）。理由：維持「最多兩條 WebSocket」的現況成本；代價是切回分頁時會重新載入歷史。

### 以 HTML5 拖放實作，拖放資料用自訂 MIME

拖曳來源（側欄列、分頁）設定 `dataTransfer` 型別 `application/x-miniapp-session`，內容為 JSON `{ sessionId, fromGroup }`；放置目標（分頁列、群組內容區）讀它並派發動作。群組內容區把放置位置分成中央與四個邊緣（以相對座標，邊緣寬度為該方向尺寸的 25%）：中央＝加入該群組、邊緣＝在只有一個群組時建立第二群組（左右邊緣設定為水平方向、上下邊緣為垂直方向，並把被拖曳的會話放在該邊緣那側）；已有兩個群組時，內容區所有位置都視為加入該群組。放置預覽用絕對定位的半透明覆蓋層。替代方案：引入拖放函式庫 → 專案無建置流程、無依賴管理，一個覆蓋層加幾個事件處理即可，不引入。

### 分頁列放在群組外框，不動 ChatView

新增群組外框元件：上方分頁列、下方 `ChatView`。`SplitChatContainer` 的 `primaryContent`／`secondaryContent` 改傳群組外框，單一群組時也顯示分頁列（與 VSCode 一致）。`ChatView` 的 props 與內部不變。

### 持久化與舊狀態遷移

新鍵 `miniapp_tab_groups_v1` 存整個狀態。載入時若新鍵不存在而舊鍵 `miniapp_split_chat_state` 存在，轉成：群組 0 放目前主欄會話、若 `enabled` 且有 `secondarySessionId` 則建群組 1；舊鍵不刪除，也不再寫入。會話列表載入完成後執行 `prune`，移除已被刪除的會話；群組變空就移除該群組。

### Ctrl+\ 與群組焦點快捷鍵

`Ctrl+\`：單一群組且該群組有至少兩個分頁 → 把作用中分頁移到新的第二群組；只有一個分頁則不動作；兩個群組 → 合併為一個（第二群組的分頁接在後面）。`Ctrl+Alt+←/→` 沿用，改為派發 `focus`。

## Implementation Contract

**Behavior**

- 桌面寬螢幕：每個群組頂端有分頁列；分頁顯示會話名稱、agent 圖示、狀態點、關閉鈕；作用中分頁有明顯樣式；中鍵點分頁等同關閉；分頁太多時分頁列水平捲動。
- 點側欄會話：在聚焦群組開啟或聚焦既有分頁。
- 拖曳側欄會話到分頁列：在滑鼠所在位置插入；到群組內容區中央：加入該群組；到畫面邊緣（只有一個群組時）：建立第二群組。拖曳過程顯示放置預覽，離開或取消後預覽消失。
- 拖曳分頁：同列重新排序；拖到另一群組的分頁列或內容區：移動；群組最後一個分頁離開後該群組消失，版面回單一群組。
- 關閉作用中分頁：改為相鄰分頁（優先右側，否則左側）；關閉某群組最後一個分頁：該群組消失；全部關閉：回到無選取會話的狀態（與現在 `session` 為 null 相同）。
- 重新整理後還原分頁、作用中分頁、聚焦群組、方向與比例；刪除會話後對應分頁自動消失。
- 窄螢幕（<1024px）：不顯示分頁列、不可拖放，行為與現在相同。

**Interface / data shape**

- 狀態：`{ groups: [{ tabs: string[], active: string|null }], focused: number, orientation: 'horizontal'|'vertical', ratio: number }`；`groups.length` 為 1 或 2；`ratio` 限制在 0.2 到 0.8。
- 動作（`type` 欄位）：`open`、`close`、`move`、`splitDrop`、`focus`、`setRatio`、`setOrientation`、`toggleSplit`、`prune`；reducer 為純函式，不修改傳入的狀態。
- 拖放資料：`dataTransfer` 型別 `application/x-miniapp-session`，內容 JSON `{ "sessionId": string, "fromGroup": number|null }`（來自側欄時 `fromGroup` 為 null）。
- localStorage 鍵 `miniapp_tab_groups_v1`（新）；`miniapp_split_chat_state`（舊，只讀一次做遷移）。

**Failure modes**

- localStorage 內容損毀或形狀不符 → 退回單一空群組，不報錯。
- 拖放資料缺失或 `sessionId` 不在目前會話列表 → 忽略該次放置。
- 側欄尚未載入完成時不執行 `prune`，避免誤刪分頁。
- 某群組作用中分頁指向不存在的會話 → 改選相鄰分頁，仍無則該群組消失。

**Acceptance criteria**

- `node --test scripts/tab-groups.test.cjs` 通過，涵蓋：開啟去重、關閉後選取鄰近、關閉最後分頁使群組消失、`splitDrop` 方向與位置、跨群組 `move`、群組折疊回單一、`toggleSplit` 兩種方向、`prune`、損毀資料退回預設、舊狀態遷移。
- 手動（暫存埠加兩個以上測試會話）：點開多個會話出現多個分頁；拖側欄會話到分頁列、內容區中央、右邊緣、下邊緣各一次結果正確；拖分頁重排與跨群組；重新整理後還原；刪除會話後分頁消失；縮到 <1024px 無分頁列且可正常使用。
- 既有的 `scripts/*.test.cjs` 全部仍通過，`go build ./cmd/server` 通過。

**Scope**

- In：分頁群組 reducer 與測試、`App` 狀態改接、分頁列與群組外框元件、側欄拖曳與點擊、快速切換器、`SplitLayout` 的放置預覽、持久化與遷移、`index.html` 與 `AGENT.md` 登記。
- Out：後端、`ChatView` 內部、觸控拖曳、三個以上群組、背景分頁的 WebSocket、分頁釘選。

## Risks / Trade-offs

- [重構 `App` 的核心 state，影響窄螢幕與所有呼叫 `selectSession` 的地方] → 以衍生值保留舊介面、分階段遷移（先加 reducer 與衍生值並讓行為不變，再加分頁列與拖放），每階段手動回歸窄螢幕與雙欄。
- [切回分頁要重新載入歷史與重連 WebSocket] → 接受；若實測不佳，再評估對最近使用的分頁保留掛載（另案）。
- [HTML5 拖放在不同瀏覽器的 dragleave 與子元素冒泡易抖動] → 放置預覽用事件計數器或只依 `dragover` 的座標更新，手動在 Chrome 驗證；Safari／Firefox 列為已知未驗證。
- [邊緣放置誤觸] → 邊緣區只在拖曳進行中啟用，且寬度固定為 25%，預覽讓使用者放手前可見結果。
- [一個會話只開一個分頁，使用者無法並排看同一會話的兩處] → 接受，理由見上。

## Open Questions

- 無。
