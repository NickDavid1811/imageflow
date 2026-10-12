package config

import (
	"errors"
	"os"
	"strings"
)

const defaultHTTPAddress = ":8080"
const defaultDatabaseURL = "postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable"

type Config struct {
	HTTPAddress string
	DatabaseURL string
}

func FromEnv() (Config, error) {
	address := os.Getenv("IMAGEFLOW_HTTP_ADDRESS")
	if address == "" {
		address = defaultHTTPAddress
	}
	databaseURL := os.Getenv("IMAGEFLOW_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}
	if !strings.HasPrefix(databaseURL, "postgres://") && !strings.HasPrefix(databaseURL, "postgresql://") {
		return Config{}, errors.New("IMAGEFLOW_DATABASE_URL must be a PostgreSQL URL")
	}
	return Config{HTTPAddress: address, DatabaseURL: databaseURL}, nil
}
