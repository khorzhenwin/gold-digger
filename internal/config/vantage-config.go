package config

import (
	"fmt"
	"os"
	"strings"
)

type VantageConfig struct {
	ApiKey        string
	ApiKeyBackups []string
	BaseUrl       string
}

func LoadVantageConfig() (*VantageConfig, error) {
	cfg := &VantageConfig{
		ApiKey: strings.TrimSpace(os.Getenv("ALPHA_VANTAGE_API_KEY")),
		ApiKeyBackups: []string{
			os.Getenv("ALPHA_VANTAGE_API_KEY_1"),
			os.Getenv("ALPHA_VANTAGE_API_KEY_2"),
		},
		BaseUrl: strings.TrimSpace(os.Getenv("ALPHA_VANTAGE_BASE_URL")),
	}

	if cfg.ApiKey == "" || cfg.BaseUrl == "" {
		return nil, fmt.Errorf("incomplete Vantage config")
	}

	return cfg, nil
}

func (c *VantageConfig) GetGlobalQuoteUrl(symbol string, apiKey string) string {
	return fmt.Sprintf(
		"%s/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		c.BaseUrl, symbol, apiKey,
	)
}
