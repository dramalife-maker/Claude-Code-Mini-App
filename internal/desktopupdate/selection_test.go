package desktopupdate

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

// TestDesktopUpdateSelection 驗證穩定版 provider 的正式版與資產選擇、checksum 強制要求。
func TestDesktopUpdateSelection(t *testing.T) {
	req := updater.CheckRequest{CurrentVersion: "1.0.0", Platform: "windows", Arch: "amd64"}

	check := func(t *testing.T, f *fixture) (*updater.Release, error) {
		t.Helper()
		return newTestProvider(t, f).Check(context.Background(), req)
	}

	t.Run("v1.0.0 偵測到 v1.0.1 時只提供桌面 EXE，並附上 notes 與 sha256", func(t *testing.T) {
		f := newFixture(t)
		exe := desktopExe("1.0.1")
		zip := legacyZip("1.0.1")
		server := fakeAsset{Name: "claude-miniapp-server-v1.0.1-linux-amd64", Body: []byte("server-bytes")}
		f.setLatest(stableRelease("1.0.1", zip, server, exe, sumsFor(zip, server, exe)))

		rel, err := check(t, f)
		if err != nil {
			t.Fatalf("Check error = %v", err)
		}
		if rel == nil {
			t.Fatal("want v1.0.1 offered, got none")
		}
		if rel.Version != "1.0.1" {
			t.Errorf("Version = %q, want 1.0.1", rel.Version)
		}
		if rel.Channel != "stable" {
			t.Errorf("Channel = %q, want stable", rel.Channel)
		}
		if rel.Artifact.Filename != exe.Name {
			t.Errorf("Artifact.Filename = %q, want %q", rel.Artifact.Filename, exe.Name)
		}
		if rel.Notes != "release notes for 1.0.1" {
			t.Errorf("Notes = %q", rel.Notes)
		}
		if rel.Verification == nil {
			t.Fatal("Verification = nil, want sha256 digest")
		}
		if rel.Verification.DigestAlgo != "sha256" || len(rel.Verification.Digest) != 32 {
			t.Fatalf("Verification = %s/%d bytes, want sha256/32", rel.Verification.DigestAlgo, len(rel.Verification.Digest))
		}
		if got := hex.EncodeToString(rel.Verification.Digest); got != sha256Hex(exe.Body) {
			t.Errorf("digest = %s, want %s", got, sha256Hex(exe.Body))
		}
		if n := f.latestCallCount(); n != 1 {
			t.Errorf("latest endpoint calls = %d, want 1 (fixed repository only)", n)
		}
		if h := f.seenAuthHeaders(); len(h) != 0 {
			t.Errorf("Authorization headers sent = %v, want none (no PAT)", h)
		}
	})

	excluded := []struct {
		name string
		rel  *fakeRelease
	}{
		{"同版本 v1.0.0 不提供", stableRelease("1.0.0", withSums(desktopExe("1.0.0"))...)},
		{"較舊版本 v0.9.9 不提供", stableRelease("0.9.9", withSums(desktopExe("0.9.9"))...)},
		{"預覽版 v1.0.1 不提供", &fakeRelease{Tag: "v1.0.1", Prerelease: true, Assets: withSums(desktopExe("1.0.1"))}},
		{"草稿 v1.0.1 不提供", &fakeRelease{Tag: "v1.0.1", Draft: true, Assets: withSums(desktopExe("1.0.1"))}},
	}
	for _, tc := range excluded {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.setLatest(tc.rel)
			rel, err := check(t, f)
			if err != nil {
				t.Fatalf("Check error = %v, want nil (no candidate)", err)
			}
			if rel != nil {
				t.Fatalf("offered %q (%s), want no update", rel.Version, rel.Artifact.Filename)
			}
		})
	}

	t.Run("沒有正式版（404）視為沒有候選，不回報錯誤", func(t *testing.T) {
		f := newFixture(t)
		rel, err := check(t, f)
		if err != nil || rel != nil {
			t.Fatalf("Check = (%v, %v), want (nil, nil)", rel, err)
		}
	})

	errCases := []struct {
		name     string
		setup    func(f *fixture)
		wantCode ErrorCode
	}{
		{
			name: "只有 Linux 與 arm64 的桌面資產 → asset_missing（錯平台）",
			setup: func(f *fixture) {
				linux := fakeAsset{Name: "claude-miniapp-desktop-v1.0.1-linux-amd64", Body: []byte("l")}
				arm := fakeAsset{Name: "claude-miniapp-desktop-v1.0.1-windows-arm64.exe", Body: []byte("a")}
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: withSums(linux, arm)})
			},
			wantCode: CodeAssetMissing,
		},
		{
			name: "只有 ZIP 與伺服器執行檔 → asset_missing",
			setup: func(f *fixture) {
				server := fakeAsset{Name: "claude-miniapp-server-v1.0.1-linux-amd64", Body: []byte("s")}
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: withSums(legacyZip("1.0.1"), server)})
			},
			wantCode: CodeAssetMissing,
		},
		{
			name: "完全沒有資產 → asset_missing",
			setup: func(f *fixture) {
				f.setLatest(&fakeRelease{Tag: "v1.0.1"})
			},
			wantCode: CodeAssetMissing,
		},
		{
			name: "EXE 檔名版本與 tag 不符 → asset_missing",
			setup: func(f *fixture) {
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: withSums(desktopExe("1.0.2"))})
			},
			wantCode: CodeAssetMissing,
		},
		{
			name: "缺 checksums.txt → checksum_missing",
			setup: func(f *fixture) {
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: []fakeAsset{desktopExe("1.0.1")}})
			},
			wantCode: CodeChecksumMissing,
		},
		{
			name: "checksums.txt 沒有該 EXE 的行 → checksum_missing",
			setup: func(f *fixture) {
				exe := desktopExe("1.0.1")
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: []fakeAsset{exe, sumsFor(legacyZip("1.0.1"))}})
			},
			wantCode: CodeChecksumMissing,
		},
		{
			name: "checksum 不是十六進位 → checksum_missing",
			setup: func(f *fixture) {
				exe := desktopExe("1.0.1")
				bad := fakeAsset{Name: checksumAsset, Body: []byte(strings.Repeat("z", 64) + "  " + exe.Name + "\n")}
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: []fakeAsset{exe, bad}})
			},
			wantCode: CodeChecksumMissing,
		},
		{
			name: "checksum 長度不是 SHA256 → checksum_missing",
			setup: func(f *fixture) {
				exe := desktopExe("1.0.1")
				short := fakeAsset{Name: checksumAsset, Body: []byte("abcd  " + exe.Name + "\n")}
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: []fakeAsset{exe, short}})
			},
			wantCode: CodeChecksumMissing,
		},
		{
			name: "checksums.txt 下載失敗（HTTP 500）→ check_failed，不是 checksum_missing",
			setup: func(f *fixture) {
				f.setLatest(&fakeRelease{Tag: "v1.0.1", Assets: withSums(desktopExe("1.0.1"))})
				f.failDownloads(checksumAsset, 1)
			},
			wantCode: CodeCheckFailed,
		},
		{
			name: "API 限流（HTTP 403）→ check_failed，不回報已是最新",
			setup: func(f *fixture) {
				f.setLatestStatus(403)
			},
			wantCode: CodeCheckFailed,
		},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.setup(f)
			rel, err := check(t, f)
			if err == nil {
				t.Fatalf("Check = (%v, nil), want error code %s", rel, tc.wantCode)
			}
			if rel != nil {
				t.Fatalf("Check returned release %q alongside error", rel.Version)
			}
			var pe *Error
			if !errors.As(err, &pe) {
				t.Fatalf("error %T %v is not *Error", err, err)
			}
			if pe.Code != tc.wantCode {
				t.Errorf("code = %s, want %s (err: %v)", pe.Code, tc.wantCode, err)
			}
		})
	}
}

func withSums(assets ...fakeAsset) []fakeAsset {
	return append(assets, sumsFor(assets...))
}
