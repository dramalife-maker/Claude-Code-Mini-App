# POC: kiro-cli 整合驗證

## 驗證目標

### A. `--no-interactive`（既有）

1. `kiro-cli chat --no-interactive` 可在 Go 環境透過 `exec.Command` 正常啟動
2. stdout 回應格式（`> ` 前綴剝除）可被正確解析
3. 首回合後能透過 `--list-sessions` 取得 session id
4. 以 `--resume-id` 能穩定延續對話

### B. ACP（2026-07-17 實測通過 → 已整合 `kiroacp` agent）

```bash
node poc/kiro-cli/acp_roundtrip.js <workDir>
```

| 檢查項 | 結果（kiro-cli 2.12.1） |
|---|---|
| `session/new` → `sessionId` | 通過（無需 `--list-sessions`） |
| `models.currentModelId` | 通過（例：`claude-sonnet-5`） |
| `agent_message_chunk` / `tool_call` 分離 | 通過 |
| chunk 文字含 `` ``` `` fence | 通過 |
| 跨進程 `session/load` resume | **2.12.1 失敗（timeout）**；**2.16.0 通過**（見下） |

產出：`samples_acp_report.json`、`samples_acp_chunks.md`  
跨進程 resume 探測：`acp_resume_probe.js`  
同 cwd resume 完整 PoC：`acp_same_cwd_resume_poc.js` → `samples_acp_same_cwd_resume.json`

### B.1 同 directory resume（2026-08-03 / kiro-cli 2.16.0）

```bash
node poc/kiro-cli/acp_same_cwd_resume_poc.js <workDir>
```

密語記憶驗證（`POCTOKEN_*`）：

| Case | 結果 |
|---|---|
| A 同進程連續 prompt（不 load） | ✅ 記得 |
| B 同進程 `session/load` 同 cwd | ✅ 34ms，記得 |
| C 跨進程 `session/load` 同 cwd | ✅ 551ms，記得 |
| D 跨進程 `session/load` **不同** cwd | ✅ load 成功，且仍記得 |

結論：官方「可 resume」在 2.16.0 **成立**；且 cwd 是否相同**不是** resume 成敗條件（至少對對話記憶而言）。  
附註：`proc.kill()` 後 `~/.kiro/sessions/cli/<id>.lock` 常殘留，但 2.16.0 仍可 load。

調查結論：`docs/plan/done/kiro-output-markdown-and-acp.md`

## 前置需求

- `kiro-cli` 已安裝且在 PATH 中
- Kiro 帳號已登入（`kiro-cli whoami`）

## 腳本說明

| 腳本 | 用途 |
|---|---|
| `capture_session_id.ps1` / `resume_smoke_test.ps1` / `run_all_poc.ps1` | `--no-interactive` session 流程 |
| `classify_output.ps1` | TTY `> ` 分類規則 |
| `markdown_experiment.py` | heuristic 補 fence（非正式） |
| `acp_probe.js` | 早期 initialize／session/new 探測 |
| `acp_roundtrip.js` | 完整 prompt roundtrip 成功標準 |
| `acp_resume_probe.js` | 跨進程 session/load（早期探測） |
| `acp_same_cwd_resume_poc.js` | 同／異 cwd × 同／跨進程 resume 密語驗證 |

## 已確認規格（`--no-interactive`）

| 項目 | 結果 |
|---|---|
| stdout 格式 | `> <回應文字>` |
| session id | `--list-sessions` 的 **stderr** |
| resume | `--resume-id <UUID>` |

## ACP → 產品整合

- agent_type：`kiroacp`（與 `kiro` 並存）
- 實作：`internal/kiroacp/`
- UI：新建 Session 選 **Kiro ACP**
- 注意：`session/load` 在 **2.16.0+** 已可跨進程 resume；舊版 2.12.1 會 timeout（見 GitHub #6753）
