package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	now := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	job, err := NewJob("job_123", "https://example.com/image.jpg", []Operation{{Type: "resize", Width: 800}}, now)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}
	if job.Status != StatusQueued {
		t.Fatalf("status = %q, want %q", job.Status, StatusQueued)
	}
	if !job.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", job.CreatedAt, now)
	}
}

func TestNewJobRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		sourceURL  string
		operations []Operation
		want       error
	}{
		{name: "relative source", sourceURL: "/image.jpg", operations: []Operation{{Type: "resize", Width: 100}}, want: ErrInvalidSource},
		{name: "missing operations", sourceURL: "https://example.com/image.jpg", want: ErrMissingOperations},
		{name: "invalid resize", sourceURL: "https://example.com/image.jpg", operations: []Operation{{Type: "resize"}}, want: ErrInvalidOperation},
		{name: "invalid format", sourceURL: "https://example.com/image.jpg", operations: []Operation{{Type: "convert", Format: "bmp"}}, want: ErrInvalidOperation},
		{name: "cover without both dimensions", sourceURL: "https://example.com/image.jpg", operations: []Operation{{Type: "resize", Width: 100, Fit: "cover"}}, want: ErrInvalidOperation},
		{name: "invalid crop", sourceURL: "https://example.com/image.jpg", operations: []Operation{{Type: "crop", X: -1, Width: 100, Height: 100}}, want: ErrInvalidOperation},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewJob("job_123", test.sourceURL, test.operations, time.Now())
			if !errors.Is(err, test.want) {
				t.Fatalf("NewJob() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestJobLifecycle(t *testing.T) {
	createdAt := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	job, err := NewJob("job_123", "https://example.com/image.jpg", []Operation{{Type: "resize", Width: 800}}, createdAt)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	startedAt := createdAt.Add(time.Minute)
	if err := job.Start(startedAt); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if job.Status != StatusProcessing || job.Attempts != 1 {
		t.Fatalf("after Start(): status = %q, attempts = %d", job.Status, job.Attempts)
	}
	if err := job.SetProgress(50, startedAt.Add(time.Minute)); err != nil {
		t.Fatalf("SetProgress() error = %v", err)
	}
	if job.Progress != 50 {
		t.Fatalf("progress = %d, want 50", job.Progress)
	}

	completedAt := startedAt.Add(2 * time.Minute)
	if err := job.Complete(completedAt); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if job.Status != StatusCompleted || job.Progress != 100 || job.CompletedAt == nil {
		t.Fatalf("completed job = %+v", job)
	}
	if err := job.Start(completedAt.Add(time.Minute)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Start() after completion error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestFailedJobCanBeRetried(t *testing.T) {
	now := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	job, err := NewJob("job_123", "https://example.com/image.jpg", []Operation{{Type: "resize", Width: 800}}, now)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}
	if err := job.Start(now.Add(time.Minute)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := job.Fail("download_failed", "source unavailable", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if err := job.Retry(now.Add(3 * time.Minute)); err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if job.Status != StatusQueued || job.ErrorCode != "" || job.StartedAt != nil {
		t.Fatalf("retried job = %+v", job)
	}
}
