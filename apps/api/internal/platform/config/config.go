package config

import (
	"errors"
	"os"
	"strings"
)

const defaultHTTPAddress = ":8080"
const defaultDatabaseURL = "postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable"
const defaultNATSURL = "nats://localhost:4222"
const defaultNATSStream = "IMAGEFLOW_JOBS"
const defaultNATSSubject = "imageflow.jobs.requested.v1"
const defaultNATSConsumer = "imageflow-workers"

type Config struct {
	HTTPAddress  string
	DatabaseURL  string
	NATSURL      string
	NATSStream   string
	NATSSubject  string
	NATSConsumer string
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
	natsURL := valueOrDefault("IMAGEFLOW_NATS_URL", defaultNATSURL)
	if !strings.HasPrefix(natsURL, "nats://") && !strings.HasPrefix(natsURL, "tls://") {
		return Config{}, errors.New("IMAGEFLOW_NATS_URL must be a NATS URL")
	}
	return Config{
		HTTPAddress:  address,
		DatabaseURL:  databaseURL,
		NATSURL:      natsURL,
		NATSStream:   valueOrDefault("IMAGEFLOW_NATS_STREAM", defaultNATSStream),
		NATSSubject:  valueOrDefault("IMAGEFLOW_NATS_SUBJECT", defaultNATSSubject),
		NATSConsumer: valueOrDefault("IMAGEFLOW_NATS_CONSUMER", defaultNATSConsumer),
	}, nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
