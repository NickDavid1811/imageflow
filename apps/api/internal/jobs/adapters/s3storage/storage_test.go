package s3storage

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestStorageLifecycle(t *testing.T) {
	bucketExists := false
	objects := map[string][]byte{}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(request.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("missing SigV4 authorization")
		}
		status, body := http.StatusOK, ""
		switch {
		case request.Method == http.MethodHead && request.URL.Path == "/imageflow":
			if !bucketExists {
				status = http.StatusNotFound
			}
		case request.Method == http.MethodPut && request.URL.Path == "/imageflow":
			bucketExists = true
		case request.Method == http.MethodPut:
			objects[request.URL.Path], _ = io.ReadAll(request.Body)
		case request.Method == http.MethodGet:
			data, found := objects[request.URL.Path]
			if !found {
				status = http.StatusNotFound
			} else {
				body = string(data)
			}
		default:
			status = http.StatusMethodNotAllowed
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	storage, err := New(Config{Endpoint: "http://minio:9000", Region: "us-east-1", Bucket: "imageflow", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	storage.client.Transport = transport
	if err := storage.EnsureBucket(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(context.Background(), "job.png", []byte("image")); err != nil {
		t.Fatal(err)
	}
	reader, err := storage.Open(context.Background(), "job.png")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	if string(data) != "image" {
		t.Fatalf("data = %q", data)
	}
}

func TestRejectsUnsafeObjectKey(t *testing.T) {
	storage, err := New(Config{Endpoint: "http://minio:9000", Region: "us-east-1", Bucket: "imageflow", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(context.Background(), "../secret", nil); err == nil {
		t.Fatal("Save() accepted path traversal")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
