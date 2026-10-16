#!/usr/bin/env python3
"""解析 GitHub Releases 列表 JSON（從 stdin 讀取，只讀取、不發出任何請求）。

用法：
  release_state.py find   <tag>                      -> 輸出 "none" 或 "<draft|published> <id>"
  release_state.py assets <tag>                      -> 每行 "<名稱> <asset id> <大小>"
  release_state.py verify <tag> <dist目錄> <名稱...>  -> 名稱集合必須完全相同且大小相符，輸出 "<名稱> <asset id>"

由 scripts/release/publish.sh 呼叫；失敗時以非零結束碼與 stderr 訊息停止發佈。
"""
import json
import os
import sys


def fail(msg):
    print("release_state: " + msg, file=sys.stderr)
    sys.exit(1)


def release_for(tag, releases):
    for r in releases:
        if r.get("tag_name") == tag:
            return r
    return None


def main(argv):
    if len(argv) < 2:
        fail("用法錯誤")
    cmd, tag = argv[0], argv[1]
    releases = json.load(sys.stdin)
    r = release_for(tag, releases)

    if cmd == "find":
        if r is None:
            print("none")
        else:
            print(("draft" if r.get("draft") else "published"), r["id"])
        return

    if r is None:
        fail("找不到 tag " + tag + " 的 Release")

    if cmd == "assets":
        for a in r.get("assets", []):
            print(a["name"], a["id"], a["size"])
        return

    if cmd == "verify":
        if len(argv) < 4:
            fail("verify 需要 <dist目錄> 與預期檔名")
        dist, expected = argv[2], argv[3:]
        assets = {a["name"]: a for a in r.get("assets", [])}
        if sorted(assets) != sorted(expected):
            fail("資產名稱集合不符：實際 %s，預期 %s" % (sorted(assets), sorted(expected)))
        for name in expected:
            local = os.path.getsize(os.path.join(dist, name))
            if local != assets[name]["size"]:
                fail("%s 大小不符：本地 %d，遠端 %d" % (name, local, assets[name]["size"]))
            print(name, assets[name]["id"])
        return

    fail("未知指令 " + cmd)


if __name__ == "__main__":
    main(sys.argv[1:])
