// POC: agy 是否支援 ACP (Agent Client Protocol / JSON-RPC over stdio)
// 對齊 poc/kiro-cli/acp_probe.js 的 initialize 請求。
// 用法: node probe_acp.js
const { spawn } = require("child_process");

const AGY = process.env.CC_AGY_BIN || "agy";
const proc = spawn(AGY, ["acp"], { stdio: ["pipe", "pipe", "pipe"] });

let buf = "";
let gotResponse = false;

proc.stdout.on("data", (d) => {
  buf += d.toString();
  console.log("[stdout]", JSON.stringify(d.toString()));
  // 嘗試找出 JSON-RPC 回應
  for (const line of buf.split("\n")) {
    const t = line.trim();
    if (!t) continue;
    try {
      const msg = JSON.parse(t);
      if (msg.jsonrpc === "2.0" && (msg.result || msg.error || msg.id !== undefined)) {
        console.log("=> ACP JSON-RPC 回應偵測到:", JSON.stringify(msg));
        gotResponse = true;
      }
    } catch { /* 非 JSON,忽略 */ }
  }
});
proc.stderr.on("data", (d) => console.error("[stderr]", d.toString()));
proc.on("error", (e) => console.error("[spawn error]", e.message));
proc.on("exit", (code, sig) => console.log(`[exit] code=${code} sig=${sig}`));

// 送 ACP initialize
const init = {
  jsonrpc: "2.0",
  id: 0,
  method: "initialize",
  params: {
    protocolVersion: 1,
    clientCapabilities: { fs: { readTextFile: true, writeTextFile: true }, terminal: true },
    clientInfo: { name: "agy-acp-probe", version: "0.0.1" },
  },
};
proc.stdin.write(JSON.stringify(init) + "\n");

// 8 秒後裁決
setTimeout(() => {
  console.log(gotResponse
    ? "\nRESULT: PASS — agy acp 回應了 ACP initialize (支援 ACP)"
    : "\nRESULT: FAIL — 8s 內無 ACP JSON-RPC 回應 (不支援或格式不符)");
  try { proc.kill(); } catch {}
  process.exit(gotResponse ? 0 : 1);
}, 8000);
