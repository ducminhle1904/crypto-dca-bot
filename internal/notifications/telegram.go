package notifications

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type TelegramNotifier struct {
	token  string
	chatID string
}

func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		token:  token,
		chatID: chatID,
	}
}

func (t *TelegramNotifier) SendAlert(level, message string) error {
	emoji := "ℹ️"
	switch level {
	case "warning":
		emoji = "⚠️"
	case "error":
		emoji = "🚨"
	case "success":
		emoji = "✅"
	}

	text := fmt.Sprintf("%s *DCA Bot Alert*\n\n%s", emoji, message)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	data := url.Values{}
	data.Set("chat_id", t.chatID)
	data.Set("text", text)
	data.Set("parse_mode", "Markdown")

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
