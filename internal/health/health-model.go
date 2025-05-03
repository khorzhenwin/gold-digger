package health

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
