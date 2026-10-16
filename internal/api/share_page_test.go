package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
)

// TestSharePageAssets 確認分享頁從注入的 fs.FS 讀 index.html，並保留 base 注入與安全 header；
// 網址有無結尾斜線時，相對資源路徑都要解析回站台根目錄。
func TestSharePageAssets(t *testing.T) {
	assets := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><html><head><title>share</title></head><body>app</body></html>")},
	}
	h := NewShareHandler(nil, ShareHooks{}, assets)
	app := fiber.New()
	app.Get("/share/:token", h.Page)

	cases := []struct {
		name     string
		path     string
		wantBase string
	}{
		{name: "無結尾斜線", path: "/share/tok123", wantBase: `<base href="../" />`},
		{name: "有結尾斜線", path: "/share/tok123/", wantBase: `<base href="../../" />`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := app.Test(httptest.NewRequest(http.MethodGet, tc.path, nil), -1)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			b, _ := io.ReadAll(res.Body)
			body := string(b)

			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d，want 200（body=%q）", res.StatusCode, body)
			}
			if !strings.Contains(body, tc.wantBase) {
				t.Errorf("缺少 base 注入 %s：%q", tc.wantBase, body)
			}
			if !strings.Contains(body, ">app</body>") {
				t.Errorf("頁面不是來自注入的 fs.FS：%q", body)
			}
			if got := res.Header.Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q，want no-store", got)
			}
			if got := res.Header.Get("Referrer-Policy"); got != "no-referrer" {
				t.Errorf("Referrer-Policy = %q，want no-referrer", got)
			}
			if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
				t.Errorf("Content-Type = %q，want text/html", got)
			}
		})
	}
}
