// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds every setting the API needs. Fields are added when a caller
// exists — auth and storage settings arrive in the phases that use them.
type Config struct {
	Env             string        `env:"APP_ENV" envDefault:"development"`
	Port            int           `env:"PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	RedisURL        string        `env:"REDIS_URL,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

// Load reads the environment into a Config. A missing required variable is a
// startup failure, not a nil dereference later at request time.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	return cfg, nil
}

// IsProduction reports whether the API runs with production behaviour, which
// currently means plain-text logs are disabled in favour of JSON.
func (c Config) IsProduction() bool {
	return c.Env == "production"
}
