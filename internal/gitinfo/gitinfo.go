// Package gitinfo 依伺服器本機路徑偵測 Git 工作樹並解析目前分支（需 PATH 中有 git）。
package gitinfo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const runTimeout = 3 * time.Second

// Branch 若 workDir 為 Git 工作樹則回傳目前分支名稱（detached 時可能為 "HEAD"）；否則 ok 為 false。
func Branch(workDir string) (branch string, ok bool) {
	if strings.TrimSpace(workDir) == "" {
		return "", false
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	if !insideWorkTree(ctx, abs) {
		return "", false
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), runTimeout)
	defer cancel2()
	out, err := exec.CommandContext(ctx2, "git", "-C", abs, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", false
	}
	branch = strings.TrimSpace(string(out))
	if branch == "" {
		return "", false
	}
	return branch, true
}

func insideWorkTree(ctx context.Context, dir string) bool {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}
