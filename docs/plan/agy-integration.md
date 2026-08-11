# Antigravity CLI（agy）整合計畫

> 狀態：規劃中（POC 通過）
> 日期：2026-08-11
> 實測環境：Windows · **agy 1.1.12**（前次評估 1.0.14）
> 關聯：`docs/spec/antigravity-cli.md`、`internal/antigravity/`、`poc/antigravity-cli/`
> POC 證據：`poc/antigravity-cli/samples/poc-results.md`、`samples/stream-json-1.1.12.jsonl`

---

## 1. 為何現在可做（先前封鎖已解除）

前次（1.0.14）兩大 blocker 於 1.1.12 皆修復（已實測）：

| Blocker | 1.0.14 | 1.1.12 |
|---|---|---|
| Issue #76：非 TTY pipe 下 `--print` 空 stdout | FAIL | **PASS**（stdout 有內容） |
| 無 stream-json | SKIP | **PASS**（`--output-format stream-json` 真 NDJSON） |

新增可用旗標：`--output-format`、`--mode`(accept-edits/plan)、`--effort`、`--json-schema`。
`--conversation` 續接、`agy models` 亦實測可用。

**ACP 不在路徑上**：agy 無 `--acp`/`acp` server（POC FAIL）。整合走 **stream-json + 每訊息 spawn**（與 `CLAUDE.md`「用後即棄」一致，不做常駐進程）。

---

## 2. 目標與範圍

**目標**：重新啟用 `agent_type=antigravity`，讓使用者能新建 Antigravity session、串流回覆、續接對話、選 model 與模式。

**MVP 範圍**：
- stream-json runner 改寫（對齊 1.1.12 事件 schema）
- 後端解除停用、前端 composer 可選
- model / `--mode` / `--effort` 旗標對應
- 單元測試 + 手動對照

**非目標（暫緩）**：
- 結構化工具事件端到端顯示（WS + 前端 tool UI）→ Phase 2
- 常駐多工進程 → 不做
- 配額徽章 → agy 無配額概念，維持隱藏
- thinking 顯示 → agy 不吐思考文字，不實作

---

## 3. 已驗證的 schema（1.1.12 與現有解析器不相容）

現有 `events.go` / `resolve.go` 假設 legacy Gemini schema（頂層 `type`、`message`/`tool_use`/`tool_result`），**全部過時**。實際：

```
event:init      → { conversation_id, init:{ cwd, tools[], permission_mode } }   // 無 model 欄位
event:step_update → { step_update:{ conversation_id, step_index, state, step_type, ... } }
    step_type:
      user_input | unknown | checkpoint | system_message
      agent_response → text_delta（該 step 的完整文字，非字元級）、usage{ thinking_tokens... }
      tool          → state ACTIVE|DONE、tool_name、tool_info{ name, parameters }（無工具輸出內容）
event:result    → { conversation_id, status: SUCCESS|ERROR, response, num_turns, usage }
```

事件對應 `agent.Event`：

| 介面事件 | 來源 | UI 是否已消費 |
|---|---|---|
| `EventSessionInit` | `init.conversation_id` | ✅ |
| `EventStreamStart` | 首個 `agent_response` | ✅ |
| `EventDelta` | `agent_response.text_delta` | ✅（整段式） |
| `EventDone` | `result.response` + `conversation_id` | ✅ |
| `EventError` | `result.status==ERROR` / stderr | ✅ |
| `EventActivity`（選用） | `tool` step 的 `tool_name` | ✅（進度提示，如 Codex/Kiro） |
| `EventToolStarted/Completed` | `tool` step ACTIVE/DONE + `tool_info.parameters` | ❌ WS 無 case，Phase 2 |
| `EventThinking` | 無文字（只有 token 數） | 不實作 |
| `EventPermDenied` | 無互動授權 | 不實作 |

---

## 4. 工作項目（對應檔案）

### 4.1 後端 runner（核心）

- [ ] **`internal/antigravity/events.go`**：重寫 `StreamEvent` 為新 schema
  （`Event`、巢狀 `Init`/`StepUpdate`/`Result`、`ConversationID`、`StepType`、`TextDelta`、`ToolInfo`）。
  刪除過時的 `type`/`message`/`IsAssistantDelta`。
- [ ] **`internal/antigravity/runner.go`**：
  - `Run` 預設走 stream-json（移除 `CC_AGY_STREAM_JSON` gate；print-text 保留為降級路徑）。
  - `appendCommonArgs`：
    - `--model`（沿用 `ArgModel`）
    - `permission_mode`：`bypassPermissions`→`--dangerously-skip-permissions`；`plan`→`--mode plan`；`acceptEdits`→`--mode accept-edits`
    - `ArgEffort`→`--effort low|medium|high`
    - `--add-dir <WorkDir>`：收斂 agy 工作範圍（緩解「跑去讀別的 repo」，見 §6）
  - `dispatch()`：改用 `switch e.Event`：init→SessionInit、agent_response→StreamStart(首次)+Delta、tool→（選用 Activity）、result→Done/Error。
  - sessionID 一律取 `conversation_id`。
- [ ] **`internal/antigravity/runner_test.go`**：改用 1.1.12 樣本；表格式解析測試（用 `samples/stream-json-1.1.12.jsonl`）；`mapPermissionMode` 三種模式測試。

### 4.2 啟用與 model 顯示

- [ ] **`internal/agent/factory.go`**：從 `disabledAgentTypes` 移除 `antigravity`（連帶 `gemini` alias 解封）。
- [ ] **`internal/model/resolve.go`** `ExtractFromAntigravityLines`：新 init **不帶 model** → 改為讀 `--model` flag / 預設值；stream 無法取得時回退 flag。低優先。

### 4.3 前端（多數已就緒，只補兩處）

- [ ] **`session/NewSessionComposer.js`**：`NEW_SESSION_AGENT_OPTIONS` 加入 `{ value:'antigravity', label:'Antigravity' }`；
  解除進階 model 輸入對 antigravity 的隱藏（`newForm.agent !== 'antigravity'`，因 `--model` 現可用）。
- [ ] **`ui/ui-atoms.js`**（約 L119-140）：antigravity 權限選項由舊的 `auto_edit/plan/yolo（--approval-mode）`
  改為對齊 1.1.12：`default / acceptEdits / plan / bypassPermissions`。
- 已就緒（無需改）：`core.js` AGENT_LABEL、chat-header quota 排除、`useChatSocket.js`、徽章。

### 4.4 文件

- [ ] `docs/spec/antigravity-cli.md`：移除停用橫幅，改寫新 schema 與旗標。
- [ ] `poc/antigravity-cli/README.md`：更新「最新實測」。
- [ ] `docs/plan/todo.md`：勾記；完成後移入 `docs/plan/done/`。

### 4.5 Phase 2（選用，不阻擋 MVP）

- [ ] `internal/ws/handler.go` 加 `EventToolStarted/Completed` case → 廣播 tool WS 訊息。
- [ ] 前端 tool 呼叫 UI（含 `tool_info.parameters`；注意 agy 無工具輸出內容）。
  此為**共用基礎建設**（Claude/Kiro 未來亦受惠），故獨立於 agy MVP。

---

## 5. 使用者可見表現（設定預期）

- 回覆**整段跳出**（`text_delta` 非字元級），不如 Claude/Cursor 平滑。
- agy 很 **agentic**：簡單問題也會自主多次呼叫工具，偶有 `status:ERROR` 迴圈。
- **慢且吃 token**（實測 trivial 任務 ~11 萬 input tokens / 19s）。可考慮預設 `gemini-3.x-flash` 降低成本。
- 中間 narration 會混入正文（agy 把過程話算進 `agent_response`）。
- 無 thinking 泡泡、無配額徽章。

---

## 6. 決策與風險

| 項目 | 決策 | 理由 / 殘留風險 |
|---|---|---|
| 生命週期 | 每訊息 spawn（stream-json） | 與現有架構一致；agy print 本為一問一答 |
| WorkDir 逸出 | 加 `--add-dir <WorkDir>` | 實測 agy 曾讀到別的 repo；`--add-dir` 收斂但**不保證**完全不逛出，需手動複驗 |
| 工具事件 | MVP 不接，Phase 2 | 正文已含 narration，先能用；結構化顯示是共用工程 |
| 成本/延遲 | 文件標註，預設可選 flash | 交由使用者選 model |
| print-text 降級 | 保留 | stream-json 若異常仍有一問一答回退 |

---

## 7. 完成定義（DOD / 驗證）

1. `go build ./...` 通過；`go test ./internal/antigravity/... ./internal/model/...` 通過。
2. 解析單元測試：用 `samples/stream-json-1.1.12.jsonl` 還原 init/agent_response/result → 對應事件正確。
3. 手動對照（登入 agy、重啟 server）：
   - 新建 **Antigravity** session → 送 prompt → 看到串流回覆 + 收尾。
   - 第二則訊息 `--conversation` 續接記得前文（密語測試）。
   - 進階填 `--model gemini-3.x-flash` → log 顯示旗標生效。
   - 權限模式 `plan` / `bypassPermissions` → log 顯示 `--mode plan` / `--dangerously-skip-permissions`。
4. 未啟用 Phase 2 時，工具事件被安全忽略（不報錯、不空訊息）。

---

## 8. 明確不做

- 不做 ACP（agy 不支援）。
- 不做常駐進程 / process manager。
- 不做配額徽章、thinking 顯示。
- 不上小模型清洗 narration（可 Phase 2 用確定性 marker 過濾，比照 Kiro）。
