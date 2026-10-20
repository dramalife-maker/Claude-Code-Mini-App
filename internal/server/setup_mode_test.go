package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/clicheck"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/spf13/viper"
)

func TestStartRouting(t *testing.T) {
	if !setupOnMissing(config.ErrConfigMissing) {
		t.Fatal("缺檔應進入設定模式")
	}
	if !setupOnMissing(fmt.Errorf("讀取: %w", config.ErrConfigMissing)) {
		t.Fatal("包住的缺檔錯誤也應進入設定模式")
	}
	if setupOnMissing(errors.New("yaml: line 1")) || setupOnMissing(errors.New("boom")) {
		t.Fatal("其他錯誤不該進入設定模式")
	}
}

func TestStartBrokenConfigDoesNotEnterSetup(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Chdir(t.TempDir())
	if err := os.WriteFile("config.yaml", []byte("bot_token: [\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load()
	if err == nil || errors.Is(err, config.ErrConfigMissing) || setupOnMissing(err) {
		t.Fatalf("壞 YAML 應是普通啟動失敗，得到 %v", err)
	}
}

func TestSetupModeSurface(t *testing.T) {
	dir := t.TempDir()
	before := dirNames(t, dir)
	t.Chdir(dir)
	port := freePort(t)
	wantTools := []clicheck.Tool{{
		AgentType: "claude", Label: "Claude", Command: "claude", Found: true, InstallURL: "https://example.test/claude",
	}}
	srv, err := startSetup(context.Background(), port, setupOptions{
		check: func(context.Context) []clicheck.Tool { return wantTools },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
		waitErr := make(chan error, 1)
		go func() { waitErr <- srv.Wait() }()
		select {
		case <-waitErr:
		case <-time.After(5 * time.Second):
			t.Error("Shutdown 後 Wait 沒返回")
		}
	})
	base := srv.URL()
	host := fmt.Sprintf("127.0.0.1:%d", port)
	token := setupTokenFromURL(t, srv.EntryURL())

	status, body := setupDo(t, http.MethodGet, base+"/sessions", host, "")
	if status != 503 || jsonCode(t, body) != "setup_required" || !contains(body, "需要先完成首次設定") {
		t.Fatalf("sessions = %d %s", status, body)
	}
	// 設定網址帶憑證，回應不可讓瀏覽器把它當 Referer 外送。
	if resp, err := http.Get(base + "/setup/status"); err != nil {
		t.Fatal(err)
	} else {
		resp.Body.Close()
		if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
			t.Fatalf("Referrer-Policy = %q，預期 no-referrer", got)
		}
	}
	status, body = setupDo(t, http.MethodGet, base+"/setup/status", host, "")
	if status != 200 {
		t.Fatalf("status = %d %s", status, body)
	}
	var st struct {
		NeedsSetup bool   `json:"needs_setup"`
		Port       int    `json:"port"`
		ConfigPath string `json:"config_path"`
	}
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatal(err)
	}
	if !st.NeedsSetup || st.Port != port || st.ConfigPath != filepath.Join(dir, "config.yaml") {
		t.Fatalf("status body = %+v", st)
	}
	status, body = setupDo(t, http.MethodGet, base+"/setup/cli-check", host, "")
	if status != 403 || jsonCode(t, body) != "setup_forbidden" {
		t.Fatalf("無憑證 = %d %s", status, body)
	}
	status, body = setupDo(t, http.MethodGet, base+"/setup/cli-check", host, token)
	if status != 200 {
		t.Fatalf("cli-check = %d %s", status, body)
	}
	var got struct {
		Tools []clicheck.Tool `json:"tools"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Tools, wantTools) {
		t.Fatalf("tools = %+v", got.Tools)
	}
	status, body = setupDo(t, http.MethodGet, base+"/setup/status", fmt.Sprintf("attacker.example:%d", port), "")
	if status != 403 || jsonCode(t, body) != "setup_forbidden" {
		t.Fatalf("偽造 Host = %d %s", status, body)
	}
	status, body = setupDo(t, http.MethodPost, base+"/setup/save", host, token)
	if status != 501 || jsonCode(t, body) != "not_implemented" {
		t.Fatalf("save = %d %s", status, body)
	}
	if after := dirNames(t, dir); !reflect.DeepEqual(before, after) {
		t.Fatalf("工作目錄變了：之前 %v 之後 %v", before, after)
	}
}

func TestNormalModeHidesSetup(t *testing.T) {
	base := startTestServer(t, "no_auth: true\n")
	resp, err := http.Get(base + "/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !contains(body, `"needs_setup":false`) {
		t.Fatalf("status = %d %s", resp.StatusCode, body)
	}
	for _, item := range []struct{ method, path string }{
		{http.MethodGet, "/setup/cli-check"},
		{http.MethodPost, "/setup/save"},
	} {
		req, err := http.NewRequest(item.method, base+item.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("%s %s = %d", item.method, item.path, res.StatusCode)
		}
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func setupDo(t *testing.T, method, rawURL, host, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	if token != "" {
		req.Header.Set("X-Setup-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func setupTokenFromURL(t *testing.T, raw string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	token := req.URL.Query().Get("setup")
	if len(token) != 64 {
		t.Fatalf("憑證長度 %d：%s", len(token), raw)
	}
	return token
}

func jsonCode(t *testing.T, body []byte) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	code, _ := m["code"].(string)
	return code
}

func contains(b []byte, s string) bool {
	return strings.Contains(string(b), s)
}
