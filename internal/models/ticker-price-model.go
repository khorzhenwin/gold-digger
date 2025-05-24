package models

import "time"

type TickerPrice struct {
	ID        uint   `gorm:"primaryKey"`
	Symbol    string `gorm:"index"`
	Price     float64
	Timestamp time.Time `gorm:"index"`
}
