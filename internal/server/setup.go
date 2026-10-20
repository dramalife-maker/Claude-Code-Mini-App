package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/clicheck"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/static"
)

// setupOptions 讓測試替換前端資源與 CLI 檢查，避免設定模式去跑真實 CLI 或讀磁碟資源。
type setupOptions struct {
	assets fs.FS
	check  func(context.Context) []clicheck.Tool
}

// setupOnMissing 只在設定檔不存在時進入設定模式。壞掉的 YAML 不是缺檔。
func setupOnMissing(err error) bool {
	return errors.Is(err, config.ErrConfigMissing)
}

// startSetup 只提供首次設定需要的頁面與三支 API，不開資料庫、不啟動代理或 MCP。
func startSetup(ctx context.Context, port int, opts setupOptions) (*Server, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	assets := opts.assets
	if assets == nil {
		assets = static.Assets
	}
	check := opts.check
	if check == nil {
		check = clicheck.Check
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("產生設定憑證失敗: %w", err)
	}
	token := hex.EncodeToString(buf)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(wd, "config.yaml")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	// 設定網址帶一次性憑證，而頁面會載入第三方腳本；不讓瀏覽器把這個網址當 Referer 送出去。
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Referrer-Policy", "no-referrer")
		return c.Next()
	})
	app.Use(static.Handler(assets))
	app.Get("/setup/status", setupGuard(port, token, false), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"needs_setup": true,
			"port":        port,
			"config_path": configPath,
		})
	})
	app.Get("/setup/cli-check", setupGuard(port, token, true), func(c *fiber.Ctx) error {
		tools := check(c.UserContext())
		if tools == nil {
			tools = []clicheck.Tool{}
		}
		return c.JSON(fiber.Map{"tools": tools})
	})
	app.Post("/setup/save", setupGuard(port, token, true), func(c *fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"error": "尚未實作", "code": "not_implemented"})
	})
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(503).JSON(fiber.Map{"error": "需要先完成首次設定", "code": "setup_required"})
	})

	// 只綁 loopback：設定模式沒有登入，區網不該連得到，即使防火牆沒擋。
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("無法監聽 %s（埠可能被占用）: %w", addr, err)
	}
	s := &Server{
		app:      app,
		url:      "http://" + addr,
		entryURL: fmt.Sprintf("http://%s/?setup=%s", addr, token),
		done:     make(chan struct{}),
	}
	go func() {
		s.serveErr = app.Listener(ln)
		close(s.done)
	}()
	// 憑證只放在這一行與 EntryURL，不要再寫進其他 log 或錯誤訊息。
	slog.Info("首次使用，請在這台電腦開啟此網址完成設定：" + s.EntryURL())
	return s, nil
}
