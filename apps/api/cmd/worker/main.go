package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/httpfetch"
	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/imaging"
	"github.com/nickdavid1811/imageflow/internal/jobs/adapters/localstorage"
	natsadapter "github.com/nickdavid1811/imageflow/internal/jobs/adapters/nats"
	jobpostgres "github.com/nickdavid1811/imageflow/internal/jobs/adapters/postgres"
	jobapplication "github.com/nickdavid1811/imageflow/internal/jobs/application"
	outboxpostgres "github.com/nickdavid1811/imageflow/internal/outbox/adapters/postgres"
	outboxapplication "github.com/nickdavid1811/imageflow/internal/outbox/application"
	"github.com/nickdavid1811/imageflow/internal/platform/config"
	"github.com/nickdavid1811/imageflow/internal/platform/database"
	"github.com/nickdavid1811/imageflow/internal/platform/messaging"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("worker stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}

func run(logger *slog.Logger) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupCtx, startupCancel := context.WithTimeout(ctx, 10*time.Second)
	defer startupCancel()
	pool, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	jetStream, err := messaging.Open(
		cfg.NATSURL, cfg.NATSStream, cfg.NATSSubject, cfg.NATSConsumer,
		cfg.NATSDeadLetterStream, cfg.NATSDeadLetterSubject, logger,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := jetStream.Close(); err != nil {
			logger.Error("close nats failed", "error", err)
		}
	}()
	subscription, err := jetStream.Subscribe()
	if err != nil {
		return err
	}

	jobs := jobpostgres.NewJobRepository(pool)
	outputs, err := localstorage.New(cfg.StoragePath)
	if err != nil {
		return err
	}
	processor := jobapplication.NewProcessor(jobs, httpfetch.New(), imaging.NewProcessor(), outputs)
	consumer := natsadapter.NewConsumer(subscription, processor, jetStream, cfg.NATSMaxDeliveries, logger)
	outbox := outboxpostgres.NewRepository(pool)
	publisher := outboxapplication.NewPublisher(outbox, jetStream, logger)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errorsChannel := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		errorsChannel <- publisher.Run(runCtx)
	}()
	go func() {
		defer group.Done()
		errorsChannel <- consumer.Run(runCtx)
	}()

	logger.Info("worker started", "stream", cfg.NATSStream, "subject", cfg.NATSSubject, "consumer", cfg.NATSConsumer)
	var runErr error
	select {
	case <-ctx.Done():
		cancel()
	case err := <-errorsChannel:
		cancel()
		runErr = err
	}

	done := make(chan struct{})
	go func() { group.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("worker shutdown timed out")
	}
	return runErr
}
