package grpcapi

import (
	"context"
	"errors"
	"strings"

	golddiggerv1 "github.com/khorzhenwin/gold-digger/gen/proto/golddigger/v1"
	"github.com/khorzhenwin/gold-digger/internal/models"
	ticker_price "github.com/khorzhenwin/gold-digger/internal/ticker-price"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type HealthServer struct {
	golddiggerv1.UnimplementedHealthServiceServer
}

func (s *HealthServer) GetHealth(context.Context, *golddiggerv1.GetHealthRequest) (*golddiggerv1.GetHealthResponse, error) {
	return &golddiggerv1.GetHealthResponse{
		Health: &golddiggerv1.HealthResponse{
			Status:  "OK",
			Message: "The server is up",
		},
	}, nil
}

type TickerPriceServer struct {
	golddiggerv1.UnimplementedTickerPriceServiceServer
	service *ticker_price.Service
}

func NewTickerPriceServer(service *ticker_price.Service) *TickerPriceServer {
	return &TickerPriceServer{service: service}
}

func (s *TickerPriceServer) GetTickerPrice(_ context.Context, req *golddiggerv1.GetTickerPriceRequest) (*golddiggerv1.TickerPrice, error) {
	if strings.TrimSpace(req.GetTicker()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker is required")
	}

	tickerPrice := s.service.FindBySymbol(req.GetTicker())
	if tickerPrice == nil {
		return nil, status.Error(codes.NotFound, "ticker not found")
	}

	return &golddiggerv1.TickerPrice{
		Symbol:    tickerPrice.Symbol,
		Price:     tickerPrice.Price,
		Timestamp: timestamppb.New(tickerPrice.Timestamp),
	}, nil
}

type WatchlistServer struct {
	golddiggerv1.UnimplementedWatchlistServiceServer
	service *watchlist.Service
}

func NewWatchlistServer(service *watchlist.Service) *WatchlistServer {
	return &WatchlistServer{service: service}
}

func (s *WatchlistServer) ListWatchlist(context.Context, *golddiggerv1.ListWatchlistRequest) (*golddiggerv1.ListWatchlistResponse, error) {
	tickers, err := s.service.FindAll()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve tickers")
	}

	items := make([]*golddiggerv1.WatchlistItem, 0, len(tickers))
	for _, t := range tickers {
		items = append(items, mapTickerToProto(t))
	}

	return &golddiggerv1.ListWatchlistResponse{Items: items}, nil
}

func (s *WatchlistServer) CreateWatchlistItem(_ context.Context, req *golddiggerv1.CreateWatchlistItemRequest) (*golddiggerv1.OperationStatus, error) {
	if req.GetTicker() == nil || strings.TrimSpace(req.GetTicker().GetSymbol()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker.symbol is required")
	}

	ticker := models.Ticker{
		Symbol: req.GetTicker().GetSymbol(),
		Notes:  req.GetTicker().GetNotes(),
	}

	if err := s.service.CreateTicker(&ticker); err != nil {
		return nil, status.Error(codes.Internal, "failed to create ticker")
	}

	return &golddiggerv1.OperationStatus{Message: "created"}, nil
}

func (s *WatchlistServer) UpdateWatchlistItem(_ context.Context, req *golddiggerv1.UpdateWatchlistItemRequest) (*golddiggerv1.OperationStatus, error) {
	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.GetTicker() == nil || strings.TrimSpace(req.GetTicker().GetSymbol()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker.symbol is required")
	}

	err := s.service.UpdateTicker(uint(req.GetId()), models.Ticker{
		Symbol: req.GetTicker().GetSymbol(),
		Notes:  req.GetTicker().GetNotes(),
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "ticker not found")
		}
		return nil, status.Error(codes.Internal, "failed to update ticker")
	}

	return &golddiggerv1.OperationStatus{Message: "updated"}, nil
}

func (s *WatchlistServer) DeleteWatchlistItem(_ context.Context, req *golddiggerv1.DeleteWatchlistItemRequest) (*emptypb.Empty, error) {
	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.service.DeleteTicker(uint(req.GetId())); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "no record found") {
			return nil, status.Error(codes.NotFound, "ticker not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete ticker")
	}

	return &emptypb.Empty{}, nil
}

func mapTickerToProto(t models.Ticker) *golddiggerv1.WatchlistItem {
	return &golddiggerv1.WatchlistItem{
		Id:        uint64(t.ID),
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
		Symbol:    t.Symbol,
		Notes:     t.Notes,
	}
}
