# 外觀設定：後端持久化

> 狀態：已完成
> 日期：2026-08-21
> 現況：Settings 彈窗已把 heading / bold / inline code / 自訂 CSS 存 `localStorage`（`cc_appearance_v1`）
> 目標：同一份設定跟 SQLite 走，換裝置／清快取／重開瀏覽器仍在

---

## 目標與範圍

**要做**：把目前外觀 JSON 存進 SQLite，REST 讀寫；localStorage 改成快取。

**不做（這輪）**：
- 側欄寬度、收合、header 收合（裝置相關，留 localStorage）
- 依 Telegram 使用者分設定
- WebSocket 推播
- MCP tool
- 把 oklch 改成 hex

---

## 實作摘要

### 後端
- [x] `settings` 表（KV）：`key=appearance`，value 為外觀 JSON
- [x] `GetAppearance` / `PutAppearance`（`stored` 區分是否曾寫入）
- [x] `GET` / `PUT /settings/appearance`（authMiddleware）
- [x] 長度檢查：顏色 128、customCss 32KiB、整包 64KiB
- [x] 單元測試：缺列 stored=false、roundtrip、超長拒絕

### 前端
- [x] 進頁先套本機；`authed` 後 hydrate
- [x] 伺服器未存過且本機有舊設定 → PUT 種子遷移
- [x] 儲存：樂觀寫本機 + PUT；失敗顯示「伺服器未存到」

---

## API

| 方法 | 路徑 | 說明 |
|---|---|---|
| `GET` | `/settings/appearance` | 無列 → 預設 + `stored:false` |
| `PUT` | `/settings/appearance` | 整包覆寫，回 `stored:true` |
