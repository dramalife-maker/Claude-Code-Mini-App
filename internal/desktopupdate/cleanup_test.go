package desktopupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// 建立測試目錄與檔案；回傳目錄與「目前 exe」的完整路徑（檔案本身也會建立）。
func asideFixture(t *testing.T, files ...string) (dir, exe string) {
	t.Helper()
	dir = t.TempDir()
	exe = filepath.Join(dir, "app.exe")
	if err := os.WriteFile(exe, []byte("current"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, exe
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCleanOldAsides_RemovesEveryAside(t *testing.T) {
	dir, exe := asideFixture(t, "app.exe.old.100", "app.exe.old.300", "app.exe.old.200")
	removed, err := CleanOldAsides(exe)
	if err != nil {
		t.Fatalf("不應有錯誤: %v", err)
	}
	sort.Strings(removed)
	want := []string{
		filepath.Join(dir, "app.exe.old.100"),
		filepath.Join(dir, "app.exe.old.200"),
		filepath.Join(dir, "app.exe.old.300"),
	}
	if !equal(removed, want) {
		t.Fatalf("removed = %v, want %v（一個都不留，包含最新的）", removed, want)
	}
	if got, want := names(t, dir), []string{"app.exe"}; !equal(got, want) {
		t.Fatalf("剩下 %v, want %v（只剩目前的 exe）", got, want)
	}
}

func TestCleanOldAsides_SingleAsideIsRemovedToo(t *testing.T) {
	dir, exe := asideFixture(t, "app.exe.old.5")
	removed, err := CleanOldAsides(exe)
	if err != nil || len(removed) != 1 {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if got, want := names(t, dir), []string{"app.exe"}; !equal(got, want) {
		t.Fatalf("剩下 %v, want %v", got, want)
	}
}

func TestCleanOldAsides_NeverTouchesOtherFiles(t *testing.T) {
	others := []string{
		"app.exe.old.abc",                     // 後綴不是純數字
		"app.exe.old.",                        // 空後綴
		"app.exe.old.1.bak",                   // 後面還有東西
		"app.exe.old.-5",                      // 負號
		"app.exe.old.+7",                      // 正號
		"app.exe.old. 8",                      // 空白
		"other.exe.old.5",                     // 別的檔案的 aside
		"xapp.exe.old.7",                      // 前綴多一個字
		"app.exe.OLD.9",                       // 大小寫不同
		"app.exe.bak",                         // Wails 的 .bak（失敗還原用，不是我們的）
		"config.yaml",                         // 使用者檔案
		"claude-miniapp.db",                   // 資料庫
		"app.exe.old.99999999999999999999999", // 超過 uint64，不可能是 UnixNano
	}
	dir, exe := asideFixture(t, append([]string{"app.exe.old.1", "app.exe.old.2"}, others...)...)
	if err := os.MkdirAll(filepath.Join(dir, "app.exe.old.50"), 0o755); err != nil { // 同名資料夾
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.exe.old.50", "keep.txt"), []byte("k"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := CleanOldAsides(exe)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(removed)
	want := []string{filepath.Join(dir, "app.exe.old.1"), filepath.Join(dir, "app.exe.old.2")}
	if !equal(removed, want) {
		t.Fatalf("只應刪除 .old.1 與 .old.2，實際 %v", removed)
	}
	for _, f := range append(others, "app.exe", filepath.Join("app.exe.old.50", "keep.txt")) {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s 不該被動到: %v", f, err)
		}
	}
}

func TestCleanOldAsides_NothingToDo(t *testing.T) {
	dir, exe := asideFixture(t) // 沒有 aside
	removed, err := CleanOldAsides(exe)
	if err != nil || len(removed) != 0 {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if got, want := names(t, dir), []string{"app.exe"}; !equal(got, want) {
		t.Fatalf("剩下 %v, want %v", got, want)
	}
}

func TestCleanOldAsides_MissingDirIsAnErrorNotAPanic(t *testing.T) {
	removed, err := CleanOldAsides(filepath.Join(t.TempDir(), "nope", "app.exe"))
	if err == nil || len(removed) != 0 {
		t.Fatalf("目錄不存在應回錯誤：removed=%v err=%v", removed, err)
	}
}

func TestCleanOldAsides_NeverDeletesTheRunningExecutable(t *testing.T) {
	// 目前的 exe 本身永遠不吻合 "<自己的檔名>.old.<數字>"；即使是從 aside 檔本身啟動
	// （名稱是 app.exe.old.6），也只會找 "app.exe.old.6.old.<數字>"，不會刪到自己或 app.exe。
	dir, _ := asideFixture(t, "app.exe.old.5", "app.exe.old.6")
	self := filepath.Join(dir, "app.exe.old.6")
	removed, err := CleanOldAsides(self)
	if err != nil || len(removed) != 0 {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if got, want := names(t, dir), []string{"app.exe", "app.exe.old.5", "app.exe.old.6"}; !equal(got, want) {
		t.Fatalf("剩下 %v, want %v", got, want)
	}
}

func TestCleanOldAsides_SkipsSymlinks(t *testing.T) {
	dir, exe := asideFixture(t, "app.exe.old.1")
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "app.exe.old.5")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("無法建立符號連結（Windows 需要權限）: %v", err)
	}
	removed, err := CleanOldAsides(exe)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != filepath.Join(dir, "app.exe.old.1") {
		t.Fatalf("只應刪除一般檔案 app.exe.old.1，實際 %v", removed)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("符號連結不該被刪: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("符號連結指向的檔案不該被動到: %v", err)
	}
}

func TestCleanOldAsides_LockedFileDoesNotAbortTheRest(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("只有 Windows 會因為檔案被開啟而無法刪除")
	}
	dir, exe := asideFixture(t, "app.exe.old.1", "app.exe.old.2", "app.exe.old.3")
	held, err := os.Open(filepath.Join(dir, "app.exe.old.1")) // Go 的 Open 不含 FILE_SHARE_DELETE，Windows 上無法刪除
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	removed, err := CleanOldAsides(exe)
	if err == nil {
		t.Fatal("有檔案刪不掉時應回傳錯誤")
	}
	sort.Strings(removed)
	want := []string{filepath.Join(dir, "app.exe.old.2"), filepath.Join(dir, "app.exe.old.3")}
	if !equal(removed, want) {
		t.Fatalf("鎖住的 1 刪不掉，但 2 與 3 仍應被刪：removed=%v", removed)
	}
	if got, want := names(t, dir), []string{"app.exe", "app.exe.old.1"}; !equal(got, want) {
		t.Fatalf("剩下 %v, want %v", got, want)
	}
}
