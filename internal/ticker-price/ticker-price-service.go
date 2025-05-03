package ticker_price

import (
	"encoding/json"
	"fmt"
	"github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/khorzhenwin/gold-digger/internal/kafka"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"github.com/khorzhenwin/gold-digger/internal/notification"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Service struct {
	watchlistService watchlist.Service
	vantageConfig    config.VantageConfig
	kafkaConfig      config.KafkaConfig
}

func NewService(watchlistService *watchlist.Service, vantageConfig *config.VantageConfig, kafkaConfig *config.KafkaConfig) *Service {
	return &Service{watchlistService: *watchlistService, vantageConfig: *vantageConfig, kafkaConfig: *kafkaConfig}
}

func (s *Service) FindBySymbol(symbol string) *models.TickerPrice {
	vantageApiUrl := s.vantageConfig.GetGlobalQuoteUrl(symbol)
	tickerPrice, _ := fetchPrice(vantageApiUrl, symbol)
	return tickerPrice
}

func getTickersFromWatchlist(watchlistService *watchlist.Service) ([]string, error) {
	tickers, err := watchlistService.FindAll()
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

	return &models.TickerPrice{
		Symbol:    symbol,
		Price:     price,
		Timestamp: timestamp + "T00:00:00Z", // Add time if needed
	}, nil
}

func pollPrices(tickerService *Service, symbols []string, results chan<- models.TickerPrice) {
	for _, symbol := range symbols {
		go func(s string) {
			vantageApiUrl := tickerService.vantageConfig.GetGlobalQuoteUrl(symbol)
			resp, err := fetchPrice(vantageApiUrl, s)
			log.Printf("Raw response : " + fmt.Sprintf("%+v", resp))
			if err != nil {
				log.Printf("❌ Error fetching %s: %v", s, err)
				return
			}
			results <- *resp
		}(symbol)
	}
}

func PollAndPushToKafka(tickerService *Service, watchlistService *watchlist.Service, kafkaConfig *config.KafkaConfig) {
	// poll every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	results := make(chan models.TickerPrice)

	log.Println("📈 Ticker-price fetcher started")

	// Start first run immediately
	tickerList, _ := getTickersFromWatchlist(watchlistService)
	go pollPrices(tickerService, tickerList, results)

	for {
		select {
		case res := <-results:
			bytes, _ := json.Marshal(res)
			log.Printf("✅ Price: %s", bytes)

			go kafka.PushToKafkaTopic(kafkaConfig.TickerPriceTopic, res, res.Symbol)

		case <-ticker.C:
			if IsTradingHours(time.Now()) || os.Getenv("FORCE_POLL") == "true" {
				log.Println("🔄 Polling watchlist...")
				go pollPrices(tickerService, tickerList, results)
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

	priceWindows := make(map[string][]PriceEntry)
	lastSignal := make(map[string]time.Time)
	var mu sync.Mutex

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case msg := <-input:
				parsedTime, err := time.Parse(time.RFC3339, msg.Timestamp)
				if err != nil {
					log.Printf("⚠️ Invalid timestamp: %v", msg.Timestamp)
					continue
				}
				price, err := strconv.ParseFloat(msg.Price, 64)
				if err != nil {
					log.Printf("⚠️ Invalid price: %v", msg.Price)
					continue
				}
				entry := PriceEntry{Timestamp: parsedTime, Price: price}

				mu.Lock()
				priceWindows[msg.Symbol] = append(priceWindows[msg.Symbol], entry)

				// Keep only the latest 10 observations
				if len(priceWindows[msg.Symbol]) > 10 {
					priceWindows[msg.Symbol] = priceWindows[msg.Symbol][len(priceWindows[msg.Symbol])-10:]
				}
				mu.Unlock()

			case <-ticker.C:
				mu.Lock()
				now := time.Now()

				for symbol, window := range priceWindows {
					if len(window) < 5 {
						continue
					}

					oldest := window[0]
					latest := window[len(window)-1]

					// % total movement
					totalChange := (latest.Price - oldest.Price) / oldest.Price

					// Count rises and falls
					increaseCount := 0
					decreaseCount := 0
					for i := 1; i < len(window); i++ {
						if window[i].Price > window[i-1].Price {
							increaseCount++
						} else if window[i].Price < window[i-1].Price {
							decreaseCount++
						}
					}

					// Cooldown: skip if recently signaled
					if last, ok := lastSignal[symbol]; ok && now.Sub(last) < time.Hour {
						continue
					}

					// 🚀 Uptrend Buy Signal
					if totalChange >= 0.02 && increaseCount >= 4 {
						message := fmt.Sprintf("🚀 BUY SIGNAL for %s - Strong uptrend (%.2f%% increase)", symbol, totalChange*100)
						log.Printf(message)
						err := notificationService.Send(message)

						if err != nil {
							return
						}
						lastSignal[symbol] = now
						continue
					}

					// 🔻 Downtrend Buy-the-Dip Signal
					if totalChange <= -0.02 && decreaseCount >= 4 {
						message := fmt.Sprintf("🔻 BOGDANOFF HAS DOUMP IT. BUY THE DIP for %s - Strong downtrend (%.2f%% decrease)", symbol, totalChange*100)
						log.Printf(message)
						err := notificationService.Send(message)

						if err != nil {
							return
						}

						lastSignal[symbol] = now
						continue
					}
				}
				mu.Unlock()
			}
		}
	}()
}
