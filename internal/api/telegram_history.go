package api

import (
	"errors"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/tgclient"
)

// telegramMessages 處理 Telegram session 的 GET /sessions/:id/messages。
// paginated＝帶 pagination=telegram（回 envelope）；否則只回最新一頁的陣列。
func (h *SessionHandler) telegramMessages(c *fiber.Ctx, s *db.Session, paginated bool) error {
	limit, before := tgclient.DefaultHistoryLimit, ""
	if paginated {
		args := c.Context().QueryArgs()
		if args.Has("limit") {
			n, err := strconv.Atoi(string(args.Peek("limit")))
			if err != nil || n < 1 || n > tgclient.MaxHistoryLimit {
				return historyError(c, tgclient.ErrHistoryInvalidLimit)
			}
			limit = n
		}
		if args.Has("before") {
			if before = string(args.Peek("before")); before == "" {
				return historyError(c, tgclient.ErrHistoryInvalidCursor)
			}
		}
	}

	var res *tgclient.HistoryPageResult
	var err error
	if h.TelegramHistory == nil {
		err = tgclient.ErrHistoryUnavailable
	} else {
		res, err = h.TelegramHistory(c.UserContext(), s.ID, limit, before)
	}
	if err != nil {
		return historyError(c, err)
	}
	if res.Messages == nil {
		res.Messages = []*db.Message{}
	}
	if !paginated {
		return c.JSON(res.Messages)
	}
	return c.JSON(res)
}

// historyError 把 HistoryPage／快照錯誤對應成 HTTP；絕不把錯誤變成空頁或 has_more=false。
func historyError(c *fiber.Ctx, err error) error {
	fail := func(status int, code, msg string) error {
		return c.Status(status).JSON(fiber.Map{"error": msg, "code": code})
	}
	var rl *tgclient.HistoryRateLimitedError
	switch {
	case errors.As(err, &rl):
		secs := int(math.Ceil(rl.RetryAfter.Seconds()))
		c.Set("Retry-After", strconv.Itoa(secs))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Telegram 要求稍後再試", "code": "telegram_rate_limited", "retry_after_seconds": secs,
		})
	case errors.Is(err, tgclient.ErrHistoryInvalidLimit):
		return fail(400, "invalid_limit", "limit 必須是 1 到 100 的整數")
	case errors.Is(err, tgclient.ErrHistoryInvalidCursor):
		return fail(400, "invalid_cursor", "before 游標無效")
	case errors.Is(err, tgclient.ErrHistoryNotTelegram):
		return fail(400, "not_telegram_session", "此會話不是 Telegram 會話")
	case errors.Is(err, tgclient.ErrHistorySessionNotFound):
		return fail(404, "session_not_found", "會話不存在")
	case errors.Is(err, tgclient.ErrHistoryUnavailable), errors.Is(err, tgclient.ErrHistoryNoBinding):
		return fail(503, "telegram_unavailable", "Telegram 目前無法讀取此頁，請稍後再試")
	}
	return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}
