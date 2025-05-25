package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	DSN      string
}

func LoadAWSConfig() (*DBConfig, error) {
	cfg := &DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSL"),
	}

	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&sslrootcert=certs/global-bundle.pem", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	// Validate required fields
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.Name == "" {
		return nil, fmt.Errorf("incomplete DB config")
	}

	return cfg, nil
}

func LoadLocalDBConfig() (*DBConfig, error) {
	cfg := &DBConfig{
		Host:     os.Getenv("LOCAL_DB_HOST"),
		Port:     os.Getenv("LOCAL_DB_PORT"),
		User:     os.Getenv("LOCAL_DB_USER"),
		Password: os.Getenv("LOCAL_DB_PASSWORD"),
		Name:     os.Getenv("LOCAL_DB_NAME"),
		SSLMode:  os.Getenv("LOCAL_DB_SSL"),
	}

	cfg.DSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.Name == "" {
		return nil, fmt.Errorf("incomplete LOCAL DB config")
	}

	return cfg, nil
}

func (c *DBConfig) GetFormattedDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode,
	)
}
