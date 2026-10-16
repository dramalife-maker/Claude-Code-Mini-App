package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestClaudePermAuth(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Post("/mcp/perm", claudePermAuth("private-test-token"), func(c *fiber.Ctx) error { return c.SendStatus(204) })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.Shutdown() })
	for _, token := range []string{"", "public-mcp-token", "private-test-token"} {
		req, _ := http.NewRequest("POST", "http://"+ln.Addr().String()+"/mcp/perm", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := 401
		if token == "private-test-token" {
			want = 204
		}
		if res.StatusCode != want {
			t.Fatalf("token case returned %d, want %d", res.StatusCode, want)
		}
	}
	// app.Test's direct remote address is 0.0.0.0; forwarded headers cannot make it loopback.
	req := httptest.NewRequest("POST", "/mcp/perm", nil)
	req.Header.Set("Authorization", "Bearer private-test-token")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Fatalf("forwarded IP bypassed loopback check: %d", res.StatusCode)
	}
}
