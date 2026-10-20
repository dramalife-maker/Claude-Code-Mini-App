package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SetupInput 是設定精靈要寫進 config.yaml 的欄位。JSON tag 與 POST /setup/save 的請求相同。
type SetupInput struct {
	Password string       `json:"password"`
	Port     int          `json:"port"`
	DBPath   string       `json:"db_path"`
	AllowLAN bool         `json:"allow_lan"`
	MiniApp  SetupMiniApp `json:"miniapp"`
}

// SetupMiniApp 關閉時不寫 bot_token 與白名單。
type SetupMiniApp struct {
	Enabled        bool    `json:"enabled"`
	BotToken       string  `json:"bot_token"`
	WhitelistTgIDs []int64 `json:"whitelist_tg_ids"`
}

// FieldError 是單一欄位的驗證錯誤，給 API 回 400 時帶 field。
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string {
	if e == nil {
		return ""
	}
	return e.Field + ": " + e.Message
}

// ValidateSetup 檢查精靈輸入。密碼規則與 Load 共用 PasswordValid。回傳第一個錯誤。
func ValidateSetup(in SetupInput) *FieldError {
	if err := PasswordValid(in.Password); err != nil {
		return &FieldError{Field: "password", Message: err.Error()}
	}
	if in.Port < 1 || in.Port > 65535 {
		return &FieldError{Field: "port", Message: "連接埠必須在 1 到 65535"}
	}
	if err := checkDBDir(in.DBPath); err != nil {
		return err
	}
	if !in.MiniApp.Enabled {
		return nil
	}
	if strings.TrimSpace(in.MiniApp.BotToken) == "" {
		return &FieldError{Field: "miniapp.bot_token", Message: "開啟 Mini App 時 bot_token 不可為空"}
	}
	for _, id := range in.MiniApp.WhitelistTgIDs {
		if id <= 0 {
			return &FieldError{Field: "miniapp.whitelist_tg_ids", Message: "白名單必須是正整數"}
		}
	}
	return nil
}

// checkDBDir 確認資料庫路徑是可建立的檔案，不是已存在的資料夾。相對路徑依工作目錄解析。
func checkDBDir(p string) *FieldError {
	notFile := &FieldError{Field: "db_path", Message: "資料庫路徑是資料夾，請指定檔名"}
	fail := &FieldError{Field: "db_path", Message: "資料庫目錄不存在或不可寫"}
	if strings.TrimSpace(p) == "" {
		fail.Message = "資料庫路徑不可為空"
		return fail
	}
	// Clean／Join 會吃掉結尾斜線，必須在解析前先擋下。
	if strings.HasSuffix(p, "/") || strings.HasSuffix(p, `\`) {
		return notFile
	}
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return fail
		}
		p = filepath.Join(wd, p)
	}
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return notFile
	}
	dir := filepath.Dir(p)
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return fail
	}
	// 只試寫一個暫存檔再刪掉，避免把「目錄存在」誤當成可寫。
	f, err := os.CreateTemp(dir, ".setup-dbcheck-*")
	if err != nil {
		return fail
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

type setupWeb struct {
	Password     string   `yaml:"password"`
	AllowedCIDRs []string `yaml:"allowed_cidrs"`
}

type setupServer struct {
	Port int `yaml:"port"`
}

type setupDB struct {
	Path string `yaml:"path"`
}

// RenderConfig 只輸出精靈要寫的欄位。其餘設定沿用程式預設值，所以不出現在檔案裡。
func RenderConfig(in SetupInput) ([]byte, error) {
	cidrs := []string{"127.0.0.0/8", "::1/128"}
	if in.AllowLAN {
		cidrs = append(cidrs, "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16")
	}
	web := setupWeb{Password: in.Password, AllowedCIDRs: cidrs}
	if !in.MiniApp.Enabled {
		body := struct {
			Web    setupWeb    `yaml:"web"`
			Server setupServer `yaml:"server"`
			DB     setupDB     `yaml:"db"`
			NoAuth bool        `yaml:"no_auth"`
		}{
			Web: web, Server: setupServer{Port: in.Port}, DB: setupDB{Path: in.DBPath}, NoAuth: false,
		}
		return yaml.Marshal(body)
	}
	ids := in.MiniApp.WhitelistTgIDs
	if ids == nil {
		ids = []int64{}
	}
	body := struct {
		BotToken       string      `yaml:"bot_token"`
		WhitelistTgIDs []int64     `yaml:"whitelist_tg_ids"`
		Web            setupWeb    `yaml:"web"`
		Server         setupServer `yaml:"server"`
		DB             setupDB     `yaml:"db"`
		NoAuth         bool        `yaml:"no_auth"`
	}{
		BotToken: strings.TrimSpace(in.MiniApp.BotToken), WhitelistTgIDs: ids,
		Web: web, Server: setupServer{Port: in.Port}, DB: setupDB{Path: in.DBPath}, NoAuth: false,
	}
	out, err := yaml.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("產生設定: %w", err)
	}
	return out, nil
}
