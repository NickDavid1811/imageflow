//go:build integration

package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
	outboxpostgres "github.com/nickdavid1811/imageflow/internal/outbox/adapters/postgres"
	"github.com/nickdavid1811/imageflow/migrations"
)

func TestJobAndOutboxPersistence(t *testing.T) {
	databaseURL := integrationDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := migrations.Run(ctx, databaseURL, "up"); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)

	now := time.Now().UTC().Truncate(time.Microsecond)
	job, err := domain.NewJob(
		"job_integration_"+strings.ReplaceAll(now.Format(time.RFC3339Nano), ":", ""),
		"https://example.com/source.jpg",
		[]domain.Operation{{Type: "resize", Width: 640}},
		now,
	)
	if err != nil {
		t.Fatalf("create domain job: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM jobs WHERE id = $1`, job.ID)
	})

	jobs := NewJobRepository(pool)
	if err := jobs.Create(ctx, job); err != nil {
		t.Fatalf("persist job: %v", err)
	}

	events, err := outboxpostgres.NewRepository(pool).ClaimBatch(ctx, 10, now.Add(time.Second), time.Minute)
	if err != nil {
		t.Fatalf("claim outbox event: %v", err)
	}
	if len(events) != 1 || events[0].AggregateID != job.ID {
		t.Fatalf("claimed events = %#v, want one event for %q", events, job.ID)
	}

	claimed, err := jobs.ClaimForProcessing(ctx, job.ID, now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if !claimed {
		t.Fatal("job was not claimed")
	}
	if err := jobs.UpdateProgress(ctx, job.ID, 60, now.Add(3*time.Second)); err != nil {
		t.Fatalf("update progress: %v", err)
	}
	output := domain.Output{Key: job.ID + ".png", ContentType: "image/png", Format: "png", SizeBytes: 42, Width: 640, Height: 480}
	if err := jobs.Complete(ctx, job.ID, output, now.Add(4*time.Second)); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	stored, err := jobs.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if stored.Status != domain.StatusCompleted || stored.Progress != 100 || stored.Attempts != 1 {
		t.Fatalf("stored job = %#v", stored)
	}
	if stored.Output == nil || stored.Output.Key != output.Key {
		t.Fatalf("stored output = %#v", stored.Output)
	}
}

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("IMAGEFLOW_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("IMAGEFLOW_TEST_DATABASE_URL is not configured")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse IMAGEFLOW_TEST_DATABASE_URL: %v", err)
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatalf("refusing to use database %q: test database name must end in _test", config.ConnConfig.Database)
	}
	return databaseURL
}
