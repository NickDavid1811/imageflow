package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

func TestProcessorCompletesClaimedJob(t *testing.T) {
	repository := &processingRepositoryStub{claimed: true, status: domain.StatusQueued, job: domain.Job{ID: "job_123", SourceURL: "https://example.com/a.png", Operations: []domain.Operation{{Type: "convert", Format: "png"}}}}
	processor := NewProcessor(repository, fetcherStub{}, imageProcessorStub{}, &outputWriterStub{})

	if err := processor.Process(context.Background(), "job_123"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !repository.completed {
		t.Fatal("job was not completed")
	}
	want := []int{30, 80}
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
	processor := NewProcessor(repository, fetcherStub{}, imageProcessorStub{}, &outputWriterStub{})

	err := processor.Process(context.Background(), "job_123")
	if !errors.Is(err, ErrJobBusy) {
		t.Fatalf("Process() error = %v, want %v", err, ErrJobBusy)
	}
}

func TestProcessorAcceptsCompletedDuplicate(t *testing.T) {
	repository := &processingRepositoryStub{claimed: false, status: domain.StatusCompleted}
	processor := NewProcessor(repository, fetcherStub{}, imageProcessorStub{}, &outputWriterStub{})

	if err := processor.Process(context.Background(), "job_123"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
}

func TestProcessorReleasesFailedAttemptForRetry(t *testing.T) {
	repository := &processingRepositoryStub{claimed: true, job: domain.Job{ID: "job_123", SourceURL: "https://example.com/a.png"}}
	processor := NewProcessor(repository, failingFetcherStub{}, imageProcessorStub{}, &outputWriterStub{})
	if err := processor.Process(context.Background(), "job_123"); err == nil {
		t.Fatal("Process() error = nil")
	}
	if !repository.retried {
		t.Fatal("failed attempt was not released for retry")
	}
}

type processingRepositoryStub struct {
	claimed   bool
	status    domain.Status
	progress  []int
	completed bool
	job       domain.Job
	output    domain.Output
	retried   bool
}

func (repository *processingRepositoryStub) FindByID(context.Context, string) (domain.Job, error) {
	return repository.job, nil
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

func (repository *processingRepositoryStub) Complete(_ context.Context, _ string, output domain.Output, _ time.Time) error {
	repository.completed = true
	repository.output = output
	return nil
}

type fetcherStub struct{}

func (fetcherStub) Fetch(context.Context, string) ([]byte, error) { return []byte("source"), nil }

type imageProcessorStub struct{}

func (imageProcessorStub) Process(context.Context, []byte, []domain.Operation) ([]byte, domain.Output, error) {
	return []byte("result"), domain.Output{Format: "png", ContentType: "image/png", SizeBytes: 6, Width: 10, Height: 10}, nil
}

type failingFetcherStub struct{}

func (failingFetcherStub) Fetch(context.Context, string) ([]byte, error) {
	return nil, errors.New("unavailable")
}

type outputWriterStub struct{}

func (*outputWriterStub) Save(context.Context, string, []byte) error { return nil }

func (repository *processingRepositoryStub) Retry(context.Context, string, string, string, time.Time) error {
	repository.retried = true
	return nil
}
