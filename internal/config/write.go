package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// ErrConfigExists 表示目標設定檔已經存在，WriteNew 不會改它。
var ErrConfigExists = errors.New("設定檔已存在")

// WriteNew 把內容寫到 path，且只在 path 尚不存在時成功。
// 先在同目錄寫暫存檔並 Sync，再用硬連結放到目標（出現即完整）。
// 檔案系統不支援硬連結時，改以 O_EXCL 建立目標，失敗就刪掉半成品。
func WriteNew(path string, content []byte) error {
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
	if err := os.Link(tmpName, path); err != nil {
		if os.IsExist(err) {
			return ErrConfigExists
		}
		if !linkUnsupported(err) {
			return err
		}
		return writeExclusive(path, content)
	}
	keep = true
	return os.Remove(tmpName)
}

func writeExclusive(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return ErrConfigExists
		}
		return err
	}
	ok := false
	defer func() {
		if !ok {
			os.Remove(path)
		}
	}()
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func linkUnsupported(err error) bool {
	if errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP) || errors.Is(err, syscall.ENOSYS) {
		return true
	}
	var errno syscall.Errno
	// Windows 上 exFAT 等不支援硬連結時常見這兩個碼。
	return errors.As(err, &errno) && (errno == 1 || errno == 50)
}
