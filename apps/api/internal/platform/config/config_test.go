package config

import "testing"

func TestFromEnvUsesDeadLetterDefaults(t *testing.T) {
	t.Setenv("IMAGEFLOW_NATS_MAX_DELIVERIES", "")
	t.Setenv("IMAGEFLOW_NATS_DLQ_STREAM", "")
	t.Setenv("IMAGEFLOW_NATS_DLQ_SUBJECT", "")

	config, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if config.NATSMaxDeliveries != 10 {
		t.Fatalf("NATSMaxDeliveries = %d, want 10", config.NATSMaxDeliveries)
	}
	if config.NATSDeadLetterStream != "IMAGEFLOW_JOBS_DLQ" {
		t.Fatalf("NATSDeadLetterStream = %q", config.NATSDeadLetterStream)
	}
	if config.NATSDeadLetterSubject != "imageflow.jobs.failed.v1" {
		t.Fatalf("NATSDeadLetterSubject = %q", config.NATSDeadLetterSubject)
	}
	if config.StorageDriver != "s3" || config.S3Endpoint != "http://localhost:9000" || config.S3Bucket != "imageflow" {
		t.Fatalf("storage defaults = %#v", config)
	}
}

func TestFromEnvRejectsInvalidMaxDeliveries(t *testing.T) {
	t.Setenv("IMAGEFLOW_NATS_MAX_DELIVERIES", "zero")
	if _, err := FromEnv(); err == nil {
		t.Fatal("FromEnv() error = nil, want validation error")
	}
}

func TestFromEnvRejectsInvalidStorageDriver(t *testing.T) {
	t.Setenv("IMAGEFLOW_STORAGE_DRIVER", "filesystem")
	if _, err := FromEnv(); err == nil {
		t.Fatal("FromEnv() error = nil, want validation error")
	}
}
