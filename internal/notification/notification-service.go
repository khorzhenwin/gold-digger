package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"net/http"
)

type Service struct {
	notificationConfig models.TelegramNotifier
}

func NewService(notificationConfig *models.TelegramNotifier) *Service {
	return &Service{notificationConfig: *notificationConfig}
}

func (s Service) Send(message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.notificationConfig.BotToken)

	payload := map[string]string{
		"chat_id": s.notificationConfig.ChatID,
		"text":    message,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("error: %v", err)
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API responded with status: %d", resp.StatusCode)
	}
	return nil
}
