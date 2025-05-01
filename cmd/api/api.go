package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/khorzhenwin/gold-digger/internal/health"
	"github.com/swaggo/http-swagger"
	//applicationConfig "github.com/khorzhenwin/go-chujang/internal/config"
	"log"
	"net/http"
)

func (app *application) run() error {
	// 1. Load Configs
	_ = godotenv.Load() // Loads from .env file

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
		//watchlist.RegisterRoutes(r, watchlistService)
	})

	// 6. Serve Swagger (if generated)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	log.Println("Starting server on", app.config.ADDRESS)
	return server.ListenAndServe()
}
