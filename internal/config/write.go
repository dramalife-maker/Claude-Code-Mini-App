package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrConfigExists 表示目標設定檔已經存在，WriteNew 不會改它。
var ErrConfigExists = errors.New("設定檔已存在")

// WriteNew 把內容寫到 path，且只在 path 尚不存在時成功。
// 先在同目錄寫暫存檔並 Sync，再用硬連結放到目標（出現即完整）。
// 檔案系統不支援硬連結時回報錯誤，避免留下可被讀取的半份設定檔。
func WriteNew(path string, content []byte) error {
	return writeNew(path, content, os.Link)
}

func writeNew(path string, content []byte, link func(string, string) error) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".setup-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0600); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if err := link(tmpName, path); err != nil {
		if os.IsExist(err) {
			return ErrConfigExists
		}
		return err
	}
	keep = true
	return os.Remove(tmpName)
}
