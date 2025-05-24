package ticker_price

import (
	"github.com/khorzhenwin/gold-digger/internal/models"
	"gorm.io/gorm"
	"time"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(price models.TickerPrice) error {
	return r.db.Create(&price).Error
}

func (r *Repository) GetLatest(symbol string, limit int) ([]models.TickerPrice, error) {
	var prices []models.TickerPrice
	err := r.db.Where("symbol = ?", symbol).
		Order("timestamp DESC").
		Limit(limit).
		Find(&prices).Error
	return prices, err
}

func (r *Repository) GetSince(symbol string, since time.Time) ([]models.TickerPrice, error) {
	var prices []models.TickerPrice
	err := r.db.Where("symbol = ? AND timestamp >= ?", symbol, since).
		Order("timestamp ASC").
		Find(&prices).Error
	return prices, err
}
