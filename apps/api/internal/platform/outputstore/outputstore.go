// Package outputstore selects and initializes the configured artifact storage adapter.
package outputstore

import (
	"context"
	"io"

	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/localstorage"
	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/s3storage"
	"github.com/nickdavid1811/imageflow/internal/platform/config"
)

// Store is the shared capability required by the API and worker.
type Store interface {
	Save(context.Context, string, []byte) error
	Open(context.Context, string) (io.ReadCloser, error)
}

// Open builds the selected adapter and prepares its backing storage.
func Open(ctx context.Context, config config.Config) (Store, error) {
	if config.StorageDriver == "local" {
		return localstorage.New(config.StoragePath)
	}
	storage, err := s3storage.New(s3storage.Config{
		Endpoint: config.S3Endpoint, Region: config.S3Region, Bucket: config.S3Bucket,
		AccessKey: config.S3AccessKey, SecretKey: config.S3SecretKey,
	})
	if err != nil {
		return nil, err
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		return nil, err
	}
	return storage, nil
}
