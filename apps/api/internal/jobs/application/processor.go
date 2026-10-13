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
	FindByID(context.Context, string) (domain.Job, error)
	Status(context.Context, string) (domain.Status, error)
	UpdateProgress(context.Context, string, int, time.Time) error
	Complete(context.Context, string, domain.Output, time.Time) error
	Retry(context.Context, string, string, string, time.Time) error
	FailPermanently(context.Context, string, string, string, time.Time) error
}

// ImageFetcher obtains a source image from an external location.
type ImageFetcher interface {
	Fetch(context.Context, string) ([]byte, error)
}

// ImageProcessor applies a validated operation pipeline to an image.
type ImageProcessor interface {
	Process(context.Context, []byte, []domain.Operation) ([]byte, domain.Output, error)
}

// OutputWriter persists a generated binary artifact.
type OutputWriter interface {
	Save(context.Context, string, []byte) error
}

// FailPermanently records exhaustion of delivery attempts.
func (processor *Processor) FailPermanently(ctx context.Context, jobID string, cause error) error {
	return processor.repository.FailPermanently(ctx, jobID, "delivery_attempts_exhausted", cause.Error(), processor.now())
}

// Processor claims jobs and executes their processing lifecycle.
type Processor struct {
	repository ProcessingRepository
	fetcher    ImageFetcher
	images     ImageProcessor
	outputs    OutputWriter
	now        func() time.Time
}

// NewProcessor constructs a processor using the production clock.
func NewProcessor(repository ProcessingRepository, fetcher ImageFetcher, images ImageProcessor, outputs OutputWriter) *Processor {
	return &Processor{repository: repository, fetcher: fetcher, images: images, outputs: outputs, now: time.Now}
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

	job, err := processor.repository.FindByID(ctx, jobID)
	if err != nil {
		return processor.fail(ctx, jobID, "load_job_failed", err)
	}
	source, err := processor.fetcher.Fetch(ctx, job.SourceURL)
	if err != nil {
		return processor.fail(ctx, jobID, "source_fetch_failed", err)
	}
	if err := processor.progress(ctx, jobID, 30); err != nil {
		return err
	}

	data, output, err := processor.images.Process(ctx, source, job.Operations)
	if err != nil {
		return processor.fail(ctx, jobID, "image_processing_failed", err)
	}
	if err := processor.progress(ctx, jobID, 80); err != nil {
		return err
	}

	output.Key = jobID + "." + output.Format
	output.URL = "/v1/jobs/" + jobID + "/result"
	if err := processor.outputs.Save(ctx, output.Key, data); err != nil {
		return processor.fail(ctx, jobID, "output_storage_failed", err)
	}
	if err := processor.repository.Complete(ctx, jobID, output, processor.now()); err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

func (processor *Processor) progress(ctx context.Context, jobID string, value int) error {
	if err := processor.repository.UpdateProgress(ctx, jobID, value, processor.now()); err != nil {
		return processor.fail(ctx, jobID, "progress_update_failed", err)
	}
	return nil
}

func (processor *Processor) fail(ctx context.Context, jobID, code string, cause error) error {
	failCtx := ctx
	if ctx.Err() != nil {
		failCtx = context.Background()
	}
	if err := processor.repository.Retry(failCtx, jobID, code, cause.Error(), processor.now()); err != nil {
		return fmt.Errorf("%s: %v; record failure: %w", code, cause, err)
	}
	return fmt.Errorf("%s: %w", code, cause)
}
