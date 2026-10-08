package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/denis-sofonov/go-task-api/db/migrations"
	"github.com/denis-sofonov/go-task-api/internal/config"
	"github.com/denis-sofonov/go-task-api/internal/platform/logger"
	"github.com/denis-sofonov/go-task-api/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "up", "down", "status":
	default:
		return fmt.Errorf("unknown command %q (want up, down or status)", command)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	switch command {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}
		for _, r := range results {
			log.Info("applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration.String())
		}
		if len(results) == 0 {
			log.Info("no pending migrations")
		}
	case "down":
		r, err := provider.Down(ctx)
		if err != nil {
			if errors.Is(err, goose.ErrNoNextVersion) {
				log.Info("nothing to roll back")
				return nil
			}
			return fmt.Errorf("migrate down: %w", err)
		}
		log.Info("rolled back", "version", r.Source.Version, "file", r.Source.Path)
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("migrate status: %w", err)
		}
		for _, s := range statuses {
			log.Info("migration", "version", s.Source.Version, "file", s.Source.Path, "state", string(s.State))
		}
	}
	return nil
}
