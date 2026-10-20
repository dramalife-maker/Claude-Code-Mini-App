## Why

Telegram 聊天原本只轉發收到的文字；在手機或電腦版送出的訊息不會出現在此處，也不能往前讀舊紀錄，多媒體只剩文字佔位。需要讓同一個私聊能跨裝置延續，並沿用現有上傳介面傳送照片與檔案，同時避免一次抓取全部歷史或自動下載大型內容。

原提案把 Telegram 歷史鏡像進本機 messages 表，為了讓鏡像不重複、不漏、不混，連帶長出水位、重連補漏、outbox、legacy 標記、快照 scope 等機制，規模遠超過需求。改為「直通」：Telegram 是唯一資料來源，本機不保存 Telegram 聊天內容。

## What Changes

- 歷史直通：往前捲動時，後端以遠端訊息 ID 為游標直接呼叫 Telegram getHistory 分頁，轉成 DTO 回傳，不寫入 messages。開啟聊天只載入最新一頁，上捲才載更早的。
- 即時訊息直通：gotd update（含手機、電腦版發出的訊息）轉成同一份 DTO，經 WebSocket 推送，不寫資料庫。自己送出的在右、對方在左，前端只以遠端訊息 ID 合併去重。
- 文字送出：用記憶體中的 random_id；收到 updateMessageID 後以真實 ID 取代待傳氣泡；失敗標為 failed，由使用者重送。
- 媒體：圖片進入可視範圍才載入預覽；影片與一般檔案只在使用者點擊後下載；後端經 upload.getFile 串流轉發，不落地快取。上傳沿用現有附件流程，改呼叫 sendMedia（照片走 photo，其餘走 document）。
- Telegram 會話不開放分享：建立分享一律 400，關閉前已建立的分享，其訪客請求一律 403。
- Telegram 會話不計未讀數。
- 移除原鏡像設計的所有實作：messages 的鏡像欄位、outbox、水位與補漏、legacy 回填、歷史頁快取、快照 scope、JSON 游標；既有資料庫殘留的欄位不 DROP，新資料庫不再建立。

## Non-Goals

- 離線讀歷史；未連線一律回 503，不退回本機資料。
- 重啟後自動補送、FLOOD_WAIT 自動重試送出、多附件相簿發送。
- 編輯／刪除／反應／已讀回條的跨裝置同步、群組、頻道、Secret Chat、搜尋、影片內嵌播放。
- 與 Telegram 已讀狀態同步的未讀數（可日後以 getPeerDialogs 實作）。

## Capabilities

### New Capabilities

- `telegram-chat-sync`: 指定 Telegram 私聊的跨裝置雙向顯示、歷史游標分頁、即時更新、文字送出與待傳取代、前端時間線合併，以及 Telegram 會話不開放分享。
- `telegram-chat-media`: 私聊照片、影片與檔案的結構化附件、圖片自動預覽、影片／檔案明確下載、串流轉發，以及沿用附件上傳介面的 Telegram 發送。

### Modified Capabilities

- `unread-message-badge`: Telegram 會話不計未讀，其他 agent 的未讀語意不變。

## Impact

- Telegram MTProto：`internal/tgclient/`（history.go、events.go、send.go、media.go、manager.go），沿用 gotd/td。
- API：`internal/api/telegram_history.go`、`telegram_media.go`、`share.go`，`internal/server/server.go`、`guest.go`；新增 `GET /sessions/:id/tg-media/:msgId`。
- WebSocket 與前端：`internal/ws/telegram.go`、`handler.go`、`sync.go`，`internal/static/js/hooks/useChatSocket.js`、`telegramTimeline.js`、`chat/ChatView.js`。
- 資料庫：`internal/db/session.go` 的 unread 查詢排除 Telegram；不再新增 Telegram 專用表或欄位。
- 相依：新增 gotd/td（go.mod 內其餘升級為 go mod tidy 的必然結果）。
