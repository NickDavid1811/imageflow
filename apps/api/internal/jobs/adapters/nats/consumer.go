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

// JobProcessor processes jobs and records terminal delivery exhaustion.
type JobProcessor interface {
	Process(context.Context, string) error
	FailPermanently(context.Context, string, error) error
}

// DeadLetterPublisher stores a message that cannot be processed successfully.
type DeadLetterPublisher interface {
	PublishDeadLetter(context.Context, string, string, []byte, string, uint64) error
}

// Consumer pulls durable job messages and coordinates acknowledgments and retries.
type Consumer struct {
	subscription  *nats.Subscription
	processor     JobProcessor
	logger        *slog.Logger
	deadLetters   DeadLetterPublisher
	maxDeliveries uint64
}

// NewConsumer constructs a durable job consumer.
func NewConsumer(subscription *nats.Subscription, processor JobProcessor, deadLetters DeadLetterPublisher, maxDeliveries int, logger *slog.Logger) *Consumer {
	return &Consumer{
		subscription: subscription, processor: processor, deadLetters: deadLetters,
		maxDeliveries: uint64(maxDeliveries), logger: logger,
	}
}

// Run fetches and processes messages until the context is cancelled or fetching fails.
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
			consumer.scheduleRedelivery(message, event.JobID)
			return
		}
		metadata, metadataErr := message.Metadata()
		if metadataErr != nil {
			consumer.logger.Error("read message metadata failed", "job_id", event.JobID, "error", metadataErr)
			consumer.scheduleRedelivery(message, event.JobID)
			return
		}
		if metadata.NumDelivered >= consumer.maxDeliveries {
			consumer.deadLetter(ctx, message, event.JobID, err, metadata.NumDelivered)
			return
		}
		consumer.logger.Error("job processing failed; scheduling redelivery", "job_id", event.JobID, "error", err)
		consumer.scheduleRedelivery(message, event.JobID)
		return
	}
	if err := message.Ack(); err != nil {
		consumer.logger.Error("job acknowledgment failed", "job_id", event.JobID, "error", err)
		return
	}
	consumer.logger.Info("job processed", "job_id", event.JobID)
}

func (consumer *Consumer) deadLetter(ctx context.Context, message *nats.Msg, jobID string, cause error, deliveries uint64) {
	eventID := message.Header.Get(nats.MsgIdHdr)
	if eventID == "" {
		eventID = jobID
	}
	if err := consumer.deadLetters.PublishDeadLetter(ctx, eventID, message.Subject, message.Data, cause.Error(), deliveries); err != nil {
		consumer.logger.Error("publish dead-letter event failed", "job_id", jobID, "error", err)
		consumer.scheduleRedelivery(message, jobID)
		return
	}
	if err := consumer.processor.FailPermanently(ctx, jobID, cause); err != nil {
		consumer.logger.Error("mark exhausted job failed", "job_id", jobID, "error", err)
		consumer.scheduleRedelivery(message, jobID)
		return
	}
	if err := message.Term(); err != nil {
		consumer.logger.Error("terminate exhausted message failed", "job_id", jobID, "error", err)
		return
	}
	consumer.logger.Error("job moved to dead-letter queue", "job_id", jobID, "deliveries", deliveries, "error", cause)
}

func (consumer *Consumer) scheduleRedelivery(message *nats.Msg, jobID string) {
	if err := message.NakWithDelay(5 * time.Second); err != nil {
		consumer.logger.Error("delayed acknowledgment failed", "job_id", jobID, "error", err)
	}
}
