package desktopupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// CleanOldAsides 清掉執行檔旁邊殘留的「<exe>.old.<時間戳>」舊檔，只保留最新的 keep 份。
//
// 這些檔案是 Wails 更新時把「執行中的舊 exe」改名放到一旁（Windows 不能覆蓋或刪除執行中的 exe）。
// Wails 在下一次更新時也會掃掉較舊的；這裡是在啟動時再做一次，處理它沒清乾淨（例如當時仍被鎖住）的殘留。
// 最新的那份保留，因為它是上一個版本的完整 exe，可當作手動還原用。
//
// 刪除規則刻意收得很窄：檔名必須嚴格等於 "<目前 exe 的檔名>.old.<純數字>"，而且是一般檔案
// （資料夾與符號連結一律不碰）；keep 小於 1 時視為 1，永遠不會全刪。單一檔案刪不掉（例如仍被開啟）
// 不會中斷其餘的清理，錯誤會彙整後回傳。回傳實際刪除的完整路徑。
func CleanOldAsides(exePath string, keep int) (removed []string, err error) {
	if keep < 1 {
		keep = 1
	}
	dir := filepath.Dir(exePath)
	re, err := regexp.Compile(`^` + regexp.QuoteMeta(filepath.Base(exePath)) + `\.old\.([0-9]+)$`)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("讀取安裝目錄失敗: %w", err)
	}

	type aside struct {
		path  string
		stamp uint64
	}
	var asides []aside
	for _, e := range entries {
		m := re.FindStringSubmatch(e.Name())
		if m == nil || !e.Type().IsRegular() {
			continue
		}
		stamp, perr := strconv.ParseUint(m[1], 10, 64)
		if perr != nil { // 超過 uint64 的數字不可能是 UnixNano，視為不是我們要處理的檔案
			continue
		}
		asides = append(asides, aside{filepath.Join(dir, e.Name()), stamp})
	}
	if len(asides) <= keep {
		return nil, nil
	}
	sort.Slice(asides, func(i, j int) bool { return asides[i].stamp > asides[j].stamp }) // 新的在前（數值比較）

	var errs []error
	for _, a := range asides[keep:] {
		if rerr := os.Remove(a.path); rerr != nil {
			errs = append(errs, rerr)
			continue
		}
		removed = append(removed, a.path)
	}
	return removed, errors.Join(errs...)
}
