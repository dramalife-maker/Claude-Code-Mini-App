package server

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSetupGuard(t *testing.T) {
	const port = 8080
	token := strings.Repeat("ab", 32)
	loop4 := net.ParseIP("127.0.0.1")
	loop6 := net.ParseIP("::1")
	tests := []struct {
		name, host, header string
		ip                 net.IP
		need, ok           bool
	}{
		{"非 loopback IPv4", "127.0.0.1:8080", token, net.ParseIP("8.8.8.8"), true, false},
		{"非 loopback IPv6", "[::1]:8080", token, net.ParseIP("2001:db8::1"), true, false},
		{"loopback IPv4", "127.0.0.1:8080", token, loop4, true, true},
		{"loopback IPv6", "[::1]:8080", token, loop6, true, true},
		{"localhost", "localhost:8080", token, loop4, true, true},
		{"LOCALHOST 大小寫", "LOCALHOST:8080", token, loop4, true, true},
		{"偽造 Host", "attacker.example:8080", token, loop4, true, false},
		{"Host 不帶埠", "127.0.0.1", token, loop4, true, false},
		{"Host 錯誤埠", "127.0.0.1:9090", token, loop4, true, false},
		{"缺憑證", "127.0.0.1:8080", "", loop4, true, false},
		{"錯誤憑證", "127.0.0.1:8080", strings.Repeat("cd", 32), loop4, true, false},
		{"憑證長度不同", "127.0.0.1:8080", "abc", loop4, true, false},
		{"不需憑證", "127.0.0.1:8080", "", loop4, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setupRequestAllowed(tt.ip, tt.host, port, tt.header, token, tt.need)
			if got != tt.ok {
				t.Fatalf("allowed = %v，預期 %v", got, tt.ok)
			}
		})
	}
}

func TestSetupGuardIgnoresForwardedFor(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	app := fiber.New()
	app.Get("/setup/status", setupGuard(8080, token, false), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	// fiber 的 app.Test 連線遠端固定是 0.0.0.0，忽略 req.RemoteAddr。
	// Host 正確、X-Forwarded-For 卻是 127.0.0.1：若守衛誤讀轉發標頭會放行。
	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 403 {
		t.Fatalf("status = %d body = %s，轉發標頭不該把非本機當成 loopback", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"code":"setup_forbidden"`) {
		t.Fatalf("body = %s", body)
	}
}
