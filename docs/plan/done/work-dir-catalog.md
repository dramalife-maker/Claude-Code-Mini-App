# 工作目錄清單（獨立於 Session）

> 狀態：已完成
> 日期：2026-09-04
> 現況：新建 Session 的目錄下拉是從現存 session 的 `work_dir` 推的，砍掉最後一個用該目錄的 session 後，下拉就沒了
> 目標：目錄清單獨立保存；session 只抄路徑；刪 session 不刪清單

---

## 目標與範圍

**要做**：獨立 `work_dirs` 表當 option 清單；建立 session 時寫入；下拉改讀 `GET /work-dirs`。

**不做（這輪）**：
- 暱稱／釘選／從清單移除／最近使用排序
- `sessions.work_dir` 改 FK
- 轉發新建的目錄下拉

---

## 實作摘要

### 後端
- [x] `work_dirs (path PK)`
- [x] 啟動時從既有 session `INSERT OR IGNORE` 補種
- [x] `CreateSession` 非空 `work_dir` → 寫入清單（trim）
- [x] `GET /work-dirs`（authMiddleware）
- [x] 刪 session 不動這張表

### 前端
- [x] 新建 Session 下拉讀目錄清單
- [x] 仍可「自訂目錄…」手打新路徑（建立時寫回清單）
