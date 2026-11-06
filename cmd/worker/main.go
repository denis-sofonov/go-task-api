package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/denis-sofonov/go-task-api/internal/config"
	"github.com/denis-sofonov/go-task-api/internal/platform/kafka"
	"github.com/denis-sofonov/go-task-api/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "task-events", "task-worker")
	defer consumer.Close()

	log.Info("worker started")
	err = consumer.Run(ctx, func(ctx context.Context, value []byte) error {
		log.Info("event received", "value", string(value))
		return nil
	})
	if err != nil {
		return fmt.Errorf("run consumer: %w", err)
	}

	log.Info("worker stopped")
	return nil
}
