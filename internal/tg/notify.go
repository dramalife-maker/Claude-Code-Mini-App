package tg

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Notify 透過 Bot API 傳送純文字訊息給指定使用者
func Notify(botToken string, chatID int64, text string) error {
	if botToken == "" || chatID == 0 {
		return nil
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"Markdown"},
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
