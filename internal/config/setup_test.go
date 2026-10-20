package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestValidateSetup(t *testing.T) {
	dir := t.TempDir()
	base := func() SetupInput {
		return SetupInput{
			Password: "correct-horse",
			Port:     8080,
			DBPath:   filepath.Join(dir, "app.db"),
		}
	}
	tests := []struct {
		name  string
		in    SetupInput
		field string
	}{
		{"change-me", func() SetupInput { in := base(); in.Password = "change-me"; return in }(), "password"},
		{"7 字元密碼", func() SetupInput { in := base(); in.Password = "abcdefg"; return in }(), "password"},
		{"埠超出範圍", func() SetupInput { in := base(); in.Port = 0; return in }(), "port"},
		{"資料庫目錄不存在", func() SetupInput {
			in := base()
			in.DBPath = filepath.Join(dir, "missing", "app.db")
			return in
		}(), "db_path"},
		{"資料庫路徑是既有資料夾", func() SetupInput { in := base(); in.DBPath = dir; return in }(), "db_path"},
		{"不存在的檔名在可寫目錄", func() SetupInput {
			in := base()
			in.DBPath = filepath.Join(dir, "new.db")
			return in
		}(), ""},
		{"資料庫路徑以斜線結尾", func() SetupInput {
			in := base()
			in.DBPath = filepath.Join(dir, "app.db") + string(filepath.Separator)
			return in
		}(), "db_path"},
		{"Mini App token 為空", func() SetupInput {
			in := base()
			in.MiniApp.Enabled = true
			in.MiniApp.BotToken = "  "
			return in
		}(), "miniapp.bot_token"},
		{"白名單含 0", func() SetupInput {
			in := base()
			in.MiniApp = SetupMiniApp{Enabled: true, BotToken: "123:ABC", WhitelistTgIDs: []int64{0}}
			return in
		}(), "miniapp.whitelist_tg_ids"},
		{"白名單含負數", func() SetupInput {
			in := base()
			in.MiniApp = SetupMiniApp{Enabled: true, BotToken: "123:ABC", WhitelistTgIDs: []int64{-3}}
			return in
		}(), "miniapp.whitelist_tg_ids"},
		{"合法", base(), ""},
		{"Mini App 關閉時不檢查 token 與白名單", func() SetupInput {
			in := base()
			in.MiniApp = SetupMiniApp{Enabled: false, BotToken: "", WhitelistTgIDs: []int64{0, -1}}
			return in
		}(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSetup(tt.in)
			if tt.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || err.Field != tt.field {
				t.Fatalf("err = %#v，預期欄位 %s", err, tt.field)
			}
			if tt.name == "資料庫路徑是既有資料夾" || tt.name == "資料庫路徑以斜線結尾" {
				if err.Message != "資料庫路徑是資料夾，請指定檔名" {
					t.Fatalf("message = %q", err.Message)
				}
			}
		})
	}
}

func TestRenderConfigRoundTrip(t *testing.T) {
	const special = "p@ss: x #y\\ 中文 "
	dir := t.TempDir()
	cases := []struct {
		name string
		in   SetupInput
		lan  bool
		mini bool
	}{
		{"僅本機且關閉 Mini App", SetupInput{
			Password: special, Port: 8090, DBPath: filepath.Join(dir, "a.db"), AllowLAN: false,
		}, false, false},
		{"允許區網且開啟 Mini App", SetupInput{
			Password: "correct-horse", Port: 8081, DBPath: filepath.Join(dir, "b.db"), AllowLAN: true,
			MiniApp: SetupMiniApp{Enabled: true, BotToken: "999:TOKEN", WhitelistTgIDs: []int64{7, 8}},
		}, true, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := RenderConfig(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if !tt.mini {
				if bytes.Contains(raw, []byte("bot_token")) || bytes.Contains(raw, []byte("whitelist_tg_ids")) {
					t.Fatalf("關閉 Mini App 仍寫出 token 或白名單:\n%s", raw)
				}
			}
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Chdir(t.TempDir())
			if err := os.WriteFile("config.yaml", raw, 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v\n%s", err, raw)
			}
			if cfg.Web.Password != tt.in.Password {
				t.Fatalf("password = %q，預期 %q", cfg.Web.Password, tt.in.Password)
			}
			if cfg.Server.Port != tt.in.Port || cfg.DB.Path != tt.in.DBPath || cfg.NoAuth {
				t.Fatalf("載入結果不符: port=%d db=%q no_auth=%v", cfg.Server.Port, cfg.DB.Path, cfg.NoAuth)
			}
			wantCIDR := []string{"127.0.0.0/8", "::1/128"}
			if tt.lan {
				wantCIDR = append(wantCIDR, "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16")
			}
			if !reflect.DeepEqual(cfg.Web.AllowedCIDRs, wantCIDR) {
				t.Fatalf("allowed_cidrs = %v，預期 %v", cfg.Web.AllowedCIDRs, wantCIDR)
			}
			if tt.mini {
				if cfg.BotToken != tt.in.MiniApp.BotToken || !reflect.DeepEqual(cfg.WhitelistTgIDs, tt.in.MiniApp.WhitelistTgIDs) {
					t.Fatalf("mini app = token %q ids %v", cfg.BotToken, cfg.WhitelistTgIDs)
				}
			} else if cfg.BotToken != "" || len(cfg.WhitelistTgIDs) != 0 {
				t.Fatalf("關閉 Mini App 仍載入 token %q ids %v", cfg.BotToken, cfg.WhitelistTgIDs)
			}
		})
	}
}
