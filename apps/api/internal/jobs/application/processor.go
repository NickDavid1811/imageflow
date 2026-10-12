package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

var ErrJobBusy = errors.New("job is already being processed")

// ProcessingRepository is the persistence port used by the background processor.
type ProcessingRepository interface {
	ClaimForProcessing(context.Context, string, time.Time) (bool, error)
	Status(context.Context, string) (domain.Status, error)
	UpdateProgress(context.Context, string, int, time.Time) error
	Complete(context.Context, string, time.Time) error
	Fail(context.Context, string, string, string, time.Time) error
	FailPermanently(context.Context, string, string, string, time.Time) error
}

// FailPermanently records exhaustion of delivery attempts.
func (processor *Processor) FailPermanently(ctx context.Context, jobID string, cause error) error {
	return processor.repository.FailPermanently(ctx, jobID, "delivery_attempts_exhausted", cause.Error(), processor.now())
}

// Processor claims jobs and executes their processing lifecycle.
type Processor struct {
	repository ProcessingRepository
	now        func() time.Time
	delay      time.Duration
}

// NewProcessor constructs a processor using the production clock.
func NewProcessor(repository ProcessingRepository) *Processor {
	return &Processor{repository: repository, now: time.Now, delay: 600 * time.Millisecond}
}

// Process claims and processes one job and safely ignores duplicate terminal deliveries.
func (processor *Processor) Process(ctx context.Context, jobID string) error {
	claimed, err := processor.repository.ClaimForProcessing(ctx, jobID, processor.now())
	if err != nil {
		return err
	}
	if !claimed {
		status, err := processor.repository.Status(ctx, jobID)
		if err != nil {
			return err
		}
		if status == domain.StatusProcessing || status == domain.StatusQueued {
			return ErrJobBusy
		}
		return nil
	}

	for _, progress := range []int{25, 60, 90} {
		if err := wait(ctx, processor.delay); err != nil {
			_ = processor.repository.Fail(context.Background(), jobID, "processing_cancelled", err.Error(), processor.now())
			return err
		}
		if err := processor.repository.UpdateProgress(ctx, jobID, progress, processor.now()); err != nil {
			_ = processor.repository.Fail(context.Background(), jobID, "progress_update_failed", err.Error(), processor.now())
			return fmt.Errorf("set progress to %d: %w", progress, err)
		}
	}
	if err := wait(ctx, processor.delay); err != nil {
		_ = processor.repository.Fail(context.Background(), jobID, "processing_cancelled", err.Error(), processor.now())
		return err
	}
	return processor.repository.Complete(ctx, jobID, processor.now())
}

func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
