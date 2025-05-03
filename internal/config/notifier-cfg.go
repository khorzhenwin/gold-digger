package config

import (
	"fmt"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"os"
)

func LoadNotifierConfig() (*models.TelegramNotifier, error) {
	cfg := &models.TelegramNotifier{
		BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		ChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
	}

	if cfg.BotToken == "" || cfg.ChatID == "" {
		return nil, fmt.Errorf("incomplete Notifier config")
	}

	return cfg, nil
}
