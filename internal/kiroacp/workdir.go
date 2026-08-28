package kiroacp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// workDirResult 是 resolveWorkDir 的解析結果。
type workDirResult struct {
	Path     string
	Fallback bool // true 表示 session 未設 work_dir，改用 server 當下 cwd
}

// resolveWorkDir 將 session work_dir 轉成存在的絕對路徑。
// 未設定時 fallback 到 miniapp server 的 cwd，並由呼叫端記錄警告。
func resolveWorkDir(raw string) (workDirResult, error) {
	wdir := strings.TrimSpace(raw)
	fallback := false
	if wdir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return workDirResult{}, fmt.Errorf("kiroacp: 工作目錄未設定且無法取得 server cwd: %w", err)
		}
		wdir = cwd
		fallback = true
	}
	abs, err := filepath.Abs(filepath.Clean(wdir))
	if err != nil {
		return workDirResult{}, fmt.Errorf("kiroacp: 工作目錄無效: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return workDirResult{}, fmt.Errorf("kiroacp: 工作目錄不存在: %s", abs)
	}
	if !st.IsDir() {
		return workDirResult{}, fmt.Errorf("kiroacp: 工作目錄不是資料夾: %s", abs)
	}
	return workDirResult{Path: abs, Fallback: fallback}, nil
}

func logWorkDirWarning(res workDirResult) {
	if res.Fallback {
		log.Printf("[kiroacp] 警告: session 未設定 work_dir，使用 server 工作目錄 %s", res.Path)
	}
}
