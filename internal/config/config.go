// Package config loads application settings from environment variables.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds every setting the application reads at startup.
type Config struct {
	Env      string `env:"APP_ENV" envDefault:"development"`
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}
