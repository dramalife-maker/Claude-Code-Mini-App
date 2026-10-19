package desktopupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// CleanOldAsides 清掉執行檔旁邊殘留的「<exe>.old.<時間戳>」舊檔，一個都不留。
//
// 這些檔案是 Wails 更新時把「執行中的舊 exe」改名放到一旁（Windows 不能覆蓋或刪除執行中的 exe），
// 它當下通常刪不掉（舊程序剛結束、映射還沒釋放），要到下一次更新才會掃掉較舊的。
// 這裡在啟動時再清一次：此時舊程序已經結束，通常刪得掉。因為「一個都不留」，更新後就沒有上一版的
// 自動還原檔了；要能回退請在更新前自己備份目前的 exe（見 README）。
//
// 刪除規則刻意收得很窄：檔名必須嚴格等於 "<目前 exe 的檔名>.old.<純數字>"，數字要能放進 uint64
// （UnixNano 的範圍），而且是一般檔案（資料夾與符號連結一律不碰）；目前的 exe 本身、<exe>.bak
// 與其他任何檔案都不會吻合。單一檔案刪不掉（例如仍被開啟）不會中斷其餘的清理，錯誤彙整後回傳，
// 下次啟動會再試。回傳實際刪除的完整路徑。
func CleanOldAsides(exePath string) (removed []string, err error) {
	dir := filepath.Dir(exePath)
	re, err := regexp.Compile(`^` + regexp.QuoteMeta(filepath.Base(exePath)) + `\.old\.([0-9]+)$`)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("讀取安裝目錄失敗: %w", err)
	}

	var errs []error
	for _, e := range entries {
		m := re.FindStringSubmatch(e.Name())
		if m == nil || !e.Type().IsRegular() {
			continue
		}
		if _, perr := strconv.ParseUint(m[1], 10, 64); perr != nil { // 超過 uint64 的數字不可能是 UnixNano
			continue
		}
		p := filepath.Join(dir, e.Name())
		if rerr := os.Remove(p); rerr != nil {
			errs = append(errs, rerr)
			continue
		}
		removed = append(removed, p)
	}
	return removed, errors.Join(errs...)
}
