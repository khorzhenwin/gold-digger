package grpcapi

import (
	golddiggerv1 "github.com/khorzhenwin/gold-digger/gen/proto/golddigger/v1"
	ticker_price "github.com/khorzhenwin/gold-digger/internal/ticker-price"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	"google.golang.org/grpc"
)

func NewServer(watchlistService *watchlist.Service, tickerPriceService *ticker_price.Service) *grpc.Server {
	server := grpc.NewServer()

	golddiggerv1.RegisterHealthServiceServer(server, &HealthServer{})
	golddiggerv1.RegisterTickerPriceServiceServer(server, NewTickerPriceServer(tickerPriceService))
	golddiggerv1.RegisterWatchlistServiceServer(server, NewWatchlistServer(watchlistService))

	return server
}
