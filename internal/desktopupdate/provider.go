// Package desktopupdate 提供 Windows 桌面版的正式版自我更新：固定 GitHub 來源、
// 精確的桌面 EXE 選擇、強制 SHA256 檢查，以及由使用者觸發的單一更新操作協調。
package desktopupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// Repository 是桌面更新唯一允許的 GitHub 來源，不接受外部傳入。
const Repository = "jerry12122/Claude-Code-Mini-App"

const (
	checksumAsset = "checksums.txt"
	apiBaseURL    = "https://api.github.com"
	// downloadTimeout 是整個 HTTP 請求（含桌面 EXE 串流下載）的上限。
	// 理由：Wails 預設的 30 秒對安裝檔可能不足，會把慢速網路誤判為下載失敗。
	// 10 分鐘是人為設定的上限，不是量測值；逾時仍會回報 download_failed 且可重試。
	downloadTimeout = 10 * time.Minute
)

// ErrorCode 是提供給使用者與診斷用的穩定錯誤代碼。
type ErrorCode string

const (
	CodeCheckFailed        ErrorCode = "check_failed"
	CodeAssetMissing       ErrorCode = "asset_missing"
	CodeChecksumMissing    ErrorCode = "checksum_missing"
	CodeChecksumMismatch   ErrorCode = "checksum_mismatch"
	CodeDownloadFailed     ErrorCode = "download_failed"
	CodeInstallNotWritable ErrorCode = "install_not_writable"
)

// Error 攜帶錯誤代碼與原因。
type Error struct {
	Code ErrorCode
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }

// AssetName 回傳指定正式版的桌面 EXE 精確檔名。
func AssetName(version string) string {
	return "claude-miniapp-desktop-v" + version + "-windows-amd64.exe"
}

var desktopAssetPattern = regexp.MustCompile(`^claude-miniapp-desktop-v\d+\.\d+\.\d+-windows-amd64\.exe$`)

// stableProvider 包裝 Wails 的 GitHub provider，只允許固定 repo、正式版、精確桌面 EXE 與強制 SHA256。
type stableProvider struct {
	inner *github.Provider

	mu      sync.Mutex
	lastErr *Error
}

// newStableProvider 建立 provider。repo 固定為 Repository、不帶 token；baseURL 僅供測試指向本機 fixture。
func newStableProvider(baseURL string, client *http.Client) (*stableProvider, error) {
	if baseURL == "" {
		baseURL = apiBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: downloadTimeout}
	}
	guarded := *client
	guarded.Transport = releaseGuard{next: transportOf(client)}

	p := &stableProvider{}
	inner, err := github.New(github.Config{
		Repository:    Repository,
		BaseURL:       baseURL,
		AssetMatcher:  p.matchDesktopAsset,
		ChecksumAsset: checksumAsset,
		HTTPClient:    &guarded,
	})
	if err != nil {
		return nil, err
	}
	p.inner = inner
	return p, nil
}

func transportOf(c *http.Client) http.RoundTripper {
	if c.Transport != nil {
		return c.Transport
	}
	return http.DefaultTransport
}

// Name implements updater.Provider。
func (p *stableProvider) Name() string { return p.inner.Name() }

// Check 回傳嚴格新於目前版本的正式版桌面 EXE；沒有候選時回傳 (nil, nil)。
func (p *stableProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	p.setErr(nil)
	rel, err := p.inner.Check(ctx, req)
	if err != nil {
		if p.lastError() == nil {
			p.setErr(classifyCheckError(err))
		}
		return nil, p.lastError()
	}
	if rel == nil || rel.Channel != "stable" {
		return nil, nil
	}
	if verr := validateRelease(rel); verr != nil {
		p.setErr(verr)
		return nil, verr
	}
	return rel, nil
}

// matchDesktopAsset 只接受唯一一個 windows-amd64 桌面 EXE；ZIP、伺服器與其他平台一律不選。
func (p *stableProvider) matchDesktopAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	if req.Platform != "windows" || req.Arch != "amd64" {
		p.setErr(&Error{Code: CodeAssetMissing, Err: fmt.Errorf("unsupported platform %s/%s", req.Platform, req.Arch)})
		return -1
	}
	hit := -1
	for i, a := range assets {
		if !desktopAssetPattern.MatchString(a.Name) {
			continue
		}
		if hit >= 0 {
			p.setErr(&Error{Code: CodeAssetMissing, Err: errors.New("multiple desktop executables in release")})
			return -1
		}
		hit = i
	}
	if hit < 0 {
		p.setErr(&Error{Code: CodeAssetMissing, Err: errors.New("no exact windows-amd64 desktop executable in release")})
	}
	return hit
}

// validateRelease 確認選中的資產檔名與版本一致，且具備有效的 sha256 digest（下載前必須通過）。
func validateRelease(rel *updater.Release) *Error {
	want := AssetName(rel.Version)
	if rel.Artifact.Filename != want {
		return &Error{Code: CodeAssetMissing, Err: fmt.Errorf("release %s has no exact desktop asset %s", rel.Version, want)}
	}
	v := rel.Verification
	if v == nil {
		return &Error{Code: CodeChecksumMissing, Err: fmt.Errorf("%s has no entry for %s", checksumAsset, want)}
	}
	if v.DigestAlgo != "sha256" || len(v.Digest) != sha256.Size {
		return &Error{Code: CodeChecksumMissing, Err: fmt.Errorf("invalid sha256 digest for %s", want)}
	}
	return nil
}

// classifyCheckError 把 Wails provider 回傳的錯誤歸類；checksum 格式錯誤屬於 checksum_missing。
func classifyCheckError(err error) *Error {
	var invalid hex.InvalidByteError
	if errors.As(err, &invalid) || errors.Is(err, hex.ErrLength) {
		return &Error{Code: CodeChecksumMissing, Err: err}
	}
	return &Error{Code: CodeCheckFailed, Err: err}
}

// Download 委派給 Wails 的串流下載；只會對已通過 validateRelease 的候選被呼叫（由 Manager 保證）。
func (p *stableProvider) Download(ctx context.Context, rel *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	return p.inner.Download(ctx, rel, dst, onProgress)
}

func (p *stableProvider) setErr(e *Error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastErr = e
}

func (p *stableProvider) lastError() *Error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}

// releaseGuard 讓 /releases/latest 上的草稿或預覽版回應等同「沒有正式版」（404）。
//
// 為何不直接信任 provider：Wails 的 GitHub provider 不解析也不暴露 draft 欄位，
// 只能依 prerelease 分流。對公開 API，/releases/latest 本來不會回傳草稿，
// 因此這道檢查是防禦層（若 API 行為或 endpoint 改變，仍不會選到草稿），而非主要過濾。
type releaseGuard struct {
	next http.RoundTripper
}

func (g releaseGuard) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := g.next.RoundTrip(r)
	if err != nil || resp.StatusCode != http.StatusOK || !strings.HasSuffix(r.URL.Path, "/releases/latest") {
		return resp, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var meta struct {
		Draft      bool `json:"draft"`
		Prerelease bool `json:"prerelease"`
	}
	if json.Unmarshal(body, &meta) == nil && (meta.Draft || meta.Prerelease) {
		return &http.Response{
			Status:     "404 Not Found",
			StatusCode: http.StatusNotFound,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Request:    r,
		}, nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}
