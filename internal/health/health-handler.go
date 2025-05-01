package health

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"net/http"
)

func RegisterRoutes(r chi.Router) {
	r.Route("/health", func(r chi.Router) {
		r.Get("/", GetHandler)
	})
}

// GetHandler handles GET /health
// @Summary      Get health status
// @Description  Returns service health
// @Tags         health
// @Success      200  string  "OK"
// @Router       /api/v1/health [get]
func GetHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// Create a response object
	response := models.HealthResponse{
		Status:  "OK",
		Message: "The server is up",
	}

	// Set headers and send JSON response
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)

	err := json.NewEncoder(responseWriter).Encode(response)
	if err != nil {
		return
	}
}
