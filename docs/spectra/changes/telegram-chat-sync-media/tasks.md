<!--
本變更的需求已由「本機鏡像」改為「Telegram 直通」。1.x–2.4 是鏡像設計時期完成的紀錄，保留原文作為歷史，
其行為已被第 6 組取代（資料表、欄位、outbox、水位、快照 scope 均已移除）；原 2.5–5.2 為尚未開始的鏡像任務，已取消，
取消原因與去向見下方「已取消的任務」。
-->

## 1. 訊息識別與發送基礎（鏡像設計時期，已被第 6 組取代）

- [x] 1.1 建立「訊息識別與私聊方向」及 Cache, legacy and sharing boundaries 所需的持久資料：在 internal/db/db.go、message.go、attachment.go 加入 nullable account／peer／remote ID、唯一索引、原始時間、history_import、legacy、outbox 與水位資料，保留原本本機 ID 和非 Telegram 行為，新增 migration 失敗必須回報；以 TestTelegramStorageMigration 驗證重開 DB、舊列與分享引用保留、遠端唯一約束及不同帳號／peer 分隔。
- [x] 1.2 [after: 1.1] 完成 Private chat identity and cross-device direction：以 internal/tgclient/manager.go 共用 normalize／upsert 路徑處理完整及 short 私訊，其他裝置 Out 依收件 PeerUser 顯示右側、入站左側並用原始時間排序，群組／頻道／未綁 peer 不轉發；以 TestTelegramMessageIdentity 及既有 TestManagerFiltersIncomingMessages 驗證相同文字不同 ID、不相關對話、跨帳號和多個同 peer 綁定。
- [x] 1.3 [after: 1.2] 完成「發送識別與回音合併」及 Durable message identity and send reconciliation 的純文字切片：internal/ws/handler.go、internal/tgclient/runner.go 與 manager.go 以持久 tg_outbox_id／random_id 綁定待傳泡泡、RPC 短回傳、UpdateMessageID 和 Out 更新，交易內合併回音先到的資料，結果不明保持 unknown，不新增成功氣泡；以 TestTelegramTextSendReconciliation 驗證回音前後競態、重播、重啟後原識別重試及沒有文字猜測。

## 2. 歷史分頁與跨裝置補漏（鏡像設計時期，已被第 6 組取代）

- [x] 2.1 [after: 1.3] 完成「歷史游標與重連補漏」的有界歷史查詢及 Bounded initial history and lazy older pages：為指定已綁 peer 使用 MessagesGetHistory，每頁預設 50、允許 1–100，以排他遠端 ID 游標處理正序顯示、相同秒排序、服務／不支援訊息推進及到頂，不自動掃完整歷史；以 TestTelegramHistoryPagination 驗證最新頁、前頁邊界、空可見頁、重疊頁和初次只查最新頁。
- [x] 2.2 [after: 2.1] 在 SessionHandler.Messages 提供 design 的 pagination=telegram envelope、游標綁定驗證及離線／429／503 錯誤契約，完成 Cache, legacy and sharing boundaries 的查詢行為，snapshot 以純本機 keyset 游標只查原 ID 上限及建立時已驗證 account／peer 範圍，沒有當前有效範圍的新快照回 409 且附件拒絕，升級前 legacy 快照仍可讀，所有快照路徑不呼叫遠端；非 Telegram 舊 API 陣列相容；以 TestTelegramHistoryAPI 驗證無效 limit／跨 session cursor、未知終點、快照外舊時間新匯入列、改綁後凍結範圍、stale binding、僅 legacy／空會話的新快照拒絕、舊 legacy 引用與零遠端呼叫。
- [x] 2.3 [after: 2.2] 完成「時間線合併與未讀」的 HTTP／WS 合併部分及 Stable paginated timeline：在 internal/ws/handler.go、broadcast.go 及 useChatSocket.js 讓 Telegram sync 只帶有界頁，完整 upsert 按穩定識別合併而非覆蓋舊頁，pending 以 replaces_client_message_id 取代；以 scripts/telegram-chat.test.cjs 驗證 history、live、sync 競態、重複事件與已載入頁保留。
- [x] 2.4 [after: 2.3] 在 ChatView 與 useChatSocket 提供頂部按需載入、可點擊的較早訊息／重試控制、單頁請求去重、到底停止及切換取消，維持 Stable paginated timeline 的可見訊息錨點、底部跟隨與新訊息提示，分割窗格游標互不污染；以 scripts/telegram-chat.test.cjs 驗證 session 切換與分頁狀態，並以瀏覽器手動驗證補頁和圖片載入後不跳捲動位置。

## 已取消的任務（不再執行，非已完成）

- 2.5 重連補漏：取消。直通不保存本機歷史，重新整理即重新載入最新頁，沒有需要補的缺口，見 6.2。
- 3.1–3.3 附件中繼資料與延後下載：取消。改為不存任何附件資料、由 `tg-media` 串流轉發，見 6.4。
- 4.1 現有上傳流程的 Telegram 發送：由 6.4 取代。4.2 outbox group 與原識別重試：取消，改為「已送出的保留、其餘標失敗」，見 6.4。
- 5.1 歷史匯入的未讀語意：取消。Telegram 會話不計未讀，見 6.6。
- 5.2 整合驗收：由 6.7 取代，只記錄實際跑過的驗證與限制。

## 6. Telegram 直通設計

- [x] 6.1 [after: 2.4] 拆除鏡像實作：移除 messages 鏡像欄位與 migration、outbox、水位與重連補漏、legacy 回填、歷史頁快取、快照 scope、版本化 JSON 游標，新資料庫不再建立這些表與欄位，既有資料庫殘留的不 DROP。驗證：`go build ./...` 與 `go test -short ./internal/...` 通過，程式碼中不再引用 telegram_watermarks、telegram_outbox、telegram_history_pages。
- [x] 6.2 [after: 6.1] 完成 Pass-through bounded history 與 Live updates over WebSocket（歷史與即時直通）：`GET /sessions/:id/messages?pagination=telegram` 以遠端訊息 ID 游標呼叫 getHistory，回 `{messages,next_cursor,has_more,sync_state}`，錯誤對應 400／429／503 且不當成到頂；gotd update 轉同一 DTO 經 WS 推送，`telegram_scope` 事件只帶 scope 與 sync_state，嚴格 PeerUser 與帳號範圍檢查保留。驗證：`TestTelegramHistoryPagination`、`TestTelegramHistoryAPI`、`TestTelegramSyncCarriesScopeOnly`、`TestManagerFiltersIncomingMessages`、`TestManagerHandleUpdates`。
- [x] 6.3 [after: 6.2] 完成 Text send with in-memory reconciliation（文字送出）：記憶體 random_id、updateMessageID 以 `replaces_client_message_id` 取代待傳氣泡，失敗（含未連線、帳號不符、FLOOD_WAIT）標 failed 且不自動重試。驗證：`TestTelegramTextSendMemory`，以及 `scripts/telegram-chat.test.cjs` 的待傳取代與送達事件案例。
- [x] 6.4 [after: 6.2] 完成 Structured private-chat attachments、Automatic visible image previews、Explicit video and file download、Streamed transfer without local storage 與 Send uploaded photos and files（媒體）：`GET /sessions/:id/tg-media/:msgId` 以 upload.getFile 串流轉發，不落地，image 預設 inline、其餘 attachment、`no-store` 與 `nosniff`；上傳沿用現有流程，jpeg／png／webp 走 photo、其餘走 document，多檔串行、已送出的保留、其餘標 failed。驗證：`TestTelegramMediaPassThrough`；影片與檔案點擊前零請求、圖片進畫面才載入僅由前端程式審查支撐，見驗證限制。
- [x] 6.5 [after: 6.2] 完成 Stable paginated timeline（前端時間線）：依遠端 ID 與 client_message_id 合併、不覆蓋已載入頁，上捲載入較早訊息（單頁 inflight 去重、到底停止、錯誤可重試、429 尊重 Retry-After），分割窗格游標互不污染，帳號切換丟棄舊 scope。驗證：`node --test scripts/*.test.cjs`（telegram-chat 22 項）。
- [x] 6.6 [after: 6.1] 完成 Sharing is disabled for Telegram sessions（設計主題「不開放分享」）與 Unread message count calculation 的 Telegram 排除（關閉分享與未讀）：Telegram 會話建立分享（live、snapshot）一律 400，`guestAuth` 對 Telegram 會話的訪客請求一律 403 `share_forbidden`，前端隱藏分享按鈕；unread 計算排除 Telegram 會話。驗證：`TestTelegramHistoryAPI` 的分享拒絕與舊分享 403 案例；未讀排除只由查詢審查支撐，沒有專屬測試，見驗證限制。
- [x] 6.7 [after: 6.5] 記錄驗證結果：`go build ./...`、`go vet ./internal/...`、`go test -short -count=1 ./...` 全通過，`node --test scripts/*.test.cjs` 33 項全通過。驗證：以上指令的實際輸出。

## 7. 尚未完成的驗證（未完成前不宣稱端到端完成）

- [ ] 7.1 [after: 6.7] 以真實已登入、使用者授權的 Telegram 測試私聊與瀏覽器驗證：手機／電腦送出的訊息即時出現、往前載入與捲動錨點、圖片進畫面才載入、影片與檔案點擊前無內容請求、從既有上傳按鈕送圖與送檔、重啟 miniapp 後仍可載入最新頁。驗證：逐項記錄實測結果與觀察到的限制；缺少條件的項目保留為未驗證。
- [ ] 7.2 [after: 6.6] 補上 Telegram 會話不計未讀的專屬測試。驗證：新增的測試在移除 unread 查詢中的 Telegram 排除條件時失敗，恢復後通過。
