package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/khorzhenwin/gold-digger/docs"
	applicationConfig "github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/khorzhenwin/gold-digger/internal/db"
	"github.com/khorzhenwin/gold-digger/internal/grpcapi"
	"github.com/khorzhenwin/gold-digger/internal/health"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"github.com/khorzhenwin/gold-digger/internal/notification"
	"github.com/khorzhenwin/gold-digger/internal/ticker-price"
	"github.com/khorzhenwin/gold-digger/internal/watchlist"
	_ "github.com/swaggo/files"
	"github.com/swaggo/http-swagger"
	"gorm.io/gorm"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func (app *application) run() error {
	// 1. Load Configs
	_ = godotenv.Load() // Loads from .env file

	cloudDbCfg, dbErr := applicationConfig.LoadAWSConfig()
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	localDbCfg, dbErr := applicationConfig.LoadLocalDBConfig()
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	vantageCfg, vErr := applicationConfig.LoadVantageConfig()
	if vErr != nil {
		log.Fatal(vErr)
	}

	notifierCfg, nErr := applicationConfig.LoadNotifierConfig()
	if nErr != nil {
		log.Fatal(nErr)
	}

	// 2. Initialize DB
	cloudConn, err := db.NewAWSClient(cloudDbCfg)
	if err != nil {
		log.Fatal(err)
	}

	maxAttempts := 10
	var localConn *gorm.DB
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		localConn, err = db.NewLocalDbClient(localDbCfg)
		if err == nil {
			log.Println("✅ Connected to TimescaleDB!")
			break
		}
		log.Printf("⏳ Waiting for TimescaleDB... attempt %d/%d", attempts, maxAttempts)
		time.Sleep(5 * time.Second)
	}

	if localConn == nil {
		log.Fatalf("❌ Failed to connect to TimescaleDB after %d attempts: %v", maxAttempts, err)
	}

	// 3. Run Migrations & initialize Repository
	if err := cloudConn.AutoMigrate(&models.Ticker{}); err != nil {
		log.Fatalf("❌ AutoMigrate failed: %v", err)
	}
	if err := localConn.AutoMigrate(&models.TickerPrice{}); err != nil {
		log.Fatalf("❌ AutoMigrate for TickerPrice failed: %v", err)
	}
	// Convert to hypertable
	localConn.Exec("SELECT create_hypertable('ticker_prices', 'timestamp', if_not_exists => TRUE);")

	watchlistRepo := watchlist.NewRepository(cloudConn)
	watchlistService := watchlist.NewService(watchlistRepo)
	notificationService := notification.NewService(notifierCfg)
	tickerPriceRepository := ticker_price.NewRepository(localConn)
	tickerPriceService := ticker_price.NewService(watchlistService, vantageCfg, tickerPriceRepository)
	grpcServer := grpcapi.NewServer(watchlistService, tickerPriceService)

	// 3.1 Initialize Poller
	go tickerPriceService.PollAndPersist()

	// 3.2 Initialize Worker
	tickerChan := make(chan models.TickerPrice, 100)
	go ticker_price.StartSignalWorker(tickerChan, notificationService)

	// 4. Setup Router config
	r := chi.NewRouter()

	grpcListener, err := net.Listen("tcp", app.config.GRPC_ADDRESS)
	if err != nil {
		log.Fatalf("failed to listen for gRPC on %s: %v", app.config.GRPC_ADDRESS, err)
	}
	go func() {
		log.Println("Starting gRPC server on", app.config.GRPC_ADDRESS)
		if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
			log.Fatalf("gRPC server failed: %v", serveErr)
		}
	}()

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

	// 6. Serve REST + gRPC OpenAPI docs in separate channels.
	r.Get("/openapi/rest/swagger.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, docs.SwaggerInfo.ReadDoc())
	})
	r.Get("/openapi/grpc/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		const grpcOpenAPIPath = "docs/openapi/grpc/proto/golddigger/v1/api.swagger.json"
		if _, statErr := os.Stat(grpcOpenAPIPath); statErr != nil {
			http.Error(w, "gRPC OpenAPI spec not found. Run `make proto`.", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, req, grpcOpenAPIPath)
	})
	r.Get("/swagger/rest/*", httpSwagger.Handler(httpSwagger.URL("/openapi/rest/swagger.json")))
	r.Get("/swagger/grpc/*", httpSwagger.Handler(httpSwagger.URL("/openapi/grpc/swagger.json")))
	r.Get("/swagger/*", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/rest/index.html", http.StatusTemporaryRedirect)
	})

	log.Println("Starting server on", app.config.ADDRESS)
	return server.ListenAndServe()
}
