package application

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/nickdavid1811/imageflow/internal/outbox/domain"
)

type Repository interface {
	ClaimBatch(context.Context, int, time.Time, time.Duration) ([]domain.Event, error)
	MarkPublished(context.Context, int64, time.Time) error
	MarkFailed(context.Context, int64, string, time.Time) error
}

type EventPublisher interface {
	Publish(context.Context, string, []byte) error
}

type Publisher struct {
	repository Repository
	messages   EventPublisher
	logger     *slog.Logger
	interval   time.Duration
	now        func() time.Time
}

func NewPublisher(repository Repository, messages EventPublisher, logger *slog.Logger) *Publisher {
	return &Publisher{repository: repository, messages: messages, logger: logger, interval: 500 * time.Millisecond, now: time.Now}
}

func (publisher *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(publisher.interval)
	defer ticker.Stop()
	for {
		if err := publisher.publishBatch(ctx); err != nil && ctx.Err() == nil {
			publisher.logger.Error("outbox batch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (publisher *Publisher) publishBatch(ctx context.Context) error {
	events, err := publisher.repository.ClaimBatch(ctx, 25, publisher.now(), time.Minute)
	if err != nil {
		return err
	}
	for _, event := range events {
		eventID := "outbox_" + strconv.FormatInt(event.ID, 10)
		if err := publisher.messages.Publish(ctx, eventID, event.Payload); err != nil {
			delay := retryDelay(event.Attempts)
			if releaseErr := publisher.repository.MarkFailed(ctx, event.ID, err.Error(), publisher.now().Add(delay)); releaseErr != nil {
				return fmt.Errorf("publish event %d: %v; release: %w", event.ID, err, releaseErr)
			}
			publisher.logger.Warn("outbox event scheduled for retry", "event_id", event.ID, "delay", delay, "error", err)
			continue
		}
		if err := publisher.repository.MarkPublished(ctx, event.ID, publisher.now()); err != nil {
			return err
		}
		publisher.logger.Info("outbox event published", "event_id", event.ID, "event_type", event.Type)
	}
	return nil
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}
