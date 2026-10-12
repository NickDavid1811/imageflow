package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func (repository *JobRepository) ClaimForProcessing(ctx context.Context, id string, now time.Time) (bool, error) {
	result, err := repository.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'processing', attempts = attempts + 1,
		    started_at = $2, updated_at = $2,
		    error_code = NULL, error_message = NULL
		WHERE id = $1
		  AND (
		      status = 'queued'
		      OR (status = 'processing' AND updated_at < ($2::timestamptz - interval '15 seconds'))
		  )`, id, now.UTC())
	if err != nil {
		return false, fmt.Errorf("claim job for processing: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (repository *JobRepository) Status(ctx context.Context, id string) (domain.Status, error) {
	var status domain.Status
	err := repository.pool.QueryRow(ctx, `SELECT status FROM jobs WHERE id = $1`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", application.ErrJobNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query job status: %w", err)
	}
	return status, nil
}

func (repository *JobRepository) UpdateProgress(ctx context.Context, id string, progress int, now time.Time) error {
	result, err := repository.pool.Exec(ctx, `
		UPDATE jobs SET progress = $2, updated_at = $3
		WHERE id = $1 AND status = 'processing'`, id, progress, now.UTC())
	if err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("update job progress: %w", application.ErrJobNotFound)
	}
	return nil
}

func (repository *JobRepository) Complete(ctx context.Context, id string, now time.Time) error {
	result, err := repository.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'completed', progress = 100, completed_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'processing'`, id, now.UTC())
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("complete job: %w", application.ErrJobNotFound)
	}
	return nil
}

func (repository *JobRepository) Fail(ctx context.Context, id, code, message string, now time.Time) error {
	_, err := repository.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'failed', error_code = $2, error_message = $3, updated_at = $4
		WHERE id = $1 AND status = 'processing'`, id, code, message, now.UTC())
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

func (repository *JobRepository) FailPermanently(ctx context.Context, id, code, message string, now time.Time) error {
	result, err := repository.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'failed', error_code = $2, error_message = $3, updated_at = $4
		WHERE id = $1 AND status IN ('queued', 'processing')`, id, code, message, now.UTC())
	if err != nil {
		return fmt.Errorf("permanently fail job: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("permanently fail job: %w", application.ErrJobNotFound)
	}
	return nil
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (repository *JobRepository) Create(ctx context.Context, job domain.Job) error {
	operations, err := json.Marshal(job.Operations)
	if err != nil {
		return fmt.Errorf("encode operations: %w", err)
	}
	eventPayload, err := json.Marshal(map[string]string{"job_id": job.ID})
	if err != nil {
		return fmt.Errorf("encode outbox event: %w", err)
	}

	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	_, err = transaction.Exec(ctx, `
		INSERT INTO jobs (
			id, source_url, operations, status, progress, attempts,
			error_code, error_message, created_at, started_at, completed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10, $11, $12)`,
		job.ID, job.SourceURL, operations, job.Status, job.Progress, job.Attempts,
		job.ErrorCode, job.ErrorMessage, job.CreatedAt, job.StartedAt, job.CompletedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	_, err = transaction.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_id, event_type, payload, created_at, available_at)
		VALUES ($1, 'job.requested.v1', $2, $3, $3)`, job.ID, eventPayload, job.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (repository *JobRepository) FindByID(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	var operations []byte
	err := repository.pool.QueryRow(ctx, `
		SELECT id, source_url, operations, status, progress, attempts,
		       COALESCE(error_code, ''), COALESCE(error_message, ''),
		       created_at, started_at, completed_at, updated_at
		FROM jobs
		WHERE id = $1`, id).Scan(
		&job.ID, &job.SourceURL, &operations, &job.Status, &job.Progress, &job.Attempts,
		&job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.StartedAt, &job.CompletedAt, &job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, application.ErrJobNotFound
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("query job: %w", err)
	}
	if err := json.Unmarshal(operations, &job.Operations); err != nil {
		return domain.Job{}, fmt.Errorf("decode operations: %w", err)
	}
	return job, nil
}
