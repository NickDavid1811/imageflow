package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultHTTPAddress = ":8080"
const defaultDatabaseURL = "postgres://imageflow:imageflow_dev@localhost:5432/imageflow?sslmode=disable"
const defaultNATSURL = "nats://localhost:4222"
const defaultNATSStream = "IMAGEFLOW_JOBS"
const defaultNATSSubject = "imageflow.jobs.requested.v1"
const defaultNATSConsumer = "imageflow-workers"
const defaultNATSDeadLetterStream = "IMAGEFLOW_JOBS_DLQ"
const defaultNATSDeadLetterSubject = "imageflow.jobs.failed.v1"
const defaultNATSMaxDeliveries = 10
const defaultStoragePath = "../../infra/docker/data/objects"
const defaultStorageDriver = "s3"
const defaultS3Endpoint = "http://localhost:9000"
const defaultS3Region = "us-east-1"
const defaultS3Bucket = "imageflow"
const defaultS3AccessKey = "imageflow"
const defaultS3SecretKey = "imageflow_dev_secret"

// Config contains runtime settings shared by ImageFlow processes.
type Config struct {
	HTTPAddress           string
	DatabaseURL           string
	NATSURL               string
	NATSStream            string
	NATSSubject           string
	NATSConsumer          string
	NATSDeadLetterStream  string
	NATSDeadLetterSubject string
	NATSMaxDeliveries     int
	StoragePath           string
	StorageDriver         string
	S3Endpoint            string
	S3Region              string
	S3Bucket              string
	S3AccessKey           string
	S3SecretKey           string
}

// FromEnv loads IMAGEFLOW_* variables and safe local-development defaults.
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
	maxDeliveries, err := positiveIntOrDefault("IMAGEFLOW_NATS_MAX_DELIVERIES", defaultNATSMaxDeliveries)
	if err != nil {
		return Config{}, err
	}
	storageDriver := valueOrDefault("IMAGEFLOW_STORAGE_DRIVER", defaultStorageDriver)
	if storageDriver != "local" && storageDriver != "s3" {
		return Config{}, errors.New("IMAGEFLOW_STORAGE_DRIVER must be local or s3")
	}
	return Config{
		HTTPAddress:           address,
		DatabaseURL:           databaseURL,
		NATSURL:               natsURL,
		NATSStream:            valueOrDefault("IMAGEFLOW_NATS_STREAM", defaultNATSStream),
		NATSSubject:           valueOrDefault("IMAGEFLOW_NATS_SUBJECT", defaultNATSSubject),
		NATSConsumer:          valueOrDefault("IMAGEFLOW_NATS_CONSUMER", defaultNATSConsumer),
		NATSDeadLetterStream:  valueOrDefault("IMAGEFLOW_NATS_DLQ_STREAM", defaultNATSDeadLetterStream),
		NATSDeadLetterSubject: valueOrDefault("IMAGEFLOW_NATS_DLQ_SUBJECT", defaultNATSDeadLetterSubject),
		NATSMaxDeliveries:     maxDeliveries,
		StoragePath:           valueOrDefault("IMAGEFLOW_STORAGE_PATH", defaultStoragePath),
		StorageDriver:         storageDriver,
		S3Endpoint:            valueOrDefault("IMAGEFLOW_S3_ENDPOINT", defaultS3Endpoint),
		S3Region:              valueOrDefault("IMAGEFLOW_S3_REGION", defaultS3Region),
		S3Bucket:              valueOrDefault("IMAGEFLOW_S3_BUCKET", defaultS3Bucket),
		S3AccessKey:           valueOrDefault("IMAGEFLOW_S3_ACCESS_KEY", defaultS3AccessKey),
		S3SecretKey:           valueOrDefault("IMAGEFLOW_S3_SECRET_KEY", defaultS3SecretKey),
	}, nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func positiveIntOrDefault(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, errors.New(name + " must be a positive integer")
	}
	return parsed, nil
}
