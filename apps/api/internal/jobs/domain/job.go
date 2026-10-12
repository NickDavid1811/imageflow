package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidID         = errors.New("job id is required")
	ErrInvalidSource     = errors.New("source_url must be an absolute http or https URL")
	ErrMissingOperations = errors.New("at least one operation is required")
	ErrInvalidOperation  = errors.New("invalid image operation")
	ErrInvalidTransition = errors.New("invalid job status transition")
	ErrInvalidProgress   = errors.New("job progress must be between 0 and 100")
)

// Status is the lifecycle state of an image-processing job.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Operation describes one transformation in the requested processing pipeline.
type Operation struct {
	Type    string `json:"type"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Format  string `json:"format,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// Job is the persisted representation of an asynchronous image-processing request.
type Job struct {
	ID           string      `json:"id"`
	SourceURL    string      `json:"source_url"`
	Operations   []Operation `json:"operations"`
	Status       Status      `json:"status"`
	Progress     int         `json:"progress"`
	Attempts     int         `json:"attempts"`
	ErrorCode    string      `json:"error_code,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// NewJob validates input and creates a queued job.
func NewJob(id, sourceURL string, operations []Operation, now time.Time) (Job, error) {
	if strings.TrimSpace(id) == "" {
		return Job{}, ErrInvalidID
	}
	parsedURL, err := url.ParseRequestURI(sourceURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return Job{}, ErrInvalidSource
	}
	if len(operations) == 0 {
		return Job{}, ErrMissingOperations
	}
	for _, operation := range operations {
		if err := operation.Validate(); err != nil {
			return Job{}, err
		}
	}

	return Job{
		ID:         id,
		SourceURL:  sourceURL,
		Operations: append([]Operation(nil), operations...),
		Status:     StatusQueued,
		CreatedAt:  now.UTC(),
		UpdatedAt:  now.UTC(),
	}, nil
}

// Start moves a queued job into processing.
func (job *Job) Start(now time.Time) error {
	if job.Status != StatusQueued {
		return ErrInvalidTransition
	}
	now = now.UTC()
	job.Status = StatusProcessing
	job.Attempts++
	job.StartedAt = &now
	job.UpdatedAt = now
	return nil
}

// SetProgress records progress for a processing job.
func (job *Job) SetProgress(progress int, now time.Time) error {
	if job.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	if progress < 0 || progress > 100 {
		return ErrInvalidProgress
	}
	job.Progress = progress
	job.UpdatedAt = now.UTC()
	return nil
}

// Complete marks a processing job as successfully completed.
func (job *Job) Complete(now time.Time) error {
	if job.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	now = now.UTC()
	job.Status = StatusCompleted
	job.Progress = 100
	job.CompletedAt = &now
	job.UpdatedAt = now
	return nil
}

// Fail marks a processing job as failed with diagnostic information.
func (job *Job) Fail(code, message string, now time.Time) error {
	if job.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	job.Status = StatusFailed
	job.ErrorCode = code
	job.ErrorMessage = message
	job.UpdatedAt = now.UTC()
	return nil
}

// Retry returns a failed job to the queue and clears its previous failure.
func (job *Job) Retry(now time.Time) error {
	if job.Status != StatusFailed {
		return ErrInvalidTransition
	}
	job.Status = StatusQueued
	job.Progress = 0
	job.ErrorCode = ""
	job.ErrorMessage = ""
	job.StartedAt = nil
	job.CompletedAt = nil
	job.UpdatedAt = now.UTC()
	return nil
}

// Cancel marks a queued or processing job as cancelled.
func (job *Job) Cancel(now time.Time) error {
	if job.Status != StatusQueued && job.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	job.Status = StatusCancelled
	job.UpdatedAt = now.UTC()
	return nil
}

func (operation Operation) Validate() error {
	switch operation.Type {
	case "resize":
		if operation.Width <= 0 && operation.Height <= 0 {
			return ErrInvalidOperation
		}
	case "convert":
		switch operation.Format {
		case "jpeg", "png", "webp", "avif":
		default:
			return ErrInvalidOperation
		}
	default:
		return ErrInvalidOperation
	}
	if operation.Quality < 0 || operation.Quality > 100 {
		return ErrInvalidOperation
	}
	return nil
}
