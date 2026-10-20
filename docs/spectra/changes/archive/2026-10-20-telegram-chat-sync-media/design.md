## Context

Telegram 會話（`agent_type = telegram`）綁定一個私聊對象，原本只轉發收到的文字。需求是跨裝置雙向顯示、往前讀歷史（不一次全載）、圖片自動預覽、影片與檔案點擊才下載、可上傳照片與檔案。Telegram 連線使用 gotd/td，既有的嚴格 PeerUser 篩選必須保留。

協定依據為 Telegram 官方 [歷史查詢](https://core.telegram.org/method/messages.getHistory)、[更新處理](https://core.telegram.org/api/updates)、[檔案傳輸](https://core.telegram.org/api/files) 與 [傳送媒體](https://core.telegram.org/method/messages.sendMedia)，欄位以 go.mod 已安裝的 gotd/td 型別為準。

## Goals / Non-Goals

**Goals（in scope）：**

- 指定私聊的雙向訊息顯示、原始時間順序、最新頁加按需往前分頁。
- 圖片自動預覽；影片與檔案只在點擊後下載；從現有上傳流程送出照片與檔案。
- 沿用既有驗證與 session／peer 隔離。

**Non-Goals（out of scope）：**

- 離線讀歷史、重啟後補送、送出遇 FLOOD_WAIT 的自動重試、相簿發送、未讀數同步、Telegram 會話的任何分享。
- 編輯／刪除／反應／已讀回條、群組、頻道、Secret Chat、搜尋、影片內嵌播放。

## Decisions

### 直通而非鏡像（Supersedes：本機鏡像設計）

Telegram 是唯一資料來源，歷史與即時訊息都不寫入 messages。取代的是原先把歷史匯入資料庫的設計：該設計為了防重複、防漏、防混而需要遠端唯一索引、水位與重連補漏、outbox、legacy 標記、快照 scope、版本化 JSON 游標與歷史頁快取，規模與維護成本遠超過需求；直通後這些前提消失。

代價：未連線就看不到歷史；全域訊息搜尋與 `list_activity` 看不到 Telegram 內容；Telegram 端刪除或編輯會直接反映在下次載入。這三點在需求確認時已被接受。

### 歷史分頁

`GET /sessions/:id/messages?pagination=telegram&limit=50&before=<遠端訊息ID>`。limit 預設 50、允許 1–100（固定契約值，不是效能量測）。游標是排他的遠端訊息 ID 整數字串，範圍 1..MaxInt32（TL 的 offset_id 是 int32，超出不能被截斷後送上游）。後端以 session 綁定的 PeerUser 與目前授權帳號呼叫 getHistory，轉成 DTO 後正序回傳；客戶端不能指定遠端 peer。每次請求恰好查一頁，不自動掃到起點；服務訊息或不支援訊息也推進游標；上游整頁都在邊界之外時回 503，避免游標原地踏步。

### 即時更新

gotd update 經嚴格 PeerUser 與帳號範圍篩選後轉成同一份 DTO，用 WS 推送，包含手機與電腦版發出的訊息。`telegram_scope` 事件只帶目前 scope（account／peer）與 `sync_state`（ready／offline）。帳號切換後前端丟棄舊 scope 的資料與待傳項目，不向新 scope 送舊游標。前端重新整理即重新載入最新頁，不需要補漏。

### 文字送出

送出時產生 random_id，與待傳氣泡的對應只放記憶體。收到 updateMessageID 後以 `replaces_client_message_id` 取代待傳氣泡；RPC 直接失敗、未連線、帳號不符或 FLOOD_WAIT 都標為 `failed`，由使用者重送，不自動重試。重啟後對應消失；程序在「已送出、未收到確認」時結束，使用者重送可能造成重複，這是為了省去持久化 outbox 而接受的取捨。

### 媒體

`GET /sessions/:id/tg-media/:msgId` 以 upload.getFile 串流轉發，不落地。圖片（Content-Type 為 image/*）預設 `inline`，帶 `download=1` 時為 attachment；其他一律 attachment。回應標頭保留 `Cache-Control: private, no-store` 與 `X-Content-Type-Options: nosniff`，使用者檔名只作顯示。「圖片進畫面才載入、影片與檔案點擊才請求」是前端行為：影片與檔案在點擊前不產生任何內容請求。

上傳沿用既有 POST uploads 與 WS input，Telegram 分支改呼叫 sendMedia：image/jpeg、image/png、image/webp 走 photo，其餘走 document 並保留原檔名；可只附檔；多檔按順序串行發送，每個為獨立訊息，已送出的保留，失敗的標 failed。

### 不開放分享

Telegram 是私聊，建立分享（live 與 snapshot）一律回 400；`guestAuth` 對綁定在 Telegram 會話上的訪客請求一律回 403 `share_forbidden`，使關閉前建立的舊分享也失效。前端隱藏分享按鈕。

### 未讀

`db/session.go` 的 unread 計算以 `agent_type != 'telegram'` 排除 Telegram 會話。因為 Telegram 內容不在 messages，原本的計數方式對它沒有意義。

## Implementation Contract

**HTTP：**

- 分頁回傳 `{messages, next_cursor: string|null, has_more, sync_state}`。無效 limit 或游標回 400（`invalid_limit`／`invalid_cursor`）；FLOOD_WAIT 回 429 加 `Retry-After` 與 `retry_after_seconds`（`telegram_rate_limited`）；未連線或上游失敗回 503（`telegram_unavailable`）。錯誤不得被當成已到歷史起點。
- 非 Telegram 的 messages 回傳維持既有陣列。Telegram 不帶分頁參數的舊客戶端只收到最新有界頁的陣列。
- `tg-media` 的 msgId 必須是正整數，否則 400。

**WebSocket：**

- Telegram 的 `message` 事件含 `telegram`（account_id、peer_id、message_id）、`delivery`、`client_message_id`、`attachments`，需要取代待傳時帶 `replaces_client_message_id`；`delivery` 事件帶 `client_message_id` 與錯誤訊息。

**驗證目標：**

- `TestTelegramHistoryPagination`、`TestTelegramHistoryAPI`：游標、邊界、限流、離線與錯誤、分享拒絕與舊分享訪客 403。
- `TestTelegramMediaPassThrough`、`TestTelegramTextSendMemory`、`TestTelegramSyncCarriesScopeOnly`：媒體串流、文字送出與取代、sync 只帶 scope。
- `scripts/telegram-chat.test.cjs`：前端合併、待傳取代、分頁載入與重試、session 切換與分割窗格隔離、帳號切換。

## Risks / Trade-offs

- [重啟或崩潰時的待傳訊息] → 無持久化識別，可能重複；個人用且機率低，接受。
- [每次翻頁都是一次 Telegram 請求] → 受 FLOOD_WAIT 限制；以 429 與 `Retry-After` 回報，前端顯示等待後可重試。
- [多附件部分失敗] → 已送出的保留、其餘標 failed，重送可能重複已送出的項目。
- [Telegram 會話不在 messages] → 全域搜尋、`list_activity`、MCP `get_messages` 看不到其內容。
- [沒有端到端驗證] → 見下方限制。

## 驗證限制

- 沒有開瀏覽器實際操作（捲動錨點、圖片可視範圍載入、點擊前零請求只由 Node 測試與程式審查支撐）。
- 沒有以真實 Telegram 帳號驗證跨裝置收發、媒體串流與上傳；MTProto 行為只對照 gotd 型別與 fake。
- 沒有重啟運行中的 miniapp 驗證實際載入。

## Migration Plan

沒有 migration：不再新增 Telegram 專用表或欄位。先前版本曾建立的欄位與表（messages 的鏡像欄位、outbox、水位等）若存在於既有資料庫，保留不動、不再讀寫；新資料庫不建立。回退可換回前一版二進位，舊資料不受影響。

## Open Questions

無。日後若要未讀數同步，可用 getPeerDialogs 取得 Telegram 端的未讀與已讀位置。
