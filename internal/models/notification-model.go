package models

type Notifier interface {
	Send(message string) error
}

type TelegramNotifier struct {
	BotToken string
	ChatID   string
}
