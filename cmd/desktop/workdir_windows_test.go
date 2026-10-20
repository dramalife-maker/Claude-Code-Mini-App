//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkDirForFirstRun(t *testing.T) {
	exe := filepath.Join("D:", "app")
	cwd := filepath.Join("D:", "shortcut")
	has := func(paths ...string) func(string) bool {
		set := map[string]bool{}
		for _, p := range paths {
			set[p] = true
		}
		return func(p string) bool { return set[p] }
	}
	exeCfg := filepath.Join(exe, "config.yaml")
	exeExample := filepath.Join(exe, "config.example.yaml")
	cwdCfg := filepath.Join(cwd, "config.yaml")

	if got := firstRunWorkDir(exe, cwd, has(exeCfg)); got != exe {
		t.Fatalf("exe 旁已有 config.yaml 應切到 exe，得到 %q", got)
	}
	if got := firstRunWorkDir(exe, cwd, has(exeExample)); got != exe {
		t.Fatalf("只有 config.example.yaml 且目前目錄沒有設定檔應切到 exe，得到 %q", got)
	}
	if got := firstRunWorkDir(exe, cwd, has(cwdCfg, exeExample)); got != "" {
		t.Fatalf("目前目錄已有 config.yaml 不該切換，得到 %q", got)
	}
	if got := firstRunWorkDir(exe, cwd, has()); got != "" {
		t.Fatalf("兩邊都沒有設定檔不該切換，得到 %q", got)
	}
}

func TestWindowURL(t *testing.T) {
	if got := windowURL("http://127.0.0.1:8080"); got != "http://127.0.0.1:8080/" {
		t.Fatalf("正常網址應補斜線，得到 %q", got)
	}
	entry := "http://127.0.0.1:8080/?setup=" + strings.Repeat("ab", 32)
	if got := windowURL(entry); got != entry {
		t.Fatalf("含憑證的網址不該再補斜線，得到 %q", got)
	}
}
