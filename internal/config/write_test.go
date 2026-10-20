package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
)

func TestWriteNewUnsupportedLinkLeavesNoConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := writeNew(path, []byte("complete config"), func(string, string) error {
		return syscall.ENOTSUP
	})
	if !errors.Is(err, syscall.ENOTSUP) {
		t.Fatalf("不支援原子寫入時應回報錯誤，得到 %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("失敗時不應建立設定檔：%v", err)
	}
	assertNoTemp(t, dir, "")
}

func TestWriteNewAtomic(t *testing.T) {
	t.Run("已存在不覆寫", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
		err := WriteNew(path, []byte("new"))
		if !errors.Is(err, ErrConfigExists) {
			t.Fatalf("err = %v，預期 ErrConfigExists", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "old" {
			t.Fatalf("內容被改成 %q", got)
		}
		assertNoTemp(t, dir, path)
	})

	t.Run("併發只有一個成功", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		var wg sync.WaitGroup
		var mu sync.Mutex
		var winner []byte
		okN := 0
		for i := 0; i < 8; i++ {
			content := []byte(fmt.Sprintf("body-%d\n", i))
			wg.Add(1)
			go func(content []byte) {
				defer wg.Done()
				err := WriteNew(path, content)
				if err == nil {
					mu.Lock()
					okN++
					winner = append([]byte(nil), content...)
					mu.Unlock()
					return
				}
				if !errors.Is(err, ErrConfigExists) {
					t.Errorf("非預期錯誤: %v", err)
				}
			}(content)
		}
		wg.Wait()
		if okN != 1 {
			t.Fatalf("成功 %d 次，預期 1", okN)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, winner) {
			t.Fatalf("內容 = %q，預期成功者 %q", got, winner)
		}
		assertNoTemp(t, dir, path)
		if runtime.GOOS != "windows" {
			st, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if st.Mode().Perm() != 0600 {
				t.Fatalf("權限 %o，預期 0600", st.Mode().Perm())
			}
		}
	})
}

func assertNoTemp(t *testing.T, dir, keep string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Join(dir, e.Name()) != keep {
			t.Fatalf("殘留檔案 %s", e.Name())
		}
	}
}
