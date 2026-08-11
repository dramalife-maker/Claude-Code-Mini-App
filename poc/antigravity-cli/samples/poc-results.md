# Antigravity POC 實測結果

## 2026-08-11 · agy 1.1.12（重跑）

**環境：** Windows · agy **1.1.12**（前次 1.0.14）· `C:\Users\user\AppData\Local\agy\bin\agy.exe`

### 摘要

| 探測 | 前次(1.0.14) | 本次(1.1.12) | 說明 |
|---|---|---|---|
| `probe_flags` | PASS | **PASS** | 新增 `--output-format`（text/json/stream-json）、`--mode`、`--effort`、`--json-schema` |
| `probe_headless` | **FAIL(#76)** | **PASS** | `--print` + pipe：exit 0，stdout **691 bytes**（非空）→ Issue #76 已修復 |
| `probe_stream_json` | SKIP | **PASS** | 收到 6 行 NDJSON（真實事件流） |

### 兩大 blocker 皆解除

1. **Issue #76（非 TTY 空 stdout）** → 修復。子進程 pipe 下 `--print` 有輸出。
2. **無 stream-json** → 修復。`--output-format stream-json` 可用，輸出真正 NDJSON。

### ⚠️ 事件 schema 已改變（與現有 runner 不相容）

實測 NDJSON（見 `samples/stream-json-1.1.12.jsonl`）**不是** legacy Gemini 格式。
現有 `internal/antigravity/events.go` 解析器**無法**正確處理：

| 現有 runner 預期（Gemini 相容） | agy 1.1.12 實際 |
|---|---|
| 頂層欄位 `type` | 頂層欄位 **`event`** |
| `type:init` + `session_id` | `event:init` + `conversation_id`，巢狀於 `init.{cwd,tools[],permission_mode}` |
| `type:message` + `role/content/delta` | **無**；助理文字來自 `event:step_update` 且 `step_type:"agent_response"` 的 `text_delta` |
| `type:tool_use` / `type:tool_result` | **無**；工具併入 `step_update`（`step_type`：user_input/agent_response/checkpoint/unknown） |
| `type:result` | `event:result` + 巢狀 `result.{status,response,num_turns,usage}` |

**後果：** 現有 `dispatch()` 用 `switch e.Type`，而新流每行 `e.Type` 皆為空 → 所有事件被忽略 →
stream-json 路徑 UI 收不到任何內容。**CLI 已解除封鎖，但 runner 事件解析器需改寫才能支援介面。**

### 對應 agent.Event 的改寫要點

- `event:init` → `EventSessionInit`（SessionID 取 `conversation_id`）
- `event:step_update` 且 `step_type=="agent_response"` → `EventStreamStart`(首次) + `EventDelta`(text_delta)
- `event:result` → `EventDone`（ResultText 取 `result.response`，SessionID 取 `conversation_id`）
- `--mode plan|accept-edits` 現可對應 `permission_mode` 的 `plan` / `acceptEdits`（前版無此旗標）

---

## 2026-06-30 · agy 1.0.14（歷史）

**環境：** Windows · agy 1.0.14 · 已登入 `jerry90522@gmail.com`

| 探測 | 結果 | 說明 |
|---|---|---|
| `probe_flags` | PASS | 無 `--output-format stream-json` |
| `probe_headless` | FAIL(#76) | `--print` + pipe：exit 0，stdout 0 bytes |
| `probe_stream_json` | SKIP | 1.0.14 不支援 stream-json 旗標 |

## 重跑

```powershell
$env:CC_AGY_BIN = "$env:LOCALAPPDATA\agy\bin\agy.exe"
cd poc/antigravity-cli
./run_all_poc.ps1
```
