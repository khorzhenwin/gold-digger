package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/khorzhenwin/gold-digger/docs" // <-- this is required for Swagger to embed docs
	applicationConfig "github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/khorzhenwin/gold-digger/internal/db"
	"github.com/khorzhenwin/gold-digger/internal/health"
	"github.com/khorzhenwin/gold-digger/internal/kafka"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"github.com/khorzhenwin/gold-digger/internal/notification"
	"github.com/khorzhenwin/gold-digger/internal/ticker-price"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	_ "github.com/swaggo/files"
	"github.com/swaggo/http-swagger"
	"log"
	"net/http"
)

func (app *application) run() error {
	// 1. Load Configs
	_ = godotenv.Load() // Loads from .env file

	dbCfg, dbErr := applicationConfig.LoadDBConfig()
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	vantageCfg, vErr := applicationConfig.LoadVantageConfig()
	if vErr != nil {
		log.Fatal(vErr)
	}

	kafkaCfg, kErr := applicationConfig.LoadKafkaConfig()
	if kErr != nil {
		log.Fatal(kErr)
	}

	notifierCfg, nErr := applicationConfig.LoadNotifierConfig()
	if nErr != nil {
		log.Fatal(nErr)
	}

	// 2. Initialize DB & Kafka
	conn, err := db.New(dbCfg)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Run Migrations & initialize Repository
	if err := conn.AutoMigrate(&models.Ticker{}); err != nil {
		log.Fatalf("❌ AutoMigrate failed: %v", err)
	}

	watchlistRepo := watchlist.NewRepository(conn)
	watchlistService := watchlist.NewService(watchlistRepo)
	notificationService := notification.NewService(notifierCfg)
	tickerPriceService := ticker_price.NewService(watchlistService, vantageCfg, kafkaCfg)

	// 3.1 Initialize Kafka Producer & Start Producer
	kafka.InitKafkaProducer(kafkaCfg)
	defer kafka.CloseKafkaProducer()
	go ticker_price.PollAndPushToKafka(tickerPriceService, watchlistService, kafkaCfg)

	// 3.2 Initialize Kafka Consumer & Start Consumer
	tickerChan := make(chan models.TickerPrice, 100)
	go kafka.StartKafkaConsumer(kafkaCfg, "gochujang-signals-group", tickerChan)
	go ticker_price.StartSignalWorker(tickerChan, notificationService)

	// 4. Setup Router config
	r := chi.NewRouter()
	server := &http.Server{
		Addr:         app.config.ADDRESS,
		Handler:      r,
		WriteTimeout: app.config.writeTimeout,
		ReadTimeout:  app.config.readTimeout,
	}

	// 5. Register all API routes
	r.Route(app.config.BASE_PATH, func(r chi.Router) {
		health.RegisterRoutes(r)
		watchlist.RegisterRoutes(r, watchlistService)
		ticker_price.RegisterRoutes(r, tickerPriceService)
	})

	// 6. Serve Swagger (if generated)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	log.Println("Starting server on", app.config.ADDRESS)
	return server.ListenAndServe()
}
