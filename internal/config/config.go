// Package config loads application settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds every setting the application reads at startup.
type Config struct {
	Env             string        `env:"APP_ENV" envDefault:"development"`
	HTTPAddr        string        `env:"HTTP_ADDR" envDefault:":8080"`
	JWTSecretKey    string        `env:"JWT_SECRET_KEY" envDefault:"change-me"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	DatabaseURL     string        `env:"DATABASE_URL,required,notEmpty"`
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.Env {
	case "development", "production", "test":
	default:
		return fmt.Errorf("APP_ENV must be development, production or test, got %q", c.Env)
	}
	if c.Env == "production" && len(c.JWTSecretKey) < 32 {
		return errors.New("JWT_SECRET_KEY must be at least 32 characters in production")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("SHUTDOWN_TIMEOUT must be positive")
	}
	return nil
}
