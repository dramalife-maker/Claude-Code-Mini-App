package tg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
)

// User 是從 initData 解析出的 Telegram 使用者
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

// Verify 驗證 Telegram WebApp initData 的 HMAC-SHA256 簽名，
// 成功時回傳解析後的 User，失敗時回傳 error。
func Verify(initData, botToken string) (*User, error) {
	if initData == "" {
		return nil, errors.New("initData 為空")
	}

	vals, err := url.ParseQuery(initData)
	if err != nil {
		return nil, errors.New("initData 格式錯誤")
	}

	hash := vals.Get("hash")
	if hash == "" {
		return nil, errors.New("缺少 hash 欄位")
	}
	vals.Del("hash")

	// 構建 data-check-string（按 key 字母排序，換行分隔）
	pairs := make([]string, 0, len(vals))
	for k, v := range vals {
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC_SHA256(bot_token, "WebAppData")
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	// expected_hash = HMAC_SHA256(data_check_string, secret_key)
	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac2.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(hash)) {
		return nil, errors.New("簽名驗證失敗")
	}

	// 解析 user 欄位
	userJSON := vals.Get("user")
	if userJSON == "" {
		return nil, errors.New("缺少 user 欄位")
	}
	var u User
	if err := json.Unmarshal([]byte(userJSON), &u); err != nil {
		return nil, errors.New("user JSON 格式錯誤")
	}
	return &u, nil
}
