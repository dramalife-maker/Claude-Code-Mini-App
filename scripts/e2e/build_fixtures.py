#!/usr/bin/env python3
"""建立 e2e 用建置物。

- 從 repo 工作目錄複製（git ls-files，遵守 .gitignore；排除 uploads/logs/workspace/db）到暫存目錄。
- 在「暫存副本」內把版本改為 1.0.0 / 1.0.1、在 index.html 加上前端標記，並 git init 一次
  （只為讓 Go 寫入 vcs.revision；不會對 repo 做任何寫入）。
- 建置 e2e 版桌面（-tags "production e2e"）、正式桌面與 server 到暫存目錄。
- 輸出 manifest.json（各檔 SHA256、版本、revision）。

用法：python scripts/e2e/build_fixtures.py [--work DIR]
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import stat
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
REPO = HERE.parent.parent
VERSIONS = ["1.0.0", "1.0.1"]
EXCLUDE_PREFIXES = (".git/", "internal/static/uploads/", "logs/", "workspace/")
DESKTOP_ASSET = "claude-miniapp-desktop-v{ver}-windows-amd64.exe"


def run(cmd: list[str], cwd: pathlib.Path, env: dict | None = None) -> str:
    res = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True)
    if res.returncode != 0:
        sys.stderr.write(f"命令失敗：{' '.join(cmd)}\n{res.stdout}\n{res.stderr}\n")
        raise SystemExit(1)
    return res.stdout


def sha256(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def source_files() -> list[str]:
    out = subprocess.run(
        ["git", "-C", str(REPO), "ls-files", "-co", "--exclude-standard", "-z"],
        capture_output=True, check=True,
    ).stdout.decode("utf-8")
    files = []
    for rel in out.split("\0"):
        if not rel or rel.startswith(EXCLUDE_PREFIXES):
            continue
        if (REPO / rel).is_file():
            files.append(rel)
    return files


def _clear_readonly_and_retry(func, path, _exc) -> None:
    # Windows 上 .git/objects 為唯讀，需先清除唯讀屬性才能刪除。
    os.chmod(path, stat.S_IWRITE)
    func(path)


def make_source(ver: str, work: pathlib.Path, files: list[str]) -> pathlib.Path:
    dst = work / "src" / ver
    if dst.exists():
        shutil.rmtree(dst, onerror=_clear_readonly_and_retry)
    for rel in files:
        target = dst / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(REPO / rel, target)

    version_go = dst / "internal" / "version" / "version.go"
    text = version_go.read_text(encoding="utf-8")
    text, n = re.subn(r'const Version = "[^"]*"', f'const Version = "{ver}"', text)
    assert n == 1, "version.go 找不到 const Version"
    version_go.write_text(text, encoding="utf-8", newline="\n")

    index = dst / "internal" / "static" / "index.html"
    html = index.read_text(encoding="utf-8")
    assert html.count("</head>") == 1, "index.html 需要恰好一個 </head>"
    html = html.replace("</head>", f"<!-- cma-e2e-frontend:{ver} -->\n</head>")
    index.write_text(html, encoding="utf-8", newline="\n")

    git_env = dict(os.environ, GIT_AUTHOR_NAME="cma-e2e", GIT_AUTHOR_EMAIL="e2e@localhost",
                   GIT_COMMITTER_NAME="cma-e2e", GIT_COMMITTER_EMAIL="e2e@localhost")
    run(["git", "init", "-q"], dst)
    run(["git", "add", "-A"], dst)
    run(["git", "commit", "-q", "-m", f"e2e fixture {ver}"], dst, git_env)
    return dst


def go_build(src: pathlib.Path, out: pathlib.Path, tags: str | None, ldflags: str | None, pkg: str) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    cmd = ["go", "build"]
    if tags:
        cmd += ["-tags", tags]
    if ldflags:
        cmd += ["-ldflags", ldflags]
    cmd += ["-o", str(out), pkg]
    run(cmd, src)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--work", default=os.path.join(tempfile.gettempdir(), "cma-e2e"))
    args = ap.parse_args()
    work = pathlib.Path(args.work).resolve()
    dist = work / "dist"
    dist.mkdir(parents=True, exist_ok=True)

    files = source_files()
    print(f"[build] 來源檔案數：{len(files)}（來自 repo 工作目錄，已排除 uploads/logs/db）")
    manifest: dict = {"versions": {}, "production": {}}

    for ver in VERSIONS:
        src = make_source(ver, work, files)
        rev = run(["git", "rev-parse", "HEAD"], src).strip()
        exe = dist / ver / "claude-miniapp-desktop.exe"
        go_build(src, exe, "production e2e", "-H windowsgui", "./cmd/desktop")
        entry = {"revision": rev, "desktop_exe": str(exe), "desktop_sha256": sha256(exe),
                 "asset_name": DESKTOP_ASSET.format(ver=ver)}
        if ver == VERSIONS[0]:
            server = dist / "server" / "claude-miniapp-server.exe"
            go_build(src, server, "production", None, "./cmd/server")
            entry["server_exe"] = str(server)
            entry["server_sha256"] = sha256(server)
        manifest["versions"][ver] = entry
        print(f"[build] {ver} e2e 桌面 sha256={entry['desktop_sha256'][:16]} rev={rev[:12]}")

    # 正式建置（自 repo 工作目錄，輸出到暫存；不寫入 repo）
    prod_desktop = dist / "production" / "claude-miniapp-desktop.exe"
    go_build(REPO, prod_desktop, "production", "-H windowsgui", "./cmd/desktop")
    prod_server = dist / "production" / "claude-miniapp-server.exe"
    go_build(REPO, prod_server, "production", None, "./cmd/server")
    manifest["production"] = {
        "desktop_exe": str(prod_desktop), "desktop_sha256": sha256(prod_desktop),
        "server_exe": str(prod_server), "server_sha256": sha256(prod_server),
    }

    # 守門：正式 EXE 不得含 e2e 環境變數名稱；e2e EXE 必須含（正向對照，確認 hook 確實編入）。
    marker = b"CMA_E2E_"
    prod_hits = prod_desktop.read_bytes().count(marker)
    e2e_hits = pathlib.Path(manifest["versions"]["1.0.0"]["desktop_exe"]).read_bytes().count(marker)
    manifest["guard"] = {"production_desktop_CMA_E2E_hits": prod_hits, "e2e_desktop_CMA_E2E_hits": e2e_hits}
    if prod_hits != 0 or e2e_hits == 0:
        sys.stderr.write(f"守門失敗：production={prod_hits} e2e={e2e_hits}\n")
        return 3
    print(f"[guard] production 內 CMA_E2E_ 命中 {prod_hits}；e2e 內 {e2e_hits}")
    print(f"[build] 正式建置完成：desktop={prod_desktop.name} server={prod_server.name}")

    (dist / "manifest.json").write_text(json.dumps(manifest, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"[build] manifest：{dist / 'manifest.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
