package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/memory"
	"github.com/nickdavid1811/imageflow/internal/jobs/application"
)

func TestCreateAndGetJob(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := memory.NewJobRepository()
	service := application.NewJobService(repository)
	router := NewHandler(service, readyCheck{}, logger).Routes()

	body := []byte(`{"source_url":"https://example.com/image.jpg","operations":[{"type":"resize","width":300},{"type":"convert","format":"webp","quality":80}]}`)
	request := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 202 {
		t.Fatalf("status = %d, want %d", recorder.Code, 202)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	request = httptest.NewRequest("GET", "/v1/jobs/"+created.ID, nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want %d", recorder.Code, 200)
	}
}

type readyCheck struct{}

func (readyCheck) Ping(context.Context) error { return nil }
