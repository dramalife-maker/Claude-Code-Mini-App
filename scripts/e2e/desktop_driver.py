"""e2e 驅動共用函式：程序（只以 PID 操作）、port、REST、WebView2 CDP、SQLite。

安全規則（由 run_e2e.py 遵守）：
- 只結束自己啟動、且執行檔路徑吻合的 PID；不以名稱結束程序。
- 不碰 port 8080 與使用者的實際 config／DB／workspace；所有路徑都在暫存目錄。
"""
from __future__ import annotations

import json
import os
import pathlib
import socket
import sqlite3
import subprocess
import time
import urllib.error
import urllib.request

DETACHED_PROCESS = 0x00000008
CREATE_NEW_PROCESS_GROUP = 0x00000200
CREATE_NO_WINDOW = 0x08000000


def ps(cmd: str) -> str:
    res = subprocess.run(["powershell", "-NoProfile", "-NonInteractive", "-Command", cmd],
                         capture_output=True, text=True)
    return res.stdout.strip()


def exe_path_of(pid: int) -> str:
    return ps(f"(Get-Process -Id {pid} -ErrorAction SilentlyContinue).Path")


def pids_with_path(path: str) -> list[int]:
    out = ps(f"Get-Process -ErrorAction SilentlyContinue | Where-Object {{ $_.Path -eq '{path}' }} "
             f"| ForEach-Object {{ $_.Id }}")
    return [int(x) for x in out.split() if x.isdigit()]


def listening_pids(port: int) -> set[int]:
    out = subprocess.run(["netstat", "-ano", "-p", "tcp"], capture_output=True, text=True).stdout
    pids: set[int] = set()
    for line in out.splitlines():
        parts = line.split()
        if len(parts) >= 5 and parts[3] == "LISTENING" and parts[1].rsplit(":", 1)[-1] == str(port):
            pids.add(int(parts[4]))
    return pids


def port_free(port: int) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.bind(("127.0.0.1", port))
        return True
    except OSError:
        return False
    finally:
        s.close()


def pick_port(start: int, end: int, used: set[int]) -> int:
    for p in range(start, end):
        if p not in used and port_free(p):
            used.add(p)
            return p
    raise RuntimeError("找不到可用的測試 port")


def wait_until(pred, timeout: float, interval: float = 0.5, what: str = "條件"):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        try:
            last = pred()
            if last:
                return last
        except Exception as exc:  # noqa: BLE001 - 等待期間的暫時性錯誤
            last = exc
        time.sleep(interval)
    raise TimeoutError(f"逾時等待 {what}（{timeout}s）；最後結果：{last!r}")


def start_detached(exe: pathlib.Path, cwd: pathlib.Path, env_extra: dict, log_path: pathlib.Path) -> subprocess.Popen:
    env = dict(os.environ)
    env.update(env_extra)
    log = open(log_path, "ab")
    return subprocess.Popen(
        [str(exe)], cwd=str(cwd), env=env, stdin=subprocess.DEVNULL,
        stdout=log, stderr=subprocess.STDOUT, close_fds=True,
        creationflags=DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW,
    )


def kill_pid_if_path(pid: int, expected_path: str) -> str:
    """只結束 pid 本身，且僅在路徑吻合時執行；回傳結果描述。"""
    path = exe_path_of(pid)
    if not path:
        return f"pid {pid} 已不存在"
    if os.path.normcase(path) != os.path.normcase(expected_path):
        return f"pid {pid} 路徑不符（{path}），拒絕結束"
    res = subprocess.run(["taskkill", "/PID", str(pid), "/F"], capture_output=True, text=True)
    return f"taskkill /PID {pid} exit={res.returncode}"


# ---------- HTTP ----------

def http(method: str, url: str, body=None, headers=None, raw: bool = False, timeout: float = 15):
    data = None
    hdrs = dict(headers or {})
    if body is not None and not isinstance(body, (bytes, bytearray)):
        data = json.dumps(body).encode()
        hdrs.setdefault("Content-Type", "application/json")
    elif body is not None:
        data = bytes(body)
    req = urllib.request.Request(url, data=data, method=method, headers=hdrs)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            payload = resp.read()
            code = resp.status
    except urllib.error.HTTPError as e:
        payload = e.read()
        code = e.code
    if raw:
        return code, payload
    try:
        return code, json.loads(payload or b"null")
    except json.JSONDecodeError:
        return code, payload.decode("utf-8", "replace")


def login(port: int, password: str) -> str:
    code, body = http("POST", f"http://127.0.0.1:{port}/auth/login", {"password": password})
    if code != 200 or not isinstance(body, dict) or not body.get("token"):
        raise RuntimeError(f"登入失敗 code={code} body={body}")
    return body["token"]


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def multipart_upload(port: int, token: str, session_id: str, filename: str, content: bytes) -> tuple[int, object]:
    boundary = "----cmae2e7f3a"
    head = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"{filename}\"\r\n"
            f"Content-Type: image/png\r\n\r\n").encode()
    tail = f"\r\n--{boundary}--\r\n".encode()
    body = head + content + tail
    hdrs = dict(auth(token))
    hdrs["Content-Type"] = f"multipart/form-data; boundary={boundary}"
    return http("POST", f"http://127.0.0.1:{port}/sessions/{session_id}/uploads", body, hdrs)


# ---------- SQLite（唯讀檢查；寫入只用於注入 queued 訊息，且在測試 DB 上）----------

def db_connect(db_path: pathlib.Path) -> sqlite3.Connection:
    con = sqlite3.connect(str(db_path), timeout=10)
    return con


# ---------- WebView2 CDP ----------

class CDP:
    def __init__(self, ws_url: str):
        import websocket  # websocket-client（前置需求，見 README）

        self.ws = websocket.create_connection(ws_url, timeout=30, suppress_origin=True)
        self._id = 0

    def call(self, method: str, params: dict | None = None) -> dict:
        self._id += 1
        mid = self._id
        self.ws.send(json.dumps({"id": mid, "method": method, "params": params or {}}))
        while True:
            msg = json.loads(self.ws.recv())
            if msg.get("id") == mid:
                return msg

    def eval(self, expr: str):
        res = self.call("Runtime.evaluate", {"expression": expr, "awaitPromise": True, "returnByValue": True})
        if "exceptionDetails" in res.get("result", {}):
            raise RuntimeError(f"JS 例外：{res['result']['exceptionDetails']}")
        return res.get("result", {}).get("result", {}).get("value")

    def close(self) -> None:
        try:
            self.ws.close()
        except Exception:  # noqa: BLE001
            pass


def cdp_page_ws(cdp_port: int, url_prefix: str, timeout: float) -> str:
    def find():
        code, targets = http("GET", f"http://127.0.0.1:{cdp_port}/json/list", timeout=5)
        if code != 200 or not isinstance(targets, list):
            return None
        for t in targets:
            if t.get("type") == "page" and str(t.get("url", "")).startswith(url_prefix):
                return t.get("webSocketDebuggerUrl")
        return None

    return wait_until(find, timeout, 1.0, what="WebView2 頁面")


CLICK_JS = """(function (label) {
  var b = Array.from(document.querySelectorAll('button')).find(function (x) {
    return x.innerText.trim() === label && !x.disabled;
  });
  if (!b) return false;
  b.click();
  return true;
})(%s)"""

BUTTON_EXISTS_JS = """(function (label) {
  return Array.from(document.querySelectorAll('button')).some(function (x) {
    return x.innerText.trim() === label && !x.disabled;
  });
})(%s)"""

INSTALL_LISTENER_JS = """(function () {
  if (window.__e2eStates) return true;
  window.__e2eStates = [];
  window.addEventListener('desktop-update-state', function (e) { window.__e2eStates.push(e.detail); });
  return true;
})()"""


def js_str(s: str) -> str:
    return json.dumps(s, ensure_ascii=False)
