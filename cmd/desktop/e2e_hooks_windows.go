//go:build windows && e2e

package main

import (
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 只在 e2e 建置（-tags "production e2e"）編入。讓測試用桌面程序：
//   - CMA_E2E_UNIQUE_ID：換掉單一實例 ID，避免與使用者真實桌面互相轉交／退出
//   - CMA_E2E_WEBVIEW_DATA：換掉 WebView2 使用者資料目錄（預設依 exe 名稱共用）
//   - CMA_E2E_CDP_PORT：開啟 WebView2 遠端除錯埠（--remote-debugging-port）
//   - CMA_E2E_HANG_SHUTDOWN=1：OnShutdown 卡住，驗證 ShutdownDeadline 的強制結束路徑
func e2eApplyOptions(o *application.Options) {
	if v := os.Getenv("CMA_E2E_UNIQUE_ID"); v != "" && o.SingleInstance != nil {
		o.SingleInstance.UniqueID = v
	}
	if v := os.Getenv("CMA_E2E_WEBVIEW_DATA"); v != "" {
		o.Windows.WebviewUserDataPath = v
	}
	if v := os.Getenv("CMA_E2E_CDP_PORT"); v != "" {
		// 開啟 WebView2 遠端除錯埠，供測試驅動以 CDP 點擊實際 UI。
		o.Windows.AdditionalBrowserArgs = append(o.Windows.AdditionalBrowserArgs, "--remote-debugging-port="+v)
	}
}

func e2eShutdownHang() {
	if os.Getenv("CMA_E2E_HANG_SHUTDOWN") == "1" {
		time.Sleep(time.Hour) // 只會被 watchdog 的 os.Exit 結束
	}
}
