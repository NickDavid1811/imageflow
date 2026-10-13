package httpadapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
	"github.com/nickdavid1811/imageflow/openapi"
)

const scalarDocument = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ImageFlow API Reference</title>
</head>
<body>
  <div id="app"></div>
  <script type="module" nonce="{{nonce}}">
    import { createApiReference } from 'https://cdn.jsdelivr.net/npm/@scalar/api-reference/esm.js'
    createApiReference('#app', {
      url: '/openapi.yaml',
      pageTitle: 'ImageFlow API Reference',
      withDefaultFonts: false,
      hideClientButton: true
    })
  </script>
</body>
</html>`

// Handler exposes job use cases and health checks over HTTP.
type Handler struct {
	service   *application.JobService
	readiness ReadinessChecker
	logger    *slog.Logger
}

// ReadinessChecker reports whether a required dependency is reachable.
type ReadinessChecker interface {
	Ping(context.Context) error
}

// NewHandler constructs the public HTTP adapter.
func NewHandler(service *application.JobService, readiness ReadinessChecker, logger *slog.Logger) *Handler {
	return &Handler{service: service, readiness: readiness, logger: logger}
}

// Routes returns the complete ImageFlow HTTP routing tree.
func (handler *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handler.live)
	mux.HandleFunc("GET /health/ready", handler.ready)
	mux.HandleFunc("GET /openapi.yaml", handler.openAPIDocument)
	mux.HandleFunc("GET /docs", handler.documentation)
	mux.HandleFunc("GET /docs/", handler.documentation)
	mux.HandleFunc("POST /v1/jobs", handler.createJob)
	mux.HandleFunc("GET /v1/jobs/{id}", handler.getJob)
	mux.HandleFunc("GET /v1/jobs/{id}/result", handler.getResult)
	return requestLog(handler.logger, mux)
}

func (handler *Handler) getResult(w http.ResponseWriter, request *http.Request) {
	id := strings.TrimSpace(request.PathValue("id"))
	result, err := handler.service.Result(request.Context(), id)
	if errors.Is(err, application.ErrJobNotFound) {
		writeError(w, http.StatusNotFound, "job_not_found", "job does not exist")
		return
	}
	if errors.Is(err, application.ErrResultNotReady) {
		writeError(w, http.StatusConflict, "result_not_ready", "job result is not ready")
		return
	}
	if err != nil {
		handler.logger.Error("get job result failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "the result could not be retrieved")
		return
	}
	defer result.Body.Close()
	w.Header().Set("Content-Type", result.Output.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", result.Output.SizeBytes))
	w.Header().Set("Content-Disposition", "inline; filename=\""+id+"."+result.Output.Format+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, result.Body)
}

func (handler *Handler) openAPIDocument(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"openapi.yaml\"")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openapi.Document)
}

func (handler *Handler) documentation(w http.ResponseWriter, _ *http.Request) {
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		handler.logger.Error("generate documentation nonce failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "documentation could not be rendered")
		return
	}
	nonce := base64.RawStdEncoding.EncodeToString(nonceBytes)
	document := strings.Replace(scalarDocument, "{{nonce}}", nonce, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; img-src data: https:; connect-src 'self' https://cdn.jsdelivr.net; font-src data:")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, document)
}

func (handler *Handler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) ready(w http.ResponseWriter, request *http.Request) {
	if err := handler.readiness.Ping(request.Context()); err != nil {
		handler.logger.Warn("readiness check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "not_ready", "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (handler *Handler) createJob(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
	var input struct {
		SourceURL  string             `json:"source_url"`
		Operations []domain.Operation `json:"operations"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body must not exceed 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain valid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain a single JSON object")
		return
	}

	job, err := handler.service.Create(request.Context(), application.CreateJobCommand(input))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSource) || errors.Is(err, domain.ErrMissingOperations) || errors.Is(err, domain.ErrInvalidOperation) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
		handler.logger.Error("create job failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "the job could not be created")
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (handler *Handler) getJob(w http.ResponseWriter, request *http.Request) {
	id := strings.TrimSpace(request.PathValue("id"))
	job, err := handler.service.Get(request.Context(), id)
	if errors.Is(err, application.ErrJobNotFound) {
		writeError(w, http.StatusNotFound, "job_not_found", "job does not exist")
		return
	}
	if err != nil {
		handler.logger.Error("get job failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "the job could not be retrieved")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func requestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		logger.Info("http request", "method", request.Method, "path", request.URL.Path)
		next.ServeHTTP(w, request)
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
