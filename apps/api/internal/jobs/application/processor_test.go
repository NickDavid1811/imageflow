package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

func TestProcessorCompletesClaimedJob(t *testing.T) {
	repository := &processingRepositoryStub{claimed: true, status: domain.StatusQueued}
	processor := NewProcessor(repository)
	processor.delay = 0

	if err := processor.Process(context.Background(), "job_123"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !repository.completed {
		t.Fatal("job was not completed")
	}
	want := []int{25, 60, 90}
	if len(repository.progress) != len(want) {
		t.Fatalf("progress = %v, want %v", repository.progress, want)
	}
	for index := range want {
		if repository.progress[index] != want[index] {
			t.Fatalf("progress = %v, want %v", repository.progress, want)
		}
	}
}

func TestProcessorRejectsConcurrentProcessing(t *testing.T) {
	repository := &processingRepositoryStub{claimed: false, status: domain.StatusProcessing}
	processor := NewProcessor(repository)
	processor.delay = 0

	err := processor.Process(context.Background(), "job_123")
	if !errors.Is(err, ErrJobBusy) {
		t.Fatalf("Process() error = %v, want %v", err, ErrJobBusy)
	}
}

func TestProcessorAcceptsCompletedDuplicate(t *testing.T) {
	repository := &processingRepositoryStub{claimed: false, status: domain.StatusCompleted}
	processor := NewProcessor(repository)
	processor.delay = 0

	if err := processor.Process(context.Background(), "job_123"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
}

type processingRepositoryStub struct {
	claimed   bool
	status    domain.Status
	progress  []int
	completed bool
}

func (repository *processingRepositoryStub) ClaimForProcessing(context.Context, string, time.Time) (bool, error) {
	return repository.claimed, nil
}

func (repository *processingRepositoryStub) FailPermanently(context.Context, string, string, string, time.Time) error {
	return nil
}

func (repository *processingRepositoryStub) Status(context.Context, string) (domain.Status, error) {
	return repository.status, nil
}

func (repository *processingRepositoryStub) UpdateProgress(_ context.Context, _ string, progress int, _ time.Time) error {
	repository.progress = append(repository.progress, progress)
	return nil
}

func (repository *processingRepositoryStub) Complete(context.Context, string, time.Time) error {
	repository.completed = true
	return nil
}

func (repository *processingRepositoryStub) Fail(context.Context, string, string, string, time.Time) error {
	return nil
}
