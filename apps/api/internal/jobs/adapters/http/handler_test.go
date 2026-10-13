package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authapplication "github.com/nickdavid1811/imageflow/internal/auth/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/memory"
	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

func TestCreateAndGetJob(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := memory.NewJobRepository()
	service := application.NewJobService(repository)
	router := NewHandler(service, readyCheck{}, authStub{}, logger).Routes()

	body := []byte(`{"source_url":"https://example.com/image.jpg","operations":[{"type":"resize","width":300},{"type":"convert","format":"jpeg","quality":80}]}`)
	request := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	authorize(request)
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
	authorize(request)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want %d", recorder.Code, 200)
	}
}

func TestGetCompletedJobResult(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := memory.NewJobRepository()
	job := domain.Job{ID: "job_123", ProjectID: "prj_test", Status: domain.StatusCompleted, Output: &domain.Output{Key: "job_123.png", ContentType: "image/png", Format: "png", SizeBytes: 5}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repository.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	service := application.NewJobService(repository, resultReaderStub{})
	router := NewHandler(service, readyCheck{}, authStub{}, logger).Routes()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/v1/jobs/job_123/result", nil)
	authorize(request)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "image" {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("Content-Type = %q", recorder.Header().Get("Content-Type"))
	}
}

func TestDocumentationEndpoints(t *testing.T) {
	router := newTestRouter()

	request := httptest.NewRequest("GET", "/openapi.yaml", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("OpenAPI status = %d, want 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/yaml") {
		t.Fatalf("OpenAPI Content-Type = %q", contentType)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline") {
		t.Fatalf("OpenAPI Content-Disposition = %q, want inline", disposition)
	}
	contract := recorder.Body.String()
	if !strings.Contains(contract, "openapi: 3.1.0") {
		t.Fatal("OpenAPI response does not contain the embedded contract")
	}
	for _, documentedPath := range []string{"/health/live:", "/health/ready:", "/v1/jobs:", "/v1/jobs/{id}:"} {
		if !strings.Contains(contract, documentedPath) {
			t.Errorf("OpenAPI response does not document %s", documentedPath)
		}
	}

	request = httptest.NewRequest("GET", "/docs", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("docs status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "createApiReference") {
		t.Fatal("docs response does not initialize Scalar")
	}
	policy := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "script-src 'nonce-") {
		t.Fatalf("docs CSP does not authorize its nonce: %q", policy)
	}
	if strings.Contains(recorder.Body.String(), "{{nonce}}") {
		t.Fatal("docs response contains an unresolved nonce placeholder")
	}
}

func TestCreateJobRejectsOversizedBody(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest("POST", "/v1/jobs", strings.NewReader(`{"padding":"`+strings.Repeat("x", (1<<20)+1)+`"}`))
	authorize(request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestCreateJobRejectsMultipleJSONValues(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest("POST", "/v1/jobs", strings.NewReader(`{"source_url":"https://example.com/image.jpg","operations":[{"type":"resize","width":300}]} {}`))
	authorize(request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestProtectedRouteRequiresAPIKey(t *testing.T) {
	router := newTestRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/v1/jobs/job_123", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func TestProjectCannotReadAnotherProjectJob(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := memory.NewJobRepository()
	_ = repository.Create(context.Background(), domain.Job{ID: "job_other", ProjectID: "prj_other"})
	service := application.NewJobService(repository)
	router := NewHandler(service, readyCheck{}, authStub{}, logger).Routes()
	request := httptest.NewRequest("GET", "/v1/jobs/job_other", nil)
	authorize(request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := memory.NewJobRepository()
	service := application.NewJobService(repository)
	return NewHandler(service, readyCheck{}, authStub{}, logger).Routes()
}

type readyCheck struct{}

func (readyCheck) Ping(context.Context) error { return nil }

type authStub struct{}

func (authStub) Authenticate(context.Context, string) (authapplication.Principal, error) {
	return authapplication.Principal{ProjectID: "prj_test", Environment: "test", APIKeyID: "key_test"}, nil
}

func authorize(request *http.Request) {
	request.Header.Set("Authorization", "Bearer if_test_valid_test_key_123456")
}

type resultReaderStub struct{}

func (resultReaderStub) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("image")), nil
}
