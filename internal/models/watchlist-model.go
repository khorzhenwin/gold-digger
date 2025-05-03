package models

import (
	"time"
)

type Ticker struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Symbol    string    `json:"symbol"` // e.g., AAPL
	Notes     string    `json:"notes,omitempty"`
}
