//go:build !production

package static

import (
	"io/fs"
	"os"
)

// Production 表示資源是否為編譯期嵌入（-tags production）。
const Production = false

// Assets 是前端公開資源來源：開發建置讀工作目錄下的 internal/static，改檔後重整即生效。
// 使用相對路徑，每次開檔時才依當下工作目錄解析（桌面版 chdir 之後仍正確）。
var Assets fs.FS = os.DirFS("internal/static")
