"""本機 fixture 更新來源：模擬 GitHub /releases/latest、資產下載與 checksums.txt。

只綁 127.0.0.1 的隨機高位埠，於驅動程式的執行緒中執行（不另起程序）。
可透過 set_fault() 注入故障：
  api: ok | 404 | ratelimit | 500 | drop（直接斷線，不回應）
  ck:  ok | missing | mismatch | badformat
  dl:  ok | abort（送一半後斷線）| drop（直接斷線）
"""
from __future__ import annotations

import hashlib
import json
import pathlib
import socket
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse

REPO_PATH = "/repos/jerry12122/Claude-Code-Mini-App"


def _sha256(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


class FixtureServer:
    def __init__(self, release_exe: pathlib.Path, asset_name: str, tag: str):
        self.release_exe = release_exe
        self.asset_name = asset_name
        self.tag = tag
        self.faults = {"api": "ok", "ck": "ok", "dl": "ok"}
        self.requests: list[str] = []
        self._lock = threading.Lock()
        self._httpd = None
        self._thread = None
        self.port = self._free_port()
        self.base = f"http://127.0.0.1:{self.port}"

    @staticmethod
    def _free_port() -> int:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.bind(("127.0.0.1", 0))
        port = s.getsockname()[1]
        s.close()
        return port

    def set_fault(self, **kw: str) -> None:
        with self._lock:
            self.faults.update(kw)

    def start(self) -> None:
        fx = self

        class Handler(BaseHTTPRequestHandler):
            # HTTP/1.0：每次請求後關閉連線。stop() 後新請求必定被拒（不會沿用 keep-alive 舊連線）。
            protocol_version = "HTTP/1.0"

            def log_message(self, *_args):  # 靜音
                pass

            def _drop(self):
                self.close_connection = True
                try:
                    self.connection.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass

            def _json(self, code: int, obj, extra=None):
                body = json.dumps(obj).encode()
                self.send_response(code)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                for k, v in (extra or {}).items():
                    self.send_header(k, v)
                self.end_headers()
                self.wfile.write(body)

            def do_GET(self):  # noqa: N802
                path = urlparse(self.path).path
                with fx._lock:
                    faults = dict(fx.faults)
                    fx.requests.append(path)
                if path == "/__health":
                    return self._json(200, {"ok": True})
                if path == f"{REPO_PATH}/releases/latest":
                    return self._release(faults)
                if path.startswith("/dl/"):
                    return self._download(path[len("/dl/"):], faults)
                self._json(404, {"message": "Not Found"})

            def _release(self, faults):
                api = faults["api"]
                if api == "drop":
                    return self._drop()
                if api == "404":
                    return self._json(404, {"message": "Not Found"})
                if api == "ratelimit":
                    return self._json(403, {"message": "API rate limit exceeded"},
                                      {"X-RateLimit-Remaining": "0"})
                if api == "500":
                    return self._json(500, {"message": "server error"})
                exe = fx.release_exe
                assets = [{
                    "id": 1, "name": fx.asset_name, "content_type": "application/octet-stream",
                    "size": exe.stat().st_size,
                    "browser_download_url": f"{fx.base}/dl/{fx.asset_name}",
                }]
                if faults["ck"] != "missing":
                    assets.append({
                        "id": 2, "name": "checksums.txt", "content_type": "text/plain",
                        "size": 100, "browser_download_url": f"{fx.base}/dl/checksums.txt",
                    })
                self._json(200, {
                    "tag_name": fx.tag, "name": fx.tag, "body": "e2e fixture",
                    "draft": False, "prerelease": False,
                    "published_at": "2026-10-16T00:00:00Z",
                    "html_url": f"{fx.base}/releases/tag/{fx.tag}",
                    "assets": assets,
                })

            def _download(self, name, faults):
                if name == "checksums.txt":
                    ck = faults["ck"]
                    if ck == "mismatch":
                        body = f"{'0' * 64}  {fx.asset_name}\n"
                    elif ck == "badformat":
                        body = "this is not a checksum line\n"
                    else:
                        body = f"{_sha256(fx.release_exe)}  {fx.asset_name}\n"
                    data = body.encode()
                elif name == fx.asset_name:
                    if faults["dl"] == "drop":
                        return self._drop()
                    data = fx.release_exe.read_bytes()
                else:
                    return self._json(404, {"message": "Not Found"})
                self.send_response(200)
                self.send_header("Content-Type", "application/octet-stream")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                if name == fx.asset_name and faults["dl"] == "abort":
                    self.wfile.write(data[: len(data) // 2])
                    self.wfile.flush()
                    return self._drop()
                self.wfile.write(data)

        class QuietServer(ThreadingHTTPServer):
            def handle_error(self, request, client_address):  # 客戶端中途斷線是預期故障，不印 traceback
                pass

        self._httpd = QuietServer(("127.0.0.1", self.port), Handler)
        self._httpd.daemon_threads = True
        self._thread = threading.Thread(target=self._httpd.serve_forever, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        if self._httpd is not None:
            self._httpd.shutdown()
            self._httpd.server_close()
            self._httpd = None
