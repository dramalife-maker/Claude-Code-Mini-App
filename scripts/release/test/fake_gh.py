#!/usr/bin/env python3
"""假的 gh，只供 scripts/release/test/test-release.sh 使用。

- 所有狀態存在 $FAKE_GH_STATE（暫存目錄）；絕不連線到 GitHub。
- 每個呼叫都寫入 $FAKE_GH_STATE/calls.log，寫入以 WRITE 開頭、讀取以 READ 開頭。
- 白名單外的子命令一律以 99 結束，讓測試能發現腳本用了不預期的 gh 指令。
- 故障注入：FAKE_GH_FAIL=create,upload,publish,delete（逗號分隔）；
  FAKE_GH_CORRUPT=<資產名稱,...> 讓下載內容被竄改一個位元組。
- 子命令 seed <none|draft|published> [舊資產名...] 由測試直接建立初始狀態。
"""
import json
import os
import shutil
import sys

STATE = os.environ.get("FAKE_GH_STATE", "")
FAIL = set(x for x in os.environ.get("FAKE_GH_FAIL", "").split(",") if x)
CORRUPT = set(x for x in os.environ.get("FAKE_GH_CORRUPT", "").split(",") if x)
DB = os.path.join(STATE, "state.json")
LOG = os.path.join(STATE, "calls.log")
ASSETS = os.path.join(STATE, "assets")


def die(code, msg):
    print("fake gh: " + msg, file=sys.stderr)
    sys.exit(code)


def load():
    if os.path.exists(DB):
        with open(DB, encoding="utf-8") as f:
            return json.load(f)
    return {"releases": [], "next_id": 1000}


def save(db):
    os.makedirs(STATE, exist_ok=True)
    with open(DB, "w", encoding="utf-8") as f:
        json.dump(db, f, ensure_ascii=False)


def log(kind, args):
    os.makedirs(STATE, exist_ok=True)
    with open(LOG, "a", encoding="utf-8") as f:
        f.write(kind + " " + " ".join(args) + "\n")


def new_id(db):
    db["next_id"] += 1
    return db["next_id"]


def asset_path(aid):
    return os.path.join(ASSETS, str(aid))


def release_of(db, tag):
    for r in db["releases"]:
        if r["tag_name"] == tag:
            return r
    return None


def add_asset(db, rel, name, content):
    os.makedirs(ASSETS, exist_ok=True)
    rel["assets"] = [a for a in rel["assets"] if a["name"] != name]
    aid = new_id(db)
    with open(asset_path(aid), "wb") as f:
        f.write(content)
    rel["assets"].append({"name": name, "id": aid, "size": len(content)})


def flag_value(args, flag, default=""):
    for i, a in enumerate(args):
        if a == flag and i + 1 < len(args):
            return args[i + 1]
        if a.startswith(flag + "="):
            return a.split("=", 1)[1]
    return default


def positional(args, skip_with_value):
    out, skip = [], False
    for a in args:
        if skip:
            skip = False
            continue
        if a in skip_with_value:
            skip = True
            continue
        if a.startswith("-"):
            continue
        out.append(a)
    return out


def cmd_seed(argv):
    status, stale = argv[0], argv[1:]
    db = {"releases": [], "next_id": 1000}
    if status != "none":
        rel = {"id": new_id(db), "tag_name": os.environ["GITHUB_REF_NAME"], "draft": status == "draft",
               "latest": False, "notes": "seeded", "assets": []}
        for name in stale:
            add_asset(db, rel, name, b"stale-" + name.encode())
        db["releases"].append(rel)
    save(db)
    print("seeded", status)


def cmd_api(args):
    path = positional(args, {"-H", "--jq"})[0]
    accept = flag_value(args, "-H")
    db = load()
    if path.startswith("repos/") and "/releases/assets/" in path:
        aid = int(path.rsplit("/", 1)[1])
        log("READ", ["api-asset", str(aid)])
        name = None
        for r in db["releases"]:
            for a in r["assets"]:
                if a["id"] == aid:
                    name = a["name"]
        with open(asset_path(aid), "rb") as f:
            data = f.read()
        if name in CORRUPT and data:
            data = bytes([data[0] ^ 0xFF]) + data[1:]
        sys.stdout.buffer.write(data)
        return
    if path.startswith("repos/") and "/releases?" in path:
        log("READ", ["api-releases"])
        sys.stdout.write(json.dumps(db["releases"]))
        return
    die(99, "unexpected gh api call: " + " ".join(args))


def cmd_release(args):
    sub, rest = args[0], args[1:]
    tag = positional(rest, {"-R", "--repo", "--title", "--notes-file"})[0]
    db = load()
    if sub == "create":
        log("WRITE", ["create", tag, "draft" if "--draft" in rest else "public"])
        if "create" in FAIL:
            die(1, "injected failure: create")
        if release_of(db, tag):
            die(1, "release already exists")
        with open(flag_value(rest, "--notes-file"), encoding="utf-8") as f:
            notes = f.read()
        db["releases"].append({"id": new_id(db), "tag_name": tag, "draft": True, "latest": False,
                               "notes": notes, "assets": []})
        save(db)
        return
    rel = release_of(db, tag)
    if rel is None:
        die(1, "release not found: " + tag)
    if sub == "upload":
        log("WRITE", ["upload", tag])
        files = positional(rest, {"-R", "--repo"})[1:]
        for i, path in enumerate(files):
            if "upload" in FAIL and i == 1:
                save(db)
                die(1, "injected failure: upload after first file")
            with open(path, "rb") as f:
                add_asset(db, rel, os.path.basename(path), f.read())
        save(db)
        return
    if sub == "delete-asset":
        name = positional(rest, {"-R", "--repo"})[1]
        log("WRITE", ["delete-asset", tag, name])
        if "delete" in FAIL:
            die(1, "injected failure: delete")
        rel["assets"] = [a for a in rel["assets"] if a["name"] != name]
        save(db)
        return
    if sub == "edit":
        if "--draft=false" in rest:
            log("WRITE", ["edit-publish", tag, "latest" if "--latest" in rest else "not-latest"])
            if "publish" in FAIL:
                die(1, "injected failure: publish")
            rel["draft"] = False
            rel["latest"] = "--latest" in rest
        else:
            log("WRITE", ["edit", tag])
            if "--notes-file" in rest:
                with open(flag_value(rest, "--notes-file"), encoding="utf-8") as f:
                    rel["notes"] = f.read()
        save(db)
        return
    die(99, "unexpected gh release call: " + sub)


def main(argv):
    if not argv:
        die(99, "no arguments")
    # 真的 gh 會拒絕含控制字元（例如 Windows 上殘留的 CR）的 URL 或參數；假的也必須拒絕，
    # 否則本機 harness 抓不到「輸出帶 CR」這類只在真實 CI 才爆的錯誤（v0.13.0 首次發佈就發生過）。
    for x in argv:
        if any(ord(c) < 0x20 or ord(c) == 0x7F for c in x):
            die(1, "invalid control character in URL or argument: %r" % x)
    if argv[0] == "seed":
        return cmd_seed(argv[1:])
    if argv[0] == "api":
        return cmd_api(argv[1:])
    if argv[0] == "release":
        return cmd_release(argv[1:])
    die(99, "unexpected gh command: " + " ".join(argv))


if __name__ == "__main__":
    if not STATE:
        die(99, "FAKE_GH_STATE 未設定")
    main(sys.argv[1:])
