package auth

import (
	"fmt"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RealIP 依序嘗試取得真實用戶端 IP：
// 1. CF-Connecting-IP（Cloudflare Tunnel 設定）
// 2. X-Forwarded-For 的第一個條目
// 3. 直連 IP（c.IP()）
func RealIP(c *fiber.Ctx) string {
	if ip := c.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	return c.IP()
}

// ParseCIDRs 將字串陣列解析為 net.IPNet 清單。
func ParseCIDRs(strs []string) ([]net.IPNet, error) {
	nets := make([]net.IPNet, 0, len(strs))
	for _, s := range strs {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("無效 CIDR %q: %w", s, err)
		}
		nets = append(nets, *ipNet)
	}
	return nets, nil
}

// IsAllowed 檢查 IP 是否在允許的 CIDR 清單內。
func IsAllowed(ipStr string, allowed []net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range allowed {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
