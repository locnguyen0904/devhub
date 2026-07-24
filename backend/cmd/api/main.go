// Command api serves the DevHub HTTP API.
//
// Run with the argument "openapi" to print the OpenAPI document to stdout and
// exit without starting the server or touching Postgres/Redis — that is what
// `make openapi` uses to generate the frontend's TypeScript types.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/locnguyen0904/devhub/backend/internal/config"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/logger"
	"github.com/locnguyen0904/devhub/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.IsProduction())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Neither constructor dials its dependency; both connect lazily. That is
	// what lets the openapi subcommand below run without Postgres or Redis.
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	redis, err := cache.New(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := redis.Close(); cerr != nil {
			log.Error("close redis", "error", cerr)
		}
	}()

	router, api := server.NewAPI(log, db, redis)

	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		spec, merr := api.OpenAPI().YAML()
		if merr != nil {
			return fmt.Errorf("marshal openapi: %w", merr)
		}
		fmt.Println(string(spec))
		return nil
	}

	return server.New(log, router, cfg.Port).Run(ctx, cfg.ShutdownTimeout)
}
