package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nickdavid1811/imageflow/internal/outbox/domain"
)

func TestPublishBatchMarksSuccessfulEvent(t *testing.T) {
	repository := &outboxRepositoryStub{events: []domain.Event{{ID: 7, Type: "job.requested.v1", Payload: []byte(`{"job_id":"job_1"}`), Attempts: 1}}}
	messages := &messagePublisherStub{}
	publisher := newTestPublisher(repository, messages)

	if err := publisher.publishBatch(context.Background()); err != nil {
		t.Fatalf("publishBatch() error = %v", err)
	}
	if repository.publishedID != 7 {
		t.Fatalf("published id = %d, want 7", repository.publishedID)
	}
	if messages.eventID != "outbox_7" {
		t.Fatalf("message id = %q, want outbox_7", messages.eventID)
	}
}

func TestPublishBatchSchedulesFailedEvent(t *testing.T) {
	repository := &outboxRepositoryStub{events: []domain.Event{{ID: 9, Payload: []byte(`{}`), Attempts: 2}}}
	messages := &messagePublisherStub{err: errors.New("nats unavailable")}
	publisher := newTestPublisher(repository, messages)

	if err := publisher.publishBatch(context.Background()); err != nil {
		t.Fatalf("publishBatch() error = %v", err)
	}
	if repository.failedID != 9 {
		t.Fatalf("failed id = %d, want 9", repository.failedID)
	}
	if !repository.availableAt.Equal(publisher.now().Add(2 * time.Second)) {
		t.Fatalf("available at = %v", repository.availableAt)
	}
}

func newTestPublisher(repository Repository, messages EventPublisher) *Publisher {
	publisher := NewPublisher(repository, messages, slog.New(slog.NewTextHandler(io.Discard, nil)))
	fixed := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	publisher.now = func() time.Time { return fixed }
	return publisher
}

type outboxRepositoryStub struct {
	events      []domain.Event
	publishedID int64
	failedID    int64
	availableAt time.Time
}

func (repository *outboxRepositoryStub) ClaimBatch(context.Context, int, time.Time, time.Duration) ([]domain.Event, error) {
	return repository.events, nil
}

func (repository *outboxRepositoryStub) MarkPublished(_ context.Context, id int64, _ time.Time) error {
	repository.publishedID = id
	return nil
}

func (repository *outboxRepositoryStub) MarkFailed(_ context.Context, id int64, _ string, availableAt time.Time) error {
	repository.failedID = id
	repository.availableAt = availableAt
	return nil
}

type messagePublisherStub struct {
	eventID string
	err     error
}

func (publisher *messagePublisherStub) Publish(_ context.Context, eventID string, _ []byte) error {
	publisher.eventID = eventID
	return publisher.err
}
