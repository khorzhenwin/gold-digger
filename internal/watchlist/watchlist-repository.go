package watchlist

import (
	"errors"
	"github.com/khorzhenwin/gold-digger/internal/models"

	"gorm.io/gorm"
)

type Storage interface {
	Create(ticker *models.Ticker) error
	GetAll() ([]models.Ticker, error)
	GetByID(id uint) (*models.Ticker, error)
	Update(id uint, updated models.Ticker) error
	Delete(id uint) error
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(t *models.Ticker) error {
	return r.db.Create(t).Error
}

func (r *Repository) GetAll() ([]models.Ticker, error) {
	var tickers []models.Ticker
	err := r.db.Find(&tickers).Error
	return tickers, err
}

func (r *Repository) GetByID(id uint) (*models.Ticker, error) {
	var ticker models.Ticker
	err := r.db.First(&ticker, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ticker, err
}

func (r *Repository) Update(id uint, updated models.Ticker) error {
	var existing models.Ticker
	if err := r.db.First(&existing, id).Error; err != nil {
		return err
	}

	existing.Symbol = updated.Symbol
	existing.Notes = updated.Notes
	return r.db.Save(&existing).Error
}

// Delete removes a ticker by ID
func (r *Repository) Delete(id uint) error {
	result := r.db.Delete(&models.Ticker{}, id)
	if result.RowsAffected == 0 {
		return errors.New("no record found to delete")
	}
	return result.Error
}
