## 1. 純邏輯與測試

- [x] 1.1 Tab group reducer is a pure function：新增 `internal/static/js/hooks/tabGroups.js` 定義全域 `tabGroupsReducer(state, action)` 與預設狀態，依「單一分頁群組狀態與純 reducer」實作 `open`、`close`、`move`、`splitDrop`、`focus`、`setRatio`、`setOrientation`、`toggleSplit`；實作「一個會話只在一個分頁」的去重、Closing tabs 的鄰近分頁選取與群組消失規則；比例限制在 0.2~0.8；登記到 `internal/static/index.html`（放在 `core.js` 之後、`app.js` 之前）。驗證：先寫 `scripts/tab-groups.test.cjs`（沿用 `scripts/quota-compact.test.cjs` 以 `vm` 載入的方式），先跑出失敗再實作轉綠，涵蓋開啟去重、關閉後鄰近選取、關閉最後分頁使群組消失、`splitDrop` 左右為 horizontal 且上下為 vertical 並把會話放在該邊那側、跨群組 `move`、群組折疊回單一、`toggleSplit` 兩種方向、輸入狀態未被修改；`node --test scripts/tab-groups.test.cjs` 通過。
- [x] 1.2 持久化與舊狀態遷移的純函式 [after: 1.1]：在同一檔案加入序列化／載入、`prune`（移除不存在的會話、空群組消失）與由舊鍵 `miniapp_split_chat_state` 形狀建立分頁群組的遷移函式，實作「持久化與舊狀態遷移」，並涵蓋 Tab state persists across reloads 與 Legacy split state migrates to tabs 兩項需求。驗證：在 `scripts/tab-groups.test.cjs` 加測試——損毀字串退回單一空群組、舊狀態（主 A、副 B、enabled、orientation vertical、ratio 0.4）轉出兩個群組且保留方向與比例、`prune` 移除已刪會話並使空群組消失；測試通過。

## 2. App 狀態改接

- [x] 2.1 衍生值維持舊介面：`App`（`app.js`）以 `useReducer(tabGroupsReducer)` 取代 `session`／`secondarySession`／`isSplit`／`splitRatio`／`splitOrientation`／`activePane` 這些分散的 state，並由 reducer 狀態衍生 `session`、`secondarySession`、`isSplit`、`activePane`；`selectSession`、`handleCreated`、`closeSecondaryPane`、`toggleSplit`、`toggleSplitOrientation` 保留簽名、內部改派發動作；持久化改寫新鍵 `miniapp_tab_groups_v1` 並在載入時做舊狀態遷移，會話列表載入後執行 `prune`。此階段不加分頁列，畫面應與現在一致。實作 Independent session selection per pane、Split view persistence 與 Narrow screens keep the current behavior 三項需求。驗證（手動，暫存埠加至少兩個測試會話）：現有雙欄改用拖放（側欄「在副視窗開啟」按鈕已移除）或 Ctrl+\ 開關、切換方向與拖分隔條仍正常、重新整理後雙欄與比例還原；縮到 <1024px 單欄與手機行為不變；`node --test scripts/*.test.cjs` 與 `go build ./cmd/server` 通過。[after: 1.2]

## 3. 分頁列與開啟邏輯

- [x] 3.1 Tab bar per group on wide screens：依「分頁列放在群組外框，不動 ChatView」新增群組外框元件與分頁列元件（新檔放在 `internal/static/js/ui/`，並登記到 `index.html`），`SplitChatContainer` 的 `primaryContent`／`secondaryContent` 改傳群組外框，單一群組也顯示分頁列；分頁顯示名稱、agent 圖示、狀態點（取自 `sidebarSortedSessions` 的狀態）、關閉鈕與中鍵關閉，作用中分頁有明顯樣式，溢出時水平捲動；落實 Only the active tab of a group is mounted（群組只渲染作用中分頁的 `ChatView`，以「只有作用中分頁掛載 ChatView」為準）與 Closing tabs。驗證（手動）：開三個會話出現三個分頁、點分頁切換、關閉作用中分頁後右側（無則左側）接手、關閉第二群組最後一個分頁後回單一群組、關閉全部後顯示空狀態；背景分頁的執行中狀態點會亮；切換時只剩作用中分頁的一條 chat WebSocket（以 DevTools 網路面板確認）。[after: 2.1]
- [x] 3.2 Opening a session reuses its existing tab：側欄 `SessionView` 點擊會話與 `QuickSwitcher` 選取都改為派發 `open` 到聚焦群組；點擊已在任一群組開啟的會話時聚焦該群組與分頁、不重複建立。驗證（手動）：連點同一會話只有一個分頁；會話開在第二群組時點它會切到第二群組；Ctrl+P 在聚焦第二群組時選會話會開在第二群組。[after: 3.1]

## 4. 拖放

- [x] 4.1 Drag a session from the sidebar：依「以 HTML5 拖放實作，拖放資料用自訂 MIME」讓側欄會話列可拖曳，`dataTransfer` 型別 `application/x-miniapp-session`、內容 `{sessionId, fromGroup}`；分頁列接受放置並在滑鼠位置插入，群組內容區把位置分為中央與四個 25% 邊緣——中央加入該群組，只有單一群組時邊緣建立第二群組（左右為 horizontal、上下為 vertical），已有兩群組時一律加入該群組；拖曳時顯示放置預覽覆蓋層，取消或離開後消失；未知 sessionId 的放置忽略。驗證（手動，Chrome）：從側欄分別拖到分頁列兩個分頁之間、內容區中央、右邊緣、下邊緣，結果與 spec 情境一致；兩群組時拖到任一內容區只會加入該群組；按 Esc 取消後預覽消失；窄螢幕側欄不可拖。[after: 3.2]
- [x] 4.2 Drag tabs to reorder and move between groups：分頁本身可拖曳，同一分頁列內重新排序、拖到另一群組的分頁列或內容區即移動；群組最後一個分頁被移走後該群組消失並回單一群組，沿用 4.1 的放置目標與預覽。驗證（手動）：同列拖曳改變順序、跨群組移動、把第二群組唯一的分頁拖回第一群組後只剩一個群組。[after: 4.1]

## 5. 快捷鍵

- [x] 5.1 Keyboard shortcuts for groups：依「Ctrl+\ 與群組焦點快捷鍵」實作：單一群組且至少兩個分頁時 `Ctrl+\` 把作用中分頁移到新的第二群組、兩個群組時合併回一個（第二群組分頁接在後面）、只有一個分頁時不動作；`Ctrl+Alt+←/→` 改派發 `focus`。驗證：在 `scripts/tab-groups.test.cjs` 補 `toggleSplit` 與 `focus` 的案例並通過；手動按快捷鍵確認分割與合併。[after: 3.1]

## 6. 收尾

- [x] 6.1 整體驗收與文件：把新增檔案登記到 `internal/static/AGENT.md` 的檔案表；`node --test scripts/*.test.cjs` 與 `go build ./cmd/server` 通過；以暫存埠完整手動走一遍所有 spec 情境（分頁、去重、關閉、側欄拖放、分頁拖放、重新整理還原、刪除會話後分頁消失、舊狀態遷移、<1024px 行為），暫存服務結束後關閉。驗證：測試與 build 輸出貼在回報中，手動結果逐項列出。[after: 4.2, 5.1]
