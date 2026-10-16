package desktopupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fixture 是本機 httptest 的 GitHub API 替身；測試永不連到公開 GitHub。
type fixture struct {
	srv *httptest.Server

	mu            sync.Mutex
	latest        *fakeRelease // nil 表示沒有任何正式版（404）
	latestStatus  int          // 非 0：/releases/latest 直接回傳此狀態碼
	latestCalls   int
	authHeaders   []string
	downloadReqs  map[string]int // 每個資產名稱收到的下載請求數
	downloadFails map[string]int // 每個資產名稱接下來要故意回 500 的次數
	downloadCode  int            // 非 0：所有下載請求回傳此狀態碼
	gate          chan struct{}  // 非 nil：下載會等到 gate 關閉後才回應
}

type fakeAsset struct {
	Name string
	Body []byte
	URL  string // 非空時覆寫 browser_download_url（用於模擬連線失敗）
}

type fakeRelease struct {
	Tag        string
	Notes      string
	Prerelease bool
	Draft      bool
	Assets     []fakeAsset
}

type apiAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type apiRelease struct {
	TagName    string     `json:"tag_name"`
	Name       string     `json:"name"`
	Body       string     `json:"body"`
	Draft      bool       `json:"draft"`
	Prerelease bool       `json:"prerelease"`
	HTMLURL    string     `json:"html_url"`
	Assets     []apiAsset `json:"assets"`
}

const latestPath = "/repos/" + Repository + "/releases/latest"

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		downloadReqs:  map[string]int{},
		downloadFails: map[string]int{},
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fixture) setLatest(r *fakeRelease) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.latest = r
}

func (f *fixture) setLatestStatus(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.latestStatus = code
}

func (f *fixture) setDownloadCode(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.downloadCode = code
}

// failDownloads 讓接下來 n 次對 name 的請求回 500。
func (f *fixture) failDownloads(name string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.downloadFails[name] = n
}

// holdDownloads 讓之後的下載請求等待，直到回傳的函式被呼叫（可安全重複呼叫）。
func (f *fixture) holdDownloads() (open func()) {
	gate := make(chan struct{})
	f.mu.Lock()
	f.gate = gate
	f.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { close(gate) }) }
}

func (f *fixture) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	if a := r.Header.Get("Authorization"); a != "" {
		f.authHeaders = append(f.authHeaders, a)
	}
	f.mu.Unlock()

	switch {
	case r.URL.Path == latestPath:
		f.serveLatest(w)
	case strings.HasPrefix(r.URL.Path, "/download/"):
		f.serveDownload(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *fixture) serveLatest(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.latestCalls++
	if f.latestStatus != 0 {
		w.WriteHeader(f.latestStatus)
		return
	}
	if f.latest == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	out := apiRelease{
		TagName:    f.latest.Tag,
		Name:       f.latest.Tag,
		Body:       f.latest.Notes,
		Draft:      f.latest.Draft,
		Prerelease: f.latest.Prerelease,
		HTMLURL:    f.srv.URL + "/releases/tag/" + f.latest.Tag,
	}
	for i, a := range f.latest.Assets {
		url := a.URL
		if url == "" {
			url = f.srv.URL + "/download/" + a.Name
		}
		out.Assets = append(out.Assets, apiAsset{
			ID:                 int64(i + 1),
			Name:               a.Name,
			ContentType:        "application/octet-stream",
			Size:               int64(len(a.Body)),
			BrowserDownloadURL: url,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (f *fixture) serveDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/download/")

	f.mu.Lock()
	f.downloadReqs[name]++
	if f.downloadCode != 0 {
		code := f.downloadCode
		f.mu.Unlock()
		w.WriteHeader(code)
		return
	}
	if f.downloadFails[name] > 0 {
		f.downloadFails[name]--
		f.mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	gate := f.gate
	f.mu.Unlock()

	if gate != nil {
		<-gate
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.latest != nil {
		for _, a := range f.latest.Assets {
			if a.Name == name {
				_, _ = w.Write(a.Body)
				return
			}
		}
	}
	http.NotFound(w, r)
}

func (f *fixture) latestCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.latestCalls
}

func (f *fixture) downloadCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.downloadReqs[name]
}

func (f *fixture) seenAuthHeaders() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.authHeaders...)
}

// desktopExe 回傳指定版本的桌面 EXE 資產（位元組內容固定，便於比對雜湊）。
func desktopExe(version string) fakeAsset {
	return fakeAsset{
		Name: AssetName(version),
		Body: []byte("desktop-exe-bytes-" + version),
	}
}

// legacyZip 模擬舊版 Windows ZIP 安裝包（只能被拒絕，不能被選中）。
func legacyZip(version string) fakeAsset {
	return fakeAsset{Name: "claude-miniapp-v" + version + "-windows-amd64.zip", Body: []byte("zip-bytes")}
}

// sumsFor 依 sha256sum 格式為給定資產產生 checksums.txt。
func sumsFor(assets ...fakeAsset) fakeAsset {
	var b strings.Builder
	for _, a := range assets {
		sum := sha256.Sum256(a.Body)
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), a.Name)
	}
	return fakeAsset{Name: checksumAsset, Body: []byte(b.String())}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// stableRelease 組出 v<version> 的正式版發佈；assets 之外不自動加入任何資產。
func stableRelease(version string, assets ...fakeAsset) *fakeRelease {
	return &fakeRelease{Tag: "v" + version, Notes: "release notes for " + version, Assets: assets}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// newTestProvider 指向本機 fixture 建立穩定版 provider。
func newTestProvider(t *testing.T, f *fixture) *stableProvider {
	t.Helper()
	p, err := newStableProvider(f.srv.URL, nil)
	if err != nil {
		t.Fatalf("newStableProvider: %v", err)
	}
	return p
}
