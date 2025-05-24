package ticker_price

import (
	"encoding/json"
	"fmt"
	"github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"github.com/khorzhenwin/gold-digger/internal/notification"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	watchlistService      watchlist.Service
	vantageConfig         config.VantageConfig
	tickerPriceRepository *Repository
	selectedApiKeyIndex   *int
}

func NewService(watchlistService *watchlist.Service, vantageConfig *config.VantageConfig, tickerPriceRepository *Repository) *Service {
	return &Service{watchlistService: *watchlistService, vantageConfig: *vantageConfig, tickerPriceRepository: tickerPriceRepository, selectedApiKeyIndex: new(int)}
}

func (s *Service) FindBySymbol(symbol string) *models.TickerPrice {
	apiKey := s.vantageConfig.ApiKey
	if s.selectedApiKeyIndex != nil && *s.selectedApiKeyIndex > 0 {
		apiKey = s.vantageConfig.ApiKeyBackups[*s.selectedApiKeyIndex-1]
	}

	vantageApiUrl := s.vantageConfig.GetGlobalQuoteUrl(symbol, apiKey)
	tickerPrice, _ := fetchPrice(vantageApiUrl, symbol)
	return tickerPrice
}

func (s *Service) getTickersFromWatchlist() ([]string, error) {
	tickers, err := s.watchlistService.FindAll()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, t := range tickers {
		symbols = append(symbols, t.Symbol)
	}

	return symbols, nil
}

func fetchPrice(externalApiUrl string, symbol string) (*models.TickerPrice, error) {
	resp, err := http.Get(externalApiUrl)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("❌ failed to read response: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Printf("❌ Failed to unmarshal response: %v", err)
		log.Printf("🔎 Raw response: %s", string(body))
		return nil, fmt.Errorf("❌ failed to decode JSON: %w", err)
	}

	// Handle known error formats from Alpha Vantage
	if note, ok := raw["Note"]; ok {
		log.Printf("⚠️ Alpha Vantage Note: %v", note)
		return nil, fmt.Errorf("rate limited or API error: %v", note)
	}
	if errMsg, ok := raw["Error Message"]; ok {
		log.Printf("⚠️ Alpha Vantage Error: %v", errMsg)
		return nil, fmt.Errorf("api error: %v", errMsg)
	}

	// Convert the nested quote safely
	globalQuote, ok := raw["Global Quote"].(map[string]interface{})
	if !ok || len(globalQuote) == 0 {
		log.Printf("⚠️ Missing or invalid Global Quote: %v", raw)
		return nil, fmt.Errorf("missing or invalid Global Quote")
	}

	price, _ := globalQuote["05. price"].(string)
	timestamp, _ := globalQuote["07. latest trading day"].(string)

	if price == "" || timestamp == "" {
		return nil, fmt.Errorf("empty price or timestamp, skipping symbol %s", symbol)
	}
	priceFloat, err := strconv.ParseFloat(price, 64)
	parsedTimestamp, err := time.Parse("2006-01-02", timestamp)

	if err != nil {
		log.Printf("❌ Failed to parse price or timestamp for %s: %v", symbol, err)
		return nil, fmt.Errorf("failed to parse price or timestamp: %w", err)
	}

	return &models.TickerPrice{
		Symbol:    symbol,
		Price:     priceFloat,
		Timestamp: parsedTimestamp,
	}, nil
}

func pollPrices(tickerService *Service, symbols []string, results chan<- models.TickerPrice) {
	for _, symbol := range symbols {
		go func(s string) {
			apiKey := tickerService.vantageConfig.ApiKey
			if tickerService.selectedApiKeyIndex != nil && *tickerService.selectedApiKeyIndex > 0 {
				apiKey = tickerService.vantageConfig.ApiKeyBackups[*tickerService.selectedApiKeyIndex-1]
			}

			vantageApiUrl := tickerService.vantageConfig.GetGlobalQuoteUrl(symbol, apiKey)
			resp, err := fetchPrice(vantageApiUrl, s)
			log.Printf("Raw response : " + fmt.Sprintf("%+v", resp))

			if err != nil {
				log.Printf("❌ Error fetching %s: %v", s, err)

				// if err contains substring of "API rate limit" switch to the next API key
				if strings.Contains(err.Error(), "rate limit") {
					*tickerService.selectedApiKeyIndex++
					if *tickerService.selectedApiKeyIndex > len(tickerService.vantageConfig.ApiKeyBackups) {
						*tickerService.selectedApiKeyIndex = 0
					}
					return
				}
			}
			results <- *resp
		}(symbol)
	}
}

func (s *Service) PollAndPersist() {
	// poll every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	results := make(chan models.TickerPrice)

	log.Println("📈 Ticker-price fetcher started")

	// Start first run immediately
	tickerList, _ := s.getTickersFromWatchlist()
	go pollPrices(s, tickerList, results)

	for {
		select {
		case res := <-results:
			bytes, _ := json.Marshal(res)
			log.Printf("✅ Price: %s", bytes)

			// save to TSDB
			err := s.tickerPriceRepository.Save(models.TickerPrice{
				Symbol:    res.Symbol,
				Price:     res.Price,
				Timestamp: res.Timestamp,
			})

			if err != nil {
				return
			}

		case <-ticker.C:
			if IsTradingHours(time.Now()) || os.Getenv("FORCE_POLL") == "true" {
				log.Println("🔄 Polling watchlist...")
				go pollPrices(s, tickerList, results)
			}
		}
	}
}

// StartSignalWorker Refer to ADR-001
func StartSignalWorker(input <-chan models.TickerPrice, notificationService *notification.Service) {
	type PriceEntry struct {
		Timestamp time.Time
		Price     float64
	}

	var (
		priceWindows = make(map[string][]PriceEntry)
		lastSignal   = make(map[string]time.Time)
		mu           sync.Mutex
	)

	evaluateSignal := func(symbol string, window []PriceEntry) {
		now := time.Now()
		if len(window) < 5 {
			return
		}

		oldest := window[0]
		latest := window[len(window)-1]
		totalChange := (latest.Price - oldest.Price) / oldest.Price

		increaseCount, decreaseCount := 0, 0
		for i := 1; i < len(window); i++ {
			if window[i].Price > window[i-1].Price {
				increaseCount++
			} else if window[i].Price < window[i-1].Price {
				decreaseCount++
			}
		}

		if last, ok := lastSignal[symbol]; ok && now.Sub(last) < time.Hour {
			return
		}

		var message string
		if totalChange >= 0.02 && increaseCount >= 4 {
			message = fmt.Sprintf("🚀 BUY SIGNAL for %s - Strong uptrend (%.2f%% increase)", symbol, totalChange*100)
		} else if totalChange <= -0.02 && decreaseCount >= 4 {
			message = fmt.Sprintf("🔻 BOGDANOFF HAS DOUMP IT. BUY THE DIP for %s - Strong downtrend (%.2f%% decrease)", symbol, totalChange*100)
		} else {
			return
		}

		log.Println(message)
		if err := notificationService.Send(message); err != nil {
			log.Printf("⚠️ Failed to send notification: %v", err)
			return
		}
		lastSignal[symbol] = now
	}

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case msg := <-input:
				entry := PriceEntry{Timestamp: msg.Timestamp, Price: msg.Price}

				mu.Lock()
				priceWindows[msg.Symbol] = append(priceWindows[msg.Symbol], entry)
				if len(priceWindows[msg.Symbol]) > 10 {
					priceWindows[msg.Symbol] = priceWindows[msg.Symbol][len(priceWindows[msg.Symbol])-10:]
				}
				mu.Unlock()

			case <-ticker.C:
				mu.Lock()
				for symbol, window := range priceWindows {
					evaluateSignal(symbol, window)
				}
				mu.Unlock()
			}
		}
	}()
}
