# Session 列表：右側只留進行中／未讀點

> 狀態：已完成
> 日期：2026-08-21
> 範圍：純前端，不動後端

---

## 現況（改前）

Session 列有兩個點：

```
[agent] [未讀紫點] [名稱] ........ [右側狀態點]
```

右側 `ra-status-dot`：
- `idle` → 綠（幾乎每個待命 session 都有，沒資訊量）
- `running` / `awaiting_confirm` → 琥珀
- 其他 → 灰

未讀紫點夾在 badge 跟名稱中間。

---

## 目標

右側同一個位置只顯示「有事發生」：

| 狀況 | 右側 |
|---|---|
| `running` / `awaiting_confirm` | 琥珀點（進行中／待授權） |
| idle 且未讀（且不是目前開啟的 session） | 紫點（未讀） |
| idle 且已讀 | **不顯示** |

進行中優先於未讀。

```
[agent] [名稱] ........ [● 或空白]
```

---

## 實作

- [x] `SessionView.js`：拿掉名稱左側未讀點；右側單一 slot（進行中 > 未讀 > 無）
- [x] `ui-atoms.js`：`idle` 不再回傳綠點 class／「待命」tooltip
- [x] `index.html`：刪 `.ra-status-dot.idle` / `.done`，加 `.unread`
- [x] 列的 unread 底色＋粗體名稱保留；hover 仍隱藏右側點

---

## 不做

- 聊天室頂欄 `SessionStateChip` 的「待命」綠點
- 後端 `status` / `last_read_at` schema
- 兩個點並排
