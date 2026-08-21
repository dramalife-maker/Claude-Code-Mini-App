# 外觀設定：後端持久化

> 狀態：規劃中
> 日期：2026-08-21
> 現況：Settings 彈窗已把 heading / bold / inline code / 自訂 CSS 存 `localStorage`（`cc_appearance_v1`）
> 目標：同一份設定跟 SQLite 走，換裝置／清快取／重開瀏覽器仍在

---

## 目標與範圍

**要做**：把目前外觀 JSON 存進 SQLite，REST 讀寫；localStorage 改成快取。

**不做（這輪）**：
- 側欄寬度、收合、header 收合（裝置相關，留 localStorage）
- 依 Telegram 使用者分設定（見下方決策）
- WebSocket 推播（設定不是串流）
- MCP tool
- 把 oklch 改成 hex（先前已決議先不動）

---

## 決策

### 1. 實例級，不是 per-user、也不是 per-session

現有 `sessions`、`last_read_at`、`model_options` 都是「這台伺服器一份」。`users` 只是白名單。Web 登入還可能 `tg_id=0`。

外觀是整站 UI，跟 session 無關。這輪跟現況對齊：**整份 SQLite 一份 appearance**。之後若真的要多使用者隔離，再把 `settings` 加上 scope，不必現在發明。

### 2. KV 表，不要加欄位到 `users`

Settings 左選單之後會再加分類。用 key/value，新分類只多一個 key，不用 migrate 欄位。

```sql
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '{}'
);
```

- `key = 'appearance'`
- `value` = 跟前端現有物件同一形狀的 JSON：

```json
{
  "headingColor": "oklch(0.78 0.15 65)",
  "boldColor": "oklch(0.72 0.07 145)",
  "codeColor": "oklch(0.63 0.12 275)",
  "customCss": ""
}
```

後端把這包當不透明 JSON（camelCase 跟 localStorage 一致，前端不用對欄位）。只檢查：是 object、已知欄位是字串、總大小有上限。多出來的 key 保留，之後加欄位不必改 Go struct。

### 3. REST，沿用現有 `apiFetch`

| 方法 | 路徑 | 說明 |
|---|---|---|
| `GET` | `/settings/appearance` | 沒存過 → 200 + 預設值（跟 `APPEARANCE_DEFAULTS` 相同） |
| `PUT` | `/settings/appearance` | 整包覆寫；回傳存進去的物件 |

驗證：
- `headingColor` / `boldColor` / `codeColor`：字串，最長 128；**不解析顏色**（oklch / hex / rgb 都合法）
- `customCss`：字串，最長 32 KiB（自架個人實例，CSS 注入是使用者自己的；只擋過大）
- 整包 JSON 最長 64 KiB

空字串顏色：後端原樣存；前端 `applyAppearance` 已 fallback 成預設。

---

## 同步（localStorage 當快取）

伺服器為準。localStorage 只為了第一次 paint 不要等 API、離線時還看得到上次的色。

```
開啟 App
  1. 立刻 apply(localStorage 或預設)     ← 不變，避免 FOUC
  2. GET /settings/appearance
     - 成功且伺服器有「真的存過」的資料 → apply + 寫回 localStorage
     - 成功但是預設（從未 PUT）且 localStorage 有舊資料
         → PUT 一次把本機當種子（遷移）
     - 失敗 → 維持本機，不擋畫面

按儲存
  1. 寫 localStorage + apply             ← 樂觀，跟現在 unread 一樣
  2. PUT /settings/appearance
     - 失敗 → 本機已存；按鈕旁提示「伺服器未存到」即可，不必 rollback
```

「從未 PUT」怎麼分：DB 沒有 `appearance` 這列。GET 此時回預設，並帶 `"stored": false`，前端才知道該不該上傳本機種子。有列之後一律 `stored: true`，即使內容剛好等於預設。

```json
{ "stored": true, "headingColor": "...", "boldColor": "...", "codeColor": "...", "customCss": "..." }
```

衝突：單人實例，不做 timestamp。後開的分頁以 GET 覆蓋本機。

---

## 工作項目

### 後端
- [ ] `internal/db/db.go`：migrate `settings` 表
- [ ] `internal/db/settings.go`：`GetSetting` / `PutSetting`；appearance 預設常數（與前端 `APPEARANCE_DEFAULTS` 字串一致）
- [ ] `internal/api/settings.go`：GET/PUT handler + 長度檢查
- [ ] `cmd/server/main.go`：掛 `/settings/appearance`（走現有 `authMiddleware`）
- [ ] `internal/db/settings_test.go`：roundtrip、oversized 拒絕、缺列時 stored=false

### 前端
- [ ] `core.js`：`fetchAppearance()` / `saveStoredAppearance` 改成本機 + PUT；GET 回包含 `stored`
- [ ] `app.js`：mount 仍先套本機，再 hydrate
- [ ] `SettingsModal.js`：儲存失敗顯示錯誤字；成功邏輯不變
- [ ] 左選單／色欄 UI **不動**

### 文件
- [ ] 實作完成後把本檔移到 `docs/plan/done/`，更新 `docs/plan/todo.md`

---

## 不需要做的事
- 不動 `users` / `sessions` schema
- 不改 WebSocket
- 不重寫 Settings 彈窗
- 不清 `cc_appearance_v1`（當快取；種子遷移靠第一次 GET）
