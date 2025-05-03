package ticker_price

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Handler struct {
	Service Service
}

func RegisterRoutes(r chi.Router, service *Service) {
	h := &Handler{Service: *service}

	r.Route("/ticker-price", func(r chi.Router) {
		r.Get("/{ticker}", h.GetTickerPrice)
	})
}

// GetTickerPrice handles GET /ticker-price/{ticker}
// @Summary      Get price of a ticker
// @Description  Returns the current price of a ticker
// @Tags         ticker-price
// @Produce      json
// @Param        ticker  path  string  true  "Ticker Symbol"
// @Success      200     {object}  TickerPrice
// @Failure      400     {string}  string  "Invalid ticker symbol"
// @Router       /api/v1/ticker-price/{ticker} [get]
func (h *Handler) GetTickerPrice(w http.ResponseWriter, r *http.Request) {
	tickerSymbol := chi.URLParam(r, "ticker")

	if tickerSymbol == "" {
		http.Error(w, "Ticker is required", http.StatusBadRequest)
		return
	}

	tickerPrice := h.Service.FindBySymbol(tickerSymbol)
	if tickerPrice == nil {
		http.Error(w, "Ticker not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(tickerPrice)
	if err != nil {
		return
	}
}
