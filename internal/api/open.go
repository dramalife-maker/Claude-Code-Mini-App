package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

// OpenHandler 提供「在伺服器主機開啟 VSCode／檔案總管」，與 shell.enabled 共用同一開關
// （語義相同：允許伺服器端啟動本機程序），指令與參數皆固定，不接受使用者輸入拼接。
type OpenHandler struct {
	db      *db.DB
	enabled bool
}

func NewOpenHandler(database *db.DB, shellEnabled bool) *OpenHandler {
	return &OpenHandler{db: database, enabled: shellEnabled}
}

func jsonErr(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// resolveWorkDir 取得 session 的 work_dir，並確認功能已啟用、目錄存在。
// 失敗時回傳 HTTP 狀態與錯誤訊息（status != 0）。
func (h *OpenHandler) resolveWorkDir(sessionID string) (string, int, string) {
	if !h.enabled {
		return "", 403, "shell.enabled 未開啟，無法使用此功能"
	}
	sess, err := h.db.GetSession(sessionID)
	if err != nil {
		return "", 404, "session 不存在"
	}
	raw := strings.TrimSpace(sess.WorkDir)
	if raw == "" {
		return "", 400, "此 session 未設定 work_dir"
	}
	abs, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", 400, "work_dir 無效"
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", 400, "work_dir 不存在或不是目錄"
	}
	return abs, 0, ""
}

func startDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func folderOpenCommand(workDir string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", workDir)
	case "darwin":
		return exec.Command("open", workDir)
	default:
		return exec.Command("xdg-open", workDir)
	}
}

// OpenVSCode POST /sessions/:id/open-vscode — 在伺服器主機以 `code <work_dir>` 開啟 VSCode。
func (h *OpenHandler) OpenVSCode(c *fiber.Ctx) error {
	workDir, status, msg := h.resolveWorkDir(c.Params("id"))
	if status != 0 {
		return jsonErr(c, status, msg)
	}
	if err := startDetached(exec.Command("code", workDir)); err != nil {
		return jsonErr(c, 500, "啟動 code 失敗（需在伺服器 PATH 內安裝 VSCode CLI）: "+err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// OpenFolder POST /sessions/:id/open-folder — 在伺服器主機開啟 work_dir 所在的檔案總管。
func (h *OpenHandler) OpenFolder(c *fiber.Ctx) error {
	workDir, status, msg := h.resolveWorkDir(c.Params("id"))
	if status != 0 {
		return jsonErr(c, status, msg)
	}
	if err := startDetached(folderOpenCommand(workDir)); err != nil {
		return jsonErr(c, 500, "開啟檔案總管失敗: "+err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}
