/**
 * ACP 互動授權 POC：不帶 --trust-all-tools，觀察 kiro-cli acp 是否發出
 * session/request_permission（server→client 請求），記錄其 schema 並回覆放行，
 * 驗證「互動式授權」對 kiroacp runner 是否可行。
 *
 * Usage: node poc/kiro-cli/acp_permission_probe.js
 * Exit 0 = 收到 request_permission 且成功回覆放行後完成；非 0 = 未觀察到或失敗。
 */
const { spawn } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "acp-perm-"));
const reportPath = path.join(__dirname, "samples_acp_permission.json");
const PROMPT =
  "請用工具在目前工作目錄建立一個名為 hello.txt 的檔案，內容為 HELLO。完成後回覆 done。";

const proc = spawn("kiro-cli", ["acp"], { cwd: workDir, stdio: ["pipe", "pipe", "pipe"] });

let buf = "";
let nextId = 1;
const pending = new Map();
const stderrChunks = [];
const permissionRequests = [];
let sessionId = "";

function send(method, params, isNotification = false) {
  const msg = { jsonrpc: "2.0", method, params };
  if (!isNotification) msg.id = nextId++;
  proc.stdin.write(JSON.stringify(msg) + "\n");
  return msg.id;
}
function request(method, params, timeoutMs = 180000) {
  return new Promise((resolve, reject) => {
    const id = send(method, params);
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`timeout ${method}`)); }, timeoutMs);
    pending.set(id, { resolve: (m) => { clearTimeout(timer); resolve(m); }, reject: (e) => { clearTimeout(timer); reject(e); } });
  });
}

function handleLine(line) {
  if (!line.trim()) return;
  let msg;
  try { msg = JSON.parse(line); } catch { stderrChunks.push("[non-json] " + line.slice(0, 200)); return; }

  // server→client 請求
  if (msg.method && msg.id !== undefined && !msg.result && !msg.error) {
    if (/permission/i.test(msg.method)) {
      console.log(`\n>>> 收到授權請求: ${msg.method}`);
      console.log(JSON.stringify(msg.params, null, 2));
      permissionRequests.push({ method: msg.method, params: msg.params });
      // 依 ACP schema 選一個「允許」選項回覆
      const opts = msg.params?.options || [];
      const allow = opts.find((o) => /allow/i.test(o.kind || o.optionId || o.name || "")) || opts[0];
      const outcome = allow
        ? { outcome: { outcome: "selected", optionId: allow.optionId } }
        : { outcome: { outcome: "cancelled" } };
      console.log(`<<< 回覆放行 optionId=${allow?.optionId}`);
      proc.stdin.write(JSON.stringify({ jsonrpc: "2.0", id: msg.id, result: outcome }) + "\n");
    } else {
      // 其他 client method（如 fs）：最小拒絕
      proc.stdin.write(JSON.stringify({ jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: "not implemented: " + msg.method } }) + "\n");
    }
    return;
  }
  if (msg.method === "session/update") return; // 本 POC 不關心文字
  if (msg.id !== undefined && pending.has(msg.id)) {
    const p = pending.get(msg.id); pending.delete(msg.id);
    if (msg.error) p.reject(new Error(JSON.stringify(msg.error))); else p.resolve(msg);
  }
}

proc.stdout.on("data", (chunk) => {
  buf += chunk.toString("utf8");
  let idx;
  while ((idx = buf.indexOf("\n")) >= 0) { const l = buf.slice(0, idx); buf = buf.slice(idx + 1); handleLine(l); }
});
proc.stderr.on("data", (d) => stderrChunks.push(d.toString()));

(async () => {
  const report = { workDir, sawPermissionRequest: false, permissionRequests: [], fileCreated: false, error: null };
  try {
    await request("initialize", { protocolVersion: 1, clientCapabilities: { fs: { readTextFile: false, writeTextFile: false } }, clientInfo: { name: "acp-perm-probe", version: "0.1.0" } });
    send("initialized", {}, true);
    const sess = await request("session/new", { cwd: workDir, mcpServers: [] });
    sessionId = sess.result?.sessionId || "";
    await request("session/prompt", { sessionId, prompt: [{ type: "text", text: PROMPT }] }, 180000);
    report.sawPermissionRequest = permissionRequests.length > 0;
    report.permissionRequests = permissionRequests;
    report.fileCreated = fs.existsSync(path.join(workDir, "hello.txt"));
  } catch (e) {
    report.error = String(e.message || e);
  } finally {
    report.stderrTail = stderrChunks.join("").slice(-800);
    fs.writeFileSync(reportPath, JSON.stringify(report, null, 2), "utf8");
    console.log("\n=== RESULT ===");
    console.log("sawPermissionRequest:", report.sawPermissionRequest);
    console.log("fileCreated:", report.fileCreated);
    console.log("report:", reportPath);
    try { proc.kill(); } catch {}
    try { fs.rmSync(workDir, { recursive: true, force: true }); } catch {}
    process.exit(report.sawPermissionRequest ? 0 : 1);
  }
})();
