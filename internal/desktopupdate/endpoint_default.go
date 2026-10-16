//go:build !e2e

package desktopupdate

// defaultAPIBase 是一般與正式建置唯一的 GitHub API 來源，不接受任何外部覆寫。
// 本檔不含任何環境變數讀取；e2e 覆寫只在 endpoint_e2e.go（-tags e2e）中。
func defaultAPIBase() string { return apiBaseURL }

// e2eRestartFault 在一般建置永遠回傳 nil（不注入失敗）。
func e2eRestartFault() error { return nil }
