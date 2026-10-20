//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// firstRunWorkDir 決定桌面版要不要把工作目錄切到 exe 旁邊。
// 回傳空字串表示維持目前目錄。
// 已安裝的 exe 旁邊有 config.yaml 時沿用舊行為；發布包只有 config.example.yaml、
// 而且捷徑的工作目錄還沒有設定檔時，也切過去，避免第一次把 config.yaml 寫到別處。
// go run 的暫存 exe 旁邊沒有這兩個檔，就不會切。
func firstRunWorkDir(exeDir, cwd string, exists func(string) bool) string {
	if exists(filepath.Join(exeDir, "config.yaml")) {
		return exeDir
	}
	if !exists(filepath.Join(cwd, "config.yaml")) && exists(filepath.Join(exeDir, "config.example.yaml")) {
		return exeDir
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// windowURL 給視窗開啟。正常模式的 EntryURL 沒有結尾斜線，要補上；
// 設定模式已經是 /?setup=憑證，再補斜線會把憑證弄壞。
func windowURL(entry string) string {
	if strings.Contains(entry, "?") {
		return entry
	}
	if strings.HasSuffix(entry, "/") {
		return entry
	}
	return entry + "/"
}
