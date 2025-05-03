package watchlist

import "github.com/khorzhenwin/gold-digger/internal/models"

type Service struct {
	store Storage
}

func NewService(store Storage) *Service {
	return &Service{store: store}
}

func (s *Service) FindAll() ([]models.Ticker, error) {
	tickers, err := s.store.GetAll()
	if err != nil {
		return nil, err
	}
	return tickers, nil
}

func (s *Service) CreateTicker(ticker *models.Ticker) error {
	err := s.store.Create(ticker)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateTicker(id uint, updated models.Ticker) error {
	return s.store.Update(id, updated)
}

func (s *Service) DeleteTicker(id uint) error {
	return s.store.Delete(id)
}
