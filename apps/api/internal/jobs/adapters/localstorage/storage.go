// Package localstorage stores generated artifacts on the local filesystem.
package localstorage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage is a local-development output store.
type Storage struct{ root string }

// New creates the storage directory and returns a filesystem-backed adapter.
func New(root string) (*Storage, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}
	return &Storage{root: root}, nil
}

// Save atomically writes an artifact below the configured root.
func (storage *Storage) Save(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := storage.path(key)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(storage.root, ".imageflow-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err = temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write output: %w", err)
	}
	if err = temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return fmt.Errorf("set output permissions: %w", err)
	}
	if err = temporary.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	if err = os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("publish output: %w", err)
	}
	return nil
}

// Open returns an artifact stream.
func (storage *Storage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := storage.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open output: %w", err)
	}
	return file, nil
}

func (storage *Storage) path(key string) (string, error) {
	if key == "" || filepath.Base(key) != key || strings.ContainsAny(key, `/\\`) {
		return "", fmt.Errorf("invalid output key")
	}
	return filepath.Join(storage.root, key), nil
}
