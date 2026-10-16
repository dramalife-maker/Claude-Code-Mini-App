# Antigravity CLI（agy）整合規格

> **取代 legacy Gemini CLI。** Gemini CLI 已退役；本專案以 `agent_type=antigravity`、子進程 `agy` 整合。
> **應用狀態（2026-10-12）：** 已啟用 Antigravity，預設解析 `init`／`step_update`／`result` 串流；回歸測試與 Chrome 元件驗證通過，真實 CLI／Telegram WebView 本輪驗收待補。
> 實作：`internal/antigravity/` · POC 與驗證：`poc/antigravity-cli/`

## 官方文件

- https://antigravity.google/docs/cli-overview
- https://github.com/google-antigravity/antigravity-cli
- 遷移指南：CLI Overview → *Migrating from Gemini CLI*

## Provider 定位

| 項目 | 規格 |
|---|---|
| 類型 | Local subprocess（`agy`，用後即棄） |
| 通訊格式 | `--output-format stream-json`（NDJSON 串流） |
| Prompt | **stdin pipe**（避開 Windows argv 截斷，支援多行輸入） |
| Session 延續 | `--conversation <session_id>` 跨輪延續對話歷史 |
| 權限模式 | 支援 `--mode accept-edits`、`--mode plan` 與 `--dangerously-skip-permissions` |
| 模型設定 | `--model <name>`（可選用 `gemini-3.8-flash-high` 等） |
| 推理強度 | `--effort <level>`，沿用會話設定 |
| 工作目錄 | 設定子進程 `cmd.Dir`，並以 `--add-dir <work_dir>` 加入工作區 |
| 配額用量 | `agy -p /usage` 解析 TSV，分別顯示 Gemini 與 Claude/GPT 的 5 小時與每週額度 |

## 啟動命令範本

```bash
agy --output-format stream-json \
  [--conversation <session_id>] \
  [--model <model>] \
  [--effort <level>] \
  [--mode accept-edits | plan] \
  [--dangerously-skip-permissions] \
  [--add-dir <work_dir>]
# prompt 經 stdin 管道餵入
```

## 事件協議（NDJSON）

`agy` 輸出的頂層事件欄位以 `event` 區分：

1. **`init`**：
   ```json
   {"event":"init","conversation_id":"<id>","init":{"cwd":"...","tools":[...],"permission_mode":"..."}}
   ```
2. **`step_update` (文字增量)**：
   ```json
   {"event":"step_update","step_update":{"step_type":"agent_response","state":"ACTIVE","text_delta":"..."}}
   ```
3. **`step_update` (工具呼叫與結果)**：
   ```json
   {"event":"step_update","step_update":{"step_type":"tool","state":"ACTIVE","tool_name":"...","tool_info":{"parameters":{...}}}}
   {"event":"step_update","step_update":{"step_type":"tool","state":"DONE","tool_name":"...","tool_info":{"output":"..."}}}
   ```
4. **`result`**：
   ```json
   {"event":"result","result":{"status":"SUCCESS","response":"...","usage":{"input_tokens":...,"output_tokens":...}}}
   ```

## Permission Mode 對應

| Mini App `permission_mode` | agy 命令列旗標 | 行為說明 |
|---|---|---|
| `default` | *(無額外旗標)* | 預設審核模式；無權限工具軟性拒絕 |
| `acceptEdits` | `--mode accept-edits` | 自動允許檔案讀寫與編輯 |
| `plan` | `--mode plan` | 唯讀規劃模式 |
| `bypassPermissions` / `yolo` | `--dangerously-skip-permissions` | 全自動放行所有工具 |

## agent_type 遷移

| 舊值 | 新值 |
|---|---|
| `gemini` | `antigravity`（factory 自動別名導向 Antigravity runner） |

舊 `gemini` Session 可繼續執行，但不可用此舊值新建 Session。新建使用 `antigravity`。

## 模型與用量顯示

- 啟動及設定頁「重新抓取」會透過 `agy models` 同步模型選單（30 秒逾時，支援 `CC_AGY_BIN`）；抓取失敗、格式錯誤或空清單時保留既有選項並回報失敗。重新開啟對話即可載入更新的選單。
- 模型選擇優先序：會話 `model` → `cli_extra_args` 的 `--model` → 標籤 fallback `gemini-3.8-flash-high`。fallback 是應用推論值；沒有指定 `--model` 時，CLI 仍使用自己的預設設定。
- 切換模型時立即發送 `model_update`，執行前更新持久化資訊；重新連線與清除選擇後重新解析，避免沿用過期標籤。
- 配額是帳戶級快取，將 CLI 回報的剩餘比例換算為已使用比例（限制在 0–100%），以 Gemini／Claude/GPT 標籤區分兩組窗口；5 小時窗口附重置倒數。
- 手機精簡徽章取兩組窗口的最高使用率，展開頂欄及 tooltip 保留各組明細；100% 使用率沿用紅色警示，支援手動刷新。

## 錯誤收尾與驗證

- `CC_AGY_FORCE_PRINT_TEXT=1` 純文字後備模式若 exit 0、未取消且 stdout 為空白，發送 `EventError` 並保留 stderr，交由 WebSocket 既有流程收尾。
- `result.usage` 已可解析，但目前沒有透過 `agent.Event` 傳遞回合 Token 統計，帳戶配額顯示另由 `/usage` 取得。
- 驗證範圍與命令見 [審查修復紀錄](../plan/done/agy-review-fixes.md)。

## 環境變數

| 變數 | 用途 |
|---|---|
| `CC_AGY_BIN` | 覆寫 agy 執行檔路徑 |
| `CC_AGY_FORCE_PRINT_TEXT=1` | 強制降級為純文字 print 模式（後備方案） |

舊 `CC_AGY_STREAM_JSON` 不再控制 runner；未強制純文字時預設使用串流。
