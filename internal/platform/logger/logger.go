// Package logger builds the application's structured logger.
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON logger. Development logs at debug level, everything else at info.
func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
