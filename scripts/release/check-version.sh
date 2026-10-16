#!/usr/bin/env bash
# 發佈前一致性檢查（CI 與本機測試共用）：
#   - tag 必須是 vX.Y.Z，且等於根目錄 VERSION 的 v<版本>
#   - internal/version/version.go 的 Version 必須等於 VERSION
#   - tag 必須指向觸發這次 workflow 的 commit（GITHUB_SHA）
# 任一不符即失敗，且不會產生任何產物或 Release。
set -euo pipefail

fail() { echo "release check failed: $*" >&2; exit 1; }

tag="${GITHUB_REF_NAME:?GITHUB_REF_NAME 未設定}"
sha="${GITHUB_SHA:?GITHUB_SHA 未設定}"

ver="$(tr -d '\r\n' < VERSION)"
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "VERSION 必須是 X.Y.Z（桌面更新資產名稱只接受這種格式），實際為 '$ver'"
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "tag '$tag' 必須是 vX.Y.Z"
[ "$tag" = "v$ver" ] || fail "tag '$tag' 必須等於 v$ver（VERSION）"

go_ver="$(sed -n 's/^const Version = "\([^"]*\)"\r\{0,1\}$/\1/p' internal/version/version.go)"
[ "$go_ver" = "$ver" ] || fail "internal/version/version.go 的 Version 為 '$go_ver'，與 VERSION '$ver' 不符"

commit="$(git rev-parse "$tag^{commit}")"
[ "$commit" = "$sha" ] || fail "tag $tag 指向 $commit，但觸發建置的是 $sha"

echo "release check ok: $tag == VERSION == version.go, commit $sha"
