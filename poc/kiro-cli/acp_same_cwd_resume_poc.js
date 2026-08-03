/**
 * PoC：驗證「ACP 同 directory 可 resume session」官方說法。
 *
 * 情境：
 *   A) 同進程、同 sessionId，連續 prompt（不 load）— 基線：對話記憶是否存在
 *   B) 同進程、同 cwd：session/new → prompt → session/load → prompt（記憶檢查）
 *   C) 跨進程、同 cwd：process A 建 session + prompt → kill → process B session/load + prompt
 *   D) 跨進程、不同 cwd：process A 建於 dir1 → process B 用 dir2 load
 *
 * Usage:
 *   node poc/kiro-cli/acp_same_cwd_resume_poc.js [workDir]
 *
 * 產出：samples_acp_same_cwd_resume.json
 */
const { spawn } = require("child_process");
const fs = require("fs");
const path = require("path");
const os = require("os");

const workDir = path.resolve(process.argv[2] || process.cwd());
const altWorkDir = fs.mkdtempSync(path.join(os.tmpdir(), "kiro-acp-alt-"));
const sessionsDir = path.join(os.homedir(), ".kiro", "sessions", "cli");
const reportPath = path.join(__dirname, "samples_acp_same_cwd_resume.json");
const SECRET = `POCTOKEN_${Date.now().toString(36).toUpperCase()}`;

const results = {
  version: "",
  workDir,
  altWorkDir,
  secret: SECRET,
  cases: {},
  disk: {},
  startedAt: new Date().toISOString(),
};

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

function sessionFiles(sessionId) {
  if (!fs.existsSync(sessionsDir)) return { exists: false, files: [] };
  const files = fs
    .readdirSync(sessionsDir)
    .filter((n) => n.startsWith(sessionId))
    .map((n) => {
      const p = path.join(sessionsDir, n);
      const st = fs.statSync(p);
      return { name: n, size: st.size, mtime: st.mtime.toISOString() };
    });
  return { exists: files.length > 0, files };
}

function collectText(updates, sessionId) {
  let text = "";
  for (const u of updates) {
    if (sessionId && u.sessionId && u.sessionId !== sessionId) continue;
    const up = u.update || {};
    if (up.sessionUpdate === "agent_message_chunk" && up.content?.text) {
      text += up.content.text;
    }
  }
  return text;
}

function runAcp(label) {
  const proc = spawn("kiro-cli", ["acp", "--trust-all-tools"], {
    cwd: workDir,
    stdio: ["pipe", "pipe", "pipe"],
  });
  let buf = "";
  let nextId = 1;
  const pending = new Map();
  const updates = [];
  const stderr = [];
  const serverRequests = [];

  proc.stderr.on("data", (c) => stderr.push(c.toString("utf8")));
  proc.stdout.on("data", (chunk) => {
    buf += chunk.toString("utf8");
    let idx;
    while ((idx = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, idx);
      buf = buf.slice(idx + 1);
      if (!line.trim()) continue;
      let msg;
      try {
        msg = JSON.parse(line);
      } catch {
        stderr.push(`[non-json] ${line.slice(0, 200)}`);
        continue;
      }
      if (msg.method === "session/update") {
        updates.push(msg.params || {});
        continue;
      }
      // server → client request（permission 等）：回拒絕／未實作，避免卡住
      if (msg.method && msg.id !== undefined && msg.result === undefined && msg.error === undefined) {
        serverRequests.push({ method: msg.method, id: msg.id });
        const isPerm = String(msg.method).includes("permission");
        const reply = isPerm
          ? {
              jsonrpc: "2.0",
              id: msg.id,
              result: { outcome: { outcome: "selected", optionId: "allow-once" } },
            }
          : {
              jsonrpc: "2.0",
              id: msg.id,
              error: { code: -32601, message: "not implemented" },
            };
        try {
          proc.stdin.write(JSON.stringify(reply) + "\n");
        } catch {}
        continue;
      }
      if (msg.id !== undefined && pending.has(msg.id)) {
        const p = pending.get(msg.id);
        pending.delete(msg.id);
        if (msg.error) p.reject(Object.assign(new Error(JSON.stringify(msg.error)), { rpc: msg.error }));
        else p.resolve(msg);
      }
    }
  });

  function request(method, params, timeoutMs = 90000) {
    return new Promise((resolve, reject) => {
      const id = nextId++;
      const payload = JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n";
      try {
        proc.stdin.write(payload);
      } catch (e) {
        reject(e);
        return;
      }
      const timer = setTimeout(() => {
        pending.delete(id);
        reject(new Error(`timeout ${method} (${timeoutMs}ms) [${label}]`));
      }, timeoutMs);
      pending.set(id, {
        resolve: (m) => {
          clearTimeout(timer);
          resolve(m);
        },
        reject: (e) => {
          clearTimeout(timer);
          reject(e);
        },
      });
    });
  }

  return {
    label,
    proc,
    updates,
    stderr,
    serverRequests,
    request,
    async init() {
      const init = await request("initialize", {
        protocolVersion: 1,
        clientCapabilities: {
          fs: { readTextFile: false, writeTextFile: false },
        },
        clientInfo: { name: "same-cwd-resume-poc", version: "0.1" },
      });
      proc.stdin.write(
        JSON.stringify({ jsonrpc: "2.0", method: "initialized", params: {} }) + "\n"
      );
      return init.result;
    },
    kill() {
      try {
        proc.kill();
      } catch {}
    },
  };
}

function rememberOk(text) {
  return text.includes(SECRET);
}

async function caseA_sameProcessNoLoad() {
  const name = "A_same_process_no_load";
  const acp = runAcp(name);
  try {
    const caps = await acp.init();
    const created = await acp.request("session/new", { cwd: workDir, mcpServers: [] });
    const sessionId = created.result.sessionId;
    await acp.request("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `記住這個密語：${SECRET}。只回 OK。不要解釋。`,
        },
      ],
    });
    const before = acp.updates.length;
    await acp.request("session/prompt", {
      sessionId,
      prompt: [{ type: "text", text: "密語是什麼？只回密語本身。" }],
    });
    const text = collectText(acp.updates.slice(before), sessionId);
    const ok = rememberOk(text);
    results.cases[name] = {
      pass: ok,
      sessionId,
      loadSessionCap: caps?.agentCapabilities?.loadSession,
      reply: text.slice(0, 300),
      disk: sessionFiles(sessionId),
      note: "同進程連打兩輪 prompt，不呼叫 session/load",
    };
    return { sessionId, ok };
  } catch (e) {
    results.cases[name] = { pass: false, error: e.message };
    return { ok: false };
  } finally {
    acp.kill();
  }
}

async function caseB_sameProcessLoadSameCwd() {
  const name = "B_same_process_load_same_cwd";
  const acp = runAcp(name);
  try {
    await acp.init();
    const created = await acp.request("session/new", { cwd: workDir, mcpServers: [] });
    const sessionId = created.result.sessionId;
    await acp.request("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `記住這個密語：${SECRET}。只回 OK。不要解釋。`,
        },
      ],
    });
    const loadStarted = Date.now();
    const loaded = await acp.request(
      "session/load",
      { sessionId, cwd: workDir, mcpServers: [] },
      45000
    );
    const loadMs = Date.now() - loadStarted;
    const before = acp.updates.length;
    await acp.request("session/prompt", {
      sessionId,
      prompt: [{ type: "text", text: "密語是什麼？只回密語本身。" }],
    });
    const text = collectText(acp.updates.slice(before), sessionId);
    const ok = rememberOk(text);
    results.cases[name] = {
      pass: ok,
      sessionId,
      loadMs,
      loadResultKeys: Object.keys(loaded.result || {}),
      reply: text.slice(0, 300),
      disk: sessionFiles(sessionId),
      note: "同進程 session/load 同 cwd 後再問密語",
    };
    return { ok };
  } catch (e) {
    results.cases[name] = { pass: false, error: e.message };
    return { ok: false };
  } finally {
    acp.kill();
  }
}

async function caseC_crossProcessSameCwd() {
  const name = "C_cross_process_same_cwd";
  let sessionId = "";
  const a = runAcp(name + "_A");
  try {
    await a.init();
    const created = await a.request("session/new", { cwd: workDir, mcpServers: [] });
    sessionId = created.result.sessionId;
    await a.request("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `記住這個密語：${SECRET}。只回 OK。不要解釋。`,
        },
      ],
    });
    results.disk.afterProcessA = sessionFiles(sessionId);
  } catch (e) {
    results.cases[name] = { pass: false, phase: "process_A", error: e.message, sessionId };
    a.kill();
    return { ok: false };
  }
  a.kill();
  await sleep(1500);
  results.disk.afterKillA = sessionFiles(sessionId);

  // 若殘留 .lock，記錄下來（可能是 hang 根因）
  const lockPath = path.join(sessionsDir, `${sessionId}.lock`);
  results.disk.lockAfterKillA = {
    path: lockPath,
    exists: fs.existsSync(lockPath),
  };

  const b = runAcp(name + "_B");
  try {
    await b.init();
    const loadStarted = Date.now();
    const loaded = await b.request(
      "session/load",
      { sessionId, cwd: workDir, mcpServers: [] },
      45000
    );
    const loadMs = Date.now() - loadStarted;
    const before = b.updates.length;
    await b.request("session/prompt", {
      sessionId,
      prompt: [{ type: "text", text: "密語是什麼？只回密語本身。" }],
    });
    const text = collectText(b.updates.slice(before), sessionId);
    const ok = rememberOk(text);
    results.cases[name] = {
      pass: ok,
      sessionId,
      loadMs,
      loadResultKeys: Object.keys(loaded.result || {}),
      reply: text.slice(0, 300),
      disk: sessionFiles(sessionId),
      note: "跨進程、同 cwd session/load —— 官方宣稱的核心情境",
    };
    return { ok };
  } catch (e) {
    results.cases[name] = {
      pass: false,
      sessionId,
      error: e.message,
      disk: sessionFiles(sessionId),
      lockStillThere: fs.existsSync(lockPath),
      stderrTail: b.stderr.join("").slice(-800),
      note: "跨進程同 cwd load 失敗",
    };
    return { ok: false };
  } finally {
    b.kill();
  }
}

async function caseD_crossProcessDifferentCwd() {
  const name = "D_cross_process_different_cwd";
  let sessionId = "";
  const a = runAcp(name + "_A");
  try {
    await a.init();
    const created = await a.request("session/new", { cwd: workDir, mcpServers: [] });
    sessionId = created.result.sessionId;
    await a.request("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `記住這個密語：${SECRET}。只回 OK。不要解釋。`,
        },
      ],
    });
  } catch (e) {
    results.cases[name] = { pass: false, phase: "process_A", error: e.message };
    a.kill();
    return { ok: false };
  }
  a.kill();
  await sleep(1500);

  const b = runAcp(name + "_B");
  try {
    await b.init();
    const loadStarted = Date.now();
    // 故意用不同 cwd
    const loaded = await b.request(
      "session/load",
      { sessionId, cwd: altWorkDir, mcpServers: [] },
      45000
    );
    const loadMs = Date.now() - loadStarted;
    const before = b.updates.length;
    await b.request("session/prompt", {
      sessionId,
      prompt: [{ type: "text", text: "密語是什麼？只回密語本身。" }],
    });
    const text = collectText(b.updates.slice(before), sessionId);
    const remembered = rememberOk(text);
    // 此 case 的「通過」定義：load 有回應（不 hang）。記憶是否保留另記。
    results.cases[name] = {
      pass: true,
      remembered,
      sessionId,
      loadMs,
      loadResultKeys: Object.keys(loaded.result || {}),
      originalCwd: workDir,
      loadCwd: altWorkDir,
      reply: text.slice(0, 300),
      note: "跨進程、不同 cwd；pass=load 未 hang；remembered=是否仍記得密語",
    };
    return { ok: true, remembered };
  } catch (e) {
    results.cases[name] = {
      pass: false,
      sessionId,
      error: e.message,
      originalCwd: workDir,
      loadCwd: altWorkDir,
      note: "不同 cwd 的 session/load 失敗／timeout",
    };
    return { ok: false };
  } finally {
    b.kill();
  }
}

(async () => {
  try {
    const ver = spawn("kiro-cli", ["--version"], { stdio: ["ignore", "pipe", "pipe"] });
    let v = "";
    ver.stdout.on("data", (c) => (v += c.toString()));
    ver.stderr.on("data", (c) => (v += c.toString()));
    await new Promise((r) => ver.on("close", r));
    results.version = v.trim();
  } catch {
    results.version = "unknown";
  }

  console.log("=== ACP same-cwd resume PoC ===");
  console.log("version :", results.version);
  console.log("workDir :", workDir);
  console.log("altDir  :", altWorkDir);
  console.log("secret  :", SECRET);
  console.log("");

  console.log("[A] same process, no load...");
  await caseA_sameProcessNoLoad();
  console.log("  →", results.cases.A_same_process_no_load);

  console.log("[B] same process, load same cwd...");
  await caseB_sameProcessLoadSameCwd();
  console.log("  →", results.cases.B_same_process_load_same_cwd);

  console.log("[C] cross process, same cwd (官方核心宣稱)...");
  await caseC_crossProcessSameCwd();
  console.log("  →", results.cases.C_cross_process_same_cwd);

  console.log("[D] cross process, different cwd...");
  await caseD_crossProcessDifferentCwd();
  console.log("  →", results.cases.D_cross_process_different_cwd);

  results.finishedAt = new Date().toISOString();
  results.verdict = {
    A_baseline_memory: !!results.cases.A_same_process_no_load?.pass,
    B_same_proc_load: !!results.cases.B_same_process_load_same_cwd?.pass,
    C_cross_proc_same_cwd: !!results.cases.C_cross_process_same_cwd?.pass,
    D_cross_proc_diff_cwd_load_ok: !!results.cases.D_cross_process_different_cwd?.pass,
    D_remembered: !!results.cases.D_cross_process_different_cwd?.remembered,
  };

  fs.writeFileSync(reportPath, JSON.stringify(results, null, 2), "utf8");
  console.log("\n=== VERDICT ===");
  console.log(JSON.stringify(results.verdict, null, 2));
  console.log("report:", reportPath);

  try {
    fs.rmSync(altWorkDir, { recursive: true, force: true });
  } catch {}

  const criticalFail =
    !results.verdict.A_baseline_memory || !results.verdict.C_cross_proc_same_cwd;
  process.exitCode = criticalFail ? 1 : 0;
})().catch((e) => {
  console.error(e);
  results.fatal = e.message;
  try {
    fs.writeFileSync(reportPath, JSON.stringify(results, null, 2), "utf8");
  } catch {}
  process.exit(1);
});
