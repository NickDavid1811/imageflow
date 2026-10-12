package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

type Handler struct {
	service   *application.JobService
	readiness ReadinessChecker
	logger    *slog.Logger
}

type ReadinessChecker interface {
	Ping(context.Context) error
}

func NewHandler(service *application.JobService, readiness ReadinessChecker, logger *slog.Logger) *Handler {
	return &Handler{service: service, readiness: readiness, logger: logger}
}

func (handler *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handler.live)
	mux.HandleFunc("GET /health/ready", handler.ready)
	mux.HandleFunc("POST /v1/jobs", handler.createJob)
	mux.HandleFunc("GET /v1/jobs/{id}", handler.getJob)
	return requestLog(handler.logger, mux)
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
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain valid JSON")
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
