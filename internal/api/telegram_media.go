package api

import (
	"fmt"
	"log/slog"
	"mime"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/tgclient"
)

// TelegramMediaHTTP GET /sessions/:id/tg-media/:msgId
// 圖片預覽與檔案下載都走這裡，內容由 Telegram 串流轉發，不寫入磁碟。
func (h *SessionHandler) TelegramMediaHTTP(c *fiber.Ctx) error {
	msgID, err := strconv.Atoi(c.Params("msgId"))
	if err != nil || msgID <= 0 {
		return historyError(c, tgclient.ErrHistoryInvalidCursor)
	}
	if h.TelegramMedia == nil {
		return historyError(c, tgclient.ErrHistoryUnavailable)
	}
	slog.Info(fmt.Sprintf("[telegram] 媒體請求 session=%s msg=%d download=%s", c.Params("id"), msgID, c.Query("download")))
	name, mimeType, body, err := h.TelegramMedia(c.UserContext(), c.Params("id"), msgID)
	if err != nil {
		return historyError(c, err)
	}
	slog.Info(fmt.Sprintf("[telegram] 媒體開始串流 msg=%d name=%s mime=%s", msgID, name, mimeType))
	// SetBodyStream 送完會關閉 body；這裡再 Close 會在寫出前把串流截斷。
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if name == "" {
		name = "file"
	}
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Type", mimeType)
	disposition := "attachment"
	if strings.HasPrefix(mimeType, "image/") && c.Query("download") != "1" {
		disposition = "inline"
	}
	c.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	// 長度未知：邊收邊送，不先把整份檔案放進記憶體或磁碟。
	c.Response().SetBodyStream(body, -1)
	return nil
}
