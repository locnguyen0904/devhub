// Package database owns the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx pool so callers depend on this package rather than on pgx
// directly, and so the pool can grow transaction helpers later.
type DB struct {
	Pool *pgxpool.Pool
}

// New builds the pool from a connection URL.
//
// It deliberately does NOT ping: pgxpool connects lazily, and that laziness is
// what lets `go run ./cmd/api openapi` emit the spec without a live database.
// Readiness is checked by the health module at request time instead.
func New(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Close releases every pooled connection.
func (db *DB) Close() {
	db.Pool.Close()
}
