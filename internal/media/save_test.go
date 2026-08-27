package media

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveBase64Image(t *testing.T) {
	t.Chdir(t.TempDir())

	data := []byte("fake-png-bytes")
	url, err := SaveBase64Image("image/png", base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatalf("SaveBase64Image: %v", err)
	}
	if !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("unexpected url: %s", url)
	}

	saved, err := os.ReadFile(filepath.Join(uploadDir, filepath.Base(url)))
	if err != nil {
		t.Fatalf("讀取存檔失敗: %v", err)
	}
	if string(saved) != string(data) {
		t.Fatalf("存檔內容不符: got %q want %q", saved, data)
	}
}

func TestSaveBase64Image_invalidBase64(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := SaveBase64Image("image/png", "not-base64!!"); err == nil {
		t.Fatal("預期無效 base64 應回傳錯誤")
	}
}
