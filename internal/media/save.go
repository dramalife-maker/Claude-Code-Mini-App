// Package media 提供跨 runner 共用的圖片落地介面：把工具回傳的 base64 圖片
// （例如 MCP tool_result 裡的 screenshot）存成靜態檔案，回傳前端可直接載入的 URL。
// 任何 runner（claude/codex/cursor/kiro/...）遇到 image content block 都呼叫同一組函式即可。
package media

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxImageBytes 為單張圖片上限；ponytail: 先擋爆量截圖塞爆磁碟，若之後需要更大圖片再放寬。
const maxImageBytes = 8 * 1024 * 1024

// uploadDir 對應 cmd/server/main.go 的 app.Static("/", "./internal/static")，
// 存在這裡即可免費取得靜態路由，不需另外註冊 server 端點。
const uploadDir = "internal/static/uploads"

var extByMediaType = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// SaveBase64Image 解碼 base64 圖片並存檔，回傳可直接放進 markdown `![](url)` 的相對 URL。
func SaveBase64Image(mediaType, b64Data string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", fmt.Errorf("media: decode base64 image: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("media: empty image data")
	}
	if len(data) > maxImageBytes {
		return "", fmt.Errorf("media: image too large (%d bytes > %d)", len(data), maxImageBytes)
	}

	ext := extByMediaType[strings.ToLower(mediaType)]
	if ext == "" {
		ext = ".png"
	}

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("media: mkdir uploads dir: %w", err)
	}

	name, err := randomFilename(ext)
	if err != nil {
		return "", fmt.Errorf("media: generate filename: %w", err)
	}

	if err := os.WriteFile(filepath.Join(uploadDir, name), data, 0o644); err != nil {
		return "", fmt.Errorf("media: write image file: %w", err)
	}

	return "/uploads/" + name, nil
}

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}
