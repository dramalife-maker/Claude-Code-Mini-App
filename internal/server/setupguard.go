package server

import (
	"crypto/subtle"
	"net"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// setupRequestAllowed 是設定 API 的三道檢查。needToken 為 false 時只看位址與 Host（/setup/status）。
// 位址必須是連線本身的遠端位址，呼叫端不可傳入從轉發標頭解析的 IP。
func setupRequestAllowed(remoteIP net.IP, host string, port int, tokenHeader, token string, needToken bool) bool {
	if remoteIP == nil || !remoteIP.IsLoopback() {
		return false
	}
	if !setupHostOK(host, port) {
		return false
	}
	if !needToken {
		return true
	}
	if len(tokenHeader) != len(token) || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tokenHeader), []byte(token)) == 1
}

// setupHostOK 要求 Host 帶上目前監聽埠，擋掉 DNS rebinding（攻擊者網域指到 127.0.0.1）。
func setupHostOK(host string, port int) bool {
	name, got, err := net.SplitHostPort(strings.TrimSpace(host))
	if err != nil || got != strconv.Itoa(port) {
		return false
	}
	switch strings.ToLower(name) {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

// setupGuard 把純函式套到 fiber。只用連線的實際遠端位址，不讀 X-Forwarded-For。
func setupGuard(port int, token string, needToken bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if setupRequestAllowed(c.Context().RemoteIP(), string(c.Request().Host()), port, c.Get("X-Setup-Token"), token, needToken) {
			return c.Next()
		}
		return c.Status(403).JSON(fiber.Map{"error": "拒絕設定請求", "code": "setup_forbidden"})
	}
}
