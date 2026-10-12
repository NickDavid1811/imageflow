package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nickdavid1811/imageflow/internal/platform/config"
	"github.com/nickdavid1811/imageflow/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		logger.Error("usage: migrate [up|down]")
		os.Exit(2)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrations.Run(ctx, cfg.DatabaseURL, os.Args[1]); err != nil {
		logger.Error("migration failed", "direction", os.Args[1], "error", err)
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("migration %s completed", os.Args[1]))
}
