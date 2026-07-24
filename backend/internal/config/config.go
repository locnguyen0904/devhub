// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds every setting the API needs. Fields are added when a caller
// exists — auth settings arrive here in Phase 1 with the auth module.
type Config struct {
	Env             string        `env:"APP_ENV" envDefault:"development"`
	Port            int           `env:"PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	RedisURL        string        `env:"REDIS_URL,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`

	// FrontendURL is where the OAuth callback sends the browser after login.
	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:5173"`

	Auth    AuthConfig
	Storage StorageConfig
}

// StorageConfig points at the S3-compatible bucket for image uploads. Defaults
// target the MinIO service in docker-compose; production overrides them.
type StorageConfig struct {
	Endpoint  string `env:"S3_ENDPOINT" envDefault:"http://localhost:9000"`
	Region    string `env:"S3_REGION" envDefault:"us-east-1"`
	Bucket    string `env:"S3_BUCKET" envDefault:"devhub"`
	AccessKey string `env:"S3_ACCESS_KEY" envDefault:"devhub"`
	SecretKey string `env:"S3_SECRET_KEY" envDefault:"devhub123"`
	// PublicURL is where an uploaded object is served from. For MinIO dev it is
	// the endpoint plus the bucket; production uses a CDN domain.
	PublicURL string `env:"S3_PUBLIC_URL" envDefault:"http://localhost:9000/devhub"`
}

// AuthConfig groups authentication settings.
type AuthConfig struct {
	JWTSecret          string        `env:"JWT_SECRET,required"`
	AccessTTL          time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTTL         time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	GitHubClientID     string        `env:"GITHUB_CLIENT_ID,required"`
	GitHubClientSecret string        `env:"GITHUB_CLIENT_SECRET,required"`
	// GitHubRedirectURL must match the callback registered on the GitHub OAuth
	// app exactly, or GitHub refuses the exchange.
	GitHubRedirectURL string `env:"GITHUB_REDIRECT_URL" envDefault:"http://localhost:8080/api/v1/auth/github/callback"`
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
// currently means JSON logs and Secure cookies.
func (c Config) IsProduction() bool {
	return c.Env == "production"
}
