package natsadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nickdavid1811/imageflow/internal/jobs/application"
)

type JobProcessor interface {
	Process(context.Context, string) error
}

type Consumer struct {
	subscription *nats.Subscription
	processor    JobProcessor
	logger       *slog.Logger
}

func NewConsumer(subscription *nats.Subscription, processor JobProcessor, logger *slog.Logger) *Consumer {
	return &Consumer{subscription: subscription, processor: processor, logger: logger}
}

func (consumer *Consumer) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		messages, err := consumer.subscription.Fetch(1, nats.MaxWait(time.Second))
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			return fmt.Errorf("fetch job message: %w", err)
		}
		for _, message := range messages {
			consumer.handle(ctx, message)
		}
	}
}

func (consumer *Consumer) handle(ctx context.Context, message *nats.Msg) {
	var event struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(message.Data, &event); err != nil || event.JobID == "" {
		consumer.logger.Error("discarding invalid job event", "error", err)
		if err := message.Term(); err != nil {
			consumer.logger.Error("terminate invalid message failed", "error", err)
		}
		return
	}
	if err := consumer.processor.Process(ctx, event.JobID); err != nil {
		if errors.Is(err, application.ErrJobBusy) {
			consumer.logger.Info("job is busy; scheduling redelivery", "job_id", event.JobID)
			if err := message.NakWithDelay(5 * time.Second); err != nil {
				consumer.logger.Error("delayed acknowledgment failed", "job_id", event.JobID, "error", err)
			}
			return
		}
		consumer.logger.Error("job processing failed; scheduling redelivery", "job_id", event.JobID, "error", err)
		if err := message.NakWithDelay(5 * time.Second); err != nil {
			consumer.logger.Error("delayed acknowledgment failed", "job_id", event.JobID, "error", err)
		}
		return
	}
	if err := message.Ack(); err != nil {
		consumer.logger.Error("job acknowledgment failed", "job_id", event.JobID, "error", err)
		return
	}
	consumer.logger.Info("job processed", "job_id", event.JobID)
}
