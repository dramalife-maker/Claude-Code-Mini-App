//go:build e2e

package desktopupdate

import (
	"errors"
	"os"
	"strings"
)

// 只在 e2e 建置（-tags "production e2e"）編入，供本機 fixture 端到端測試使用。
// 正式與一般建置不含此檔，因此 EXE 中不會有這些環境變數名稱（scripts/e2e 會檢查）。
const (
	envE2EAPIBase     = "CMA_E2E_API_BASE"
	envE2EFailRestart = "CMA_E2E_FAIL_RESTART"
)

// defaultAPIBase 允許以 CMA_E2E_API_BASE 指向本機 fixture；只接受 127.0.0.1，其餘仍用正式來源。
func defaultAPIBase() string {
	if v := os.Getenv(envE2EAPIBase); strings.HasPrefix(v, "http://127.0.0.1:") {
		return v
	}
	return apiBaseURL
}

// e2eRestartFault 在 CMA_E2E_FAIL_RESTART=1 時於 manager 層注入一次「helper 啟動失敗」，
// 不真的啟動 helper；用來驗證閘門釋放、ready 保留與錯誤碼（不驗證真實 helper 程序失敗）。
func e2eRestartFault() error {
	if os.Getenv(envE2EFailRestart) == "1" {
		return errors.New("e2e: injected helper start failure")
	}
	return nil
}
