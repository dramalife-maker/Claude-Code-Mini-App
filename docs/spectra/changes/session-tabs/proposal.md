## Why

目前桌面版同一時間只能「主欄一個會話＋可選的副欄一個會話」：從左側列表點另一個會話就會把主欄換掉，原本的對話要再找一次才切得回去；想同時照看多個 agent 時，只能靠兩欄分割，而且只有兩格。使用者想要像 VSCode 的編輯器分頁：開過的會話留在頂端分頁列、一鍵切換，並能從左側列表直接把會話拖進分頁列或畫面邊緣來分割。

## What Changes

- 桌面寬螢幕（≥1024px，即現有 `isWideScreen` 版面）每個會話群組（group）頂端新增分頁列：每個分頁顯示會話名稱、agent 圖示與狀態點（執行中／待授權），可關閉（按鈕、滑鼠中鍵），超出寬度時水平捲動。
- 群組最多兩個（沿用現有雙欄分割的上限、方向切換、分隔條拖曳與比例）；每個群組各自有一串分頁與一個作用中分頁，只有作用中分頁掛載 `ChatView`（每個群組最多一條 WebSocket，與現況相同）。
- 同一個會話同一時間只會出現在一個分頁；再次開啟已開啟的會話時，聚焦該分頁而不重複建立。
- 從左側會話列表點擊：在目前聚焦的群組開啟（或切到）該會話的分頁。
- 從左側會話列表拖曳會話：放到分頁列＝在放開位置插入分頁；放到群組內容區中央＝加入該群組；放到畫面邊緣＝分割並開出第二個群組（左右邊緣為左右分割、上下邊緣為上下分割）。拖曳時顯示放置位置預覽。
- 分頁本身可拖曳：同一分頁列內重新排序、拖到另一個群組。群組的最後一個分頁被移走或關閉時，該群組消失，版面回到單一群組。
- 分頁狀態（各群組分頁順序、作用中分頁、聚焦群組、方向、比例）存入 localStorage，重新整理後還原；舊的 `miniapp_split_chat_state`（主欄會話＋副欄會話）自動轉成分頁，已被刪除的會話自動從分頁移除。
- `Ctrl+\` 改為：單一群組時把作用中分頁移到新的第二群組；已有兩個群組時合併回一個。`Ctrl+Alt+←/→` 繼續切換群組焦點。
- 窄螢幕（<1024px）維持現行單欄／手機行為：不顯示分頁列、不支援拖曳。側欄會話列原有的「在副視窗開啟」按鈕已移除，第二群組改以拖放（或 Ctrl+\）開啟；窄螢幕本來就只有單一面板。

## Non-Goals

- 不支援三個以上群組、巢狀分割、分頁釘選、分頁群組命名。
- 不做觸控裝置上的拖曳（HTML5 拖放在觸控不可用），觸控與手機沿用現有按鈕開啟方式。
- 不保留背景分頁的 WebSocket 連線；非作用中分頁的狀態靠側欄既有的輪詢（狀態點、未讀徽章）呈現。
- 不改後端、API、WebSocket 協議與 session 資料結構。
- 不改 `ChatView` 內部（輸入列、訊息列表、頂欄）。

## Capabilities

### New Capabilities

- `session-tabs`: 桌面版會話分頁列、分頁開關與去重規則、從側欄與分頁拖放（排序、跨群組、邊緣分割）、分頁狀態持久化與舊分割狀態遷移、窄螢幕行為不變。

### Modified Capabilities

- `dual-chat-session`: 「每個 pane 獨立選擇會話」改為每個群組的作用中分頁；「分割狀態持久化」改為持久化分頁群組。

## Impact

- `internal/static/js/app.js`：以單一分頁群組狀態取代 `session`／`secondarySession`／`isSplit`／`activePane`／`splitRatio` 等分散的 state（窄螢幕路徑透過衍生值維持行為），`selectSession`、`handleCreated`、鍵盤快捷鍵、持久化與舊資料遷移都改走它。
- 新增分頁群組的純邏輯（reducer）、分頁列元件、群組外框元件（約為 `internal/static/js/hooks/`、`internal/static/js/ui/` 下的新檔），並登記到 `internal/static/index.html`。
- `internal/static/js/ui/SplitLayout.js`：群組內容改為帶分頁列的群組外框，放置預覽覆蓋層。
- `internal/static/js/session/SessionView.js`：會話列變成可拖曳、點擊走新的開啟邏輯。
- `internal/static/js/ui/QuickSwitcher.js`：選取後以分頁開啟於聚焦群組。
- `scripts/tab-groups.test.cjs`（新增）：分頁群組 reducer 的 Node 測試，沿用 `scripts/*.test.cjs` 慣例。
- `internal/static/AGENT.md`：登記新增檔案。
- 無後端、API、依賴變更。
